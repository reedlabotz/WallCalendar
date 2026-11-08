package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBatteryCache_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, ".last_battery")

	// Initially, file does not exist
	_, err := LoadCachedBattery(cachePath)
	if err == nil {
		t.Fatalf("expected error loading nonexistent cache file, got nil")
	}

	// Saving 0 or negative should be rejected
	if err := SaveCachedBattery(cachePath, 0.0); err == nil {
		t.Fatalf("expected error saving 0.0 battery, got nil")
	}
	if err := SaveCachedBattery(cachePath, -5.0); err == nil {
		t.Fatalf("expected error saving negative battery, got nil")
	}

	// Save valid battery level
	if err := SaveCachedBattery(cachePath, 74.5); err != nil {
		t.Fatalf("failed to save valid battery: %v", err)
	}

	// Load valid battery level
	loaded, err := LoadCachedBattery(cachePath)
	if err != nil {
		t.Fatalf("failed to load valid battery: %v", err)
	}
	if loaded != 74.5 {
		t.Fatalf("expected loaded battery 74.5, got %.1f", loaded)
	}

	// Test malformed content
	if err := os.WriteFile(cachePath, []byte("invalid-float"), 0644); err != nil {
		t.Fatalf("failed to write corrupted cache: %v", err)
	}
	if _, err := LoadCachedBattery(cachePath); err == nil {
		t.Fatalf("expected error loading corrupted cache, got nil")
	}
}
