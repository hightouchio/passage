// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var tunnel_pb = require('./tunnel_pb.js');
var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');

function serialize_CreateReverseTunnelRequest(arg) {
  if (!(arg instanceof tunnel_pb.CreateReverseTunnelRequest)) {
    throw new Error('Expected argument of type CreateReverseTunnelRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_CreateReverseTunnelRequest(buffer_arg) {
  return tunnel_pb.CreateReverseTunnelRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_CreateStandardTunnelRequest(arg) {
  if (!(arg instanceof tunnel_pb.CreateStandardTunnelRequest)) {
    throw new Error('Expected argument of type CreateStandardTunnelRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_CreateStandardTunnelRequest(buffer_arg) {
  return tunnel_pb.CreateStandardTunnelRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_DeleteTunnelRequest(arg) {
  if (!(arg instanceof tunnel_pb.DeleteTunnelRequest)) {
    throw new Error('Expected argument of type DeleteTunnelRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_DeleteTunnelRequest(buffer_arg) {
  return tunnel_pb.DeleteTunnelRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_GetTunnelRequest(arg) {
  if (!(arg instanceof tunnel_pb.GetTunnelRequest)) {
    throw new Error('Expected argument of type GetTunnelRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_GetTunnelRequest(buffer_arg) {
  return tunnel_pb.GetTunnelRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_Tunnel(arg) {
  if (!(arg instanceof tunnel_pb.Tunnel)) {
    throw new Error('Expected argument of type Tunnel');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_Tunnel(buffer_arg) {
  return tunnel_pb.Tunnel.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_google_protobuf_Empty(arg) {
  if (!(arg instanceof google_protobuf_empty_pb.Empty)) {
    throw new Error('Expected argument of type google.protobuf.Empty');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_google_protobuf_Empty(buffer_arg) {
  return google_protobuf_empty_pb.Empty.deserializeBinary(new Uint8Array(buffer_arg));
}


var PassageService = exports.PassageService = {
  createStandardTunnel: {
    path: '/Passage/CreateStandardTunnel',
    requestStream: false,
    responseStream: false,
    requestType: tunnel_pb.CreateStandardTunnelRequest,
    responseType: tunnel_pb.Tunnel,
    requestSerialize: serialize_CreateStandardTunnelRequest,
    requestDeserialize: deserialize_CreateStandardTunnelRequest,
    responseSerialize: serialize_Tunnel,
    responseDeserialize: deserialize_Tunnel,
  },
  createReverseTunnel: {
    path: '/Passage/CreateReverseTunnel',
    requestStream: false,
    responseStream: false,
    requestType: tunnel_pb.CreateReverseTunnelRequest,
    responseType: tunnel_pb.Tunnel,
    requestSerialize: serialize_CreateReverseTunnelRequest,
    requestDeserialize: deserialize_CreateReverseTunnelRequest,
    responseSerialize: serialize_Tunnel,
    responseDeserialize: deserialize_Tunnel,
  },
  getTunnel: {
    path: '/Passage/GetTunnel',
    requestStream: false,
    responseStream: false,
    requestType: tunnel_pb.GetTunnelRequest,
    responseType: tunnel_pb.Tunnel,
    requestSerialize: serialize_GetTunnelRequest,
    requestDeserialize: deserialize_GetTunnelRequest,
    responseSerialize: serialize_Tunnel,
    responseDeserialize: deserialize_Tunnel,
  },
  deleteTunnel: {
    path: '/Passage/DeleteTunnel',
    requestStream: false,
    responseStream: false,
    requestType: tunnel_pb.DeleteTunnelRequest,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_DeleteTunnelRequest,
    requestDeserialize: deserialize_DeleteTunnelRequest,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
};

exports.PassageClient = grpc.makeGenericClientConstructor(PassageService);
