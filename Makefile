PROJECT=gowaves
ORGANISATION=wavesplatform
SOURCE=$(shell find . -name '*.go' | grep -v vendor/)
SOURCE_DIRS = cmd pkg

VERSION=$(shell git describe --tags --always --dirty)

.PHONY: fmtcheck dep clean build gotest

all: dep build gotest fmtcheck

dep:
	dep ensure

ver:
	@echo Building version: $(VERSION)

build: build/bin/forkdetector

build/bin/forkdetector: $(SOURCE)
	@mkdir -p build/bin
	go build -o build/bin/forkdetector ./cmd/forkdetector 

gotest:
	go test -cover ./...

fmtcheck:
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi
	@gocritic check-project ./
clean:
	@rm -rf build

build-chaincmp-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/chaincmp.exe -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp

release-chaincmp: ver build-chaincmp-linux build-chaincmp-darwin build-chaincmp-windows

dist-chaincmp: release-chaincmp
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/chaincmp_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_Linux-64bit.tar.gz ./*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_macOS-64bit.tar.gz ./*

dist: clean dist-chaincmp
