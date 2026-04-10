package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sakurai-youhei/qrrun/internal/server"
)

func TestServer_ServesScriptFile(t *testing.T) {
	content := "print('hello')\n"
	srv, err := server.New([]byte(content), "text/x-python; charset=utf-8")
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := srv.Serve(ctx); err != nil {
			// Only log — the test may have already cancelled the context.
			fmt.Println("server error:", err)
		}
	}()

	// Give the server a moment to start.
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get(srv.ScriptURL())
	if err != nil {
		t.Fatalf("GET %s: %v", srv.ScriptURL(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "print") {
		t.Errorf("unexpected body: %q", string(body))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/x-python; charset=utf-8" {
		t.Errorf("unexpected Content-Type: %q", ct)
	}
}

func TestServer_URLFormat(t *testing.T) {
	srv, err := server.New([]byte(""), "text/x-python; charset=utf-8")
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	if !strings.HasPrefix(srv.URL(), "http://127.0.0.1:") {
		t.Errorf("unexpected URL: %q", srv.URL())
	}

	scriptURL := srv.ScriptURL()
	if !strings.HasPrefix(scriptURL, srv.URL()+"/") {
		t.Errorf("unexpected ScriptURL prefix: %q", scriptURL)
	}

	id := strings.TrimPrefix(scriptURL, srv.URL()+"/")
	if strings.Contains(id, ".") {
		t.Errorf("expected extensionless script id, got %q", id)
	}
	if ok, _ := regexp.MatchString("^[a-f0-9]{32}$", id); !ok {
		t.Errorf("expected 32-char lowercase hex script id, got %q", id)
	}
}
