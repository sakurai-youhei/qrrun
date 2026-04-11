// Package app wires together the server, transport and runtime, then renders
// the QR code.
package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"

	"github.com/sakurai-youhei/qrrun/internal/runtime"
	"github.com/sakurai-youhei/qrrun/internal/server"
	"github.com/sakurai-youhei/qrrun/internal/transport"
)

// Options holds the configuration for a single qrrun invocation.
type Options struct {
	TransportName   string
	RuntimeName     string
	ScriptPath      string
	KeepServing     bool
	ExitQuietPeriod time.Duration
	URLOnly         bool
	Input           io.Reader // source for script content when ScriptPath is "-"; defaults to os.Stdin
	Output          io.Writer // destination for the QR code; defaults to os.Stdout
}

const DefaultExitQuietPeriod = 500 * time.Millisecond

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
	quietPeriod := opts.ExitQuietPeriod
	if quietPeriod <= 0 {
		quietPeriod = DefaultExitQuietPeriod
	}

	t, err := transport.New(opts.TransportName)
	if err != nil {
		return err
	}
	if opts.URLOnly {
		if cf, ok := t.(*transport.Cloudflared); ok {
			cf.CommandLog = io.Discard
		}
	}

	rt, err := runtime.New(opts.RuntimeName)
	if err != nil {
		return err
	}

	scriptBytes, err := loadScriptContent(opts.ScriptPath, opts.Input)
	if err != nil {
		return err
	}

	bearerToken, err := generateBearerToken()
	if err != nil {
		return fmt.Errorf("generate bearer token: %w", err)
	}

	requestLog := io.Writer(os.Stdout)
	if opts.URLOnly {
		requestLog = io.Discard
	}

	srv, err := server.New(scriptBytes, "text/x-python; charset=utf-8", bearerToken, requestLog)
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
		transportErrCh <- t.Expose(ctx, srv.OriginURL(), urlCh)
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
	scriptPublicURL := rt.QRCodeURL(replaceBase(srv.ScriptURL(), publicURL), bearerToken)

	if opts.URLOnly {
		fmt.Fprintln(opts.Output, scriptPublicURL)
	} else {
		fmt.Fprintf(opts.Output, "\nScan the QR code below with your phone:\n\n")
		qrterminal.GenerateWithConfig(scriptPublicURL, qrterminal.Config{
			Level:     qrterminal.M,
			Writer:    opts.Output,
			BlackChar: qrterminal.BLACK,
			WhiteChar: qrterminal.WHITE,
			QuietZone: 1,
		})
		if opts.KeepServing {
			fmt.Fprintf(opts.Output, "\nURL: %s\n\nKeep-serving mode enabled. Press Ctrl+C to stop.\n", scriptPublicURL)
		} else {
			fmt.Fprintf(opts.Output, "\nURL: %s\n\nDefault mode: qrrun exits after the last successful content delivery (quiet period: %s).\n", scriptPublicURL, quietPeriod)
		}
	}

	if !opts.KeepServing {
		// Default mode: wait for successful delivery, then exit after a quiet period.
		timer := time.NewTimer(24 * time.Hour)
		timer.Stop()
		defer timer.Stop()
		hasDelivery := false

		select {
		case <-ctx.Done():
			return nil
		case <-srv.DeliveryEvents():
			hasDelivery = true
			timer.Reset(quietPeriod)
		case err := <-serverErrCh:
			if err != nil {
				return err
			}
			return nil
		case err := <-transportErrCh:
			if err != nil {
				return err
			}
			return nil
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-srv.DeliveryEvents():
				hasDelivery = true
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(quietPeriod)
			case <-timer.C:
				if hasDelivery {
					return nil
				}
			case err := <-serverErrCh:
				if err != nil {
					return err
				}
				return nil
			case err := <-transportErrCh:
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	// Multi-request mode: keep running until cancelled.
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

func generateBearerToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func loadScriptContent(scriptPath string, input io.Reader) ([]byte, error) {
	if scriptPath == "-" {
		b, err := io.ReadAll(input)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return b, nil
	}

	b, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("script path: %w", err)
	}
	return b, nil
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
