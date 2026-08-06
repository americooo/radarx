# RadarX build tasks. Pure Go, CGO-free — cross-compiles anywhere.

BINARY   := radarx
PKG      := ./cmd/radarx
DIST     := dist
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -s -w -X main.version=$(VERSION)

export CGO_ENABLED=0

.PHONY: all build run test vet fmt check clean release

all: check build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

check: fmt vet test
	@gofmt -l . | grep . && (echo "unformatted files" && exit 1) || echo "format OK"

clean:
	rm -rf $(BINARY) $(BINARY).exe $(DIST)

# Cross-compile release binaries + SHA-256 checksums for each.
release: clean
	mkdir -p $(DIST)
	@for target in linux/amd64 windows/amd64 darwin/arm64 darwin/amd64; do \
		os=$${target%/*}; arch=$${target#*/}; \
		out=$(DIST)/$(BINARY)-$(VERSION)-$$os-$$arch; \
		[ $$os = windows ] && out=$$out.exe; \
		echo "building $$out"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out $(PKG); \
		sha256sum $$out > $$out.sha256; \
	done
	@echo "release artifacts in $(DIST)/"
