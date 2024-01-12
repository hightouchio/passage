#!/usr/bin/env sh

PROTO_DIR="../../tunnel/proto"
OUTPUT_DIR="./src"

# Generate Node code
grpc_tools_node_protoc \
  --js_out=import_style=commonjs,binary:$OUTPUT_DIR \
  --grpc_out=grpc_js:$OUTPUT_DIR \
  --plugin=protoc-gen-grpc=`which grpc_tools_node_protoc_plugin` \
  -I $PROTO_DIR \
  $PROTO_DIR/*.proto

# Generate TypeScript types
protoc \
  --plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
  --ts_out=grpc_js:$OUTPUT_DIR \
  -I $PROTO_DIR \
  $PROTO_DIR/*.proto