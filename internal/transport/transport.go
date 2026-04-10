// Package transport defines the interface for tunneling transports and provides
// a factory to create transports by name.
package transport

import (
	"context"
	"fmt"
)

// Transport exposes a local URL as a publicly reachable URL.
type Transport interface {
	// Expose starts the transport and returns the public URL that maps to
	// localURL.  It blocks until ctx is cancelled or an unrecoverable error
	// occurs.  The public URL is written to urlCh exactly once before Expose
	// blocks.
	Expose(ctx context.Context, localURL string, urlCh chan<- string) error
}

// New returns a Transport for the given name or an error when the name is
// unknown.
func New(name string) (Transport, error) {
	switch name {
	case "cloudflared":
		return &Cloudflared{}, nil
	default:
		return nil, fmt.Errorf("unknown transport %q (available: cloudflared)", name)
	}
}
