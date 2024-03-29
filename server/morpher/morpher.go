package morpher

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2021 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"pkg.re/essentialkaos/ek.v12/knf"
	"pkg.re/essentialkaos/ek.v12/log"
	"pkg.re/essentialkaos/ek.v12/sortutil"
	"pkg.re/essentialkaos/ek.v12/version"

	"github.com/essentialkaos/pkgre/refs"
	"github.com/essentialkaos/pkgre/repo"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	MAIN_DOMAIN    = "main:domain"
	HTTP_IP        = "http:ip"
	HTTP_PORT      = "http:port"
	HTTP_REDIRECT  = "http:redirect"
	HTTP_REUSEPORT = "http:reuserport"
)

const USER_AGENT = "PkgRE-Morpher"

const DOC_QUERY_ARG = "docs"

// ////////////////////////////////////////////////////////////////////////////////// //

// PkgInfo is struct with package info
type PkgInfo struct {
	Path       string
	TargetName string
	Domain     string
	RepoInfo   *repo.Info
	RefsInfo   *refs.Info
	TargetType refs.RefType
}

// Metrics is struct with metrics data
type Metrics struct {
	Hits      uint64
	Misses    uint64
	Errors    uint64
	Redirects uint64
	Docs      uint64
	Goget     uint64
}

// ////////////////////////////////////////////////////////////////////////////////// //

var (
	UAGit = []byte("git/")            // Git User-Agent
	UAGo  = []byte("Go-http-client/") // Go User-Agent
)

// ////////////////////////////////////////////////////////////////////////////////// //

// majorVerRegExp regexp for extracting major version
var majorVerRegExp = regexp.MustCompile(`^[a-zA-Z]{0,}([0-9]{1}.*)`)

// goGetTemplate is template used for go get command response
var goGetTemplate = template.Must(template.New("").Parse(`<html>
  <head>
    <meta name="go-import" content="{{.Domain}}/{{.RepoInfo.Root}} git https://{{.Domain}}/{{.RepoInfo.Root}}" />
    {{$root := .RepoInfo.GitHubRoot}}{{$tree := .TargetName}}<meta name="go-source" content="{{.Domain}}/{{.RepoInfo.Root}} _ https://{{$root}}/tree/{{$tree}}{/dir} https://{{$root}}/blob/{{$tree}}{/dir}/{file}#L{line}" />
  </head>
  <body>
    go get {{.Domain}}/{{.RepoInfo.FullPath}}
  </body>
</html>
`))

// server is main HTTP server
var server *fasthttp.Server

// client is default client for all http requests
var client *fasthttp.Client

// client for proxying requests to GitHub.com
var proxyClient *fasthttp.Client

// daemonVersion is current morpher version
var daemonVersion string

// domain is main service domain
var domain string

// metrics contains morpher metrics
var metrics = &Metrics{}

// ////////////////////////////////////////////////////////////////////////////////// //

// Start starts HTTP server
func Start(version string) error {
	daemonVersion = version
	domain = knf.GetS(MAIN_DOMAIN)

	initHTTPClients()

	addr := knf.GetS(HTTP_IP) + ":" + knf.GetS(HTTP_PORT)

	log.Info("Morpher HTTP server will be started on %s", addr)

	server = &fasthttp.Server{
		Name:    USER_AGENT + "/" + daemonVersion,
		Handler: requestHandler,
	}

	var err error
	var ln net.Listener

	if knf.GetB(HTTP_REUSEPORT, false) {
		ln, err = net.Listen("tcp4", addr)
	} else {
		ln, err = reuseport.Listen("tcp4", addr)
	}

	if err != nil {
		return fmt.Errorf("Can't create listener on %s: %v", addr, err)
	}

	return server.Serve(ln)
}

// Stop stops HTTP server
func Stop() error {
	if server == nil {
		return nil
	}

	return server.Shutdown()
}

// ////////////////////////////////////////////////////////////////////////////////// //

// initHTTPClients initializes basic clients
func initHTTPClients() {
	client = &fasthttp.Client{
		Name:                USER_AGENT + "/" + daemonVersion,
		MaxIdleConnDuration: 5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		MaxConnsPerHost:     150,
	}

	proxyClient = &fasthttp.Client{
		Name:                USER_AGENT + "/" + daemonVersion,
		MaxIdleConnDuration: 10 * time.Second,
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        15 * time.Second,
		MaxConnsPerHost:     50,
	}
}

