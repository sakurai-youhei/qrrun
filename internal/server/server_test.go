package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sakurai-youhei/qrrun/internal/server"
)

func TestServer_ServesScriptFile(t *testing.T) {
	// Write a temporary script file.
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "hello.py")
	content := "print('hello')\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	srv, err := server.New(scriptPath)
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
}

func TestServer_URLFormat(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "test.py")
	if err := os.WriteFile(scriptPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	srv, err := server.New(scriptPath)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	if !strings.HasPrefix(srv.URL(), "http://127.0.0.1:") {
		t.Errorf("unexpected URL: %q", srv.URL())
	}
	if !strings.HasSuffix(srv.ScriptURL(), "/test.py") {
		t.Errorf("unexpected ScriptURL: %q", srv.ScriptURL())
	}
}
