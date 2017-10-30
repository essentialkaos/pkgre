package morpher

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"pkg.re/essentialkaos/ek.v9/knf"
	"pkg.re/essentialkaos/ek.v9/log"
	"pkg.re/essentialkaos/ek.v9/sortutil"
	"pkg.re/essentialkaos/ek.v9/version"

	"github.com/essentialkaos/pkgre/refs"
	"github.com/essentialkaos/pkgre/repo"

	"github.com/valyala/fasthttp"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	HTTP_IP       = "http:ip"
	HTTP_PORT     = "http:port"
	HTTP_REDIRECT = "http:redirect"
)

const USER_AGENT = "PkgRE-Morpher/3.4"

// ////////////////////////////////////////////////////////////////////////////////// //

// PkgInfo is struct with package info
type PkgInfo struct {
	RepoInfo   *repo.Info
	RefsInfo   *refs.Info
	Path       string
	TargetType refs.RefType
	TargetName string
}

// ////////////////////////////////////////////////////////////////////////////////// //

// majorVerRegExp regexp for extracting major version
var majorVerRegExp = regexp.MustCompile(`^[a-zA-Z]{0,}([0-9]{1}.*)`)

// goGetTemplate is template used for go get command response
var goGetTemplate = template.Must(template.New("").Parse(`<html>
  <head>
    <meta name="go-import" content="pkg.re/{{.RepoInfo.Root}} git https://pkg.re/{{.RepoInfo.Root}}" />
    {{$root := .RepoInfo.GitHubRoot}}{{$tree := .TargetName}}<meta name="go-source" content="pkg.re/{{.RepoInfo.Root}} _ https://{{$root}}/tree/{{$tree}}{/dir} https://{{$root}}/blob/{{$tree}}{/dir}/{file}#L{line}" />
  </head>
  <body>
    go get pkg.re/{{.RepoInfo.FullPath}}
  </body>
</html>
`))

// client is default client for all http requests
var client *fasthttp.Client

// metrics
var (
	counterHits      uint64
	counterMisses    uint64
	counterErrors    uint64
	counterRedirects uint64
	counterDocs      uint64
	counterGoget     uint64
)

// client for proxying requests to GitHub.com
var proxyClient *fasthttp.Client

// ////////////////////////////////////////////////////////////////////////////////// //

// Start start HTTP server
func Start() error {
	initHTTPClients()

	log.Info("Morpher HTTP server will be started on %s:%s", knf.GetS(HTTP_IP), knf.GetS(HTTP_PORT))

	return fasthttp.ListenAndServe(knf.GetS(HTTP_IP)+":"+knf.GetS(HTTP_PORT), requestHandler)
}

// ////////////////////////////////////////////////////////////////////////////////// //

func initHTTPClients() {
	client = &fasthttp.Client{
		Name:                USER_AGENT,
		MaxIdleConnDuration: 5 * time.Second,
		ReadTimeout:         3 * time.Second,
		WriteTimeout:        3 * time.Second,
		MaxConnsPerHost:     150,
	}

	proxyClient = &fasthttp.Client{
		Name:                USER_AGENT,
		MaxIdleConnDuration: 10 * time.Second,
		ReadTimeout:         15 * time.Second,
		WriteTimeout:        15 * time.Second,
		MaxConnsPerHost:     50,
	}
}

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

	// Redirect to documentation
	if strings.Contains(path, "#") {
		processDocsRequest(ctx, start, path)
		return
	}

	repoInfo, err := repo.ParsePath(path)

	if err != nil {
		atomic.AddUint64(&counterErrors, 1)
		log.Warn("Can't parse repo path: %v", err)
		appendProcHeader(ctx, start)
		notFoundResponse(ctx, err.Error())
		return
	}

	// Return rewritten pack
	if repoInfo.Path == "git-upload-pack" {
		processUploadPackRequest(ctx, start, repoInfo)
		return
	}

	refsInfo, err := fetchRefs(repoInfo)

	if err != nil {
		atomic.AddUint64(&counterErrors, 1)
		log.Warn("Can't process refs data for %s: %v", repoInfo.GitHubRoot(), err)
		appendProcHeader(ctx, start)
		notFoundResponse(ctx, err.Error())
		return
	}

	targetType, targetName := suggestHead(repoInfo, refsInfo)
	pkgInfo := &PkgInfo{repoInfo, refsInfo, path, targetType, targetName}

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

	atomic.AddUint64(&counterRedirects, 1)

	// Redirect to github
	appendProcHeader(ctx, start)

	// Proxying allowed only for GoDoc bot
	if strings.HasPrefix(string(ctx.UserAgent()), "GoDocBot") {
		proxyRequest(ctx, repoInfo.GitHubURL(pkgInfo.TargetName))
	} else {
		redirectRequest(ctx, repoInfo.GitHubURL(pkgInfo.TargetName))
	}
}

// processBasicRequest redirect requests from main page to page defined in config
func processBasicRequest(ctx *fasthttp.RequestCtx, start time.Time) {
	appendProcHeader(ctx, start)
	redirectRequest(ctx, knf.GetS(HTTP_REDIRECT))
}

