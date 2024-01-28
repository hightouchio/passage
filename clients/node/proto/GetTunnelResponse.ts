// Original file: proto/service.proto

import type { Tunnel as _Tunnel, Tunnel__Output as _Tunnel__Output } from './Tunnel';
import type { TunnelInstance as _TunnelInstance, TunnelInstance__Output as _TunnelInstance__Output } from './TunnelInstance';

export interface GetTunnelResponse {
  'tunnel'?: (_Tunnel | null);
  'instances'?: (_TunnelInstance)[];
}

export interface GetTunnelResponse__Output {
  'tunnel'?: (_Tunnel__Output);
  'instances'?: (_TunnelInstance__Output)[];
}
