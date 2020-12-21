<p align="center"><img src="https://gh.kaos.st/pkgre.svg"/></p>

<p align="center">
  <a href="https://github.com/essentialkaos/pkgre/actions"><img src="https://github.com/essentialkaos/pkgre/workflows/CI/badge.svg" alt="GitHub Actions Status" /></a>
  <a href="https://github.com/essentialkaos/pkgre/actions?query=workflow%3ACodeQL"><img src="https://github.com/essentialkaos/pkgre/workflows/CodeQL/badge.svg" /></a>
  <a href="https://goreportcard.com/report/github.com/essentialkaos/pkgre"><img src="https://goreportcard.com/badge/github.com/essentialkaos/pkgre" /></a>
  <a href="https://codebeat.co/projects/github-com-essentialkaos-pkgre-master"><img alt="codebeat badge" src="https://codebeat.co/badges/f29ed07b-af32-4d45-a342-59b20e3bfcf9" /></a>
  <a href="#"><img src="https://healthchecks.io/badge/6f454deb-5215-40aa-933f-f91a8e579a07/sKjRtflJ-2/server.svg" /></a>
  <a href="#"><img src="https://healthchecks.io/badge/6f454deb-5215-40aa-933f-f91a8e579a07/2FbciL3K-2/morpher.svg" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<p align="center"><a href="#routing-examples">Routing examples</a> • <a href="#contributing">Contributing</a> • <a href="#license">License</a></p>

<br/>

The [pkg.re](https://pkg.re) service provides versioned URLs that offer the proper metadata for redirecting the go tool onto well defined GitHub repositories. Developers that choose to use this service are strongly encouraged to not make any backwards incompatible changes without also changing the version in the package URL. This convention improves the chances that dependent code will continue to work while depended upon packages evolve.


The advantage of using pkg.re is that the URL is cleaner, shorter, redirects to the package documentation at godoc.org when opened with a browser, handles git branches and tags for versioning, and most importantly encourages the adoption of stable versioned package APIs.


Note that pkg.re does not hold the package code.


[pkg.re](https://pkg.re) have backward compatibility with [gopkg.in](https://gopkg.in) service.

### Routing examples

```
go get pkg.re/essentialkaos/ek.v1        → github.com/essentialkaos/ek tag/branch v1.x.x
go get pkg.re/essentialkaos/ek.v1.6      → github.com/essentialkaos/ek tag/branch v1.6.x
go get pkg.re/essentialkaos/ek.v1.6.8    → github.com/essentialkaos/ek tag/branch v1.6.8
go get pkg.re/essentialkaos/ek.develop   → github.com/essentialkaos/ek tag/branch develop
go get pkg.re/check.v1                   → github.com/go-check/check tag/branch v1.x.x
https://pkg.re/essentialkaos/ek.v1       → https://github.com/essentialkaos/ek/tree/v1.x.x
https://pkg.re/essentialkaos/ek.v1?docs  → https://pkg.go.dev/pkg.re/essentialkaos/ek.v1
```

`x` - latest available version

### Contributing

Before contributing to this project please read our [Contributing Guidelines](https://github.com/essentialkaos/contributing-guidelines#contributing-guidelines).

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>
