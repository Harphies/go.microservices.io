package authentication

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/harphies/go.microservices.io/utils"
	"go.uber.org/zap"
	"net/http"
)

type OauthServiceProvider struct {
	clientID     string
	clientSecret string
	logger       *zap.Logger
}

type OauthAccessResponse struct {
	AccessToken  string `json:"access_token"`
	IdToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

// Oauth2ServiceProvider ... Oauth2.0 Service Provider Interface that can be implemented by
// any Oauth Service Provider.
type Oauth2ServiceProvider interface {
	GenerateToken(ctx context.Context, logger *zap.Logger, endpoint, code string) string
	GenerateCode(ctx context.Context, logger *zap.Logger, endpoint string) string
}

// NewOauthServiceProvider instantiate a new Oauth Service Provider.
func NewOauthServiceProvider(logger *zap.Logger, clientId, clientSecret string) *OauthServiceProvider {
	return &OauthServiceProvider{
		clientID:     clientId,
		clientSecret: clientSecret,
		logger:       logger,
	}
}

// GenerateTokenWithCode generates an access token with code grant flow.
// Takes code, client_id, client_secret as query Params.
func (oauth *OauthServiceProvider) GenerateTokenWithCode(ctx context.Context, endpoint string, qs, headers map[string]string) string {

	response, err := utils.HTTPRequest(ctx, oauth.logger, http.MethodPost, endpoint, "", nil, qs, headers)
	if err != nil {
		fmt.Println(err)
	}
	// Get the actual access token
	var resp OauthAccessResponse
	_ = json.Unmarshal(response, &resp)

	return resp.AccessToken
}

// GenerateToken generates an access token with client credentials grant flow.
// Takes client_id, client_secret as query Params.
func (oauth *OauthServiceProvider) GenerateToken(ctx context.Context, endpoint string, qs, headers map[string]string) string {

	response, err := utils.HTTPRequest(ctx, oauth.logger, http.MethodPost, endpoint, "", nil, qs, nil)
	if err != nil {
		fmt.Println(err)
	}
	// Get the actual access token
	var resp OauthAccessResponse
	_ = json.Unmarshal(response, &resp)

	return resp.AccessToken
}
