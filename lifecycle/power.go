package lifecycle

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

var defaultFlagFiles = []string{
	"/boot/firmware/DEBUG",
	"/tmp/DEBUG",
}

// ShutdownOptions configures the defensive shutdown evaluator.
type ShutdownOptions struct {
	EnableShutdown bool
	IsRTCWake      bool
	FlagFiles      []string
	CheckSSH       func() bool
	PoweroffFn     func() error
}

func hasActiveSSHInReader(r io.Reader) bool {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// fields[1] is local_address:port (hex), fields[3] is connection state (hex)
		// port 22 is hex 0016; state 01 is TCP_ESTABLISHED
		if len(fields) > 3 && strings.HasSuffix(fields[1], ":0016") && fields[3] == "01" {
			return true
		}
	}
	return false
}

// HasActiveSSH checks /proc/net/tcp and /proc/net/tcp6 for established SSH connections.
func HasActiveSSH() bool {
	files := []string{"/proc/net/tcp", "/proc/net/tcp6"}
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			continue
		}
		hasSSH := hasActiveSSHInReader(file)
		file.Close()
		if hasSSH {
			return true
		}
	}
	return false
}

// HasDebugFlag checks whether any of the specified debug flag files exist.
func HasDebugFlag(flagFiles ...string) bool {
	for _, path := range flagFiles {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// EvaluateShutdown checks all safety conditions before executing system poweroff.
// Returns shouldPoweroff, reason description, and any error encountered.
func EvaluateShutdown(opts ShutdownOptions) (bool, string, error) {
	// Layer 0: Explicit command-line or config opt-in
	if !opts.EnableShutdown {
		return false, "Shutdown disabled by flag", nil
	}

	// Layer 1: Flag files (/boot/firmware/DEBUG or /tmp/DEBUG)
	flagFiles := opts.FlagFiles
	if len(flagFiles) == 0 {
		flagFiles = defaultFlagFiles
	}
	if HasDebugFlag(flagFiles...) {
		return false, "Debug flag file present", nil
	}

	// Layer 2: Interactive terminal (Active SSH session on port 22)
	checkSSH := opts.CheckSSH
	if checkSSH == nil {
		checkSSH = HasActiveSSH
	}
	if checkSSH() {
		return false, "Active SSH session detected", nil
	}

	// Layer 3: PiSugar wake reason (must be RTC alarm)
	if !opts.IsRTCWake {
		return false, "Manual button boot or PiSugar unavailable", nil
	}

	// All conditions met: execute clean poweroff
	poweroffFn := opts.PoweroffFn
	if poweroffFn == nil {
		poweroffFn = func() error {
			log.Println("Scheduled cycle finished. Invoking systemctl poweroff...")
			return exec.Command("systemctl", "poweroff").Run()
		}
	}

	err := poweroffFn()
	return true, "Powering down...", err
}