// requestHandler is a main request handler
func requestHandler(ctx *fasthttp.RequestCtx) {
	start := time.Now()

	defer requestRecover(ctx, start)

	path := string(ctx.Path())

	if path == "/" {
		processBasicRequest(ctx, start)
		return
	}

	// Return metrics
	if path == "/_metrics" {
		processMetricsRequest(ctx, start)
		return
	}

	repoInfo, err := repo.ParsePath(path)

	if err != nil {
		atomic.AddUint64(&metrics.Errors, 1)
		log.Warn("Can't parse repository path (%s): %v", path, err)
		appendProcHeader(ctx, start)
		notFoundResponse(ctx, err.Error())
		return
	}

	err = repoInfo.Validate()

	if err != nil {
		atomic.AddUint64(&metrics.Errors, 1)
		log.Warn("Repository path validation error (%s): %v", path, err)
		appendProcHeader(ctx, start)
		notFoundResponse(ctx, err.Error())
		return
	}

	if repoInfo.Target == "" {
		ghURL := repoInfo.GitHubURL("")
		atomic.AddUint64(&metrics.Redirects, 1)
		log.Debug("Redirecting request to %s", ghURL)
		redirectRequest(ctx, ghURL)
		return
	}

	// Return rewritten pack
	if repoInfo.Path == "git-upload-pack" {
		processUploadPackRequest(ctx, start, repoInfo)
		return
	}

	refsInfo, err := fetchRefs(repoInfo)

	if err != nil {
		atomic.AddUint64(&metrics.Errors, 1)
		log.Warn("Can't process refs data for %s: %v", repoInfo.GitHubRoot(), err)
		appendProcHeader(ctx, start)
		notFoundResponse(ctx, err.Error())
		return
	}

	targetType, targetName := suggestHead(repoInfo, refsInfo)
	pkgInfo := &PkgInfo{
		RepoInfo: repoInfo, RefsInfo: refsInfo,
		TargetType: targetType, TargetName: targetName,
		Path: path, Domain: domain,
	}

	// Rewrite refs
	if repoInfo.Path == "info/refs" {
		processRefsRequest(ctx, start, pkgInfo)
		return
	}

	// Return info for "go get" request
	if len(ctx.FormValue("go-get")) != 0 {
		processGoGetRequest(ctx, start, pkgInfo)
		return
	}

	// Redirect to pkg.go.dev
	if ctx.QueryArgs().Has(DOC_QUERY_ARG) {
		processDocsRequest(ctx, start, path, pkgInfo)
		return
	}

	appendProcHeader(ctx, start)

	ghURL := repoInfo.GitHubURL(pkgInfo.TargetName)

	// Proxy only requests from Go and Git
	if bytes.HasPrefix(ctx.UserAgent(), UAGit) || bytes.HasPrefix(ctx.UserAgent(), UAGo) {
		log.Debug("Proxying request to %s", ghURL)
		proxyRequest(ctx, ghURL)
	} else {
		atomic.AddUint64(&metrics.Redirects, 1)
		log.Debug("Redirecting request to %s", ghURL)
		redirectRequest(ctx, ghURL)
	}
}

// processBasicRequest redirect requests from main page to page defined in config
func processBasicRequest(ctx *fasthttp.RequestCtx, start time.Time) {
	appendProcHeader(ctx, start)
	redirectRequest(ctx, knf.GetS(HTTP_REDIRECT))
}

// processMetricsRequest writes metrics response
func processMetricsRequest(ctx *fasthttp.RequestCtx, start time.Time) {
	appendProcHeader(ctx, start)

	ctx.WriteString("{\n")
	ctx.WriteString("  \"hits\": " + strconv.FormatUint(atomic.LoadUint64(&metrics.Hits), 10) + ",\n")
	ctx.WriteString("  \"misses\": " + strconv.FormatUint(atomic.LoadUint64(&metrics.Misses), 10) + ",\n")
	ctx.WriteString("  \"errors\": " + strconv.FormatUint(atomic.LoadUint64(&metrics.Errors), 10) + ",\n")
	ctx.WriteString("  \"redirects\": " + strconv.FormatUint(atomic.LoadUint64(&metrics.Redirects), 10) + ",\n")
	ctx.WriteString("  \"docs\": " + strconv.FormatUint(atomic.LoadUint64(&metrics.Docs), 10) + ",\n")
	ctx.WriteString("  \"goget\": " + strconv.FormatUint(atomic.LoadUint64(&metrics.Goget), 10) + "\n")
	ctx.WriteString("}\n")
}

