package runtime

import (
	"fmt"
)

// Pythonista produces URLs for Pythonista runtimes.
// It embeds Python code via the `exec` URL parameter.
type Pythonista struct {
	Scheme string
}

// QRCodeURL converts a raw script URL into a Pythonista 3 deep-link URL.
func (p *Pythonista) QRCodeURL(publicURL string, bearerToken string) string {
	code := fmt.Sprintf("exec(__import__(\"requests\").get(%q,headers={\"Authorization\":\"Bearer %s\"}).text)", publicURL, bearerToken)
	return fmt.Sprintf("%s://?exec=%s", p.Scheme, code)
}
