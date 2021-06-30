################################################################################

# This Makefile generated by GoMakeGen 1.3.2 using next command:
# gomakegen --dep --metalinter .
#
# More info: https://kaos.sh/gomakegen

################################################################################

.DEFAULT_GOAL := help
.PHONY = fmt vet all clean git-config deps deps-test test gen-fuzz dep-init dep-update metalinter help

################################################################################

all: morpher-server ## Build all binaries

morpher-server: ## Build morpher-server binary
	go build morpher-server.go

install: ## Install all binaries
	cp morpher-server /usr/bin/morpher-server

uninstall: ## Uninstall all binaries
	rm -f /usr/bin/morpher-server

git-config: ## Configure git redirects for stable import path services
	git config --global http.https://pkg.re.followRedirects true

deps: git-config dep-update ## Download dependencies

deps-test: deps ## Download dependencies for tests

test: ## Run tests
	go test -covermode=count ./refs

gen-fuzz: ## Generate archives for fuzz testing
	which go-fuzz-build &>/dev/null || go get -u -v github.com/dvyukov/go-fuzz/go-fuzz-build
	go-fuzz-build -o refs-fuzz.zip github.com/essentialkaos/pkgre/refs

dep-init: ## Initialize dep workspace
	which dep &>/dev/null || go get -u -v github.com/golang/dep/cmd/dep
	dep init

dep-update: ## Update packages and dependencies through dep
	which dep &>/dev/null || go get -u -v github.com/golang/dep/cmd/dep
	test -s Gopkg.toml || dep init
	test -s Gopkg.lock && dep ensure -update || dep ensure

fmt: ## Format source code with gofmt
	find . -name "*.go" -exec gofmt -s -w {} \;

vet: ## Runs go vet over sources
	go vet -composites=false -printfuncs=LPrintf,TLPrintf,TPrintf,log.Debug,log.Info,log.Warn,log.Error,log.Critical,log.Print ./...

metalinter: ## Install and run gometalinter
	test -s $(GOPATH)/bin/gometalinter || (go get -u github.com/alecthomas/gometalinter ; $(GOPATH)/bin/gometalinter --install)
	$(GOPATH)/bin/gometalinter --deadline 30s

clean: ## Remove generated files
	rm -f morpher-server

help: ## Show this info
	@echo -e '\n\033[1mSupported targets:\033[0m\n'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[33m%-16s\033[0m %s\n", $$1, $$2}'
	@echo -e ''
	@echo -e '\033[90mGenerated by GoMakeGen 1.3.2\033[0m\n'

################################################################################