// processDocsRequest redirects request to godoc.org
func processDocsRequest(ctx *fasthttp.RequestCtx, start time.Time, path string, pkgInfo *PkgInfo) {
	atomic.AddUint64(&metrics.Docs, 1)
	appendProcHeader(ctx, start)
	redirectRequest(ctx, genGoDevURL(path, pkgInfo.TargetName))
}

// processUploadPackRequest redirects git-upload-pack request to GitHub
func processUploadPackRequest(ctx *fasthttp.RequestCtx, start time.Time, repoInfo *repo.Info) {
	appendProcHeader(ctx, start)

	url := "https://" + repoInfo.GitHubRoot() + "/git-upload-pack"

	log.Debug("Proxying git-upload-pack request to %s", url)
	proxyRequest(ctx, url)
}

// processRefsRequest processes request for refs
func processRefsRequest(ctx *fasthttp.RequestCtx, start time.Time, pkgInfo *PkgInfo) {
	if pkgInfo.TargetName != "" {
		switch pkgInfo.TargetType {
		case refs.TYPE_TAG:
			atomic.AddUint64(&metrics.Hits, 1)
			log.Debug(
				"%s -> T:%s (%s)", pkgInfo.Path, pkgInfo.TargetName,
				pkgInfo.RefsInfo.GetTagSHA(pkgInfo.TargetName, true),
			)
		case refs.TYPE_BRANCH:
			atomic.AddUint64(&metrics.Hits, 1)
			log.Debug(
				"%s -> B:%s (%s)", pkgInfo.Path, pkgInfo.TargetName,
				pkgInfo.RefsInfo.GetBranchSHA(pkgInfo.TargetName, true),
			)
		default:
			atomic.AddUint64(&metrics.Misses, 1)
			log.Warn("%s -> master (proper tag/branch not found)", pkgInfo.Path)
		}
	} else {
		atomic.AddUint64(&metrics.Misses, 1)
		log.Info("%s -> master (no target version)", pkgInfo.Path)
	}

	appendProcHeader(ctx, start)
	ctx.Response.Header.Set("Content-Type", "application/x-git-upload-pack-advertisement")
	ctx.Write(pkgInfo.RefsInfo.Rewrite(pkgInfo.TargetName, pkgInfo.TargetType))
}

// processGoGetRequest processes "go get" requests
func processGoGetRequest(ctx *fasthttp.RequestCtx, start time.Time, pkgInfo *PkgInfo) {
	appendProcHeader(ctx, start)

	if pkgInfo.TargetType == refs.TYPE_UNKNOWN {
		atomic.AddUint64(&metrics.Misses, 1)
		ctx.Response.Header.Add("Content-Type", "text/plain; charset=utf-8")
		ctx.SetStatusCode(http.StatusNotFound)
		ctx.WriteString(fmt.Sprintf(
			"GitHub repository at https://%s has no proper branch or tag",
			pkgInfo.RepoInfo.GitHubRoot()),
		)
		return
	}

	ctx.Response.Header.Add("Content-Type", "text/html; charset=utf-8")

	err := goGetTemplate.Execute(ctx, pkgInfo)

	if err != nil {
		atomic.AddUint64(&metrics.Errors, 1)
		log.Error("Can't render go get template: %v", err)
	}

	atomic.AddUint64(&metrics.Goget, 1)
}

// notFoundResponse writes 404 response
func notFoundResponse(ctx *fasthttp.RequestCtx, data string) {
	ctx.SetStatusCode(http.StatusNotFound)
	ctx.WriteString(data + "\n")
}

// appendProcHeader appends header with processing time
func appendProcHeader(ctx *fasthttp.RequestCtx, start time.Time) {
	ctx.Response.Header.Set("Server", "PKGRE Morpher")
	ctx.Response.Header.Add("X-Morpher-Time", fmt.Sprintf("%s", time.Since(start)))
}

