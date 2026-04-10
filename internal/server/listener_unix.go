//go:build !windows

package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func newOriginListener() (net.Listener, string, string, func(), error) {
	socketDir, err := os.MkdirTemp("", "qrrun-sock-*")
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("server: create socket dir: %w", err)
	}
	if err := os.Chmod(socketDir, 0o700); err != nil {
		_ = os.RemoveAll(socketDir)
		return nil, "", "", nil, fmt.Errorf("server: chmod socket dir: %w", err)
	}

	socketPath := filepath.Join(socketDir, "origin.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		_ = os.RemoveAll(socketDir)
		return nil, "", "", nil, fmt.Errorf("server: listen: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = ln.Close()
		_ = os.RemoveAll(socketDir)
		return nil, "", "", nil, fmt.Errorf("server: chmod socket: %w", err)
	}

	cleanup := func() {
		_ = os.Remove(socketPath)
		_ = os.RemoveAll(socketDir)
	}

	return ln, "http://localhost", "unix://" + socketPath, cleanup, nil
}
