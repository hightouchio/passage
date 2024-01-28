// Original file: proto/service.proto

import type * as grpc from '@grpc/grpc-js'
import type { MethodDefinition } from '@grpc/proto-loader'
import type { CreateReverseTunnelRequest as _CreateReverseTunnelRequest, CreateReverseTunnelRequest__Output as _CreateReverseTunnelRequest__Output } from './CreateReverseTunnelRequest';
import type { CreateStandardTunnelRequest as _CreateStandardTunnelRequest, CreateStandardTunnelRequest__Output as _CreateStandardTunnelRequest__Output } from './CreateStandardTunnelRequest';
import type { DeleteTunnelRequest as _DeleteTunnelRequest, DeleteTunnelRequest__Output as _DeleteTunnelRequest__Output } from './DeleteTunnelRequest';
import type { Empty as _google_protobuf_Empty, Empty__Output as _google_protobuf_Empty__Output } from './google/protobuf/Empty';
import type { GetTunnelRequest as _GetTunnelRequest, GetTunnelRequest__Output as _GetTunnelRequest__Output } from './GetTunnelRequest';
import type { GetTunnelResponse as _GetTunnelResponse, GetTunnelResponse__Output as _GetTunnelResponse__Output } from './GetTunnelResponse';
import type { Tunnel as _Tunnel, Tunnel__Output as _Tunnel__Output } from './Tunnel';

export interface PassageClient extends grpc.Client {
  CreateReverseTunnel(argument: _CreateReverseTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  CreateReverseTunnel(argument: _CreateReverseTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  CreateReverseTunnel(argument: _CreateReverseTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  CreateReverseTunnel(argument: _CreateReverseTunnelRequest, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createReverseTunnel(argument: _CreateReverseTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createReverseTunnel(argument: _CreateReverseTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createReverseTunnel(argument: _CreateReverseTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createReverseTunnel(argument: _CreateReverseTunnelRequest, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  
  CreateStandardTunnel(argument: _CreateStandardTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  CreateStandardTunnel(argument: _CreateStandardTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  CreateStandardTunnel(argument: _CreateStandardTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  CreateStandardTunnel(argument: _CreateStandardTunnelRequest, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createStandardTunnel(argument: _CreateStandardTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createStandardTunnel(argument: _CreateStandardTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createStandardTunnel(argument: _CreateStandardTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  createStandardTunnel(argument: _CreateStandardTunnelRequest, callback: grpc.requestCallback<_Tunnel__Output>): grpc.ClientUnaryCall;
  
  DeleteTunnel(argument: _DeleteTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  DeleteTunnel(argument: _DeleteTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  DeleteTunnel(argument: _DeleteTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  DeleteTunnel(argument: _DeleteTunnelRequest, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  deleteTunnel(argument: _DeleteTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  deleteTunnel(argument: _DeleteTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  deleteTunnel(argument: _DeleteTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  deleteTunnel(argument: _DeleteTunnelRequest, callback: grpc.requestCallback<_google_protobuf_Empty__Output>): grpc.ClientUnaryCall;
  
  GetTunnel(argument: _GetTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  GetTunnel(argument: _GetTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  GetTunnel(argument: _GetTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  GetTunnel(argument: _GetTunnelRequest, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  getTunnel(argument: _GetTunnelRequest, metadata: grpc.Metadata, options: grpc.CallOptions, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  getTunnel(argument: _GetTunnelRequest, metadata: grpc.Metadata, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  getTunnel(argument: _GetTunnelRequest, options: grpc.CallOptions, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  getTunnel(argument: _GetTunnelRequest, callback: grpc.requestCallback<_GetTunnelResponse__Output>): grpc.ClientUnaryCall;
  
}

export interface PassageHandlers extends grpc.UntypedServiceImplementation {
  CreateReverseTunnel: grpc.handleUnaryCall<_CreateReverseTunnelRequest__Output, _Tunnel>;
  
  CreateStandardTunnel: grpc.handleUnaryCall<_CreateStandardTunnelRequest__Output, _Tunnel>;
  
  DeleteTunnel: grpc.handleUnaryCall<_DeleteTunnelRequest__Output, _google_protobuf_Empty>;
  
  GetTunnel: grpc.handleUnaryCall<_GetTunnelRequest__Output, _GetTunnelResponse>;
  
}

export interface PassageDefinition extends grpc.ServiceDefinition {
  CreateReverseTunnel: MethodDefinition<_CreateReverseTunnelRequest, _Tunnel, _CreateReverseTunnelRequest__Output, _Tunnel__Output>
  CreateStandardTunnel: MethodDefinition<_CreateStandardTunnelRequest, _Tunnel, _CreateStandardTunnelRequest__Output, _Tunnel__Output>
  DeleteTunnel: MethodDefinition<_DeleteTunnelRequest, _google_protobuf_Empty, _DeleteTunnelRequest__Output, _google_protobuf_Empty__Output>
  GetTunnel: MethodDefinition<_GetTunnelRequest, _GetTunnelResponse, _GetTunnelRequest__Output, _GetTunnelResponse__Output>
}
