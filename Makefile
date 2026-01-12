build:
	go mod download
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=`git describe --tags --always | sed 's/^v//'`" -o zimserver ./cmd/zimserver

default: build

upgrade:
	go get -u -v ./...
	go mod download
	go mod tidy
	go mod verify

run:
	./zimserver

clean:
	go clean
	go mod tidy
	rm -f zimserver

install:
	mkdir -p $(DESTDIR)/usr/local/bin
	cp zimserver $(DESTDIR)/usr/local/bin/zimserver
	chmod 755 $(DESTDIR)/usr/local/bin/zimserver

uninstall:
	rm -f $(DESTDIR)/usr/local/bin/zimserver