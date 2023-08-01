package tunnel

import (
	"crypto/rand"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
	"testing"
)

// Test our key pair generation and signers
func TestSSHKeySigning(t *testing.T) {
	// Generate key pair
	keyPair, err := GenerateKeyPair()

	// Get public key
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(keyPair.PublicKey)
	if err != nil {
		t.Error(errors.Wrap(err, "parse public key"))
		return
	}

	// Get Auth Signers
	signers, err := getSignersForPrivateKey(keyPair.PrivateKey)
	if err != nil {
		t.Error(errors.Wrap(err, "get auth signers"))
		return
	}

	var observedSignatureAlgorithms []string

	// Test each signer
	for _, signer := range signers {
		signatureAlgorithm := signer.PublicKey().Type()
		observedSignatureAlgorithms = append(observedSignatureAlgorithms, signatureAlgorithm)

		t.Logf("Verify signature %s", signatureAlgorithm)
		signature, err := signers[0].Sign(rand.Reader, []byte("hello world"))
		if err != nil {
			t.Error(errors.Wrapf(err, "Sign message for key %s", signatureAlgorithm))
			return
		}

		if err := publicKey.Verify([]byte("hello world"), signature); err != nil {
			t.Error(errors.Wrapf(err, "Verify message for key %s", signatureAlgorithm))
			return
		}
	}

	// Assert that our supported signature algorithms were present
	assert.EqualValues(t, observedSignatureAlgorithms, []string{"ssh-rsa", "rsa-sha2-256", "rsa-sha2-512"})
}
