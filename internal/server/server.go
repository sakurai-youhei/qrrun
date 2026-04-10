// Package server provides a minimal HTTP file server that serves a single
// script file.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// Server is an in-memory single-script HTTP server.
type Server struct {
	scriptID    string
	scriptBytes []byte
	contentType string
	listener    net.Listener
}

// New creates a Server that serves scriptBytes on a random free port.
// The script is exposed at /<uuid-without-extension>.
func New(scriptBytes []byte, contentType string) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("server: listen: %w", err)
	}

	scriptID, err := newScriptID()
	if err != nil {
		return nil, fmt.Errorf("server: script id: %w", err)
	}

	return &Server{
		scriptID:    scriptID,
		scriptBytes: scriptBytes,
		contentType: contentType,
		listener:    ln,
	}, nil
}

// URL returns the base URL of this server (e.g. "http://127.0.0.1:54321").
func (s *Server) URL() string {
	return "http://" + s.listener.Addr().String()
}

// ScriptURL returns the full URL for the served script file.
func (s *Server) ScriptURL() string {
	return s.URL() + "/" + s.scriptID
}

// Serve starts the HTTP server and blocks until ctx is cancelled or an
// unrecoverable error occurs.
func (s *Server) Serve(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/"+s.scriptID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", s.contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(s.scriptBytes)))
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(s.scriptBytes)
	})

	srv := &http.Server{Handler: mux}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(s.listener)
	}()

	select {
	case <-ctx.Done():
		if shutdownErr := srv.Shutdown(context.Background()); shutdownErr != nil {
			return fmt.Errorf("server: shutdown: %w", shutdownErr)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server: serve: %w", err)
	}
}

func newScriptID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	// 32 hex chars, extensionless UUID-like identifier suitable for URL path.
	return hex.EncodeToString(raw), nil
}
