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
	QRCodeURL(publicURL string) string
}

// New returns a Runtime for the given name or an error when the name is
// unknown.
func New(name string) (Runtime, error) {
	switch name {
	case "pythonista3":
		return &Pythonista3{}, nil
	default:
		return nil, fmt.Errorf("unknown runtime %q (available: pythonista3)", name)
	}
}
