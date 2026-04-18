// Package server provides a minimal HTTP file server that serves a single
// script file.
package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sakurai-youhei/qrrun/internal/randomid"
)

// Server is an in-memory single-script HTTP server.
type Server struct {
	scriptID    string
	bearerToken string
	scriptBytes []byte
	contentType string
	requestLog  io.Writer
	listener    net.Listener
	tlsConfig   *tls.Config
	originCAPEM []byte
	baseURL     string
	originURL   string
	cleanup     func()
	firstReqCh  chan struct{}
	deliveryCh  chan struct{}
	firstReq    sync.Once
}

// New creates a Server that serves scriptBytes on a random free port.
// The script is exposed at /<8-char-random-id>.
func New(scriptBytes []byte, contentType string, bearerToken string, requestLog io.Writer) (*Server, error) {
	ln, baseURL, originURL, cleanup, err := newOriginListener()
	if err != nil {
		return nil, err
	}
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			cleanup()
		}
	}()

	if strings.TrimSpace(bearerToken) == "" {
		return nil, fmt.Errorf("server: bearer token is required")
	}
	if requestLog == nil {
		requestLog = os.Stdout
	}

	tlsArtifacts, err := newOriginTLSConfig()
	if err != nil {
		return nil, err
	}

	scriptID, err := newScriptID()
	if err != nil {
		return nil, fmt.Errorf("server: script id: %w", err)
	}

	cleanupOnError = false

	return &Server{
		scriptID:    scriptID,
		bearerToken: bearerToken,
		scriptBytes: scriptBytes,
		contentType: contentType,
		requestLog:  requestLog,
		listener:    ln,
		tlsConfig:   tlsArtifacts.config,
		originCAPEM: tlsArtifacts.serverCAPEM,
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

// OriginCAPEM returns the PEM-encoded CA certificate used by the local origin.
func (s *Server) OriginCAPEM() []byte {
	return append([]byte(nil), s.originCAPEM...)
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
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		queryToken := strings.TrimSpace(r.URL.Query().Get("t"))
		if auth != "Bearer "+s.bearerToken && queryToken != s.bearerToken {
			time.Sleep(1 * time.Second)
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
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

	tlsListener := tls.NewListener(s.listener, s.tlsConfig)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(tlsListener)
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
	return randomid.AlphaNum(8)
}

type originTLSArtifacts struct {
	config      *tls.Config
	serverCAPEM []byte
}

func newOriginTLSConfig() (*originTLSArtifacts, error) {
	serverCAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("server: generate server ca key: %w", err)
	}

	newSerial := func() (*big.Int, error) {
		serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
		return rand.Int(rand.Reader, serialLimit)
	}

	now := time.Now().UTC()
	serverCASerial, err := newSerial()
	if err != nil {
		return nil, fmt.Errorf("server: generate server ca serial: %w", err)
	}

	serverCATmpl := &x509.Certificate{
		SerialNumber: serverCASerial,
		Subject: pkix.Name{
			CommonName: "qrrun-origin-server-ca",
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}
	serverCADer, err := x509.CreateCertificate(rand.Reader, serverCATmpl, serverCATmpl, &serverCAKey.PublicKey, serverCAKey)
	if err != nil {
		return nil, fmt.Errorf("server: create server ca cert: %w", err)
	}
	serverCAPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCADer})
	serverCAParsed, err := x509.ParseCertificate(serverCADer)
	if err != nil {
		return nil, fmt.Errorf("server: parse server ca cert: %w", err)
	}

	serverLeafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("server: generate server leaf key: %w", err)
	}
	serverLeafSerial, err := newSerial()
	if err != nil {
		return nil, fmt.Errorf("server: generate server leaf serial: %w", err)
	}
	serverLeafTmpl := &x509.Certificate{
		SerialNumber: serverLeafSerial,
		Subject: pkix.Name{
			CommonName: "qrrun-origin",
		},
		NotBefore:             now.Add(-5 * time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}
	serverLeafDer, err := x509.CreateCertificate(rand.Reader, serverLeafTmpl, serverCAParsed, &serverLeafKey.PublicKey, serverCAKey)
	if err != nil {
		return nil, fmt.Errorf("server: create server leaf cert: %w", err)
	}
	serverLeafPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverLeafDer})
	serverLeafKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverLeafKey)})
	serverTLSCert, err := tls.X509KeyPair(serverLeafPEM, serverLeafKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("server: load server keypair: %w", err)
	}

	return &originTLSArtifacts{
		config: &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
			MinVersion:   tls.VersionTLS12,
		},
		serverCAPEM: serverCAPEM,
	}, nil
}
