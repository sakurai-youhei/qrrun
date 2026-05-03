package runtime

import (
	"fmt"
	"net/url"
	"strings"
)

// Pythonista produces URLs for Pythonista runtimes.
// It embeds Python code via the `exec` URL parameter.
type Pythonista struct {
	Scheme  string
	Python2 bool
}

// QRCodeURL converts a raw script URL into a Pythonista deep-link URL.
func (p *Pythonista) QRCodeURL(publicURL string, _ string, scriptArgv []string) string {
	argvLiteral := pythonStringListLiteral(scriptArgv)
	code := pythonista3ExecCode(publicURL, argvLiteral)
	if p.Python2 {
		code = pythonista2ExecCode(publicURL, argvLiteral)
	}
	encoded := strings.ReplaceAll(url.QueryEscape(code), "+", "%20")
	return fmt.Sprintf("%s://?exec=%s", p.Scheme, encoded)
}

func pythonista3ExecCode(publicURL, argvLiteral string) string {
	return fmt.Sprintf(
		"import sys,urllib.request as u;a=sys.argv[:];sys.argv=%s\n"+
			"try:exec(u.urlopen(%q).read().decode(),{\"__name__\":\"__main__\"})\n"+
			"finally:sys.argv=a",
		argvLiteral,
		publicURL,
	)
}

func pythonista2ExecCode(publicURL, argvLiteral string) string {
	return fmt.Sprintf(
		"import sys,urllib2 as u;a=sys.argv[:];sys.argv=%s\n"+
			"try:exec(u.urlopen(%q).read(),{\"__name__\":\"__main__\"})\n"+
			"finally:sys.argv=a",
		argvLiteral,
		publicURL,
	)
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
