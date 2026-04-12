package runtime

import (
	"fmt"
	"net/url"
	"strings"
)

// Pythonista produces URLs for Pythonista runtimes.
// It embeds Python code via the `exec` URL parameter.
type Pythonista struct {
	Scheme string
}

// QRCodeURL converts a raw script URL into a Pythonista 3 deep-link URL.
func (p *Pythonista) QRCodeURL(publicURL string, bearerToken string) string {
	code := fmt.Sprintf("import urllib.request as u;exec(u.urlopen(u.Request(%q,headers={\"Authorization\":\"Bearer %s\"})).read().decode())", publicURL, bearerToken)
	encoded := strings.ReplaceAll(url.QueryEscape(code), "+", " ")
	return fmt.Sprintf("%s://?exec=%s", p.Scheme, encoded)
}
