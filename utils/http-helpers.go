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
	"net"
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

const (
	defaultTimeout         = 10 * time.Second
	defaultMaxIdleConns    = 100
	defaultIdleConnTimeout = 90 * time.Second
)

// NewHTTPClient reuse your client for performance reasons
func NewHTTPClient(timeout time.Duration) *http.Client {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        defaultMaxIdleConns,
		IdleConnTimeout:     defaultIdleConnTimeout,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// HTTPRequest sends an HTTP request and returns the response body
func HTTPRequest(ctx context.Context, logger *zap.Logger, method, endpoint, token string, payload interface{}, queryParams, headers map[string]string) ([]byte, error) {
	client := NewHTTPClient(180 * time.Second)

	var body io.Reader
	if payload != nil {
		bodyData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(bodyData)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	q := req.URL.Query()
	for key, value := range queryParams {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			logger.Error("Failed to close response body", zap.Error(cerr))
		}
	}()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d with erro:%v", res.StatusCode, err.Error())
	}

	return responseBody, nil
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
