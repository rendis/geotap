VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build install clean release-dry

build:
	go build -ldflags "$(LDFLAGS)" -o geotap ./cmd/geotap

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/geotap

clean:
	rm -f geotap

release-dry:
	goreleaser release --snapshot --clean
