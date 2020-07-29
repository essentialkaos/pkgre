<p align="center"><img src="https://gh.kaos.st/pkgre.svg"/></p>

<p align="center">
  <a href="https://travis-ci.com/essentialkaos/pkgre"><img src="https://travis-ci.com/essentialkaos/pkgre.svg?branch=master" /></a>
  <a href="https://goreportcard.com/report/github.com/essentialkaos/pkgre"><img src="https://goreportcard.com/badge/github.com/essentialkaos/pkgre" /></a>
  <a href="https://codebeat.co/projects/github-com-essentialkaos-pkgre-master"><img alt="codebeat badge" src="https://codebeat.co/badges/f29ed07b-af32-4d45-a342-59b20e3bfcf9" /></a>
  <img src="https://github.com/essentialkaos/pkgre/workflows/CodeQL/badge.svg">
  <a href="https://essentialkaos.com/ekol"><img src="https://gh.kaos.st/ekol.svg" /></a>
</p>

<p align="center"><a href="#git-support">Git support</a> • <a href="#routing-examples">Routing examples</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a></p>

<br/>

The [pkg.re](https://pkg.re) service provides versioned URLs that offer the proper metadata for redirecting the go tool onto well defined GitHub repositories. Developers that choose to use this service are strongly encouraged to not make any backwards incompatible changes without also changing the version in the package URL. This convention improves the chances that dependent code will continue to work while depended upon packages evolve.


The advantage of using pkg.re is that the URL is cleaner, shorter, redirects to the package documentation at godoc.org when opened with a browser, handles git branches and tags for versioning, and most importantly encourages the adoption of stable versioned package APIs.


Note that pkg.re does not hold the package code. Instead, the go tool is redirected and obtains the code straight from the respective GitHub repository.


[pkg.re](https://pkg.re) have backward compatibility with [gopkg.in](https://gopkg.in) service.

### Git support

Since version 2.11.1 git [does not follow](https://github.com/git/git/commit/50d3413740d1da599cdc0106e6e916741394cc98) redirects by default. If you use git 2.11.0+ you must allow redirects for pkg.re using next command:

```bash
git config --global http.https://pkg.re.followRedirects true
```

_You can set this property for earlier versions as well._

For support fetching sources without this git configuration, we must proxy all content from source repository through our servers. This is **ABSOLUTELY NOT SECURE** and theoretically, allow to us modify the source code (_currently we just redirect all requests to Github, execept request's from GoDoc service_).

### Routing examples

```
go get pkg.re/essentialkaos/ek.v1      → github.com/essentialkaos/ek tag/branch v1.x.x
go get pkg.re/essentialkaos/ek.v1.6    → github.com/essentialkaos/ek tag/branch v1.6.x
go get pkg.re/essentialkaos/ek.v1.6.8  → github.com/essentialkaos/ek tag/branch v1.6.8
go get pkg.re/essentialkaos/ek.develop → github.com/essentialkaos/ek tag/branch develop
go get pkg.re/check.v1                 → github.com/go-check/check tag/branch v1.x.x
https://pkg.re/essentialkaos/ek.v1     → https://github.com/essentialkaos/ek/tree/v1.x.x
```

`x` - latest available version

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[EKOL](https://essentialkaos.com/ekol)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>
