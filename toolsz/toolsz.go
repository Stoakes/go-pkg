package toolsz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	// Revision git commit short
	Revision = "unknown"
	// Branch branch name
	Branch = "unknown"
	// BuildUser username@hostname
	BuildUser = "unknown"
	// Date build date
	Date = "unknown"
	// GoVersion golang version used for build
	GoVersion = "unknown"
)

// ToolsServer is a basic & extensible tools server to expose operations tooling
type ToolsServer interface {
	// Start is a blocking call starting tools server
	Start(ctx context.Context) error
}

// ToolsServer handles tools exposed by tools service.
// these tools make operations easier
type toolsServer struct {
	port        int
	zapLogLevel zap.AtomicLevel
	handlers    *map[string]http.Handler // warning on handler order, see note on New
	server      *http.Server
}

// New creates a tools server
// warning, there's no guarantee on handlers order. It's caller duty to select URLs that do no collision
func New(port int, zapLogLevel zap.AtomicLevel, customHandlers *map[string]http.Handler) ToolsServer {
	return &toolsServer{
		port:        port,
		zapLogLevel: zapLogLevel,
		handlers:    customHandlers,
	}
}

// Start starts the tools server. (blocking call)
func (t *toolsServer) Start(ctx context.Context) error {
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/logz", t.zapLogLevel)
	http.Handle("/info", versionHandler())
	if t.handlers != nil {
		for url, handler := range *t.handlers {
			http.Handle(url, handler)
		}
	}
	t.server = &http.Server{
		Addr:    ":" + strconv.Itoa(t.port),
		Handler: http.DefaultServeMux,
	}

	if err := t.server.ListenAndServe(); err != nil {
		return fmt.Errorf("Tools server stopped: " + err.Error())
	}
	select {
	case <-ctx.Done():
		return t.server.Shutdown(ctx)
	}
}

// versionHandler http handler returing build information as JSON
func versionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "application/json")

		res, err := json.Marshal(map[string]string{
			"revision":   Revision,
			"branch":     Branch,
			"build_user": BuildUser,
			"date":       Date,
			"go_version": GoVersion,
		})
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		if _, err := w.Write(res); err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	})
}
