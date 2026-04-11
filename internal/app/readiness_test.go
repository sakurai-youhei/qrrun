package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitForPublicOriginReady_SucceedsAfterTransient502(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	if err := waitForPublicOriginReady(context.Background(), ts.URL, "token", 3*time.Second); err != nil {
		t.Fatalf("waitForPublicOriginReady: %v", err)
	}
}

func TestWaitForPublicOriginReady_FailsForPersistentBadStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	err := waitForPublicOriginReady(context.Background(), ts.URL, "token", 400*time.Millisecond)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
