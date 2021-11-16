package tunnel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"golang.org/x/crypto/ssh"
)

type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

func (k KeyPair) Base64PrivateKey() string {
	return base64.StdEncoding.EncodeToString(k.PrivateKey)
}

const clientKeyPairBits = 2048

func GenerateKeyPair() (KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, clientKeyPairBits)
	if err != nil {
		return KeyPair{}, err
	}

	if err = privateKey.Validate(); err != nil {
		return KeyPair{}, err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return KeyPair{}, err
	}

	return KeyPair{
		PublicKey: ssh.MarshalAuthorizedKey(publicKey),
		PrivateKey: pem.EncodeToMemory(&pem.Block{
			Type:    "RSA PRIVATE KEY",
			Headers: nil,
			Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
		}),
	}, nil
}
