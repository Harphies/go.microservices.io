package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"go.uber.org/zap"
)

// reuse your client for performance reasons
func newHttpClient() *http.Client {
	trp := &http.Transport{
		Proxy:             http.ProxyFromEnvironment, // Get a proxy endpoint, if any, from the HTTP(S)_PROXY environment variables
		ForceAttemptHTTP2: false,                     // in case HTTP/2 is supported
		MaxConnsPerHost:   100,                       // 100 is the default
		DialContext: (&net.Dialer{
			Timeout:   2 * time.Minute,
			KeepAlive: 3 * time.Minute,
		}).DialContext,
		TLSHandshakeTimeout:   2 * time.Minute,
		ResponseHeaderTimeout: 3 * time.Minute,
		ExpectContinueTimeout: 1 * time.Minute,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{
		Timeout:   10 * time.Minute,
		Transport: trp,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	return client
}

// HTTPRequestWithTrace makes http request with http request tracing option for debugging
func HTTPRequestWithTrace(ctx context.Context, logger *zap.Logger, client *http.Client, method, endpoint, token string, payload interface{}, queryParams, headers map[string]string, enableTrace bool) ([]byte, error) {
	var body io.Reader
	if payload != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
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

	if enableTrace {
		trace := &httptrace.ClientTrace{
			GotConn: func(connInfo httptrace.GotConnInfo) {
				fmt.Printf("Connection established: reused=%v, wasIdle=%v, idleTime=%v\n",
					connInfo.Reused, connInfo.WasIdle, connInfo.IdleTime)
			},
			ConnectStart: func(network, addr string) {
				fmt.Printf("Dialing to %s\n", addr)
			},
			ConnectDone: func(network, addr string, err error) {
				if err != nil {
					fmt.Printf("Error connecting to %s: %v\n", addr, err)
				} else {
					fmt.Printf("Connected to %s\n", addr)
				}
			},
			GotFirstResponseByte: func() {
				fmt.Println("First response byte received")
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
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
		return nil, fmt.Errorf("error occurred while making the http request: %w", err)
	}
	defer func() {
		if cerr := res.Body.Close(); cerr != nil {
			logger.Error("Failed to close response body", zap.Error(cerr))
		}
	}()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return responseBody, nil
}

// shouldRetry implements http client retry logic
func shouldRetry(err error, resp *http.Response) bool {
	// drain the response body before closing the connection to re-use the connection for retry
	drainBody(resp)
	if err != nil {
		return true
	}

	if resp.StatusCode == http.StatusBadGateway ||
		resp.StatusCode == http.StatusServiceUnavailable ||
		resp.StatusCode == http.StatusGatewayTimeout {
		return true
	}
	return false
}

func drainBody(res *http.Response) {
	if res != nil || res.Body != nil {
		_, err := io.Copy(io.Discard, res.Body)
		err = res.Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %v", err)
		}
	}
}
