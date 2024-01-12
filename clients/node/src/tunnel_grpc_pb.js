// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var tunnel_pb = require('./tunnel_pb.js');

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


var PassageService = exports.PassageService = {
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
};

exports.PassageClient = grpc.makeGenericClientConstructor(PassageService);
