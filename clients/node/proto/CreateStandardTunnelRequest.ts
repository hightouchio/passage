// Original file: proto/service.proto


export interface CreateStandardTunnelRequest {
  'sshHost'?: (string);
  'sshPort'?: (number);
  'sshUser'?: (string);
  'serviceHost'?: (string);
  'servicePort'?: (number);
  'privateKeys'?: (string)[];
  'createKeyPair'?: (boolean);
  '_sshUser'?: "sshUser";
}

export interface CreateStandardTunnelRequest__Output {
  'sshHost'?: (string);
  'sshPort'?: (number);
  'sshUser'?: (string);
  'serviceHost'?: (string);
  'servicePort'?: (number);
  'privateKeys'?: (string)[];
  'createKeyPair'?: (boolean);
}
