package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestMaterializeStdinScript(t *testing.T) {
	path, cleanup, err := materializeStdinScript(strings.NewReader("print('ok')\n"))
	if err != nil {
		t.Fatalf("materializeStdinScript: %v", err)
	}
	t.Cleanup(cleanup)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp file: %v", err)
	}
	if string(content) != "print('ok')\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}
	if filepath.Ext(path) != ".py" {
		t.Fatalf("expected .py extension, got %q", path)
	}
}

func TestMaterializeStdinScript_ReadError(t *testing.T) {
	path, cleanup, err := materializeStdinScript(errorReader{})
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatal("expected read error")
	}
	if path != "" {
		t.Fatalf("expected empty path on error, got %q", path)
	}
	if cleanup != nil {
		t.Fatal("expected nil cleanup on error")
	}
}
