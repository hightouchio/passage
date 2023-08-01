package tunnel

import (
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
)

// getSignersForPrivateKey returns a list of ssh.Signers for the provided private key
func getSignersForPrivateKey(privateKeyBytes []byte) ([]ssh.Signer, error) {
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return []ssh.Signer{}, errors.Wrap(err, "could not parse private key")
	}

	return []ssh.Signer{
		signer, // Original signer
		wrapSigner{signer, ssh.SigAlgoRSASHA2256}, // Signer with SHA2-256 algorithm
		wrapSigner{signer, ssh.SigAlgoRSASHA2512}, // Signer with SHA2-512 algorithm
	}, nil
}

// WrapSigner wraps a signer and overrides its public key type with the provided algorithm
type wrapSigner struct {
	ssh.Signer
	algorithm string
}

// PublicKey returns an associated PublicKey instance.
func (s wrapSigner) PublicKey() ssh.PublicKey {
	return &wrapPublicKey{
		PublicKey: s.Signer.PublicKey(),
		algorithm: s.algorithm,
	}
}

// Sign returns raw signature for the given data. This method
// will apply the hash specified for the keytype to the data using
// the algorithm assigned for this key
func (s wrapSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	return s.Signer.(ssh.AlgorithmSigner).SignWithAlgorithm(rand, data, s.algorithm)
}

// wrapPublicKey wraps a PublicKey and overrides its type
type wrapPublicKey struct {
	ssh.PublicKey
	algorithm string
}

// Type returns the algorithm
func (k *wrapPublicKey) Type() string {
	return k.algorithm
}
