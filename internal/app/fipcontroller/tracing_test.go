package fipcontroller

import (
	"context"
	"testing"
)

func TestInitTracingDisabledWhenNoEndpoint(t *testing.T) {
	shutdown, err := InitTracing(context.Background(), "", "test", "v0")
	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected a non-nil shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown should be a no-op but returned %v", err)
	}
}
