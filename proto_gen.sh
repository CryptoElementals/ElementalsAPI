#! /bin/bash
export PATH=$PATH:~/go/bin
# protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative rpc/proto/*.proto
protoc --experimental_allow_proto3_optional --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative rpc/proto/*.proto