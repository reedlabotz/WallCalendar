package lifecycle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasActiveSSHInReader(t *testing.T) {
	// Standard /proc/net/tcp header and lines
	// Field 1 is local_address:port (hex). Port 22 is 0x0016.
	// Field 3 is st (state hex). 01 is TCP_ESTABLISHED, 0A is TCP_LISTEN.
	activeSSH := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00 00000000 0        0 1234 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0016 0100007F:C000 01 00000000:00000000 00:00 00000000 0        0 5678 1 0000000000000000 100 0 0 10 0
`
	if !hasActiveSSHInReader(strings.NewReader(activeSSH)) {
		t.Fatalf("expected active SSH to be detected")
	}

	listenOnly := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00 00000000 0        0 1234 1 0000000000000000 100 0 0 10 0
`
	if hasActiveSSHInReader(strings.NewReader(listenOnly)) {
		t.Fatalf("expected listen-only SSH not to be counted as active")
	}

	otherPortEstablished := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 0100007F:C000 01 00000000:00000000 00:00 00000000 0        0 5678 1 0000000000000000 100 0 0 10 0
`
	if hasActiveSSHInReader(strings.NewReader(otherPortEstablished)) {
		t.Fatalf("expected non-22 port not to be counted as SSH")
	}
}

func TestHasDebugFlag(t *testing.T) {
	tmpDir := t.TempDir()
	flagPath := filepath.Join(tmpDir, "DEBUG")

	if HasDebugFlag(flagPath) {
		t.Fatalf("expected flag not to exist")
	}

	if err := os.WriteFile(flagPath, []byte("debug"), 0644); err != nil {
		t.Fatalf("failed to create flag file: %v", err)
	}

	if !HasDebugFlag(flagPath) {
		t.Fatalf("expected flag to be detected")
	}
}

func TestEvaluateShutdown(t *testing.T) {
	poweroffCalled := false
	mockPoweroff := func() error {
		poweroffCalled = true
		return nil
	}

	// 1. EnableShutdown is false
	shouldPoweroff, reason, _ := EvaluateShutdown(ShutdownOptions{
		EnableShutdown: false,
		IsRTCWake:      true,
		PoweroffFn:     mockPoweroff,
	})
	if shouldPoweroff || poweroffCalled {
		t.Fatalf("expected shutdown suppressed when EnableShutdown=false, reason: %s", reason)
	}

	// 2. Debug flag present
	tmpDir := t.TempDir()
	debugFile := filepath.Join(tmpDir, "DEBUG")
	_ = os.WriteFile(debugFile, []byte("1"), 0644)

	shouldPoweroff, reason, _ = EvaluateShutdown(ShutdownOptions{
		EnableShutdown: true,
		FlagFiles:      []string{debugFile},
		IsRTCWake:      true,
		PoweroffFn:     mockPoweroff,
	})
	if shouldPoweroff || poweroffCalled {
		t.Fatalf("expected shutdown suppressed due to debug flag, reason: %s", reason)
	}

	// 3. Not an RTC wake (manual button press)
	shouldPoweroff, reason, _ = EvaluateShutdown(ShutdownOptions{
		EnableShutdown: true,
		FlagFiles:      []string{filepath.Join(tmpDir, "NONEXISTENT")},
		IsRTCWake:      false,
		PoweroffFn:     mockPoweroff,
	})
	if shouldPoweroff || poweroffCalled {
		t.Fatalf("expected shutdown suppressed due to manual button boot, reason: %s", reason)
	}

	// 4. All clear -> poweroff called
	shouldPoweroff, reason, err := EvaluateShutdown(ShutdownOptions{
		EnableShutdown: true,
		FlagFiles:      []string{filepath.Join(tmpDir, "NONEXISTENT")},
		IsRTCWake:      true,
		CheckSSH:       func() bool { return false },
		PoweroffFn:     mockPoweroff,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shouldPoweroff || !poweroffCalled {
		t.Fatalf("expected poweroff to be invoked, reason: %s", reason)
	}
}
