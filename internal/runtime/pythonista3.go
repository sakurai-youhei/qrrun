package runtime

import (
	"fmt"
	"net/url"
)

// Pythonista produces URLs for Pythonista runtimes.
// It embeds Python code via the `exec` URL parameter.
type Pythonista struct {
	Scheme string
}

// QRCodeURL converts a raw script URL into a Pythonista 3 deep-link URL.
func (p *Pythonista) QRCodeURL(publicURL string) string {
	code := fmt.Sprintf("import urllib.request\nexec(urllib.request.urlopen(%q).read().decode('utf-8'))", publicURL)
	q := url.Values{"exec": []string{code}}.Encode()
	return fmt.Sprintf("%s://?%s", p.Scheme, q)
}
