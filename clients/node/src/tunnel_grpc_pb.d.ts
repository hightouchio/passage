// package: 
// file: tunnel.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as tunnel_pb from "./tunnel_pb";

interface IPassageService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    getTunnel: IPassageService_IGetTunnel;
}

interface IPassageService_IGetTunnel extends grpc.MethodDefinition<tunnel_pb.GetTunnelRequest, tunnel_pb.Tunnel> {
    path: "/Passage/GetTunnel";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<tunnel_pb.GetTunnelRequest>;
    requestDeserialize: grpc.deserialize<tunnel_pb.GetTunnelRequest>;
    responseSerialize: grpc.serialize<tunnel_pb.Tunnel>;
    responseDeserialize: grpc.deserialize<tunnel_pb.Tunnel>;
}

export const PassageService: IPassageService;

export interface IPassageServer extends grpc.UntypedServiceImplementation {
    getTunnel: grpc.handleUnaryCall<tunnel_pb.GetTunnelRequest, tunnel_pb.Tunnel>;
}

export interface IPassageClient {
    getTunnel(request: tunnel_pb.GetTunnelRequest, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    getTunnel(request: tunnel_pb.GetTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    getTunnel(request: tunnel_pb.GetTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
}

export class PassageClient extends grpc.Client implements IPassageClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public getTunnel(request: tunnel_pb.GetTunnelRequest, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public getTunnel(request: tunnel_pb.GetTunnelRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
    public getTunnel(request: tunnel_pb.GetTunnelRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: tunnel_pb.Tunnel) => void): grpc.ClientUnaryCall;
}
