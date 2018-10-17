.PHONY: dev
dev: forkdetector
forkdetector: $(shell find . -type f -name '*.go')
	go build -o $@ ./cmd/forkdetector/...
