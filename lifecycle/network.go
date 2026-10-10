package lifecycle

import (
	"context"
	"net"
	"time"
)

// WaitForNetwork polls outbound connectivity to target (default "1.1.1.1:53")
// at the given poll interval until a connection succeeds or the context expires.
func WaitForNetwork(ctx context.Context, target string, pollInterval time.Duration) error {
	if target == "" {
		target = "1.1.1.1:53"
	}
	if pollInterval <= 0 {
		pollInterval = 250 * time.Millisecond
	}

	dialer := net.Dialer{Timeout: 500 * time.Millisecond}

	// Try immediate dial before waiting for ticker
	conn, err := dialer.DialContext(ctx, "tcp", target)
	if err == nil {
		conn.Close()
		return nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := dialer.DialContext(ctx, "tcp", target)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}
