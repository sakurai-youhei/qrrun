package server

import (
	"fmt"
	"net"
)

func newOriginListener() (net.Listener, string, string, func(), error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", "", nil, fmt.Errorf("server: listen: %w", err)
	}

	base := "https://" + ln.Addr().String()
	cleanup := func() {
		_ = ln.Close()
	}

	return ln, base, base, cleanup, nil
}
