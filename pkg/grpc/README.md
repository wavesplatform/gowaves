# Waves Node gRPC

## Installation

Install gRPC & Protobuf & Vtprotobuf plugin:

1. Install the protocol buffer compiler `protoc`
2. Install go related tools:
```bash
go get -u google.golang.org/grpc
go get -u google.golang.org/protobuf/proto
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.4.0
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
```

If you want to regenerate the code from updated schemas:

1. Make sure that tools which you installed on installation step are in the PATH

2. Run `make proto` from the root of gowaves repo

