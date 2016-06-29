########################################################################################

.PHONY = all clean install uninstall deps

########################################################################################

all: morpher

deps:
	go get -v pkg.re/essentialkaos/ek.v2
	go get -v pkg.re/essentialkaos/librato.v2

morpher:
	go build morpher-server.go

clean:
	rm -f morpher-server
