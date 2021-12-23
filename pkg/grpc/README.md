# Waves Node gRPC

## Installation

Install gRPC & Protobuf:

```bash
go get -u google.golang.org/grpc
go get -u google.golang.org/protobuf/proto
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
go install github.com/alexeykiselev/vtprotobuf/cmd/protoc-gen-go-vtproto@main
```

## Package structure

* `grpc/protobuf-schemas/` - a submodule of [protobuf-schemas](https://github.com/wavesplatform/protobuf-schemas)
  project (proto files).
* `grpc/generated` - code generated from proto files.
* `grpc/server` - gRPC server implementation (API).

## Instructions

If you want to update proto schemas:

```bash
cd pkg/grpc/protobuf-schemas
git submodule update --init
git pull
```

If you want to regenerate the code from updated schemas:

```bash
make proto # from the root of gowaves repo
```
