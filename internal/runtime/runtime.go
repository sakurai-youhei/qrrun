// Package runtime defines the interface for mobile script runtimes and
// provides a factory to create runtimes by name.
package runtime

import (
	"fmt"
)

// Runtime converts a raw public URL (pointing at a script file) into the URL
// that should be encoded in the QR code.  Some runtimes use custom URL schemes
// (e.g. pythonista3://) so that scanning the QR code opens the runtime
// directly.
type Runtime interface {
	// QRCodeURL returns the URL to encode in the QR code given the public URL
	// of the script.
	QRCodeURL(publicURL string, bearerToken string, scriptArgv []string) string
}

// New returns a Runtime for the given name or an error when the name is
// unknown.
func New(name string) (Runtime, error) {
	switch name {
	case "ashell":
		return &AShell{}, nil
	case "pythonista2":
		return &Pythonista{Scheme: "pythonista2", Python2: true}, nil
	case "pythonista3":
		return &Pythonista{Scheme: "pythonista3", Python2: false}, nil
	default:
		return nil, fmt.Errorf("unknown runtime %q (available: ashell, pythonista2, pythonista3)", name)
	}
}
