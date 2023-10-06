package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// reuse your client for performance reasons
// Red more about selection the right timeout - https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
func httpClient() *http.Client {
	client := &http.Client{Timeout: 10 * time.Second}
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
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Set("accept", "application/json")

	// API Key based Authentication - https://swagger.io/docs/specification/authentication/api-keys/
	// api key in request header
	//req.Header.Set("X-API-Key", os.Getenv("API-KEY"))
	//req.Header.Set("x-api-key", "api key value")

	// api key in request query string
	//req.URL.Query().Add("api-key", os.Getenv("API-KEY"))

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
		logger.Error("Error occurred while making the http request")
	}

	defer res.Body.Close()

	responseBody, err := ioutil.ReadAll(res.Body)
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
