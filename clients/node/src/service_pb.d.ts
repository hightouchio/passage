// package: 
// file: service.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";
import * as tunnel_pb from "./tunnel_pb";

export class CreateStandardTunnelRequest extends jspb.Message { 
    getSshHost(): string;
    setSshHost(value: string): CreateStandardTunnelRequest;
    getSshPort(): number;
    setSshPort(value: number): CreateStandardTunnelRequest;

    hasSshUser(): boolean;
    clearSshUser(): void;
    getSshUser(): string | undefined;
    setSshUser(value: string): CreateStandardTunnelRequest;
    getServiceHost(): string;
    setServiceHost(value: string): CreateStandardTunnelRequest;
    getServicePort(): number;
    setServicePort(value: number): CreateStandardTunnelRequest;
    getCreateKeyPair(): boolean;
    setCreateKeyPair(value: boolean): CreateStandardTunnelRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateStandardTunnelRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateStandardTunnelRequest): CreateStandardTunnelRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateStandardTunnelRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateStandardTunnelRequest;
    static deserializeBinaryFromReader(message: CreateStandardTunnelRequest, reader: jspb.BinaryReader): CreateStandardTunnelRequest;
}

export namespace CreateStandardTunnelRequest {
    export type AsObject = {
        sshHost: string,
        sshPort: number,
        sshUser?: string,
        serviceHost: string,
        servicePort: number,
        createKeyPair: boolean,
    }
}

export class CreateReverseTunnelRequest extends jspb.Message { 
    clearPublicKeysList(): void;
    getPublicKeysList(): Array<string>;
    setPublicKeysList(value: Array<string>): CreateReverseTunnelRequest;
    addPublicKeys(value: string, index?: number): string;
    getCreateKeyPair(): boolean;
    setCreateKeyPair(value: boolean): CreateReverseTunnelRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CreateReverseTunnelRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CreateReverseTunnelRequest): CreateReverseTunnelRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CreateReverseTunnelRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CreateReverseTunnelRequest;
    static deserializeBinaryFromReader(message: CreateReverseTunnelRequest, reader: jspb.BinaryReader): CreateReverseTunnelRequest;
}

export namespace CreateReverseTunnelRequest {
    export type AsObject = {
        publicKeysList: Array<string>,
        createKeyPair: boolean,
    }
}

export class GetTunnelRequest extends jspb.Message { 
    getId(): string;
    setId(value: string): GetTunnelRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetTunnelRequest.AsObject;
    static toObject(includeInstance: boolean, msg: GetTunnelRequest): GetTunnelRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetTunnelRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetTunnelRequest;
    static deserializeBinaryFromReader(message: GetTunnelRequest, reader: jspb.BinaryReader): GetTunnelRequest;
}

export namespace GetTunnelRequest {
    export type AsObject = {
        id: string,
    }
}

export class GetTunnelResponse extends jspb.Message { 

    hasTunnel(): boolean;
    clearTunnel(): void;
    getTunnel(): tunnel_pb.Tunnel | undefined;
    setTunnel(value?: tunnel_pb.Tunnel): GetTunnelResponse;
    clearInstancesList(): void;
    getInstancesList(): Array<tunnel_pb.TunnelInstance>;
    setInstancesList(value: Array<tunnel_pb.TunnelInstance>): GetTunnelResponse;
    addInstances(value?: tunnel_pb.TunnelInstance, index?: number): tunnel_pb.TunnelInstance;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): GetTunnelResponse.AsObject;
    static toObject(includeInstance: boolean, msg: GetTunnelResponse): GetTunnelResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: GetTunnelResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): GetTunnelResponse;
    static deserializeBinaryFromReader(message: GetTunnelResponse, reader: jspb.BinaryReader): GetTunnelResponse;
}

export namespace GetTunnelResponse {
    export type AsObject = {
        tunnel?: tunnel_pb.Tunnel.AsObject,
        instancesList: Array<tunnel_pb.TunnelInstance.AsObject>,
    }
}

export class DeleteTunnelRequest extends jspb.Message { 
    getId(): string;
    setId(value: string): DeleteTunnelRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): DeleteTunnelRequest.AsObject;
    static toObject(includeInstance: boolean, msg: DeleteTunnelRequest): DeleteTunnelRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: DeleteTunnelRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): DeleteTunnelRequest;
    static deserializeBinaryFromReader(message: DeleteTunnelRequest, reader: jspb.BinaryReader): DeleteTunnelRequest;
}

export namespace DeleteTunnelRequest {
    export type AsObject = {
        id: string,
    }
}
