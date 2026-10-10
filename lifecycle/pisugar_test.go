package lifecycle

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func startMockPiSugarServer(t *testing.T, handler func(cmd string) string) (string, func()) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	done := make(chan struct{})

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					return
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				scanner := bufio.NewScanner(c)
				for scanner.Scan() {
					cmd := strings.TrimSpace(scanner.Text())
					if cmd == "" {
						continue
					}
					resp := handler(cmd)
					fmt.Fprintf(c, "%s\n", resp)
				}
			}(conn)
		}
	}()

	cleanup := func() {
		close(done)
		l.Close()
	}
	return l.Addr().String(), cleanup
}

func TestPiSugarClient_GetBattery(t *testing.T) {
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		if cmd == "get battery" {
			return "battery: 84.5"
		}
		return "unknown"
	})
	defer cleanup()

	client := NewPiSugarClient(addr, 500*time.Millisecond)
	bat, err := client.GetBattery()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bat != 84.5 {
		t.Fatalf("expected 84.5, got %f", bat)
	}
}

func TestPiSugarClient_IsRTCWake(t *testing.T) {
	cleared := false
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		switch cmd {
		case "get rtc_alarm_flag":
			return "rtc_alarm_flag: true"
		case "set_rtc_alarm_flag 0":
			cleared = true
			return "ok"
		default:
			return "unknown"
		}
	})
	defer cleanup()

	client := NewPiSugarClient(addr, 500*time.Millisecond)
	isWake, err := client.IsRTCWake()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isWake {
		t.Fatalf("expected true, got false")
	}
	if !cleared {
		t.Fatalf("expected alarm flag to be cleared")
	}
}

func TestPiSugarClient_IsRTCWake_False(t *testing.T) {
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		if cmd == "get rtc_alarm_flag" {
			return "rtc_alarm_flag: false"
		}
		return "unknown"
	})
	defer cleanup()

	client := NewPiSugarClient(addr, 500*time.Millisecond)
	isWake, err := client.IsRTCWake()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isWake {
		t.Fatalf("expected false, got true")
	}
}

func TestPiSugarClient_SyncRTCWeb(t *testing.T) {
	synced := false
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		if cmd == "rtc_web" {
			synced = true
			return "rtc_web ok"
		}
		return "unknown"
	})
	defer cleanup()

	client := NewPiSugarClient(addr, 500*time.Millisecond)
	err := client.SyncRTCWeb()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !synced {
		t.Fatalf("expected rtc_web to be sent")
	}
}
