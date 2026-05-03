package runtime_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/sakurai-youhei/qrrun/internal/runtime"
)

func TestNew_KnownRuntime(t *testing.T) {
	known := []string{"ashell", "pythonista2", "pythonista3"}
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

func TestAshell_QRCodeURL(t *testing.T) {
	rt, err := runtime.New("ashell")
	if err != nil {
		t.Fatalf("unexpected error for ashell: %v", err)
	}

	got := rt.QRCodeURL("https://example.com/run.sh?t=test-token", "ignored-token", []string{"-", "arg1", "arg2"})
	if !strings.HasPrefix(got, "ashell:") {
		t.Fatalf("expected ashell scheme, got %q", got)
	}
	if strings.HasPrefix(got, "ashell://") {
		t.Fatalf("ashell URL must use ashell: (without //), got %q", got)
	}

	wantDecoded := "curl -sSL 'https://example.com/run.sh?t=test-token'|sh -s -- arg1 arg2"
	payload := strings.TrimPrefix(got, "ashell:")
	if payload != wantDecoded {
		t.Fatalf("unexpected ashell payload: got %q, want %q", payload, wantDecoded)
	}
	if strings.Contains(payload, " - arg1") {
		t.Fatalf("script path placeholder must not be forwarded as shell arg: %q", payload)
	}
}

func TestNew_UnknownRuntime(t *testing.T) {
	tests := []string{"unknown-runtime", "pythonista"}
	for _, name := range tests {
		_, err := runtime.New(name)
		if err == nil {
			t.Fatalf("expected error for unknown runtime %q, got nil", name)
		}
	}
}

func TestPythonista_QRCodeURL_ExecScheme(t *testing.T) {
	tests := []struct {
		name      string
		scheme    string
		isPython2 bool
	}{
		{name: "pythonista2", scheme: "pythonista2", isPython2: true},
		{name: "pythonista3", scheme: "pythonista3", isPython2: false},
	}

	rawURL := "https://example.trycloudflare.com/hello.py"
	bearerToken := "test-token-123"
	scriptArgv := []string{"hello.py", "arg1", "arg2"}

	for _, tc := range tests {
		rt, err := runtime.New(tc.name)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.name, err)
		}

		got := rt.QRCodeURL(rawURL, bearerToken, scriptArgv)
		if !strings.HasPrefix(got, tc.scheme+"://") {
			t.Errorf("expected %s:// scheme, got %q", tc.scheme, got)
		}
		if strings.Contains(got, "\n") || strings.Contains(got, "\r") {
			t.Errorf("expected single-line URL for %q, got %q", tc.name, got)
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
		if !strings.Contains(execCode, "a=sys.argv[:]") {
			t.Errorf("expected sys.argv backup in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, "sys.argv=[\"hello.py\",\"arg1\",\"arg2\"]") {
			t.Errorf("expected script argv overwrite in exec code for %q, got %q", tc.name, execCode)
		}
		if tc.isPython2 {
			if !strings.Contains(execCode, "import sys,urllib2 as u") {
				t.Errorf("expected urllib2 import in exec code for %q, got %q", tc.name, execCode)
			}
			if !strings.Contains(execCode, "exec(u.urlopen(") || !strings.Contains(execCode, ").read(),{\"__name__\":\"__main__\"})") {
				t.Errorf("expected Python2 exec(...) call in exec code for %q, got %q", tc.name, execCode)
			}
			if strings.Contains(execCode, ".decode()") {
				t.Errorf("did not expect decode() in python2 exec code for %q, got %q", tc.name, execCode)
			}
		} else {
			if !strings.Contains(execCode, "import sys,urllib.request as u") {
				t.Errorf("expected urllib.request import in exec code for %q, got %q", tc.name, execCode)
			}
			if !strings.Contains(execCode, "exec(u.urlopen(") || !strings.Contains(execCode, ".read().decode()") {
				t.Errorf("expected Python3 exec(u.urlopen(...).read().decode()) in exec code for %q, got %q", tc.name, execCode)
			}
		}
		if !strings.Contains(execCode, "finally:") || !strings.Contains(execCode, "sys.argv=a") {
			t.Errorf("expected sys.argv restore in finally for %q, got %q", tc.name, execCode)
		}
		if strings.Contains(execCode, "requests") {
			t.Errorf("did not expect requests dependency in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(execCode, rawURL) {
			t.Errorf("expected raw URL in exec code for %q, got %q", tc.name, execCode)
		}
		if !strings.Contains(rawExec, "%") {
			t.Errorf("expected encoded exec query for %q, got %q", tc.name, rawExec)
		}
		if strings.Contains(rawExec, " ") {
			t.Errorf("expected raw exec query to avoid literal spaces for %q, got %q", tc.name, rawExec)
		}
		if strings.Contains(rawExec, "+") {
			t.Errorf("expected spaces to be encoded as %%20 (no '+') for %q, got %q", tc.name, rawExec)
		}
		if !strings.Contains(rawExec, "%20") {
			t.Errorf("expected raw exec query to encode spaces as %%20 for %q, got %q", tc.name, rawExec)
		}
	}
}
