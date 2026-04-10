package runtime_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/sakurai-youhei/qrrun/internal/runtime"
)

func TestNew_KnownRuntime(t *testing.T) {
	known := []string{"pythonista", "pythonista2", "pythonista3"}
	for _, name := range known {
		rt, err := runtime.New(name)
		if err != nil {
			t.Fatalf("unexpected error for known runtime %q: %v", name, err)
		}
		if rt == nil {
			t.Fatalf("expected non-nil Runtime for %q", name)
		}
	}
}

func TestNew_UnknownRuntime(t *testing.T) {
	_, err := runtime.New("unknown-runtime")
	if err == nil {
		t.Fatal("expected error for unknown runtime, got nil")
	}
}

func TestPythonista_QRCodeURL_ExecScheme(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
	}{
		{name: "pythonista", scheme: "pythonista"},
		{name: "pythonista2", scheme: "pythonista2"},
		{name: "pythonista3", scheme: "pythonista3"},
	}

	rawURL := "https://example.trycloudflare.com/hello.py"

	for _, tc := range tests {
		rt, err := runtime.New(tc.name)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.name, err)
		}

		got := rt.QRCodeURL(rawURL)
		if !strings.HasPrefix(got, tc.scheme+"://") {
			t.Errorf("expected %s:// scheme, got %q", tc.scheme, got)
		}

		u, err := url.Parse(got)
		if err != nil {
			t.Fatalf("parse url for %q: %v", tc.name, err)
		}

		execCode := u.Query().Get("exec")
		if execCode == "" {
			t.Errorf("expected exec query parameter for %q, got %q", tc.name, got)
		}
		if !strings.Contains(execCode, "urllib.request.urlopen") {
			t.Errorf("expected urllib.request.urlopen in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, rawURL) {
			t.Errorf("expected raw URL in exec code for %q, got %q", tc.name, execCode)
		}
	}
}
