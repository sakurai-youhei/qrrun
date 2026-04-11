package transport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

// Cloudflared uses the cloudflared quick-tunnel feature to expose a local URL.
// The cloudflared binary must be available on PATH.
type Cloudflared struct {
	CommandLog io.Writer
	ExtraArgs  []string
	LogStdout  bool
	LogStderr  bool
	LogConfigSet bool
}

// tunnelURLRe matches the public URL printed by cloudflared logs.
var tunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// Expose starts a cloudflared quick tunnel pointing at localURL and sends the
// resulting public URL to urlCh.  It returns when ctx is cancelled or the
// subprocess exits unexpectedly.
func (c *Cloudflared) Expose(ctx context.Context, localURL string, urlCh chan<- string) error {
	args := c.buildArgs(localURL)
	fmt.Fprintf(c.commandLogWriter(), "transport command: cloudflared %s\n", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "cloudflared", args...)

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
	var outputCapture bytes.Buffer
	outputCaptureMax := 128 * 1024

	appendCapture := func(line string) {
		if outputCapture.Len() >= outputCaptureMax {
			return
		}
		remaining := outputCaptureMax - outputCapture.Len()
		if len(line)+1 > remaining {
			if remaining > 1 {
				outputCapture.WriteString(line[:remaining-1])
				outputCapture.WriteByte('\n')
			}
			return
		}
		outputCapture.WriteString(line)
		outputCapture.WriteByte('\n')
	}

	scanOutput := func(r io.Reader, shouldLog bool) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			if shouldLog {
				fmt.Fprintf(c.commandLogWriter(), "cloudflared: %s\n", line)
			}
			appendCapture(line)
			if m := tunnelURLRe.FindString(line); m != "" {
				select {
				case foundURLCh <- m:
				default:
				}
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		scanOutput(stdout, c.logStdoutEnabled())
	}()
	go func() {
		defer wg.Done()
		scanOutput(stderr, c.logStderrEnabled())
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
		if outputCapture.Len() == 0 {
			return fmt.Errorf("cloudflared: exited unexpectedly: %w", err)
		}
		return fmt.Errorf("cloudflared: exited unexpectedly: %w\ncloudflared output:\n%s", err, strings.TrimSpace(outputCapture.String()))
	}
	<-scanDoneCh

	if !urlSent {
		if outputCapture.Len() == 0 {
			return fmt.Errorf("cloudflared: quick tunnel URL not found in output")
		}
		return fmt.Errorf("cloudflared: quick tunnel URL not found in output\ncloudflared output:\n%s", strings.TrimSpace(outputCapture.String()))
	}
	return nil
}

func (c *Cloudflared) buildArgs(localURL string) []string {
	args := []string{"tunnel"}
	if len(c.ExtraArgs) > 0 {
		args = append(args, c.ExtraArgs...)
	}
	if strings.HasPrefix(localURL, "unix://") {
		socketPath := strings.TrimPrefix(localURL, "unix://")
		// Use unix socket URL directly so cloudflared does not fall back to localhost:80.
		return append(args, "--url", "unix:"+socketPath)
	}
	return append(args, "--url", localURL)
}

func (c *Cloudflared) commandLogWriter() io.Writer {
	if c.CommandLog == nil {
		return os.Stdout
	}
	return c.CommandLog
}

func (c *Cloudflared) logStdoutEnabled() bool {
	if !c.LogConfigSet {
		// Default behavior when not configured explicitly: print both streams.
		return true
	}
	return c.LogStdout
}

func (c *Cloudflared) logStderrEnabled() bool {
	if !c.LogConfigSet {
		// Default behavior when not configured explicitly: print both streams.
		return true
	}
	return c.LogStderr
}
