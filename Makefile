PROJECT=gowaves
ORGANISATION=wavesplatform
SOURCE=$(shell find . -name '*.go' | grep -v vendor/)
SOURCE_DIRS = cmd pkg

.PHONY: fmtcheck dep clean build gotest

all: dep build gotest fmtcheck

dep:
	dep ensure

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
	rm -rf build
