package transport

import (
	"reflect"
	"testing"
)

func TestTunnelURLRe_Match(t *testing.T) {
	line := "INF +--------------------------------------------------------------------------------------------+ https://fancy-moon-1234.trycloudflare.com"
	got := tunnelURLRe.FindString(line)
	want := "https://fancy-moon-1234.trycloudflare.com"
	if got != want {
		t.Fatalf("unexpected URL match: got %q, want %q", got, want)
	}
}

func TestTunnelURLRe_NoMatch(t *testing.T) {
	cases := []string{
		"https://example.com",
		"https://foo.trycloudflare.net",
		"no url here",
	}
	for _, line := range cases {
		if got := tunnelURLRe.FindString(line); got != "" {
			t.Fatalf("expected no match for %q, got %q", line, got)
		}
	}
}

func TestCloudflaredBuildArgs_WithExtraOptions(t *testing.T) {
	c := &Cloudflared{ExtraArgs: []string{"--loglevel", "debug"}}
	got := c.buildArgs("http://127.0.0.1:12345")
	want := []string{"tunnel", "--loglevel", "debug", "--url", "http://127.0.0.1:12345"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v, want %#v", got, want)
	}
}

func TestCloudflaredBuildArgs_UnixOrigin(t *testing.T) {
	c := &Cloudflared{ExtraArgs: []string{"--loglevel", "debug"}}
	got := c.buildArgs("unix:///tmp/qrrun.sock")
	want := []string{"tunnel", "--loglevel", "debug", "--url", "unix:/tmp/qrrun.sock"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: got %#v, want %#v", got, want)
	}
}
