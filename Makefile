PROJECT=gowaves
ORGANISATION=wavesplatform
SOURCE=$(shell find . -name '*.go')

.PHONY: dep clean build gotest

all: dep build

dep:
	dep ensure

build: build/bin/forkdetector

build/bin/forkdetector: $(SOURCE)
	@mkdir -p build/bin
	go build -o build/bin/forkdetector ./cmd/forkdetector 

gotest:
	go test -cover ./...

clean:
	rm -rf build
