GO ?= go
VERSION ?= v2.0.0
GOFLAGS_COMMON := -trimpath
LDFLAGS := -s -w -X plexlinker/internal/buildinfo.Version=$(VERSION)

.PHONY: test verify integration build demo

test:
	$(GO) test ./...

verify:
	gofmt -s -w cmd internal
	$(GO) test -race ./...
	$(GO) vet ./...
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

integration:
	PLEXLINKER_INTEGRATION=1 $(GO) test -v ./internal/aria2manager -run TestManagedAria2DownloadIntegration -count=1

build:
	mkdir -p dist
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS_COMMON) -ldflags="$(LDFLAGS)" -o dist/plexlinker-windows-amd64.exe ./cmd/plexlinker-v2
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS_COMMON) -ldflags="$(LDFLAGS)" -o dist/plexlinker-linux-amd64 ./cmd/plexlinker-v2
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS_COMMON) -ldflags="$(LDFLAGS)" -o dist/plexlinker-linux-arm64 ./cmd/plexlinker-v2
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS_COMMON) -ldflags="$(LDFLAGS)" -o dist/plexlinker-darwin-amd64 ./cmd/plexlinker-v2
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS_COMMON) -ldflags="$(LDFLAGS)" -o dist/plexlinker-darwin-arm64 ./cmd/plexlinker-v2

demo:
	$(GO) run ./cmd/plexlinker-demo-server
