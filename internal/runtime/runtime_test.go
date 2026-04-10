package runtime_test

import (
	"strings"
	"testing"

	"github.com/sakurai-youhei/qrrun/internal/runtime"
)

func TestNew_KnownRuntime(t *testing.T) {
	rt, err := runtime.New("pythonista3")
	if err != nil {
		t.Fatalf("unexpected error for known runtime: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil Runtime")
	}
}

func TestNew_UnknownRuntime(t *testing.T) {
	_, err := runtime.New("unknown-runtime")
	if err == nil {
		t.Fatal("expected error for unknown runtime, got nil")
	}
}

func TestPythonista3_QRCodeURL(t *testing.T) {
	rt, err := runtime.New("pythonista3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rawURL := "https://example.trycloudflare.com/hello.py"
	got := rt.QRCodeURL(rawURL)

	if !strings.HasPrefix(got, "pythonista3://") {
		t.Errorf("expected pythonista3:// scheme, got %q", got)
	}
	if !strings.Contains(got, "importscript") {
		t.Errorf("expected importscript host, got %q", got)
	}
	if !strings.Contains(got, "url=") {
		t.Errorf("expected url= query param, got %q", got)
	}
	if !strings.Contains(got, "example.trycloudflare.com") {
		t.Errorf("expected original host in URL, got %q", got)
	}
}
