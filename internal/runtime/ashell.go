package runtime

import (
	"net/url"
	"strings"
)

// AShell produces URLs for a-Shell runtime.
type AShell struct{}

// QRCodeURL converts script argv to an a-Shell deep-link URL.
// a-Shell uses the "ashell:" prefix (without "//").
func (a *AShell) QRCodeURL(_ string, _ string, scriptArgv []string) string {
	return "ashell:" + url.PathEscape(strings.Join(scriptArgv, " "))
}
