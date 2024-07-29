package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

/*
Refs
https://medium.com/@kdthedeveloper/golang-http-retries-fbf7abacbe27
https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
http batch
https://medium.com/@ggiovani/tcp-socket-implementation-on-golang-c38b67c5d8b
*/

// reuse your client for performance reasons
func httpClient() *http.Client {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	return client
}

func HTTPRequest(ctx context.Context, logger *zap.Logger, method, endpoint, token string, payload interface{}, queryParams, headers map[string]string) []byte {
	var req *http.Request
	var err error
	client := httpClient()

	// encode the payload
	bodyData, err := json.Marshal(payload)

	switch method {
	case http.MethodPost:
		if bodyData == nil {
			logger.Error("Unable to send Post request without Body")
		}
		req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(bodyData))
	case http.MethodGet:
		req, err = http.NewRequest(method, endpoint, nil)
		if bodyData == nil {
			logger.Error("Unable to send Post request without Body")
		}
		req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(bodyData))
	case http.MethodDelete:
		req, err = http.NewRequest(method, endpoint, nil)
	default:
		logger.Error("Request Unknown")
	}

	// set a request Context
	req = req.WithContext(ctx)
	if err != nil {
		logger.Error("Error Occurred")
	}

	// Request Headers
	// Basic Auth - https://swagger.io/docs/specification/authentication/basic-authentication/
	//username := os.Getenv("USERNAME")
	//password := os.Getenv("PASSWORD")
	//auth := fmt.Sprintf("%s:%s", username, password)
	//req.Header.Add("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))

	// Bearer Auth - https://swagger.io/docs/specification/authentication/bearer-authentication/
	if token != "" {
		req.Header.Add("Authorization", "Bearer "+token)
	}

	req.Header.Set("accept", "application/json")

	// API Key based Authentication - https://swagger.io/docs/specification/authentication/api-keys/
	// api key in request header
	//req.Header.Set("X-API-Key", os.Getenv("API-KEY"))
	//req.Header.Set("x-api-key", "api key value")

	// Additional Headers
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value) // Use Set not Add
		}
	}

	// Add Request Query Params if Any
	if queryParams != nil {
		for key, value := range queryParams {
			q := req.URL.Query()
			q.Add(key, value)
			req.URL.RawQuery = q.Encode()
		}
	}

	res, err := client.Do(req)
	if err != nil {
		logger.Error("Error occurred while making the http request", zap.Error(err))
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			logger.Error(fmt.Sprintf("failed to close processed request after transaction complete: %v", err.Error()))
		}
	}()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		logger.Error("Unable to Decode response")
	}

	return responseBody
}

func ReadRequestBody(w http.ResponseWriter, r *http.Request, destination interface{}) error {
	// specify the maximum number of request body to read
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(destination)

	if err != nil {
		// If there is an error during decoding, start the triage
		var (
			syntaxError           *json.SyntaxError
			unmarshalTypeError    *json.UnmarshalTypeError
			invalidUnmarshalError *json.InvalidUnmarshalError
		)

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON at character %d", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json unknown Field"):
			fieldName := strings.TrimPrefix(err.Error(), "json unknown field")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		case err.Error() == "http request body too large":
			return fmt.Errorf("body must not be larger than %d", maxBytes)
		default:
			return err
		}
	}
	defer r.Body.Close()

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
}

// - Reading QueryString key and Values

// ReadQueryStringKeyOfStringValue reads a string value from a query string key with value of type string
func ReadQueryStringKeyOfStringValue(qs url.Values, key, defaultValue string) string {
	value := qs.Get(key)

	if value == "" {
		return defaultValue
	}

	return value
}

// SetCookie Set Authorization Token in http cookie after user signed in for stateless cookies.
func SetCookie(w http.ResponseWriter, cookieValue, cookieName string, expirationTime time.Duration, setCookieInHeader bool) {
	cookie := &http.Cookie{
		Name:     cookieName,
		Value:    cookieValue,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}

	if expirationTime > 0 {
		cookie.Expires = time.Now().Add(expirationTime)
	} else {
		cookie.Expires = time.Now().Add(5 * time.Minute)
	}

	// How the stateless cookie is managed
	if setCookieInHeader {
		w.Header().Set("Set-Cookie", cookie.String())
		return
	}
	http.SetCookie(w, cookie)
}

