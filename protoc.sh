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
NODE_OUTPUT_DIR="./clients/node/src"
grpc_tools_node_protoc \
  --js_out=import_style=commonjs,binary:$NODE_OUTPUT_DIR \
  --grpc_out=grpc_js:$NODE_OUTPUT_DIR \
  --plugin=protoc-gen-grpc=`which grpc_tools_node_protoc_plugin` \
  -I $PROTO_DIR \
  $PROTO_DIR/*.proto

# Generate TypeScript types
protoc \
  --plugin=protoc-gen-ts=./clients/node/node_modules/.bin/protoc-gen-ts \
  --ts_out=grpc_js:$NODE_OUTPUT_DIR \
  -I $PROTO_DIR \
  $PROTO_DIR/*.proto
