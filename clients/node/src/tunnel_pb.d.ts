// package: 
// file: tunnel.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

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

export class TunnelInstance extends jspb.Message { 
    getHost(): string;
    setHost(value: string): TunnelInstance;
    getPort(): number;
    setPort(value: number): TunnelInstance;
    getStatus(): TunnelHealthcheck.Status;
    setStatus(value: TunnelHealthcheck.Status): TunnelInstance;
    clearHealthchecksList(): void;
    getHealthchecksList(): Array<TunnelHealthcheck>;
    setHealthchecksList(value: Array<TunnelHealthcheck>): TunnelInstance;
    addHealthchecks(value?: TunnelHealthcheck, index?: number): TunnelHealthcheck;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TunnelInstance.AsObject;
    static toObject(includeInstance: boolean, msg: TunnelInstance): TunnelInstance.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TunnelInstance, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TunnelInstance;
    static deserializeBinaryFromReader(message: TunnelInstance, reader: jspb.BinaryReader): TunnelInstance;
}

export namespace TunnelInstance {
    export type AsObject = {
        host: string,
        port: number,
        status: TunnelHealthcheck.Status,
        healthchecksList: Array<TunnelHealthcheck.AsObject>,
    }
}

export class TunnelHealthcheck extends jspb.Message { 
    getId(): string;
    setId(value: string): TunnelHealthcheck;
    getStatus(): TunnelHealthcheck.Status;
    setStatus(value: TunnelHealthcheck.Status): TunnelHealthcheck;
    getMessage(): string;
    setMessage(value: string): TunnelHealthcheck;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): TunnelHealthcheck.AsObject;
    static toObject(includeInstance: boolean, msg: TunnelHealthcheck): TunnelHealthcheck.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: TunnelHealthcheck, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): TunnelHealthcheck;
    static deserializeBinaryFromReader(message: TunnelHealthcheck, reader: jspb.BinaryReader): TunnelHealthcheck;
}

export namespace TunnelHealthcheck {
    export type AsObject = {
        id: string,
        status: TunnelHealthcheck.Status,
        message: string,
    }

    export enum Status {
    WARNING = 0,
    PASSING = 1,
    CRITICAL = 2,
    }

}
