package lifecycle

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// PiSugarClient manages communication with the local PiSugar server over TCP.
type PiSugarClient struct {
	addr    string
	timeout time.Duration
}

// NewPiSugarClient creates a new PiSugarClient for the given address and per-call timeout.
func NewPiSugarClient(addr string, timeout time.Duration) *PiSugarClient {
	if addr == "" {
		addr = "127.0.0.1:8423"
	}
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	}
	return &PiSugarClient{
		addr:    addr,
		timeout: timeout,
	}
}

func (c *PiSugarClient) send(cmd string) (string, error) {
	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return "", err
	}

	if _, err := fmt.Fprintf(conn, "%s\n", cmd); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("empty response from PiSugar server for command: %s", cmd)
}

// GetBattery requests the current battery percentage from PiSugar.
func (c *PiSugarClient) GetBattery() (float64, error) {
	resp, err := c.send("get battery")
	if err != nil {
		return 0, err
	}

	parts := strings.Split(resp, " ")
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected battery response format: %q", resp)
	}

	val, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse battery value from %q: %w", resp, err)
	}
	return val, nil
}

// IsRTCWake queries the PiSugar server to verify if boot was triggered by an RTC alarm.
// If it was, it clears the alarm flag so the next cycle can distinguish boot reasons.
func (c *PiSugarClient) IsRTCWake() (bool, error) {
	resp, err := c.send("get rtc_alarm_flag")
	if err != nil {
		return false, err
	}

	isAlarm := strings.Contains(strings.ToLower(resp), "true")
	if isAlarm {
		_, _ = c.send("set_rtc_alarm_flag 0")
	}
	return isAlarm, nil
}

// SyncRTCWeb instructs PiSugar to sync its RTC clock.
func (c *PiSugarClient) SyncRTCWeb() error {
	_, err := c.send("rtc_web")
	return err
}
