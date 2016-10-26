########################################################################################

.PHONY = all clean deps fmt install uninstall

########################################################################################

all: morpher-server morpher-librato

deps:
	go get -v pkg.re/essentialkaos/ek.v5
	go get -v pkg.re/essentialkaos/librato.v3
	go get -v github.com/valyala/fasthttp

fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

morpher-server:
	go build morpher-server.go

morpher-librato:
	go build morpher-librato.go

clean:
	rm -f morpher-server
	rm -f morpher-librato
