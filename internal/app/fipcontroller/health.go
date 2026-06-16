package fipcontroller

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthServer exposes liveness (/healthz) and readiness (/readyz) HTTP
// endpoints used by Kubernetes probes.
//
// Liveness reports whether the process is up and able to serve requests.
// Readiness reports whether the controller has finished its initialisation
// and is participating in leader election.
type HealthServer struct {
	server *http.Server
	logger *logrus.Logger
	ready  atomic.Bool
}

// NewHealthServer creates a HealthServer listening on the given address.
func NewHealthServer(address string, logger *logrus.Logger) *HealthServer {
	health := &HealthServer{logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health.healthzHandler)
	mux.HandleFunc("/readyz", health.readyzHandler)

	health.server = &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return health
}

// SetReady toggles the readiness state reported by the /readyz endpoint.
func (health *HealthServer) SetReady(ready bool) {
	health.ready.Store(ready)
}

func (health *HealthServer) healthzHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}

func (health *HealthServer) readyzHandler(writer http.ResponseWriter, _ *http.Request) {
	if !health.ready.Load() {
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte("not ready"))
		return
	}
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}

// Run starts the health server and blocks until the context is cancelled or
// the server fails. When the context is cancelled the server is shut down
// gracefully.
func (health *HealthServer) Run(ctx context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		health.logger.Infof("Starting health server on %s", health.server.Addr)
		if err := health.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return health.server.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
