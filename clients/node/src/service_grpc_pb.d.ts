// package: 
// file: service.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as service_pb from "./service_pb";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as tunnel_pb from "./tunnel_pb";

interface IPassageService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    createStandardTunnel: IPassageService_ICreateStandardTunnel;
    createReverseTunnel: IPassageService_ICreateReverseTunnel;
    getTunnel: IPassageService_IGetTunnel;
    deleteTunnel: IPassageService_IDeleteTunnel;
}

interface IPassageService_ICreateStandardTunnel extends grpc.MethodDefinition<service_pb.CreateStandardTunnelRequest, tunnel_pb.Tunnel> {
    path: "/Passage/CreateStandardTunnel";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<service_pb.CreateStandardTunnelRequest>;
    requestDeserialize: grpc.deserialize<service_pb.CreateStandardTunnelRequest>;
    responseSerialize: grpc.serialize<tunnel_pb.Tunnel>;
    responseDeserialize: grpc.deserialize<tunnel_pb.Tunnel>;
}
interface IPassageService_ICreateReverseTunnel extends grpc.MethodDefinition<service_pb.CreateReverseTunnelRequest, tunnel_pb.Tunnel> {
    path: "/Passage/CreateReverseTunnel";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<service_pb.CreateReverseTunnelRequest>;
    requestDeserialize: grpc.deserialize<service_pb.CreateReverseTunnelRequest>;
    responseSerialize: grpc.serialize<tunnel_pb.Tunnel>;
    responseDeserialize: grpc.deserialize<tunnel_pb.Tunnel>;
}
interface IPassageService_IGetTunnel extends grpc.MethodDefinition<service_pb.GetTunnelRequest, service_pb.GetTunnelResponse> {
    path: "/Passage/GetTunnel";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<service_pb.GetTunnelRequest>;
    requestDeserialize: grpc.deserialize<service_pb.GetTunnelRequest>;
    responseSerialize: grpc.serialize<service_pb.GetTunnelResponse>;
    responseDeserialize: grpc.deserialize<service_pb.GetTunnelResponse>;
}
interface IPassageService_IDeleteTunnel extends grpc.MethodDefinition<service_pb.DeleteTunnelRequest, google_protobuf_empty_pb.Empty> {
    path: "/Passage/DeleteTunnel";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<service_pb.DeleteTunnelRequest>;
    requestDeserialize: grpc.deserialize<service_pb.DeleteTunnelRequest>;
    responseSerialize: grpc.serialize<google_protobuf_empty_pb.Empty>;
    responseDeserialize: grpc.deserialize<google_protobuf_empty_pb.Empty>;
}

export const PassageService: IPassageService;

export interface IPassageServer extends grpc.UntypedServiceImplementation {
    createStandardTunnel: grpc.handleUnaryCall<service_pb.CreateStandardTunnelRequest, tunnel_pb.Tunnel>;
    createReverseTunnel: grpc.handleUnaryCall<service_pb.CreateReverseTunnelRequest, tunnel_pb.Tunnel>;
    getTunnel: grpc.handleUnaryCall<service_pb.GetTunnelRequest, service_pb.GetTunnelResponse>;
    deleteTunnel: grpc.handleUnaryCall<service_pb.DeleteTunnelRequest, google_protobuf_empty_pb.Empty>;
}

export interface IPassageClient {
    createStandardTunnel(request: service_pb.CreateStandardTunnelRequest, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    createStandardTunnel(request: service_pb.CreateStandardTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    createStandardTunnel(request: service_pb.CreateStandardTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    createReverseTunnel(request: service_pb.CreateReverseTunnelRequest, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    createReverseTunnel(request: service_pb.CreateReverseTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    createReverseTunnel(request: service_pb.CreateReverseTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    getTunnel(request: service_pb.GetTunnelRequest, callback: (error: grpc.ServiceError | null, response: service_pb.GetTunnelResponse) => void): grpc.ClientUnaryCall;
    getTunnel(request: service_pb.GetTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: service_pb.GetTunnelResponse) => void): grpc.ClientUnaryCall;
    getTunnel(request: service_pb.GetTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: service_pb.GetTunnelResponse) => void): grpc.ClientUnaryCall;
    deleteTunnel(request: service_pb.DeleteTunnelRequest, callback: (error: grpc.ServiceError | null, response: google_protobuf_empty_pb.Empty) => void): grpc.ClientUnaryCall;
    deleteTunnel(request: service_pb.DeleteTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: google_protobuf_empty_pb.Empty) => void): grpc.ClientUnaryCall;
    deleteTunnel(request: service_pb.DeleteTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: google_protobuf_empty_pb.Empty) => void): grpc.ClientUnaryCall;
}

export class PassageClient extends grpc.Client implements IPassageClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public createStandardTunnel(request: service_pb.CreateStandardTunnelRequest, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public createStandardTunnel(request: service_pb.CreateStandardTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public createStandardTunnel(request: service_pb.CreateStandardTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public createReverseTunnel(request: service_pb.CreateReverseTunnelRequest, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public createReverseTunnel(request: service_pb.CreateReverseTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public createReverseTunnel(request: service_pb.CreateReverseTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public getTunnel(request: service_pb.GetTunnelRequest, callback: (error: grpc.ServiceError | null, response: service_pb.GetTunnelResponse) => void): grpc.ClientUnaryCall;
    public getTunnel(request: service_pb.GetTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: service_pb.GetTunnelResponse) => void): grpc.ClientUnaryCall;
    public getTunnel(request: service_pb.GetTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: service_pb.GetTunnelResponse) => void): grpc.ClientUnaryCall;
    public deleteTunnel(request: service_pb.DeleteTunnelRequest, callback: (error: grpc.ServiceError | null, response: google_protobuf_empty_pb.Empty) => void): grpc.ClientUnaryCall;
    public deleteTunnel(request: service_pb.DeleteTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: google_protobuf_empty_pb.Empty) => void): grpc.ClientUnaryCall;
    public deleteTunnel(request: service_pb.DeleteTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: google_protobuf_empty_pb.Empty) => void): grpc.ClientUnaryCall;
}
