package authentication

//
//import (
//	"context"
//	"errors"
//	"fmt"
//	circuitbreaker "github.com/mercari/go-circuitbreaker"
//	"go.opentelemetry.io/otel"
//	"go.opentelemetry.io/otel/trace"
//	"go.uber.org/zap"
//	"golang.org/x/crypto/bcrypt"
//	"time"
//)
//
//// NewUser ...
///*
//// https://github.com/hashicorp-demoapp/product-api-go/blob/main/handlers/user.go
//// https://github.com/hashicorp-demoapp/product-api-go/blob/main/handlers/auth.go
//// https://github.com/hashicorp-demoapp/hashicups-client-go/blob/main/auth.go
//// https://github.com/hashicorp-demoapp/product-api-go/blob/main/data/connection.go#L102
//// https://github.com/Harphies/microservices/blob/main/golang-projects/myservices/greenlight-api/cmd/api/user.go
//// https://github.com/hashicorp-demoapp/product-api-go/blob/main/handlers/auth.go#L54
//// https://www.alexedwards.net/blog/basic-authentication-in-go
//// https://www.sohamkamani.com/web-security-basics/#sessions-and-cookies
//// https://www.sohamkamani.com/golang/password-authentication-and-storage/
//// https://www.sohamkamani.com/golang/session-cookie-authentication/
//*/
//func NewUser(ctx context.Context, logger *zap.Logger, opts *UserOptions) (*User, error) {
//	return &User{
//		cb: circuitbreaker.New(
//			circuitbreaker.WithOpenTimeout(2*time.Minute),
//			circuitbreaker.WithTripFunc(circuitbreaker.NewTripFuncConsecutiveFailures(3)),
//			circuitbreaker.WithOnStateChangeHookFn(func(oldState, newState circuitbreaker.State) {
//				logger.Info("state changed",
//					zap.String("old", string(oldState)),
//					zap.String("new", string(newState)))
//			})),
//		db:    opts.DB,
//		cache: opts.Cache,
//	}, nil
//}
//
//// SignUp Sign up handler registers a new user.
//// This will be called from handler and pass the parsed <request body from user> to it.
//// This then pass the value passed from handler to interface implementations
//// And Interface implementations finally persist the data in DB down the stack.
//// Likewise up the stack from DB to <User for Response>.
//func (user *User) SignUp(username, password string) {
//	// Sign up a new user up and save record in DB
//
//}
//
//// SignIn authenticate a user
//func (user *User) SignIn() {
//	// Authenticate a user
//}
//
//// Set hash a password with bcrypt hashing algorithm
//func (user *User) Set(password string) {
//	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
//	if err != nil {
//		errors.New("hashing password failed")
//	}
//	user.plainPassword = password
//	user.hashPassword = hash
//	fmt.Println("hashed password", string(user.hashPassword))
//}
//
//// Matches checks whether the provided password match the stored hash password
//func (user *User) Matches(password string) (bool, error) {
//	err := bcrypt.CompareHashAndPassword(user.hashPassword, []byte(password))
//	if err != nil {
//		switch {
//		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
//			return false, errors.New("mismatched password")
//		default:
//			return false, err
//		}
//	}
//	return true, nil
//}
//
//func newOTELSpan(ctx context.Context, name string) trace.Span {
//	_, span := otel.Tracer("authorization").Start(ctx, name)
//
//	return span
//}
