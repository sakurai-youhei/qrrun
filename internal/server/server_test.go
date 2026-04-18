package server_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sakurai-youhei/qrrun/internal/server"
)

const testBearerToken = "test-bearer-token"

func TestServer_ServesScriptFile(t *testing.T) {
	content := "print('hello')\n"
	srv, err := server.New([]byte(content), "text/x-python; charset=utf-8", testBearerToken, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	client := startServerForTest(t, srv)

	resp, err := doRequest(client, http.MethodGet, srv.ScriptURL(), testBearerToken, nil)
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
	srv, err := server.New([]byte(""), "text/x-python; charset=utf-8", testBearerToken, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	if !strings.HasPrefix(srv.URL(), "https://127.0.0.1:") {
		t.Errorf("unexpected URL: %q", srv.URL())
	}
	if !strings.HasPrefix(srv.OriginURL(), "https://127.0.0.1:") {
		t.Errorf("unexpected OriginURL: %q", srv.OriginURL())
	}

	scriptURL := srv.ScriptURL()
	if !strings.HasPrefix(scriptURL, srv.URL()+"/") {
		t.Errorf("unexpected ScriptURL prefix: %q", scriptURL)
	}

	id := strings.TrimPrefix(scriptURL, srv.URL()+"/")
	if strings.Contains(id, ".") {
		t.Errorf("expected extensionless script id, got %q", id)
	}
	if ok, _ := regexp.MatchString("^[a-zA-Z0-9]{8}$", id); !ok {
		t.Errorf("expected 8-char alnum script id, got %q", id)
	}
}

func TestServer_FirstRequestServed_IsSignaled(t *testing.T) {
	srv, err := server.New([]byte("print('ok')\n"), "text/x-python; charset=utf-8", testBearerToken, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	client := startServerForTest(t, srv)

	select {
	case <-srv.FirstRequestServed():
		t.Fatal("first request signal should not be closed before serving any request")
	default:
	}

	resp, err := doRequest(client, http.MethodGet, srv.ScriptURL(), testBearerToken, nil)
	if err != nil {
		t.Fatalf("GET %s: %v", srv.ScriptURL(), err)
	}
	_ = resp.Body.Close()

	select {
	case <-srv.FirstRequestServed():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("first request signal was not closed after first request")
	}
}

func TestServer_FirstRequestServed_HeadDoesNotSignal(t *testing.T) {
	srv, err := server.New([]byte("print('ok')\n"), "text/x-python; charset=utf-8", testBearerToken, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	client := startServerForTest(t, srv)

	req, err := http.NewRequest(http.MethodHead, srv.ScriptURL(), nil)
	if err != nil {
		t.Fatalf("create HEAD request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testBearerToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HEAD %s: %v", srv.ScriptURL(), err)
	}
	_ = resp.Body.Close()

	select {
	case <-srv.FirstRequestServed():
		t.Fatal("first request signal should not be closed by HEAD")
	default:
	}
}

func TestServer_LogsAllRequests(t *testing.T) {
	var logs bytes.Buffer
	srv, err := server.New([]byte("print('ok')\n"), "text/x-python; charset=utf-8", testBearerToken, &logs)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	client := startServerForTest(t, srv)

	if _, err := doRequest(client, http.MethodGet, srv.ScriptURL(), testBearerToken, nil); err != nil {
		t.Fatalf("GET %s: %v", srv.ScriptURL(), err)
	}

	req, err := http.NewRequest(http.MethodHead, srv.ScriptURL(), nil)
	if err != nil {
		t.Fatalf("create HEAD request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testBearerToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HEAD %s: %v", srv.ScriptURL(), err)
	}
	_ = resp.Body.Close()

	req, err = http.NewRequest(http.MethodPost, srv.ScriptURL(), strings.NewReader("x"))
	if err != nil {
		t.Fatalf("create POST request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+testBearerToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", srv.ScriptURL(), err)
	}
	_ = resp.Body.Close()

	got := logs.String()
	if !strings.Contains(got, "method=GET") {
		t.Fatalf("expected GET log, got: %q", got)
	}
	if !strings.Contains(got, "method=HEAD") {
		t.Fatalf("expected HEAD log, got: %q", got)
	}
	if !strings.Contains(got, "method=POST") {
		t.Fatalf("expected POST log, got: %q", got)
	}
}

func TestServer_RejectsUnauthorizedRequest(t *testing.T) {
	srv, err := server.New([]byte("print('ok')\n"), "text/x-python; charset=utf-8", testBearerToken, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	client := startServerForTest(t, srv)

	start := time.Now()
	resp, err := doRequest(client, http.MethodGet, srv.ScriptURL(), "wrong-token", nil)
	if err != nil {
		t.Fatalf("GET %s: %v", srv.ScriptURL(), err)
	}
	defer resp.Body.Close()
	if elapsed := time.Since(start); elapsed < 950*time.Millisecond {
		t.Fatalf("expected unauthorized response delay around 1s, got %s", elapsed)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", resp.StatusCode)
	}

	select {
	case <-srv.FirstRequestServed():
		t.Fatal("first request signal should not be closed for unauthorized request")
	default:
	}
}

func TestServer_AcceptsQueryToken(t *testing.T) {
	srv, err := server.New([]byte("print('ok')\n"), "text/x-python; charset=utf-8", testBearerToken, io.Discard)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	client := startServerForTest(t, srv)

	queryURL := srv.ScriptURL() + "?t=" + url.QueryEscape(testBearerToken)
	resp, err := doRequestRawURL(client, http.MethodGet, queryURL, "", nil)
	if err != nil {
		t.Fatalf("GET %s: %v", queryURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func doRequest(client *http.Client, method, url, bearerToken string, body io.Reader) (*http.Response, error) {
	return doRequestRawURL(client, method, url, bearerToken, body)
}

func doRequestRawURL(client *http.Client, method, rawURL, bearerToken string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	return client.Do(req)
}

func startServerForTest(t *testing.T, srv *server.Server) *http.Client {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go func() {
		_ = srv.Serve(ctx)
	}()

	client := serverHTTPClient(t, srv)
	waitForServerReady(t, client, srv.URL())
	return client
}

func waitForServerReady(t *testing.T, client *http.Client, baseURL string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := doRequestRawURL(client, http.MethodGet, baseURL+"/", "", nil)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}

	if lastErr == nil {
		t.Fatal("server did not become ready in time")
	}
	t.Fatalf("server did not become ready in time: %v", lastErr)
}

func serverHTTPClient(t *testing.T, srv *server.Server) *http.Client {
	t.Helper()

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(srv.OriginCAPEM()); !ok {
		t.Fatal("failed to append origin CA cert")
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}
	return &http.Client{Transport: tr}
}
