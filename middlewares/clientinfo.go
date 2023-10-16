package middlewares

import (
	"fmt"
	"github.com/tomasen/realip"
	"go.uber.org/zap"
	"net/http"
)

// ClientInfo ...
/*
Each log contains information such as the time the request was received,
the client's IP address, latencies, request paths, and server responses.
You can use these access logs to analyze traffic patterns and troubleshoot issues.
*/
func ClientInfo(next http.Handler, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userIp := realip.FromRequest(r)
		logger.Info(fmt.Sprintf("Client with IP %s is making a %v request to %s ", userIp, r.Method, r.URL.Path))
		next.ServeHTTP(w, r)
	})
}
