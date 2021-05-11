package tunnel

import "time"

type ReverseTunnel struct {
	ID         int       `json:"id"`
	CreatedAt  time.Time `json:"createdAt"`
	PublicKey  string    `json:"publicKey"`
	Port       uint32    `json:"port"`
	SSHPort    uint32    `json:"sshPort"`
}
