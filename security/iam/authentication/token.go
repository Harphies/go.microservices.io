package authentication

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"time"
)

// TokenProvider Interface for JWT, Oauth2.0 & OIDC Token Management between practice-projects
type TokenProvider interface {
	Generate(ctx context.Context, ttl time.Duration, scope interface{}) (*Token, error)
	Validate(ctx context.Context, token Token) (bool, error)
}

// Token Implementation
// Token is the structure of token returned when generated
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// NewToken return an instance of a token
func NewToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	// Create a Token Instance
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	return token, nil
}

// Generate ...
func (t *Token) Generate(ctx context.Context, ttl time.Duration, scope string) (*Token, error) {
	// Initiate a zero-value byte slice with a length of 16 bytes
	randomBytes := make([]byte, 16)

	token := &Token{}

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}

// Validate ...
func (t *Token) Validate(ctx context.Context, token Token) (bool, error) {

	return true, nil
}
