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

func TestPiSugarClient_SyncRTCToPi(t *testing.T) {
	synced := false
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		if cmd == "rtc_rtc2pi" {
			synced = true
			return "rtc_rtc2pi: done"
		}
		return "unknown"
	})
	defer cleanup()

	client := NewPiSugarClient(addr, 500*time.Millisecond)
	err := client.SyncRTCToPi()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !synced {
		t.Fatalf("expected rtc_rtc2pi to be sent")
	}
}

func TestPiSugarClient_IsRTCWake_FlagTrue(t *testing.T) {
	cleared := false
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		switch cmd {
		case "get rtc_alarm_flag":
			return "rtc_alarm_flag: true"
		case "rtc_clear_flag":
			cleared = true
			return "rtc_clear_flag: done"
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

func TestPiSugarClient_IsRTCWake_PiSugar3_WithinWindow(t *testing.T) {
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		switch cmd {
		case "get rtc_alarm_flag":
			return "rtc_alarm_flag: false"
		case "get rtc_alarm_enabled":
			return "rtc_alarm_enabled: true"
		case "get rtc_time":
			return "rtc_time: 2026-10-10T06:00:25.000-04:00"
		case "get rtc_alarm_time":
			return "rtc_alarm_time: 2000-01-01T06:00:00.000-04:00"
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
		t.Fatalf("expected true for boot within alarm window, got false")
	}
}

func TestPiSugarClient_IsRTCWake_PiSugar3_OutsideWindow(t *testing.T) {
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		switch cmd {
		case "get rtc_alarm_flag":
			return "rtc_alarm_flag: false"
		case "get rtc_alarm_enabled":
			return "rtc_alarm_enabled: true"
		case "get rtc_time":
			return "rtc_time: 2026-10-10T07:15:00.000-04:00"
		case "get rtc_alarm_time":
			return "rtc_alarm_time: 2000-01-01T06:00:00.000-04:00"
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
	if isWake {
		t.Fatalf("expected false for manual boot 75 minutes after alarm, got true")
	}
}

func TestPiSugarClient_IsRTCWake_AlarmDisabled(t *testing.T) {
	addr, cleanup := startMockPiSugarServer(t, func(cmd string) string {
		switch cmd {
		case "get rtc_alarm_flag":
			return "rtc_alarm_flag: false"
		case "get rtc_alarm_enabled":
			return "rtc_alarm_enabled: false"
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
	if isWake {
		t.Fatalf("expected false when alarm disabled, got true")
	}
}

func TestIsWithinAlarmWindow(t *testing.T) {
	alarm := time.Date(2000, 1, 1, 6, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		rtc     time.Time
		want    bool
	}{
		{
			name: "exact match",
			rtc:  time.Date(2026, 10, 10, 6, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "1 minute after alarm",
			rtc:  time.Date(2026, 10, 10, 6, 1, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "2 minutes before alarm (jitter)",
			rtc:  time.Date(2026, 10, 10, 5, 58, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "30 minutes after alarm",
			rtc:  time.Date(2026, 10, 10, 6, 30, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "50 minutes after alarm (outside window)",
			rtc:  time.Date(2026, 10, 10, 6, 50, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "afternoon manual boot",
			rtc:  time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinAlarmWindow(tt.rtc, alarm)
			if got != tt.want {
				t.Errorf("IsWithinAlarmWindow() = %v, want %v", got, tt.want)
			}
		})
	}

	// Test midnight rollover
	midnightAlarm := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	lateNightRTC := time.Date(2026, 10, 10, 23, 58, 0, 0, time.UTC) // 2 mins before midnight
	if !IsWithinAlarmWindow(lateNightRTC, midnightAlarm) {
		t.Errorf("expected midnight rollover to be within window")
	}

	afterMidnightRTC := time.Date(2026, 10, 10, 0, 10, 0, 0, time.UTC) // 10 mins after midnight
	if !IsWithinAlarmWindow(afterMidnightRTC, midnightAlarm) {
		t.Errorf("expected 10 mins after midnight to be within window")
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
