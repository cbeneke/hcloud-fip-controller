package fipcontroller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestHealthzHandler(t *testing.T) {
	health := NewHealthServer(":0", logrus.New())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	health.healthzHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d but got %d", http.StatusOK, recorder.Code)
	}
}

func TestReadyzHandler(t *testing.T) {
	tests := []struct {
		name       string
		ready      bool
		wantStatus int
	}{
		{name: "not ready", ready: false, wantStatus: http.StatusServiceUnavailable},
		{name: "ready", ready: true, wantStatus: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			health := NewHealthServer(":0", logrus.New())
			health.SetReady(test.ready)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)

			health.readyzHandler(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Fatalf("expected status %d but got %d", test.wantStatus, recorder.Code)
			}
		})
	}
}
