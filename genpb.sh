#!/bin/bash

protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=plugins=grpc:$GOPATH/src pkg/grpc/protobuf-schemas/proto/waves/*.proto
protoc --proto_path=pkg/grpc/protobuf-schemas/proto/ --go_out=plugins=grpc:$GOPATH/src pkg/grpc/protobuf-schemas/proto/waves/node/grpc/*.proto
