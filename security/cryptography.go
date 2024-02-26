package security

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
)

// GenerateKeyPair Generate a Private and Public Key pair for encryption and Decryption

func GenerateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Generate Private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, errors.New("unable to generate private key")
	}

	// generate templates key
	publicKey := privateKey.PublicKey

	return privateKey, &publicKey, nil
}
