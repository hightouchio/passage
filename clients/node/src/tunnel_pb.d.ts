// package: 
// file: tunnel.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_empty_pb from "google-protobuf/google/protobuf/empty_pb";

export class Tunnel extends jspb.Message { 
    getId(): string;
    setId(value: string): Tunnel;
    getType(): Tunnel.Type;
    setType(value: Tunnel.Type): Tunnel;
    getEnabled(): boolean;
    setEnabled(value: boolean): Tunnel;
    getBindPort(): number;
    setBindPort(value: number): Tunnel;

    hasStandardTunnel(): boolean;
    clearStandardTunnel(): void;
    getStandardTunnel(): Tunnel.StandardTunnel | undefined;
    setStandardTunnel(value?: Tunnel.StandardTunnel): Tunnel;

    hasReverseTunnel(): boolean;
    clearReverseTunnel(): void;
    getReverseTunnel(): Tunnel.ReverseTunnel | undefined;
    setReverseTunnel(value?: Tunnel.ReverseTunnel): Tunnel;

    getTunnelCase(): Tunnel.TunnelCase;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Tunnel.AsObject;
    static toObject(includeInstance: boolean, msg: Tunnel): Tunnel.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Tunnel, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Tunnel;
    static deserializeBinaryFromReader(message: Tunnel, reader: jspb.BinaryReader): Tunnel;
}

export namespace Tunnel {
    export type AsObject = {
        id: string,
        type: Tunnel.Type,
        enabled: boolean,
        bindPort: number,
        standardTunnel?: Tunnel.StandardTunnel.AsObject,
        reverseTunnel?: Tunnel.ReverseTunnel.AsObject,
    }


    export class StandardTunnel extends jspb.Message { 
        getSshHost(): string;
        setSshHost(value: string): StandardTunnel;
        getSshPort(): number;
        setSshPort(value: number): StandardTunnel;
        getSshUser(): string;
        setSshUser(value: string): StandardTunnel;
        getServiceHost(): string;
        setServiceHost(value: string): StandardTunnel;
        getServicePort(): number;
        setServicePort(value: number): StandardTunnel;

        serializeBinary(): Uint8Array;
        toObject(includeInstance?: boolean): StandardTunnel.AsObject;
        static toObject(includeInstance: boolean, msg: StandardTunnel): StandardTunnel.AsObject;
        static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
        static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
        static serializeBinaryToWriter(message: StandardTunnel, writer: jspb.BinaryWriter): void;
        static deserializeBinary(bytes: Uint8Array): StandardTunnel;
        static deserializeBinaryFromReader(message: StandardTunnel, reader: jspb.BinaryReader): StandardTunnel;
    }

    export namespace StandardTunnel {
        export type AsObject = {
            sshHost: string,
            sshPort: number,
            sshUser: string,
            serviceHost: string,
            servicePort: number,
        }
    }

    export class ReverseTunnel extends jspb.Message { 

        serializeBinary(): Uint8Array;
        toObject(includeInstance?: boolean): ReverseTunnel.AsObject;
        static toObject(includeInstance: boolean, msg: ReverseTunnel): ReverseTunnel.AsObject;
        static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
        static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
        static serializeBinaryToWriter(message: ReverseTunnel, writer: jspb.BinaryWriter): void;
        static deserializeBinary(bytes: Uint8Array): ReverseTunnel;
        static deserializeBinaryFromReader(message: ReverseTunnel, reader: jspb.BinaryReader): ReverseTunnel;
    }

    export namespace ReverseTunnel {
        export type AsObject = {
        }
    }


    export enum Type {
    STANDARD = 0,
    REVERSE = 1,
    }


    export enum TunnelCase {
        TUNNEL_NOT_SET = 0,
        STANDARD_TUNNEL = 6,
        REVERSE_TUNNEL = 5,
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
