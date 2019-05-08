FROM golang:1.12

WORKDIR /go/src/github.com/wavesplatform/gowaves

COPY cmd cmd
COPY pkg pkg
COPY Makefile Makefile
COPY vendor vendor

RUN make build-node-linux

EXPOSE 6863
EXPOSE 6869

CMD build/bin/linux-amd64/node run --waves-network=wavesW -d 0.0.0.0:6863 -w 0.0.0.0:6869