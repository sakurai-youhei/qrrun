package transport_test

import (
	"testing"

	"github.com/sakurai-youhei/qrrun/internal/transport"
)

func TestNew_KnownTransport(t *testing.T) {
	tr, err := transport.New("cloudflared")
	if err != nil {
		t.Fatalf("unexpected error for known transport: %v", err)
	}
	if tr == nil {
		t.Fatal("expected non-nil Transport")
	}
}

func TestNew_UnknownTransport(t *testing.T) {
	_, err := transport.New("unknown-transport")
	if err == nil {
		t.Fatal("expected error for unknown transport, got nil")
	}
}
