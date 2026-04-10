package transport

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
)

// Cloudflared uses the cloudflared quick-tunnel feature to expose a local URL.
// The cloudflared binary must be available on PATH.
type Cloudflared struct{}

// tunnelURLRe matches the public URL printed by cloudflared to stderr.
var tunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// Expose starts a cloudflared quick tunnel pointing at localURL and sends the
// resulting public URL to urlCh.  It returns when ctx is cancelled or the
// subprocess exits unexpectedly.
func (c *Cloudflared) Expose(ctx context.Context, localURL string, urlCh chan<- string) error {
	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", localURL)

	// cloudflared writes its tunnel URL to stderr.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cloudflared: pipe stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cloudflared: start: %w (is cloudflared installed?)", err)
	}

	urlSent := false
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if !urlSent {
			if m := tunnelURLRe.FindString(line); m != "" {
				urlCh <- m
				urlSent = true
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			// Cancelled by the caller — not an error.
			return nil
		}
		return fmt.Errorf("cloudflared: exited unexpectedly: %w", err)
	}
	return nil
}
