package morpher

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2015 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"

	"pkg.re/essentialkaos/ek.v2/knf"
	"pkg.re/essentialkaos/ek.v2/log"
	"pkg.re/essentialkaos/ek.v2/req"
	"pkg.re/essentialkaos/ek.v2/sortutil"
	"pkg.re/essentialkaos/ek.v2/version"

	"pkg.re/essentialkaos/librato.v2"

	"github.com/essentialkaos/pkgre/refs"
	"github.com/essentialkaos/pkgre/repo"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	HTTP_IP              = "http:ip"
	HTTP_PORT            = "http:port"
	HTTP_READ_TIMEOUT    = "http:read-timeout"
	HTTP_WRITE_TIMEOUT   = "http:write-timeout"
	HTTP_MAX_HEADER_SIZE = "http:max-header-size"
	HTTP_REDIRECT        = "http:redirect"
	LIBRATO_ENABLED      = "librato:enabled"
	LIBRATO_PREFIX       = "librato:prefix"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// PkgInfo is struct with package info
type PkgInfo struct {
	RepoInfo *repo.Info
	Target   string
}

// Stats is struct with some statistics data
type Stats struct {
	Hits      int
	Misses    int
	Errors    int
	Redirects int
	Docs      int
	Goget     int
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

// stats is struct with statistics data
var stats = &Stats{0, 0, 0, 0, 0, 0}

// ////////////////////////////////////////////////////////////////////////////////// //

// Start start HTTP server
func Start() error {
	server := &http.Server{
		Addr:           knf.GetS(HTTP_IP) + ":" + knf.GetS(HTTP_PORT),
		Handler:        http.NewServeMux(),
		ReadTimeout:    time.Duration(knf.GetI(HTTP_READ_TIMEOUT)) * time.Second,
		WriteTimeout:   time.Duration(knf.GetI(HTTP_WRITE_TIMEOUT)) * time.Second,
		MaxHeaderBytes: knf.GetI(HTTP_MAX_HEADER_SIZE),
	}

	server.Handler.(*http.ServeMux).HandleFunc("/", requestHandler)

	log.Info("Morpher HTTP server started on %s:%s", knf.GetS(HTTP_IP), knf.GetS(HTTP_PORT))

	if knf.GetB(LIBRATO_ENABLED, false) {
		librato.NewCollector(time.Minute, collectStats).ErrorHandler = collectErrorHandler
	}

	err := server.ListenAndServe()

	return err
}

// ////////////////////////////////////////////////////////////////////////////////// //

// requestHandler derfault request handler
func requestHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	path := r.URL.Path

	if path == "/" {
		appendProcHeader(w, start)
		redirectRequest(w, knf.GetS(HTTP_REDIRECT))
		return
	}

	// Redirect to documentation
	if strings.Contains(path, "#") {
		stats.Docs++
		appendProcHeader(w, start)
		redirectRequest(w, "https://godoc.org/pkg.re"+path)
		return
	}

	repoInfo, err := repo.ParsePath(path)

	if err != nil {
		stats.Errors++
		log.Warn("Can't parse repo path: %v", err)
		appendProcHeader(w, start)
		notFoundResponse(w, err.Error())
		return
	}

	if repoInfo.Path == "git-upload-pack" {
		appendProcHeader(w, start)
		redirectRequest(w, "https://"+repoInfo.GitHubRoot()+"/git-upload-pack")
		return
	}

	refsInfo, err := fetchRefs(repoInfo)

	if err != nil {
		stats.Errors++
		log.Warn("Can't process refs data for %s: %v", repoInfo.GitHubRoot(), err)
		appendProcHeader(w, start)
		notFoundResponse(w, err.Error())
		return
	}

	t, n := suggestHead(repoInfo, refsInfo)

	// Rewrite refs
	if repoInfo.Path == "info/refs" {
		if n != "" {
			switch t {
			case refs.TYPE_TAG:
				log.Info("%s -> T:%s (%s)", path, n, refsInfo.GetTagSHA(n, true))
			case refs.TYPE_BRANCH:
				log.Info("%s -> B:%s (%s)", path, n, refsInfo.GetBranchSHA(n, true))
			default:
				log.Warn("%s -> master (proper tag/branch not found)", path)
			}
		} else {
			log.Info("%s -> master (no target version)", path)
		}

		stats.Hits++

		appendProcHeader(w, start)
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		w.Write(refsInfo.Rewrite(n, t))

		return
	}

	pkgInfo := &PkgInfo{repoInfo, n}

	if r.FormValue("go-get") == "1" {
		appendProcHeader(w, start)

		w.Header().Set("Content-Type", "text/html")

		err := goGetTemplate.Execute(w, pkgInfo)

		if err != nil {
			stats.Errors++
			log.Error("Can't render go get template: %v", err)
		}

		stats.Goget++

		return
	}

	stats.Redirects++

	// Redirect to github
	appendProcHeader(w, start)
	redirectRequest(w, repoInfo.GitHubURL(n))
}

// notFoundResponse write 404 response
func notFoundResponse(w http.ResponseWriter, data string) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(data + "\n"))
}

// appendProcHeader append header with processing time
func appendProcHeader(w http.ResponseWriter, start time.Time) {
	w.Header().Add("X-Morpher-Time", fmt.Sprintf("%s", time.Since(start)))
}

// redirectRequest add redirect header to repsponse
func redirectRequest(w http.ResponseWriter, url string) {
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// fetchRefs downloads and parse refs info from github
func fetchRefs(repo *repo.Info) (*refs.Info, error) {
	resp, err := req.Request{
		URL:         "https://" + repo.GitHubRoot() + ".git/info/refs?service=git-upload-pack",
		AutoDiscard: true,
		Close:       true,
	}.Get()

	if err != nil {
		return nil, err
	}

	refsData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("Can't read GitHub response: %v", err)
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
	targetVersion := version.Parse(getCleanVer(repoInfo.Target))

	// Can't parse version
	if !targetVersion.Valid() {
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
		tagVer := version.Parse(getCleanVer(t))

		if !tagVer.Valid() {
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

// collectStats push stats from stats struct to librato
func collectStats() []librato.Measurement {
	prefix := knf.GetS(LIBRATO_PREFIX)

	metrics := []librato.Measurement{
		&librato.Gauge{Name: prefix + ".hits", Value: stats.Hits},
		&librato.Gauge{Name: prefix + ".misses", Value: stats.Misses},
		&librato.Gauge{Name: prefix + ".errors", Value: stats.Errors},
		&librato.Gauge{Name: prefix + ".redirects", Value: stats.Redirects},
		&librato.Gauge{Name: prefix + ".docs", Value: stats.Docs},
		&librato.Gauge{Name: prefix + ".goget", Value: stats.Goget},
	}

	// Clean stats counters
	stats.Hits = 0
	stats.Misses = 0
	stats.Errors = 0
	stats.Redirects = 0
	stats.Docs = 0
	stats.Goget = 0

	return metrics
}

// collectErrorHandler librato errors handler
func collectErrorHandler(errs []error) {
	if len(errs) == 0 {
		return
	}

	for _, e := range errs {
		log.Error(e.Error())
	}
}
