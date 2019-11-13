PROJECT=gowaves
ORGANISATION=wavesplatform
SOURCE=$(shell find . -name '*.go' | grep -v vendor/)
SOURCE_DIRS = cmd pkg

VERSION=$(shell git describe --tags --always --dirty)

export GO111MODULE=on

.PHONY: vendor vetcheck fmtcheck clean build gotest

all: vendor vetcheck fmtcheck build gotest mod-clean

ver:
	@echo Building version: $(VERSION)

build: build/bin/forkdetector

build/bin/forkdetector: $(SOURCE)
	@mkdir -p build/bin
	go build -o build/bin/forkdetector ./cmd/forkdetector

build-forkdetector-linux-amd64:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/forkdetector -ldflags="-X main.version=$(VERSION)" ./cmd/forkdetector

build-forkdetector-linux-arm:
	@CGO_ENABLE=0 GOOS=linux GOARCH=arm go build -o build/bin/linux-arm/forkdetector -ldflags="-X main.version=$(VERSION)" ./cmd/forkdetector

gotest:
	go test -cover ./...

fmtcheck:
	@gofmt -l -s $(SOURCE_DIRS) | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi

mod-clean:
	go mod tidy

clean:
	@rm -rf build
	go mod tidy

vendor:
	go mod vendor

vetcheck:
	go vet ./...
	golangci-lint run

build-chaincmp-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/chaincmp -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp
build-chaincmp-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/chaincmp.exe -ldflags="-X main.version=$(VERSION)" ./cmd/chaincmp

release-chaincmp: ver build-chaincmp-linux build-chaincmp-darwin build-chaincmp-windows

dist-chaincmp: release-chaincmp
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/chaincmp_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/chaincmp*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_Linux-64bit.tar.gz ./chaincmp*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/chaincmp_$(VERSION)_macOS-64bit.tar.gz ./chaincmp*

build-wmd-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/wmd -ldflags="-X main.version=$(VERSION)" ./cmd/wmd
build-wmd-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/wmd -ldflags="-X main.version=$(VERSION)" ./cmd/wmd
build-wmd-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/wmd.exe -ldflags="-X main.version=$(VERSION)" ./cmd/wmd

release-wmd: ver build-wmd-linux build-wmd-darwin build-wmd-windows

dist-wmd: release-wmd
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/wmd_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/wmd*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/wmd_$(VERSION)_Linux-64bit.tar.gz ./wmd*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/wmd_$(VERSION)_macOS-64bit.tar.gz ./wmd*

build-retransmitter-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/retransmitter -ldflags="-X main.version=$(VERSION)" ./cmd/retransmitter
build-retransmitter-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/retransmitter -ldflags="-X main.version=$(VERSION)" ./cmd/retransmitter
build-retransmitter-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/retransmitter.exe -ldflags="-X main.version=$(VERSION)" ./cmd/retransmitter

release-retransmitter: ver build-retransmitter-linux build-retransmitter-darwin build-retransmitter-windows

build-node-linux:
	@GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/node ./cmd/node
build-node-darwin:
	@GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/node ./cmd/node
build-node-windows:
	@GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/node.exe ./cmd/node

release-node: ver build-node-linux build-node-darwin build-node-windows

dist-node: release-node
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/node_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/node*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/node_$(VERSION)_Linux-64bit.tar.gz ./node*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/node_$(VERSION)_macOS-64bit.tar.gz ./node*

build-custom-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/custom ./cmd/custom
build-custom-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/custom ./cmd/custom
build-custom-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/custom.exe ./cmd/custom

build-custom: ver build-custom-linux build-custom-darwin build-custom-windows

build-docker:
	docker build -t com.wavesplatform/node-it:latest .

build-importer-linux:
	@CGO_ENABLE=0 GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/importer ./cmd/importer
build-importer-darwin:
	@CGO_ENABLE=0 GOOS=darwin GOARCH=amd64 go build -o build/bin/darwin-amd64/importer ./cmd/importer
build-importer-windows:
	@CGO_ENABLE=0 GOOS=windows GOARCH=amd64 go build -o build/bin/windows-amd64/importer.exe ./cmd/importer

release-importer: ver build-importer-linux build-importer-darwin build-importer-windows

dist-importer: release-importer
	@mkdir -p build/dist
	@cd ./build/; zip -j ./dist/importer_$(VERSION)_Windows-64bit.zip ./bin/windows-amd64/importer*
	@cd ./build/bin/linux-amd64/; tar pzcvf ../../dist/importer_$(VERSION)_Linux-64bit.tar.gz ./importer*
	@cd ./build/bin/darwin-amd64/; tar pzcvf ../../dist/importer_$(VERSION)_macOS-64bit.tar.gz ./importer*

dist: clean dist-chaincmp dist-wmd dist-importer dist-node
