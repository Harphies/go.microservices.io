package authentication

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"net/http"
	"time"
)

/*
https://github.com/chartmuseum/auth/blob/main/token.go
https://github.com/hashicorp-demoapp/product-api-go/blob/main/handlers/auth.go
https://www.sohamkamani.com/golang/jwt-authentication/
*/

type TokenClaims struct {
}

type TokenGenerationOptions struct {
	Password       string
	SecretKey      []byte
	PrivateKey     string
	PrivateKeyPath string
	PublicKey      string
	PublicKeyPath  string
	AwsKmsKeyID    string
	Cookie         bool
}

type TokenGenerator struct {
	Password       string
	SecretKey      []byte
	PrivateKey     string
	PrivateKeyPath string
	PublicKey      string
	PublicKeyPath  string
	SetCookie      bool
}

// JwtTokenProvider for JwtToken Interface Standardisation
type JwtTokenProvider interface {
	GenerateToken(expiration time.Duration, w http.ResponseWriter) (string, error)
	// DecodeToken or VerifyJWT
	DecodeToken(tokenString string) (*jwt.Token, error)
	RefreshToken(w http.ResponseWriter, r *http.Request) (string, error)
}

// NewJwtToken ...
func NewJwtToken(opts *TokenGenerationOptions) (*TokenGenerator, error) {

	// TODO: Retrieve the Private key from AWS SSM or from other storage options and use to sign the token
	return &TokenGenerator{
		Password:       opts.Password,
		SecretKey:      opts.SecretKey,
		PrivateKey:     opts.PrivateKey,
		PrivateKeyPath: opts.PrivateKeyPath,
		PublicKey:      opts.PublicKey,
		PublicKeyPath:  opts.PublicKeyPath,
		SetCookie:      opts.Cookie,
	}, nil
}

// DecodeToken returned decoded token with various options to decode
func (t *TokenGenerator) DecodeToken(tokenString string) (string, error) {
	var (
		tkn *jwt.Token
		err error
	)

	// Decode Token with RSA Public Key
	if t.PublicKey != "" || t.PublicKeyPath != "" {
		tkn, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return t.PublicKey, nil
		})
	}

	// Decode with symmetric secret Key
	tkn, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return t.SecretKey, nil
	})

	// TODO: Or Verify(decode) the Token With AWS KMS Key

	if err != nil {
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return "", errors.New("token signature invalid")
		}
	}
	if !tkn.Valid {
		return "", errors.New("token expired")
	}

	if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
		// Access the claims
		tokenExpr, err := claims.GetExpirationTime()
		if err != nil {
			return "", err
		}
		return tokenExpr.String(), nil
	}
	return "", errors.New("token expired")
}

//
//// GenerateToken ...
//func (t *TokenGenerator) GenerateToken(expiration time.Duration, w http.ResponseWriter) (string, error) {
//
//	var token *jwt.Token
//
//	standardClaims := jwt.StandardClaims{}
//
//	now := time.Now()
//	standardClaims.IssuedAt = now.Unix()
//
//	if expiration > 0 {
//		standardClaims.ExpiresAt = time.Now().Add(expiration).Unix()
//	}
//
//	// Sign the token with RSA Private Key and Signing Method: jwt.SigningMethodRS256
//	if t.PrivateKeyPath != "" || t.PrivateKey != "" {
//		token = jwt.New(jwt.SigningMethodRS256)
//		token.Claims = &TokenClaims{
//			StandardClaims: &standardClaims,
//		}
//		return token.SignedString(t.PrivateKey)
//	}
//
//	// Sign the Token with Symmetric Secret key and Signing Method: jwt.SigningMethodHS256
//	token = jwt.New(jwt.SigningMethodHS256)
//	token.Claims = &TokenClaims{
//		StandardClaims: &standardClaims,
//	}
//	signedToken, err := token.SignedString(t.SecretKey)
//
//	// TODO: Or Sign the Token With AWS KMS Key
//
//	// if SetCookie is true, add the cookie to request cookie for user as stateless token
//	if t.SetCookie == true && w != nil {
//		utils.SetCookie(w, signedToken, "refresh_token", 10*time.Second, false)
//	}
//
//	// TODO: Save token in database as stateful token
//
//	// use secret key to sign the token
//	return signedToken, err
//}

//// RefreshToken ... Returns a newly generated Token after old token have less than 30 seconds to expiry.
//func (t *TokenGenerator) RefreshToken(w http.ResponseWriter, r *http.Request, token string, expirationTime time.Duration) (string, error) {
//	var (
//		err error
//		tkn *jwt.Token
//	)
//	var oldToken string
//
//	claims := &TokenClaims{}
//
//	// Get the original Token from Cookie for Stateless Token if a token is not provided.
//	if t.SetCookie {
//		oldToken, err = utils.GetCookie(r, "refresh_token")
//		if err != nil {
//			return "", errors.New("no token retrieved")
//		}
//	}
//	//TODO: Get the Original Token to refresh from database for Stateful Token
//
//	// Use the token provided if any
//	if token != "" {
//		oldToken = token
//	}
//
//	tkn, err = jwt.ParseWithClaims(oldToken, claims, func(token *jwt.Token) (interface{}, error) {
//		return t.SecretKey, nil
//	})
//
//	if err != nil {
//		return "", errors.New("unauthorised")
//	}
//
//	// Check if the token is still valid
//	if !tkn.Valid {
//		return "", errors.New("invalid token")
//	}
//	// If old token expiration time is till more than 30 seconds, don't generate new token
//	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 30*time.Second {
//		return "", errors.New("token still have more time to expire")
//	}
//
//	// Otherwise, generate a new token
//	if expirationTime > 0 {
//		return t.GenerateToken(expirationTime, w)
//	}
//
//	return t.GenerateToken(5*time.Minute, w)
//}

// ValidateAWSCognitoAccessToken Validate a JWT Access Token generated by AWS Cognito
func ValidateAWSCognitoAccessToken(tokenString, jwksURL string) (*jwt.Token, *time.Time, error) {

	keySet, err := jwk.Fetch(context.Background(), jwksURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		key, found := keySet.LookupKeyID(kid)
		if !found {
			return nil, fmt.Errorf("key %v not found", kid)
		}

		var rawKey interface{}
		if err := key.Raw(&rawKey); err != nil {
			return nil, fmt.Errorf("failed to get raw key: %v", err)
		}

		return rawKey, nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return nil, nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, fmt.Errorf("failed to get token claims")
	}

	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return nil, nil, fmt.Errorf("failed to get expiration time")
	}

	expTime := time.Unix(int64(expFloat), 0).UTC()
	expTimeUTC1 := expTime.In(time.FixedZone("UTC+1", 1*60*60))
	fmt.Printf("Token Expiration in UTC+1: %v\n", expTimeUTC1)

	return token, &expTime, nil
}
