// Package server provides a minimal HTTP file server that serves a single
// script file.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
)

// Server is a single-file HTTP server.
type Server struct {
	scriptPath string
	listener   net.Listener
}

// New creates a Server that serves scriptPath on a random free port.
func New(scriptPath string) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("server: listen: %w", err)
	}
	return &Server{scriptPath: scriptPath, listener: ln}, nil
}

// URL returns the base URL of this server (e.g. "http://127.0.0.1:54321").
func (s *Server) URL() string {
	return "http://" + s.listener.Addr().String()
}

// ScriptURL returns the full URL for the served script file.
func (s *Server) ScriptURL() string {
	return s.URL() + "/" + filepath.Base(s.scriptPath)
}

// Serve starts the HTTP server and blocks until ctx is cancelled or an
// unrecoverable error occurs.
func (s *Server) Serve(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/"+filepath.Base(s.scriptPath), http.FileServer(http.Dir(filepath.Dir(s.scriptPath))))

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
