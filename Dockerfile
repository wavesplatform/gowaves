FROM golang:1.19.0-alpine3.16 as parent

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

FROM parent as builder

RUN apk add --no-cache make

COPY pkg pkg
COPY cmd cmd
COPY Makefile .

RUN make build-node-linux
RUN make build-integration-linux

FROM alpine:3.16.1
ENV TZ=Etc/UTC \
    APP_USER=gowaves

RUN addgroup -S $APP_USER \
    && adduser -S $APP_USER -G $APP_USER

EXPOSE 6863
EXPOSE 6869
EXPOSE 6870

USER $APP_USER

COPY --from=builder /app/build/bin/linux-amd64/node        /app/node
COPY --from=builder /app/build/bin/linux-amd64/integration /app/integration

CMD /app/integration -log-level DEBUG -node /app/node
