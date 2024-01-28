#!/usr/bin/env sh

# Compile protobufs into client/server code
PROTO_DIR="./proto"

# Generate Go code
GO_OUTPUT_DIR="./tunnel/proto"
protoc --go_out=$GO_OUTPUT_DIR \
       --go_opt=paths=source_relative \
       --go-grpc_out=$GO_OUTPUT_DIR \
       --go-grpc_opt=paths=source_relative \
       -I $PROTO_DIR \
       proto/*.proto


# Generate Node code
NODE_OUTPUT_DIR="./clients/node/proto"
cp proto/*.proto $NODE_OUTPUT_DIR
./clients/node/node_modules/.bin/proto-loader-gen-types --grpcLib=@grpc/grpc-js --outDir=$NODE_OUTPUT_DIR proto/*.proto
