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

	parts := strings.Fields(resp)
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected battery response format: %q", resp)
	}

	val, err := strconv.ParseFloat(parts[len(parts)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse battery value from %q: %w", resp, err)
	}
	return val, nil
}

// SyncRTCToPi instructs PiSugar to sync its hardware RTC clock into the Linux system clock.
func (c *PiSugarClient) SyncRTCToPi() error {
	_, err := c.send("rtc_rtc2pi")
	return err
}

// GetRTCTime queries the PiSugar server for its current hardware RTC time.
func (c *PiSugarClient) GetRTCTime() (time.Time, error) {
	resp, err := c.send("get rtc_time")
	if err != nil {
		return time.Time{}, err
	}
	return parseRFC3339Value("rtc_time", resp)
}

// GetAlarmTime queries the PiSugar server for the configured RTC wake-up alarm time.
func (c *PiSugarClient) GetAlarmTime() (time.Time, error) {
	resp, err := c.send("get rtc_alarm_time")
	if err != nil {
		return time.Time{}, err
	}
	return parseRFC3339Value("rtc_alarm_time", resp)
}

// IsAlarmEnabled checks if the PiSugar RTC alarm is enabled.
func (c *PiSugarClient) IsAlarmEnabled() (bool, error) {
	resp, err := c.send("get rtc_alarm_enabled")
	if err != nil {
		return false, err
	}
	return strings.Contains(strings.ToLower(resp), "true"), nil
}

func parseRFC3339Value(prefix, resp string) (time.Time, error) {
	parts := strings.SplitN(resp, ":", 2)
	val := strings.TrimSpace(resp)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) == prefix {
		val = strings.TrimSpace(parts[1])
	}
	t, err := time.Parse(time.RFC3339Nano, val)
	if err != nil {
		t, err = time.Parse(time.RFC3339, val)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse timestamp %q: %w", val, err)
		}
	}
	return t, nil
}

// IsWithinAlarmWindow determines if the rtcTime is within [-5m, +45m] of the alarmTime's time of day.
func IsWithinAlarmWindow(rtcTime, alarmTime time.Time) bool {
	rtcSecs := rtcTime.Hour()*3600 + rtcTime.Minute()*60 + rtcTime.Second()
	alarmSecs := alarmTime.Hour()*3600 + alarmTime.Minute()*60 + alarmTime.Second()
	diff := rtcSecs - alarmSecs
	if diff < -43200 {
		diff += 86400
	} else if diff > 43200 {
		diff -= 86400
	}
	// Allow 5 minutes before scheduled alarm (early wake/jitter) up to 45 minutes after alarm (boot duration)
	return diff >= -300 && diff <= 2700
}

// IsRTCWake queries PiSugar to determine if the current boot was triggered by an RTC alarm.
// Handles PiSugar 2 (rtc_alarm_flag register) and PiSugar 3 (alarm enabled + time window proximity).
func (c *PiSugarClient) IsRTCWake() (bool, error) {
	// First check explicit hardware flag (supported on PiSugar 2)
	resp, err := c.send("get rtc_alarm_flag")
	if err == nil && strings.Contains(strings.ToLower(resp), "true") {
		_, _ = c.send("rtc_clear_flag")
		return true, nil
	}

	// For PiSugar 3 (where read_alarm_flag is hardcoded false in firmware),
	// check if alarm is enabled and current RTC time matches alarm schedule window.
	enabled, err := c.IsAlarmEnabled()
	if err != nil || !enabled {
		return false, err
	}

	rtcTime, err := c.GetRTCTime()
	if err != nil {
		return false, err
	}

	alarmTime, err := c.GetAlarmTime()
	if err != nil {
		return false, err
	}

	return IsWithinAlarmWindow(rtcTime, alarmTime), nil
}

// SyncRTCWeb instructs PiSugar to sync its RTC clock.
func (c *PiSugarClient) SyncRTCWeb() error {
	_, err := c.send("rtc_web")
	return err
}
