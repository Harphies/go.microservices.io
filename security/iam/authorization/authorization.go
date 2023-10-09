package authorization

import "regexp"

var (
	BasicAuthorizerType  AuthorizerType = "basic"
	BearerAuthorizerType AuthorizerType = "bearer"

	bearerTokenMatch = regexp.MustCompile("(?i)bearer (.*)")
)

type (
	AuthorizerType string

	Authorizer struct {
		Type string
	}

	AuthorizerOptions struct {
		Username string
		Password string
	}

	Permission struct {
		Allowed bool
	}
)

//// NewAuthorizer return an Instance of Authorizer
//func NewAuthorizer(opts AuthorizerOptions) (*Authorizer, error) {
//
//}
