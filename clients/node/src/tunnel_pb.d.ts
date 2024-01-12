// package: 
// file: tunnel.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

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

export class Tunnel extends jspb.Message { 
    getId(): string;
    setId(value: string): Tunnel;
    getType(): Tunnel.Type;
    setType(value: Tunnel.Type): Tunnel;
    getEnabled(): boolean;
    setEnabled(value: boolean): Tunnel;
    getBindport(): number;
    setBindport(value: number): Tunnel;

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
        bindport: number,
        standardTunnel?: Tunnel.StandardTunnel.AsObject,
        reverseTunnel?: Tunnel.ReverseTunnel.AsObject,
    }


    export class StandardTunnel extends jspb.Message { 
        getSshhost(): string;
        setSshhost(value: string): StandardTunnel;
        getSshport(): number;
        setSshport(value: number): StandardTunnel;
        getServicehost(): string;
        setServicehost(value: string): StandardTunnel;
        getServiceport(): string;
        setServiceport(value: string): StandardTunnel;

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
            sshhost: string,
            sshport: number,
            servicehost: string,
            serviceport: string,
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
