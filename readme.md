## `pkg.re` [![Build Status](https://travis-ci.org/essentialkaos/pkgre.svg?branch=master)](https://travis-ci.org/essentialkaos/pkgre)

The [pkg.re](https://pkg.re) service provides versioned URLs that offer the proper metadata for redirecting the go tool onto well defined GitHub repositories. Developers that choose to use this service are strongly encouraged to not make any backwards incompatible changes without also changing the version in the package URL. This convention improves the chances that dependent code will continue to work while depended upon packages evolve.


The advantage of using pkg.re is that the URL is cleaner, shorter, redirects to the package documentation at godoc.org when opened with a browser, handles git branches and tags for versioning, and most importantly encourages the adoption of stable versioned package APIs.


Note that pkg.re does not hold the package code. Instead, the go tool is redirected and obtains the code straight from the respective GitHub repository.


pkg.re is fully compatible with [gopkg.in](https://gopkg.in) service.

#### Routing examples

````
go get pkg.re/essentialkaos/ek.v1      → github.com/essentialkaos/ek tag/branch v1.x.x
go get pkg.re/essentialkaos/ek.v1.6    → github.com/essentialkaos/ek tag/branch v1.6.x
go get pkg.re/essentialkaos/ek.v1.6.8  → github.com/essentialkaos/ek tag/branch v1.6.8
go get pkg.re/essentialkaos/ek.develop → github.com/essentialkaos/ek tag/branch develop
go get pkg.re/check.v1                 → github.com/essentialkaos/go-check/check tag/branch v1.x.x
````

`x` - latest available version

#### License

[EKOL](https://essentialkaos.com/ekol)
