# Waves Node gRPC client

## Installation

Install required packages.

```bash
go get -u google.golang.org/grpc
go get -u github.com/golang/protobuf/protoc-gen-go
```

Install Protobuf compiler.  

On Mac OS X:

```bash
brew update
brew install protobuf
```

## Package structure

* `client/grpc/proto/` - a copy of proto files from [protobuf-schemas](https://github.com/wavesplatform/protobuf-schemas) project. Files are copied from folders `proto/waves/` and `proto/waves/node/grpc`. And `import` directives updated afterwards to reflect the flat structure.
* `client/grpc/` - generated gRPC client files.

## Rebuilding

Before rebuilding of gRPC client it's required to add to all `*.proto` files the following line:

```proto
option go_package = "client/grpc";
```

Execute the following command to regenerate the code.

```bash
protoc --proto_path=pkg/client/grpc/proto --go_out=plugins=grpc:pkg/ pkg/client/grpc/proto/*.proto 
```