package lifecycle

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DefaultBatteryCacheFile is the default filename used to persist the last known valid battery level.
const DefaultBatteryCacheFile = ".last_battery"

// SaveCachedBattery persists a valid battery percentage (> 0) to disk.
func SaveCachedBattery(path string, battery float64) error {
	if path == "" {
		path = DefaultBatteryCacheFile
	}
	if battery <= 0 {
		return fmt.Errorf("refusing to cache non-positive battery level: %.1f", battery)
	}
	return os.WriteFile(path, []byte(fmt.Sprintf("%.1f\n", battery)), 0644)
}

// LoadCachedBattery reads the last known valid battery percentage from disk.
func LoadCachedBattery(path string) (float64, error) {
	if path == "" {
		path = DefaultBatteryCacheFile
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse cached battery: %w", err)
	}
	if val <= 0 {
		return 0, fmt.Errorf("cached battery level is non-positive: %.1f", val)
	}
	return val, nil
}
