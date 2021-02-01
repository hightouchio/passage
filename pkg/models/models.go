package models

import "time"

type Tunnel struct {
	ID         string     `json:"id"`
	Type       TunnelType `json:"type"`
	CreatedAt  time.Time  `json:"createdAt"`
	PublicKey  string     `json:"publicKey"`
	PrivateKey string     `json:"privateKey"`
	Port       uint32     `json:"port"`
}

type TunnelType string

const (
	TunnelTypeNormal  = TunnelType("normal")
	TunnelTypeReverse = TunnelType("reverse")
)
