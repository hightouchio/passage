package models

import "time"

type Tunnel struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	PublicKey      string    `json:"publicKey"`
	PrivateKey     string    `json:"privateKey"`
	Port           uint32    `json:"port"`
	ServiceEndpont string    `json:"serviceEndpoint"`
	ServicePort    uint32    `json:"servicePort"`
}

type ReverseTunnel struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	PublicKey  string    `json:"publicKey"`
	PrivateKey string    `json:"privateKey"`
	Port       uint32    `json:"port"`
	SSHPort    uint32    `json:"sshPort"`
}
