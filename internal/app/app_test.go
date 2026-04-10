package app_test

import (
	"io"
	"testing"

	"github.com/sakurai-youhei/qrrun/internal/app"
)

func TestRun_InvalidTransport(t *testing.T) {
	err := app.Run(app.Options{
		TransportName: "invalid-transport",
		RuntimeName:   "pythonista3",
		ScriptPath:    "script.py",
		Output:        io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for invalid transport")
	}
}

func TestRun_InvalidRuntime(t *testing.T) {
	err := app.Run(app.Options{
		TransportName: "cloudflared",
		RuntimeName:   "invalid-runtime",
		ScriptPath:    "script.py",
		Output:        io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for invalid runtime")
	}
}

func TestRun_InvalidScriptPath(t *testing.T) {
	// app.Run now validates script existence before starting transport.
	err := app.Run(app.Options{
		TransportName: "cloudflared",
		RuntimeName:   "pythonista3",
		ScriptPath:    "/nonexistent/path/script.py",
		Output:        io.Discard,
	})
	if err == nil {
		t.Fatal("expected error for invalid script path")
	}
}
