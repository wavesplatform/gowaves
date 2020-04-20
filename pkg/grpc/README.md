# Waves Node gRPC

## Installation

Install gRPC & Protobuf:

```bash
go get -u google.golang.org/grpc
go get -u github.com/golang/protobuf/proto
go get -u github.com/golang/protobuf/protoc-gen-go
```

## Package structure

* `grpc/protobuf-schemas/` - a submodule of [protobuf-schemas](https://github.com/wavesplatform/protobuf-schemas) project (proto files).
* `grpc/generated` - code generated from proto files.
* `grpc/server` - gRPC server implementation (API).
