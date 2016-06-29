########################################################################################

.PHONY = all clean install uninstall deps

########################################################################################

all: morpher-server

deps:
	go get -v pkg.re/essentialkaos/ek.v2
	go get -v pkg.re/essentialkaos/librato.v2

morpher-server:
	go build morpher-server.go

clean:
	rm -f morpher-server
