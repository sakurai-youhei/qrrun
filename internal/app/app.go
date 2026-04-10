// Package app wires together the server, transport and runtime, then renders
// the QR code.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mdp/qrterminal/v3"

	"github.com/sakurai-youhei/qrrun/internal/runtime"
	"github.com/sakurai-youhei/qrrun/internal/server"
	"github.com/sakurai-youhei/qrrun/internal/transport"
)

// Options holds the configuration for a single qrrun invocation.
type Options struct {
	TransportName string
	RuntimeName   string
	ScriptPath    string
	Input         io.Reader // source for script content when ScriptPath is "-"; defaults to os.Stdin
	Output        io.Writer // destination for the QR code; defaults to os.Stdout
}

// Run performs the full qrrun workflow:
//  1. Validates options and resolves the transport / runtime.
//  2. Starts a local HTTP server to serve the script.
//  3. Starts the tunnel via the chosen transport.
//  4. Prints the QR code to opts.Output once the public URL is known.
//  5. Blocks until the process receives SIGINT / SIGTERM.
func Run(opts Options) error {
	if opts.Input == nil {
		opts.Input = os.Stdin
	}
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	t, err := transport.New(opts.TransportName)
	if err != nil {
		return err
	}

	rt, err := runtime.New(opts.RuntimeName)
	if err != nil {
		return err
	}

	scriptPath := opts.ScriptPath
	if opts.ScriptPath == "-" {
		var cleanup func()
		scriptPath, cleanup, err = materializeStdinScript(opts.Input)
		if err != nil {
			return err
		}
		defer cleanup()
	}

	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script path: %w", err)
	}

	srv, err := server.New(scriptPath)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start the local HTTP server.
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Serve(ctx)
	}()

	// Start the tunnel and wait for the public URL.
	urlCh := make(chan string, 1)
	transportErrCh := make(chan error, 1)
	go func() {
		transportErrCh <- t.Expose(ctx, srv.URL(), urlCh)
	}()

	// Wait for the public URL or an early error.
	var publicURL string
	select {
	case publicURL = <-urlCh:
	case err := <-transportErrCh:
		return fmt.Errorf("transport: %w", err)
	case err := <-serverErrCh:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
		return nil
	}

	// Build the QR code URL: replace the local base URL with the public one,
	// then let the runtime wrap it in the appropriate URL scheme.
	scriptPublicURL := rt.QRCodeURL(replaceBase(srv.ScriptURL(), publicURL))

	fmt.Fprintf(opts.Output, "\nScan the QR code below with your phone:\n\n")
	qrterminal.GenerateWithConfig(scriptPublicURL, qrterminal.Config{
		Level:     qrterminal.M,
		Writer:    opts.Output,
		BlackChar: qrterminal.BLACK,
		WhiteChar: qrterminal.WHITE,
		QuietZone: 1,
	})
	fmt.Fprintf(opts.Output, "\nURL: %s\n\nPress Ctrl+C to stop.\n", scriptPublicURL)

	// Block until cancelled.
	select {
	case <-ctx.Done():
	case err := <-serverErrCh:
		if err != nil {
			return err
		}
	case err := <-transportErrCh:
		if err != nil {
			return err
		}
	}
	return nil
}

func materializeStdinScript(input io.Reader) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "qrrun-stdin-*.py")
	if err != nil {
		return "", nil, fmt.Errorf("create temp script: %w", err)
	}

	if _, err := io.Copy(tmpFile, input); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("read stdin: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("close temp script: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(tmpFile.Name())
	}
	return tmpFile.Name(), cleanup, nil
}

// replaceBase swaps the local base URL in rawURL with publicBase.
func replaceBase(rawURL, publicBase string) string {
	// rawURL starts with the local server URL (e.g. "http://127.0.0.1:PORT/…").
	// We swap the scheme+host+port portion with publicBase.
	const scheme = "http://"
	if len(rawURL) < len(scheme) {
		return rawURL
	}
	// Find end of host:port
	rest := rawURL[len(scheme):]
	slashIdx := len(rest)
	for i, c := range rest {
		if c == '/' {
			slashIdx = i
			break
		}
	}
	return publicBase + rest[slashIdx:]
}
