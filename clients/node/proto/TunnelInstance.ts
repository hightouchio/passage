// Original file: proto/service.proto

import type { _TunnelHealthcheck_Status, _TunnelHealthcheck_Status__Output } from './TunnelHealthcheck';
import type { TunnelHealthcheck as _TunnelHealthcheck, TunnelHealthcheck__Output as _TunnelHealthcheck__Output } from './TunnelHealthcheck';

export interface TunnelInstance {
  'host'?: (string);
  'port'?: (number);
  'status'?: (_TunnelHealthcheck_Status);
  'healthchecks'?: (_TunnelHealthcheck)[];
}

export interface TunnelInstance__Output {
  'host'?: (string);
  'port'?: (number);
  'status'?: (_TunnelHealthcheck_Status__Output);
  'healthchecks'?: (_TunnelHealthcheck__Output)[];
}
