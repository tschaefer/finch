Version := $(shell git describe --tags --dirty 2> /dev/null)
GitCommit := $(shell git rev-parse HEAD)
LDFLAGS := "-s -w -X github.com/tschaefer/finch/internal/version.Version=$(Version) -X github.com/tschaefer/finch/internal/version.GitCommit=$(GitCommit)"
GOOS   := $(if $(GOOS),$(GOOS),linux)
GOARCH := $(if $(GOARCH),$(GOARCH),amd64)

.PHONY: all
all: fmt lint dist

.PHONY: fmt
fmt:
	test -z $(shell gofmt -l .) || (echo "[WARN] Fix format issues" && exit 1)

.PHONY: lint
lint:
	test -z $(shell golangci-lint run >/dev/null || echo 1) || (echo "[WARN] Fix lint issues" && exit 1)

.PHONY: test
test:
	test -z $(shell go test -v ./... 2>&1 >/dev/null || echo 1) || (echo "[WARN] Fix test issues" && exit 1)

.PHONY: dist
dist:
	mkdir -p bin
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/finch-$(GOOS)-$(GOARCH) -ldflags $(LDFLAGS) .

.PHONY: checksum
checksum:
	cd bin && \
	for f in finch-$(GOOS)-$(GOARCH); do \
		sha256sum $$f > $$f.sha256; \
	done && \
	cd ..

.PHONY: clean
clean:
	rm -rf bin