// redirectRequest appends redirect header to response
func redirectRequest(ctx *fasthttp.RequestCtx, url string) {
	ctx.Response.Header.Set("Location", url)
	ctx.SetStatusCode(http.StatusTemporaryRedirect)
}

// proxyRequest proxies request to GitHub
func proxyRequest(ctx *fasthttp.RequestCtx, url string) {
	ctx.Request.Header.Del("Connection")
	ctx.Request.SetRequestURI(url)

	err := proxyClient.Do(&ctx.Request, &ctx.Response)

	if err != nil {
		log.Error("Can't proxy request to %s", url)
	}

	ctx.Response.Header.Del("Connection")
}

// requestRecover recovers panic in request
func requestRecover(ctx *fasthttp.RequestCtx, start time.Time) {
	r := recover()

	if r != nil {
		log.Error("Recovered internal error: %v", r)
		appendProcHeader(ctx, start)
		ctx.SetStatusCode(http.StatusInternalServerError)
	}
}

// fetchRefs downloads and parse refs info from github
func fetchRefs(repo *repo.Info) (*refs.Info, error) {
	var refsData []byte

	statusCode, refsData, err := client.Get(
		nil, "https://"+repo.GitHubRoot()+".git/info/refs?service=git-upload-pack",
	)

	if statusCode != 200 {
		return nil, fmt.Errorf("GitHub return status code <%d>", statusCode)
	}

	if len(refsData) == 0 {
		return nil, errors.New("GitHub return empty response")
	}

	refs, err := refs.Parse(refsData)

	if err != nil {
		return nil, fmt.Errorf("Can't parse refs data: %v", err)
	}

	return refs, nil
}

// suggestHead returns best fit head
func suggestHead(repoInfo *repo.Info, refsInfo *refs.Info) (refs.RefType, string) {
	// If target is empty we do not change refs head
	if repoInfo.Target == "" {
		return refs.TYPE_BRANCH, ""
	}

	// Try to parse target as version
	targetVersion, err := version.Parse(getCleanVer(repoInfo.Target))

	// Can't parse version
	if err != nil {
		// Try to find branch with given name
		if refsInfo.HasBranch(repoInfo.Target) {
			return refs.TYPE_BRANCH, repoInfo.Target
		}
	} else {
		if targetVersion.PreRelease() != "" && refsInfo.HasBranch(repoInfo.Target) {
			return refs.TYPE_BRANCH, repoInfo.Target
		}
	}

	tags := refsInfo.TagList()

	sortutil.Versions(tags)

	var fitVerson string

	// Try to find best fit tag
	for _, t := range tags {
		tagVer, err := version.Parse(getCleanVer(t))

		if err != nil {
			continue
		}

		// Find latest version
		if targetVersion.Contains(tagVer) {
			fitVerson = t
		}
	}

	if fitVerson != "" {
		return refs.TYPE_TAG, fitVerson
	}

	// Tag exact search
	if refsInfo.HasTag(repoInfo.Target) {
		return refs.TYPE_TAG, repoInfo.Target
	}

	// Branch exact search
	if refsInfo.HasBranch(repoInfo.Target) {
		return refs.TYPE_BRANCH, repoInfo.Target
	}

	return refs.TYPE_UNKNOWN, ""
}

// getCleanVer returns version digits without any prefix (v/r/ver/version/etc...)
func getCleanVer(v string) string {
	vf := majorVerRegExp.FindStringSubmatch(v)

	if len(vf) == 0 {
		return ""
	}

	return vf[1]
}

// getRealIP return remote IP
func getRealIP(ctx *fasthttp.RequestCtx) string {
	xRealIP := string(ctx.Request.Header.Peek("X-Real-IP"))

	if xRealIP != "" {
		return xRealIP
	}

	return ctx.RemoteIP().String()
}

// genGoDevURL returns URL of pkg.go.dev page with package documentation
func genGoDevURL(path, branchOrTag string) string {
	url := "https://pkg.go.dev/" + domain + "/" + path + "@" + branchOrTag

	if !strings.HasPrefix(branchOrTag, "v1.") && !strings.HasPrefix(branchOrTag, "v0.") {
		url += "+incompatible"
	}

	return url
}
