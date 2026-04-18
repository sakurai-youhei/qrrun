package runtime

import (
	"strings"
)

// AShell produces URLs for a-Shell runtime.
type AShell struct{}

// QRCodeURL converts script argv to an a-Shell deep-link URL.
// a-Shell uses the "ashell:" prefix (without "//").
func (a *AShell) QRCodeURL(publicURL string, _ string, scriptArgv []string) string {
	scriptArgs := scriptArgv
	if len(scriptArgv) > 0 {
		scriptArgs = scriptArgv[1:]
	}

	cmd := "curl -sSL " + shellSingleQuote(publicURL) + "|sh -s --"
	if len(scriptArgs) > 0 {
		escapedArgs := make([]string, 0, len(scriptArgs))
		for _, arg := range scriptArgs {
			escapedArgs = append(escapedArgs, shellEscapeWord(arg))
		}
		cmd += " " + strings.Join(escapedArgs, " ")
	}

	return "ashell:" + cmd
}

func shellEscapeWord(value string) string {
	if value == "" {
		return "''"
	}
	if isSafeShellWord(value) {
		return value
	}
	return shellSingleQuote(value)
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func isSafeShellWord(value string) bool {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case strings.ContainsRune("_@%+=:,./-", r):
		default:
			return false
		}
	}
	return true
}
