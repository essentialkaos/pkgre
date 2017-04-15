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

	"pkg.re/essentialkaos/ek.v8/knf"
	"pkg.re/essentialkaos/ek.v8/log"
	"pkg.re/essentialkaos/ek.v8/sortutil"
	"pkg.re/essentialkaos/ek.v8/version"

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

const USER_AGENT = "PkgRE-Morpher/3.3"

// ////////////////////////////////////////////////////////////////////////////////// //

// PkgInfo is struct with package info
type PkgInfo struct {
	RepoInfo *repo.Info
	Target   string
}

// ////////////////////////////////////////////////////////////////////////////////// //

// majorVerRegExp regexp for extracting major version
var majorVerRegExp = regexp.MustCompile(`^[a-zA-Z]{0,}([0-9]{1}.*)`)

// goGetTemplate is template used for go get command response
var goGetTemplate = template.Must(template.New("").Parse(`<html>
  <head>
    <meta name="go-import" content="pkg.re/{{.RepoInfo.Root}} git https://pkg.re/{{.RepoInfo.Root}}" />
    {{$root := .RepoInfo.GitHubRoot}}{{$tree := .Target}}<meta name="go-source" content="pkg.re/{{.RepoInfo.Root}} _ https://{{$root}}/tree/{{$tree}}{/dir} https://{{$root}}/blob/{{$tree}}{/dir}/{file}#L{line}" />
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

// ////////////////////////////////////////////////////////////////////////////////// //

// Start start HTTP server
func Start() error {
	initHTTPClient()

	log.Info("Morpher HTTP server will be started on %s:%s", knf.GetS(HTTP_IP), knf.GetS(HTTP_PORT))

	return fasthttp.ListenAndServe(knf.GetS(HTTP_IP)+":"+knf.GetS(HTTP_PORT), requestHandler)
}

// ////////////////////////////////////////////////////////////////////////////////// //

func initHTTPClient() {
	client = &fasthttp.Client{
		Name:                USER_AGENT,
		MaxIdleConnDuration: time.Second,
		ReadTimeout:         time.Second,
		WriteTimeout:        time.Second,
		MaxConnsPerHost:     150,
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	start := time.Now()

	defer requestRecover(ctx, start)

	path := string(ctx.Path())

	if path == "/" {
		appendProcHeader(ctx, start)
		redirectRequest(ctx, knf.GetS(HTTP_REDIRECT))
		return
	}

	if path == "/_metrics" {
		appendProcHeader(ctx, start)
		encodeMetrics(ctx)
		return
	}

	// Redirect to documentation
	if strings.Contains(path, "#") {
		atomic.AddUint64(&counterDocs, 1)
		appendProcHeader(ctx, start)
		redirectRequest(ctx, "https://godoc.org/pkg.re"+path)
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

	if repoInfo.Path == "git-upload-pack" {
		appendProcHeader(ctx, start)
		redirectRequest(ctx, "https://"+repoInfo.GitHubRoot()+"/git-upload-pack")
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

	t, n := suggestHead(repoInfo, refsInfo)

	// Rewrite refs
	if repoInfo.Path == "info/refs" {
		if n != "" {
			switch t {
			case refs.TYPE_TAG:
				atomic.AddUint64(&counterHits, 1)
				log.Debug("%s -> T:%s (%s)", path, n, refsInfo.GetTagSHA(n, true))
			case refs.TYPE_BRANCH:
				atomic.AddUint64(&counterHits, 1)
				log.Debug("%s -> B:%s (%s)", path, n, refsInfo.GetBranchSHA(n, true))
			default:
				atomic.AddUint64(&counterMisses, 1)
				log.Warn("%s -> master (proper tag/branch not found)", path)
			}
		} else {
			atomic.AddUint64(&counterMisses, 1)
			log.Info("%s -> master (no target version)", path)
		}

		appendProcHeader(ctx, start)
		ctx.Response.Header.Set("Content-Type", "application/x-git-upload-pack-advertisement")
		ctx.Write(refsInfo.Rewrite(n, t))

		return
	}

	pkgInfo := &PkgInfo{repoInfo, n}

	if len(ctx.FormValue("go-get")) != 0 {
		appendProcHeader(ctx, start)

		ctx.Response.Header.Add("Content-Type", "text/html")

		err := goGetTemplate.Execute(ctx, pkgInfo)

		if err != nil {
			atomic.AddUint64(&counterErrors, 1)
			log.Error("Can't render go get template: %v", err)
		}

		atomic.AddUint64(&counterGoget, 1)

		return
	}

	atomic.AddUint64(&counterRedirects, 1)

	// Redirect to github
	appendProcHeader(ctx, start)
	redirectRequest(ctx, repoInfo.GitHubURL(n))
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

// requestRecover recover panic in request
func requestRecover(ctx *fasthttp.RequestCtx, start time.Time) {
	r := recover()

	if r != nil {
		log.Error("Recovered internal error: %v", r)
		appendProcHeader(ctx, start)
		ctx.SetStatusCode(http.StatusInternalServerError)
	}
}

// encodeMetrics encode metrics to JSON
func encodeMetrics(ctx *fasthttp.RequestCtx) {
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
