FROM golang:1.17.8 as parent

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM parent


COPY pkg pkg
COPY cmd cmd
COPY Makefile .

RUN make build-node-linux
RUN make build-integration-linux

EXPOSE 6863
EXPOSE 6869
EXPOSE 6870

CMD ./build/bin/linux-amd64/integration -log-level DEBUG -node ./build/bin/linux-amd64/node