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
func (p *Pythonista) QRCodeURL(publicURL string, _ string, scriptArgv []string) string {
	argvLiteral := pythonStringListLiteral(scriptArgv)
	code := fmt.Sprintf(
		"import sys,urllib.request as u;a=sys.argv[:];sys.argv=%s\n"+
			"try:exec(u.urlopen(%q).read().decode(),{\"__name__\":\"__main__\"})\n"+
			"finally:sys.argv=a",
		argvLiteral,
		publicURL,
	)
	encoded := strings.ReplaceAll(url.QueryEscape(code), "+", " ")
	return fmt.Sprintf("%s://?exec=%s", p.Scheme, encoded)
}

func pythonStringListLiteral(values []string) string {
	if len(values) == 0 {
		return "[]"
	}

	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(quoted, ",") + "]"
}
