package tunnel

import "time"

type NormalTunnel struct {
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
