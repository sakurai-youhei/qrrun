// Package app wires together the server, transport and runtime, then renders
// the QR code.
package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"rsc.io/qr"

	"github.com/sakurai-youhei/qrrun/internal/runtime"
	"github.com/sakurai-youhei/qrrun/internal/server"
	"github.com/sakurai-youhei/qrrun/internal/transport"
)

// Options holds the configuration for a single QRrun invocation.
type Options struct {
	TransportName   string
	RuntimeName     string
	QRErrorLevel    string
	ScriptPath      string
	ScriptArgs      []string
	KeepServing     bool
	ExitQuietPeriod time.Duration
	Debug           bool
	TransportStdout bool
	TransportStderr bool
	PrintURL        bool
	CloudflaredOpts []string
	Input           io.Reader // source for script content when ScriptPath is "-"; defaults to os.Stdin
	Output          io.Writer // destination for the QR code; defaults to os.Stdout
}

const DefaultExitQuietPeriod = 500 * time.Millisecond

const alphaNumChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const DefaultQRErrorLevel = "M"

var SupportedQRErrorLevels = []string{"L", "M", "Q", "H"}

var qrLevelByName = map[string]qr.Level{
	"L": qr.L,
	"M": qr.M,
	"Q": qr.Q,
	"H": qr.H,
}

func SupportedQRErrorLevelsText() string {
	return strings.Join(SupportedQRErrorLevels, ", ")
}

// Run performs the full QRrun workflow:
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
	qrLevel, err := parseQRErrorLevel(opts.QRErrorLevel)
	if err != nil {
		return err
	}
	statusOutput := io.Writer(os.Stderr)
	quietPeriod := opts.ExitQuietPeriod
	if quietPeriod <= 0 {
		quietPeriod = DefaultExitQuietPeriod
	}

	t, err := transport.New(opts.TransportName)
	if err != nil {
		return err
	}
	if cf, ok := t.(*transport.Cloudflared); ok {
		if cf.CommandLog == nil {
			cf.CommandLog = statusOutput
		}
		cf.LogCommand = opts.Debug
		cf.ExtraArgs = append([]string(nil), opts.CloudflaredOpts...)
		cf.LogStdout = opts.TransportStdout
		cf.LogStderr = opts.TransportStderr
		cf.LogConfigSet = true
		if opts.PrintURL {
			cf.LogStdout = false
			cf.LogStderr = false
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

	requestLog := io.Writer(io.Discard)
	if opts.Debug {
		requestLog = statusOutput
	}

	srv, err := server.New(scriptBytes, "text/x-python; charset=utf-8", bearerToken, requestLog)
	if err != nil {
		return err
	}

	if cf, ok := t.(*transport.Cloudflared); ok {
		caPEM := srv.OriginCAPEM()
		if len(caPEM) > 0 {
			caFile, err := os.CreateTemp("", "qrrun-origin-ca-*.pem")
			if err != nil {
				return fmt.Errorf("create origin ca file: %w", err)
			}
			defer func() {
				_ = os.Remove(caFile.Name())
			}()
			if _, err := caFile.Write(caPEM); err != nil {
				_ = caFile.Close()
				return fmt.Errorf("write origin ca file: %w", err)
			}
			if err := caFile.Close(); err != nil {
				return fmt.Errorf("close origin ca file: %w", err)
			}
			cf.OriginCAPoolPath = caFile.Name()
		}
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
	scriptArgv := make([]string, 0, 1+len(opts.ScriptArgs))
	scriptArgv = append(scriptArgv, opts.ScriptPath)
	scriptArgv = append(scriptArgv, opts.ScriptArgs...)
	scriptURLWithToken := withQueryToken(replaceBase(srv.ScriptURL(), publicURL), bearerToken)
	scriptPublicURL := rt.QRCodeURL(
		scriptURLWithToken,
		bearerToken,
		scriptArgv,
	)

	if opts.PrintURL {
		fmt.Fprintln(opts.Output, scriptPublicURL)
	} else {
		if err := renderCompactQRCode(opts.Output, scriptPublicURL, qrLevel); err != nil {
			return fmt.Errorf("render qr: %w", err)
		}
		if opts.KeepServing {
			fmt.Fprintf(statusOutput, "\nKeep-serving mode enabled. Press Ctrl+C to stop.\n")
		} else {
			fmt.Fprintf(statusOutput, "\nDefault mode: QRrun exits after the last successful content delivery (quiet period: %s).\n", quietPeriod)
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
	b := make([]byte, 8)
	for i := range b {
		idx, err := randomAlphaNumIndex(len(alphaNumChars))
		if err != nil {
			return "", err
		}
		b[i] = alphaNumChars[idx]
	}
	return string(b), nil
}

func randomAlphaNumIndex(max int) (int, error) {
	if max <= 0 || max > 256 {
		return 0, fmt.Errorf("invalid max: %d", max)
	}
	limit := byte(256 - (256 % max))
	for {
		var one [1]byte
		if _, err := rand.Read(one[:]); err != nil {
			return 0, err
		}
		if one[0] < limit {
			return int(one[0]) % max, nil
		}
	}
}

func withQueryToken(rawURL, token string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := parsed.Query()
	q.Set("t", token)
	parsed.RawQuery = q.Encode()
	return parsed.String()
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
	rawParsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	publicParsed, err := url.Parse(publicBase)
	if err != nil {
		return rawURL
	}
	rawParsed.Scheme = publicParsed.Scheme
	rawParsed.Host = publicParsed.Host
	return rawParsed.String()
}

func parseQRErrorLevel(level string) (qr.Level, error) {
	normalized := strings.ToUpper(strings.TrimSpace(level))
	if normalized == "" {
		normalized = DefaultQRErrorLevel
	}
	if mapped, ok := qrLevelByName[normalized]; ok {
		return mapped, nil
	}
	return 0, fmt.Errorf("invalid qr level %q: must be one of %s", level, SupportedQRErrorLevelsText())
}

func renderCompactQRCode(w io.Writer, content string, level qr.Level) error {
	code, err := qr.Encode(content, level)
	if err != nil {
		return err
	}

	const quietZone = 1
	size := code.Size + 2*quietZone

	for y := 0; y < size; y += 2 {
		var line strings.Builder
		line.Grow(size)
		for x := 0; x < size; x++ {
			top := qrBlackAt(code, x-quietZone, y-quietZone)
			bottom := qrBlackAt(code, x-quietZone, y+1-quietZone)
			line.WriteRune(compactBlock(top, bottom))
		}
		if _, err := fmt.Fprintln(w, line.String()); err != nil {
			return err
		}
	}
	return nil
}

func qrBlackAt(code *qr.Code, x, y int) bool {
	if x < 0 || y < 0 || x >= code.Size || y >= code.Size {
		return false
	}
	return code.Black(x, y)
}

func compactBlock(top, bottom bool) rune {
	switch {
	case top && bottom:
		return '█'
	case top:
		return '▀'
	case bottom:
		return '▄'
	default:
		return ' '
	}
}
