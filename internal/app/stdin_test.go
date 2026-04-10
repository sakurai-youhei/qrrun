package app

import (
	"errors"
	"os"
	"strings"
	"testing"
)

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read error")
}

func TestLoadScriptContent_FromStdin(t *testing.T) {
	content, err := loadScriptContent("-", strings.NewReader("print('ok')\n"))
	if err != nil {
		t.Fatalf("loadScriptContent: %v", err)
	}
	if string(content) != "print('ok')\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestLoadScriptContent_StdinReadError(t *testing.T) {
	content, err := loadScriptContent("-", errorReader{})
	if err == nil {
		t.Fatal("expected read error")
	}
	if content != nil {
		t.Fatalf("expected nil content on error, got %q", string(content))
	}
	if !strings.Contains(err.Error(), "read stdin") {
		t.Fatalf("expected read stdin error, got %v", err)
	}
}

func TestLoadScriptContent_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/hello.py"
	if err := os.WriteFile(path, []byte("print('file')\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	content, err := loadScriptContent(path, strings.NewReader("ignored"))
	if err != nil {
		t.Fatalf("loadScriptContent: %v", err)
	}
	if string(content) != "print('file')\n" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestLoadScriptContent_FileReadError(t *testing.T) {
	content, err := loadScriptContent("/nonexistent/path/script.py", strings.NewReader("ignored"))
	if err == nil {
		t.Fatal("expected script path error")
	}
	if content != nil {
		t.Fatalf("expected nil content on error, got %q", string(content))
	}
	if !strings.Contains(err.Error(), "script path") {
		t.Fatalf("expected script path error, got %v", err)
	}
}
