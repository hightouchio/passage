package tunnel

import (
	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"io"
)

// wrapSigner wraps a signer and overrides its public key type with the provided algorithm
type wrapSigner struct {
	ssh.Signer
	algorithm string
}

// PublicKey returns an associated PublicKey instance.
func (s *wrapSigner) PublicKey() gossh.PublicKey {
	return &wrapPublicKey{
		PublicKey: s.Signer.PublicKey(),
		algorithm: s.algorithm,
	}
}

// Sign returns raw signature for the given data. This method
// will apply the hash specified for the keytype to the data using
// the algorithm assigned for this key
func (s *wrapSigner) Sign(rand io.Reader, data []byte) (*gossh.Signature, error) {
	return s.Signer.(gossh.AlgorithmSigner).SignWithAlgorithm(rand, data, s.algorithm)
}

// wrapPublicKey wraps a PublicKey and overrides its type
type wrapPublicKey struct {
	gossh.PublicKey
	algorithm string
}

// Type returns the algorithm
func (k *wrapPublicKey) Type() string {
	return k.algorithm
}
