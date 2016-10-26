########################################################################################

.PHONY = all clean deps fmt install uninstall

########################################################################################

all: morpher-server

deps:
	go get -v pkg.re/essentialkaos/ek.v5
	go get -v pkg.re/essentialkaos/librato.v2
	go get -v github.com/valyala/fasthttp

fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

morpher-server:
	go build morpher-server.go

clean:
	rm -f morpher-server
