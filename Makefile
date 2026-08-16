APP := orangectl
PKG := ./cmd/orangectl
DIST := dist

.PHONY: fmt vet test race check build linux-arm64 linux-arm install uninstall checksums

fmt:
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

check: fmt vet test race

build:
	go build -trimpath -o $(DIST)/$(APP) $(PKG)

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o $(DIST)/$(APP)-linux-arm64 $(PKG)

linux-arm:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -o $(DIST)/$(APP)-linux-armv7 $(PKG)

checksums:
	sha256sum $(DIST)/$(APP)-linux-arm64 $(DIST)/$(APP)-linux-armv7 > $(DIST)/SHA256SUMS

install:
	install -Dm755 $(DIST)/$(APP) $(DESTDIR)$(HOME)/.local/bin/$(APP)

uninstall:
	rm -f $(DESTDIR)$(HOME)/.local/bin/$(APP)