// GetCookie retrieve token stored in cookie for a user with the name used to store the cookie
// Use the decode method on the TokenGenerator to Decode and return the claims on that token: https://github.com/Harphies/microservices/blob/main/golang-projects/microservices-toolkits/pkg/security/authorization/jwt-token.go#L66
func GetCookie(r *http.Request, cookieName string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", errors.New("no cookie set for this user")
		}
	}
	cookieValue := cookie.Value
	return cookieValue, nil
}

// ClearCookies clear all the stateless cookies specified to clear on the user browser
func ClearCookies(w http.ResponseWriter, cookiesNameList []string) {

	// Set expiration to a time in the past
	pastTime := time.Now().Add(-24 * time.Hour) // 24 hours ago

	// Define the cookie object
	cookie := &http.Cookie{
		Value:    "",
		Path:     "/",
		Expires:  pastTime,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	// handles each cookie clearing
	clearCookie := func(name string) {
		cookie.Name = name
		http.SetCookie(w, cookie)
	}

	// range all over the cookies to clear and call the function to clear each
	for _, cookieName := range cookiesNameList {
		clearCookie(cookieName)
	}
}

func SetValueInRequestContext(r *http.Request, key string, value interface{}) *http.Request {
	ctx := context.WithValue(r.Context(), key, value)
	return r.WithContext(ctx)
}

// GetValueFromRequestContext retrieve the user from the request context
func GetValueFromRequestContext(r *http.Request, key string) *interface{} {
	value, ok := r.Context().Value(key).(interface{})
	if !ok {
		panic(fmt.Sprintf("missing %s value in the request context", key))
	}

	return &value
}

// WriteJsonResponse is a utility function to help write Go structs to JSON response
func WriteJsonResponse(w http.ResponseWriter, r *http.Request, data interface{}, headers http.Header, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	if headers != nil {
		for key, value := range headers {
			w.Header()[key] = value
		}
	}
	resp, _ := json.MarshalIndent(data, "", "\t")
	w.Write(resp)
}

// mtlsClient ...
func mtlsClient() (*http.Client, error) {
	caCert, err := ioutil.ReadFile(os.Getenv("mTLS_CERT_FILE_PATH"))
	if err != nil {
		return nil, errors.New("reading ca certificate")
	}

	caCertPool := x509.NewCertPool() //x509.SystemCertPool()
	if err != nil {
		return nil, errors.New("creating system cert pool")
	}
	caCertPool.AppendCertsFromPEM(caCert)

	// Read the new key pair to create the certificate
	cert, err := tls.LoadX509KeyPair(os.Getenv("mTLS_CERT_FILE_PATH"), os.Getenv("mTLS_CERT_KEY_PATH"))
	if err != nil {
		return nil, errors.New("reading the key pair")
	}

	// Create an HTTPS client and supply the created CA pool and certificate
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}
	return client, nil
}

// MtlsRequest ...
func MtlsRequest(req *http.Request) (*http.Response, error) {
	// Create an HTTPS client and supply the created CA pool and certificate
	client, err := mtlsClient()
	if err != nil {
		return nil, errors.New("MTLS_client_creation_failed_error")
	}
	start := time.Now()

	r, err := client.Do(req)
	if err != nil {
		// record the request failed metrics
		return nil, errors.New("request_failed_error")
	}
	duration := time.Since(start)
	// log the request duration as Info log level
	fmt.Println("API request duration %v", duration)
	// record the success request metrics
	return r, nil
}

// GenerateBasicAuth ...
func GenerateBasicAuth(username, password string) string {
	auth := fmt.Sprintf("%s:%s", username, password)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func ExtractBearerToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("access token required")
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return parts[1], nil
}

// addCORSHeaders sets the CORS headers for the response
func addCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
