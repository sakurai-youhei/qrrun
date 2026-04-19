package runtime

import (
	"strings"
)

// AShell produces URLs for a-Shell runtime.
type AShell struct{}

// QRCodeURL converts script argv to an a-Shell deep-link URL.
// a-Shell uses the "ashell:" prefix (without "//").
func (a *AShell) QRCodeURL(publicURL string, _ string, scriptArgv []string) string {
	cmd := "curl -sSL " + shellSingleQuote(publicURL) + "|bash -s --"
	if len(scriptArgv) > 1 {
		// scriptArgv[0] is the local script path (or "-") and should not be passed to bash.
		escapedArgs := make([]string, 0, len(scriptArgv)-1)
		for _, arg := range scriptArgv[1:] {
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
