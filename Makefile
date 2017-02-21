########################################################################################

.PHONY = all clean deps fmt install uninstall

########################################################################################

all: morpher-server morpher-librato

deps:
	go get -v -d pkg.re/essentialkaos/ek.v6
	go get -v -d pkg.re/essentialkaos/librato.v4
	go get -v -d github.com/valyala/fasthttp

fmt:
	find . -name "*.go" -exec gofmt -s -w {} \;

morpher-server:
	go build morpher-server.go

morpher-librato:
	go build morpher-librato.go

clean:
	rm -f morpher-server
	rm -f morpher-librato
