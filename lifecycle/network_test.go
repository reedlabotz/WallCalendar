package lifecycle

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestWaitForNetwork_Success(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = WaitForNetwork(ctx, l.Addr().String(), 50*time.Millisecond)
	if err != nil {
		t.Fatalf("expected network probe to succeed, got %v", err)
	}
}

func TestWaitForNetwork_Timeout(t *testing.T) {
	// Pick an address that will refuse connections immediately (closed local port)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = WaitForNetwork(ctx, addr, 20*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context deadline error, got %v", err)
	}
}
