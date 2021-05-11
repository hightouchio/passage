package models

import "time"

type Tunnel struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"createdAt"`
	PublicKey       string    `json:"publicKey"`
	PrivateKey      string    `json:"privateKey"`
	Port            uint32    `json:"port"`
	ServerEndpoint  string    `json:"serverEndpoint"`
	ServerPort      uint32    `json:"serverPort"`
	ServiceEndpoint string    `json:"serviceEndpoint"`
	ServicePort     uint32    `json:"servicePort"`
}

type ReverseTunnel struct {
	ID         int       `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	PublicKey  string    `json:"publicKey"`
	Port       uint32    `json:"port"`
	SSHPort    uint32    `json:"sshPort"`
}
