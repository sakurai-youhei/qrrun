package transport

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sync"
)

// Cloudflared uses the cloudflared quick-tunnel feature to expose a local URL.
// The cloudflared binary must be available on PATH.
type Cloudflared struct{}

// tunnelURLRe matches the public URL printed by cloudflared logs.
var tunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// Expose starts a cloudflared quick tunnel pointing at localURL and sends the
// resulting public URL to urlCh.  It returns when ctx is cancelled or the
// subprocess exits unexpectedly.
func (c *Cloudflared) Expose(ctx context.Context, localURL string, urlCh chan<- string) error {
	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", localURL)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cloudflared: pipe stdout: %w", err)
	}

	// cloudflared usually writes logs to stderr, but this may vary by version.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cloudflared: pipe stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cloudflared: start: %w (is cloudflared installed?)", err)
	}

	foundURLCh := make(chan string, 1)

	scanOutput := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if m := tunnelURLRe.FindString(scanner.Text()); m != "" {
				select {
				case foundURLCh <- m:
				default:
				}
				return
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanOutput(stdout)
	}()
	go func() {
		defer wg.Done()
		scanOutput(stderr)
	}()

	scanDoneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(scanDoneCh)
	}()

	urlSent := false
waitURL:
	for {
		select {
		case u := <-foundURLCh:
			urlCh <- u
			urlSent = true
			break waitURL
		case <-scanDoneCh:
			// Best-effort final read in case URL and completion raced.
			select {
			case u := <-foundURLCh:
				urlCh <- u
				urlSent = true
			default:
			}
			break waitURL
		case <-ctx.Done():
			_ = cmd.Wait()
			<-scanDoneCh
			return nil
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			// Cancelled by the caller — not an error.
			return nil
		}
		return fmt.Errorf("cloudflared: exited unexpectedly: %w", err)
	}
	<-scanDoneCh

	if !urlSent {
		return fmt.Errorf("cloudflared: quick tunnel URL not found in output")
	}
	return nil
}
