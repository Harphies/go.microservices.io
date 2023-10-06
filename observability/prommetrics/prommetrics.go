package prommetrics

import (
	"net/http"
	"net/http/pprof"
	"os"
)

var (
	// DefaultPromMetricsNamespace is the prefix for all prometheus metrics exported by your application services
	DefaultPromMetricsNamespace string = os.Getenv("APPLICATION_NAME")
)

// RegisterProfiler adds pprof endpoints to mux.
func RegisterProfiler(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
