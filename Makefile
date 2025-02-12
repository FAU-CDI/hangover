COMMANDS = hangover n2j
DIST = $(COMMANDS:%=dist/%)
.PHONY = $(DIST) all dist deps godeps clean test cross

# read the gobin variable
GOBIN = $(shell go env GOBIN)
GOBIN := $(if $(GOBIN),$(GOBIN),$(shell go env GOPATH)/bin)

# set the path to the build directory
BUILDPATH = $(GOBIN):$(PATH)

LINUX_AMD64 = $(COMMANDS:%=dist/%_linux_amd64)
DARWIN = $(COMMANDS:%=dist/%_darwin)
DARWIN_AMD64 = $(COMMANDS:%=dist/%_darwin_amd64)
DARWIN_ARM64 = $(COMMANDS:%=dist/%_darwin_arm64)
WINDOWS_AMD64 = $(COMMANDS:%=dist/%_windows_amd64.exe)

all: $(DIST)
$(DIST): dist/%: dist/%_linux_amd64 dist/%_darwin dist/%_windows_amd64.exe

$(LINUX_AMD64):dist/%_linux_amd64:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@ ./cmd/$*/
$(DARWIN):dist/%_darwin: dist/%_darwin_arm64 dist/%_darwin_amd64
	mkdir -p dist
	$(GOBIN)/lipo -output $@ -create dist/$*_darwin_arm64 dist/$*_darwin_amd64
$(DARWIN_ARM64):dist/%_darwin_arm64:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $@ ./cmd/$*/
$(DARWIN_AMD64):dist/%_darwin_amd64:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $@ ./cmd/$*/
$(WINDOWS_AMD64):dist/%_windows_amd64.exe:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $@ ./cmd/$*/

cross:
	go tool fyne-cross darwin -arch=amd64,arm64 -app-id de.fau.data.wisski.headache -env GOTOOLCHAIN=auto ./cmd/headache
	go tool fyne-cross linux -app-id de.fau.data.wisski.headache  -env GOTOOLCHAIN=auto ./cmd/headache
	go tool fyne-cross windows -arch=amd64 -app-id de.fau.data.wisski.headache  -env GOTOOLCHAIN=auto ./cmd/headache

clean:
	rm -rf dist

generate:
	PATH=$(BUILDPATH) go generate ./...

test:
	go test ./...

deps: godeps internal/assets/node_modules
godeps:
	go mod download
	go install github.com/tkw1536/lipo@latest
	go install github.com/tkw1536/gogenlicense/cmd/gogenlicense@latest

internal/assets/node_modules:
	cd internal/assets/ && yarn install --frozen-lockfile
