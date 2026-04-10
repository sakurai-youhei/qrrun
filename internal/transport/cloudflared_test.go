package transport

import "testing"

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
