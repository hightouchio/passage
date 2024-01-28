// Original file: proto/service.proto


// Original file: proto/service.proto

export const _TunnelHealthcheck_Status = {
  WARNING: 0,
  PASSING: 1,
  CRITICAL: 2,
} as const;

export type _TunnelHealthcheck_Status =
  | 'WARNING'
  | 0
  | 'PASSING'
  | 1
  | 'CRITICAL'
  | 2

export type _TunnelHealthcheck_Status__Output = typeof _TunnelHealthcheck_Status[keyof typeof _TunnelHealthcheck_Status]

export interface TunnelHealthcheck {
  'id'?: (string);
  'status'?: (_TunnelHealthcheck_Status);
  'message'?: (string);
}

export interface TunnelHealthcheck__Output {
  'id'?: (string);
  'status'?: (_TunnelHealthcheck_Status__Output);
  'message'?: (string);
}
