// Package server provides a minimal HTTP file server that serves a single
// script file.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
)

// Server is an in-memory single-script HTTP server.
type Server struct {
	scriptID    string
	scriptBytes []byte
	contentType string
	requestLog  io.Writer
	listener    net.Listener
	baseURL     string
	originURL   string
	cleanup     func()
	firstReqCh  chan struct{}
	deliveryCh  chan struct{}
	firstReq    sync.Once
}

// New creates a Server that serves scriptBytes on a random free port.
// The script is exposed at /<uuid-without-extension>.
func New(scriptBytes []byte, contentType string, requestLog io.Writer) (*Server, error) {
	ln, baseURL, originURL, cleanup, err := newOriginListener()
	if err != nil {
		return nil, err
	}
	if requestLog == nil {
		requestLog = os.Stdout
	}

	scriptID, err := newScriptID()
	if err != nil {
		return nil, fmt.Errorf("server: script id: %w", err)
	}

	return &Server{
		scriptID:    scriptID,
		scriptBytes: scriptBytes,
		contentType: contentType,
		requestLog:  requestLog,
		listener:    ln,
		baseURL:     baseURL,
		originURL:   originURL,
		cleanup:     cleanup,
		firstReqCh:  make(chan struct{}),
		deliveryCh:  make(chan struct{}, 32),
	}, nil
}

// URL returns the local HTTP base URL used for script URL composition.
func (s *Server) URL() string {
	return s.baseURL
}

// OriginURL returns the local origin URL for cloudflared.
func (s *Server) OriginURL() string {
	return s.originURL
}

// ScriptURL returns the full URL for the served script file.
func (s *Server) ScriptURL() string {
	return s.URL() + "/" + s.scriptID
}

// FirstRequestServed is closed once the script endpoint is served for the first time.
func (s *Server) FirstRequestServed() <-chan struct{} {
	return s.firstReqCh
}

// DeliveryEvents receives an event each time script content is successfully written.
func (s *Server) DeliveryEvents() <-chan struct{} {
	return s.deliveryCh
}

// Serve starts the HTTP server and blocks until ctx is cancelled or an
// unrecoverable error occurs.
func (s *Server) Serve(ctx context.Context) error {
	defer s.cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("/"+s.scriptID, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(s.requestLog, "request: method=%s path=%s remote=%s\n", r.Method, r.URL.Path, r.RemoteAddr)
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
		if _, err := w.Write(s.scriptBytes); err == nil {
			s.firstReq.Do(func() {
				close(s.firstReqCh)
			})
			select {
			case s.deliveryCh <- struct{}{}:
			default:
			}
		}
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
