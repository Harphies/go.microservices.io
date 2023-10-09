package authentication

import (
	"context"
	"encoding/json"
	"github.com/harphies/go.microservices.io/utils"
	"go.uber.org/zap"
	"net/http"
)

type (
	QueryParams map[string]string
)

type OauthServiceProvider struct {
	clientID     string
	clientSecret string
	redirectUri  string
	logger       *zap.Logger
}
type OauthAccessResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// Oauth2ServiceProvider ... Oauth2.0 Service Provider Interface that can be implemented by
// any Oauth Service Provider.
type Oauth2ServiceProvider interface {
	GenerateToken(ctx context.Context, logger *zap.Logger, endpoint, code string) string
	GenerateCode(ctx context.Context, logger *zap.Logger, endpoint string) string
}

// NewOauthServiceProvider instantiate a new Oauth Service Provider.
func NewOauthServiceProvider(logger *zap.Logger, clientId, clientSecret, redirectUri string) *OauthServiceProvider {
	return &OauthServiceProvider{
		clientID:     clientId,
		clientSecret: clientSecret,
		redirectUri:  redirectUri,
		logger:       logger,
	}
}

// GenerateCode generates a code to be used to generate access token
// Takes client_id & redirect_uri as query parameters in the request
func (oauth *OauthServiceProvider) GenerateCode(ctx context.Context, redirectUri, endpoint string) string {
	queryParams := QueryParams{
		"client_id":     oauth.clientID,
		"client_secret": oauth.clientSecret,
		"redirect_uri":  redirectUri,
	}
	response := utils.HTTPRequest(ctx, oauth.logger, http.MethodGet, endpoint, "", nil, queryParams, nil)
	return string(response)
}

// GenerateTokenWithCode generates an access token with code grant flow.
// Takes code, client_id, client_secret as query Params.
func (oauth *OauthServiceProvider) GenerateTokenWithCode(ctx context.Context, endpoint, code string) string {
	queryParams := QueryParams{
		"client_id":     oauth.clientID,
		"client_secret": oauth.clientSecret,
		"code":          code,
	}
	response := utils.HTTPRequest(ctx, oauth.logger, http.MethodPost, endpoint, "", nil, queryParams, nil)
	// Get the actual access token
	var resp OauthAccessResponse
	_ = json.Unmarshal(response, &resp)

	return resp.AccessToken
}

// GenerateToken generates an access token with client credentials grant flow.
// Takes client_id, client_secret as query Params.
func (oauth *OauthServiceProvider) GenerateToken(ctx context.Context, endpoint string) string {
	queryParams := QueryParams{
		"client_id":     oauth.clientID,
		"client_secret": oauth.clientSecret,
	}
	response := utils.HTTPRequest(ctx, oauth.logger, http.MethodPost, endpoint, "", nil, queryParams, nil)
	// Get the actual access token
	var resp OauthAccessResponse
	_ = json.Unmarshal(response, &resp)

	return resp.AccessToken
}