// processMetricsRequest return metrics
func processMetricsRequest(ctx *fasthttp.RequestCtx, start time.Time) {
	appendProcHeader(ctx, start)

	metrics := "{\n"
	metrics += "  \"hits\": " + strconv.FormatUint(atomic.LoadUint64(&counterHits), 10) + ",\n"
	metrics += "  \"misses\": " + strconv.FormatUint(atomic.LoadUint64(&counterMisses), 10) + ",\n"
	metrics += "  \"errors\": " + strconv.FormatUint(atomic.LoadUint64(&counterErrors), 10) + ",\n"
	metrics += "  \"redirects\": " + strconv.FormatUint(atomic.LoadUint64(&counterRedirects), 10) + ",\n"
	metrics += "  \"docs\": " + strconv.FormatUint(atomic.LoadUint64(&counterDocs), 10) + ",\n"
	metrics += "  \"goget\": " + strconv.FormatUint(atomic.LoadUint64(&counterGoget), 10) + "\n"
	metrics += "}\n"

	ctx.WriteString(metrics)
}

// processDocsRequest redirect request to godoc.org
func processDocsRequest(ctx *fasthttp.RequestCtx, start time.Time, path string) {
	atomic.AddUint64(&counterDocs, 1)
	appendProcHeader(ctx, start)
	redirectRequest(ctx, "https://godoc.org/pkg.re"+path)
}

// processUploadPackRequest redirect git-upload-pack request to GitHub
func processUploadPackRequest(ctx *fasthttp.RequestCtx, start time.Time, repoInfo *repo.Info) {
	appendProcHeader(ctx, start)
	redirectRequest(ctx, "https://"+repoInfo.GitHubRoot()+"/git-upload-pack")
}

// processRefsRequest process request for refs
func processRefsRequest(ctx *fasthttp.RequestCtx, start time.Time, pkgInfo *PkgInfo) {
	if pkgInfo.TargetName != "" {
		switch pkgInfo.TargetType {
		case refs.TYPE_TAG:
			atomic.AddUint64(&counterHits, 1)
			log.Debug(
				"%s -> T:%s (%s)", pkgInfo.Path, pkgInfo.TargetName,
				pkgInfo.RefsInfo.GetTagSHA(pkgInfo.TargetName, true),
			)
		case refs.TYPE_BRANCH:
			atomic.AddUint64(&counterHits, 1)
			log.Debug(
				"%s -> B:%s (%s)", pkgInfo.Path, pkgInfo.TargetName,
				pkgInfo.RefsInfo.GetBranchSHA(pkgInfo.TargetName, true),
			)
		default:
			atomic.AddUint64(&counterMisses, 1)
			log.Warn("%s -> master (proper tag/branch not found)", pkgInfo.Path)
		}
	} else {
		atomic.AddUint64(&counterMisses, 1)
		log.Info("%s -> master (no target version)", pkgInfo.Path)
	}

	appendProcHeader(ctx, start)
	ctx.Response.Header.Set("Content-Type", "application/x-git-upload-pack-advertisement")
	ctx.Write(pkgInfo.RefsInfo.Rewrite(pkgInfo.TargetName, pkgInfo.TargetType))
}

// processGoGetRequest process "go get" requests
func processGoGetRequest(ctx *fasthttp.RequestCtx, start time.Time, pkgInfo *PkgInfo) {
	appendProcHeader(ctx, start)

	if pkgInfo.TargetType == refs.TYPE_UNKNOWN {
		atomic.AddUint64(&counterMisses, 1)
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
		atomic.AddUint64(&counterErrors, 1)
		log.Error("Can't render go get template: %v", err)
	}

	atomic.AddUint64(&counterGoget, 1)
}

// notFoundResponse write 404 response
func notFoundResponse(ctx *fasthttp.RequestCtx, data string) {
	ctx.SetStatusCode(http.StatusNotFound)
	ctx.WriteString(data + "\n")
}

// appendProcHeader append header with processing time
func appendProcHeader(ctx *fasthttp.RequestCtx, start time.Time) {
	ctx.Response.Header.Set("Server", "PKGRE Morpher")
	ctx.Response.Header.Add("X-Morpher-Time", fmt.Sprintf("%s", time.Since(start)))
}

// redirectRequest add redirect header to repsponse
func redirectRequest(ctx *fasthttp.RequestCtx, url string) {
	ctx.Response.Header.Set("Location", url)
	ctx.SetStatusCode(http.StatusTemporaryRedirect)
}

// proxyRequest proxy request to GitHub
func proxyRequest(ctx *fasthttp.RequestCtx, url string) {
	ctx.Request.Header.Del("Connection")
	ctx.Request.SetRequestURI(url)

	err := proxyClient.Do(&ctx.Request, &ctx.Response)

	if err != nil {
		log.Error("Can't proxy request to %s", url)
	}

	ctx.Response.Header.Del("Connection")
}

// requestRecover recover panic in request
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
		return nil, fmt.Errorf("GitHub return status code <%s>", statusCode)
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

// suggestHead return best fit head
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

// getCleanVer return only version digits without any prefix (v/r/ver/version/etc...)
func getCleanVer(v string) string {
	vf := majorVerRegExp.FindStringSubmatch(v)

	if len(vf) == 0 {
		return ""
	}

	return vf[1]
}
