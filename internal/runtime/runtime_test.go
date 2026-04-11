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
	bearerToken := "test-token-123"

	for _, tc := range tests {
		rt, err := runtime.New(tc.name)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.name, err)
		}

		got := rt.QRCodeURL(rawURL, bearerToken)
		if !strings.HasPrefix(got, tc.scheme+"://") {
			t.Errorf("expected %s:// scheme, got %q", tc.scheme, got)
		}

		prefix := tc.scheme + "://?exec="
		if !strings.HasPrefix(got, prefix) {
			t.Errorf("expected %q prefix for %q, got %q", prefix, tc.name, got)
		}

		rawExec := strings.TrimPrefix(got, prefix)
		if rawExec == "" {
			t.Errorf("expected exec query parameter for %q, got %q", tc.name, got)
		}
		execCode, err := url.QueryUnescape(rawExec)
		if err != nil {
			t.Fatalf("failed to decode exec code for %q: %v (raw: %q)", tc.name, err, rawExec)
		}
		if !strings.HasPrefix(execCode, "exec(__import__(\"requests\").get(") {
			t.Errorf("expected single-expression __import__ prefix in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, ").text)") {
			t.Errorf("expected .text execution suffix in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, "Authorization") {
			t.Errorf("expected Authorization header in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, "Bearer "+bearerToken) {
			t.Errorf("expected Bearer token in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, rawURL) {
			t.Errorf("expected raw URL in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(rawExec, "%") {
			t.Errorf("expected encoded exec query for %q, got %q", tc.name, rawExec)
		}
		if strings.Contains(rawExec, "+") {
			t.Errorf("expected spaces to remain literal spaces (no '+') for %q, got %q", tc.name, rawExec)
		}
		if !strings.Contains(rawExec, " ") {
			t.Errorf("expected raw exec query to contain a literal space for %q, got %q", tc.name, rawExec)
		}
	}
}
