// Original file: proto/service.proto


export interface _Tunnel_ReverseTunnel {
}

export interface _Tunnel_ReverseTunnel__Output {
}

export interface _Tunnel_StandardTunnel {
  'sshHost'?: (string);
  'sshPort'?: (number);
  'sshUser'?: (string);
  'serviceHost'?: (string);
  'servicePort'?: (number);
}

export interface _Tunnel_StandardTunnel__Output {
  'sshHost'?: (string);
  'sshPort'?: (number);
  'sshUser'?: (string);
  'serviceHost'?: (string);
  'servicePort'?: (number);
}

// Original file: proto/service.proto

export const _Tunnel_Type = {
  STANDARD: 0,
  REVERSE: 1,
} as const;

export type _Tunnel_Type =
  | 'STANDARD'
  | 0
  | 'REVERSE'
  | 1

export type _Tunnel_Type__Output = typeof _Tunnel_Type[keyof typeof _Tunnel_Type]

export interface Tunnel {
  'id'?: (string);
  'type'?: (_Tunnel_Type);
  'enabled'?: (boolean);
  'bindPort'?: (number);
  'reverseTunnel'?: (_Tunnel_ReverseTunnel | null);
  'standardTunnel'?: (_Tunnel_StandardTunnel | null);
  'tunnel'?: "standardTunnel"|"reverseTunnel";
}

export interface Tunnel__Output {
  'id'?: (string);
  'type'?: (_Tunnel_Type__Output);
  'enabled'?: (boolean);
  'bindPort'?: (number);
  'reverseTunnel'?: (_Tunnel_ReverseTunnel__Output);
  'standardTunnel'?: (_Tunnel_StandardTunnel__Output);
}
