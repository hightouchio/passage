import type * as grpc from '@grpc/grpc-js';
import type { MessageTypeDefinition } from '@grpc/proto-loader';

import type { PassageClient as _PassageClient, PassageDefinition as _PassageDefinition } from './Passage';

type SubtypeConstructor<Constructor extends new (...args: any) => any, Subtype> = {
  new(...args: ConstructorParameters<Constructor>): Subtype;
};

export interface ProtoGrpcType {
  CreateReverseTunnelRequest: MessageTypeDefinition
  CreateStandardTunnelRequest: MessageTypeDefinition
  DeleteTunnelRequest: MessageTypeDefinition
  GetTunnelRequest: MessageTypeDefinition
  GetTunnelResponse: MessageTypeDefinition
  Passage: SubtypeConstructor<typeof grpc.Client, _PassageClient> & { service: _PassageDefinition }
  Tunnel: MessageTypeDefinition
  TunnelHealthcheck: MessageTypeDefinition
  TunnelInstance: MessageTypeDefinition
  google: {
    protobuf: {
      Empty: MessageTypeDefinition
    }
  }
}

