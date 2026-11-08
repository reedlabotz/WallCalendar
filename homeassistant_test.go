package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBatteryConfigStateClass(t *testing.T) {
	device := HADevice{
		Identifiers:  []string{"wall_calendar"},
		Name:         "Wall Calendar",
		Model:        "E-Ink Display",
		Manufacturer: "Custom",
	}

	batteryCfg := newBatterySensorConfig(device)
	data, err := json.Marshal(batteryCfg)
	if err != nil {
		t.Fatalf("failed to marshal battery config: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"state_class":"measurement"`) {
		t.Errorf("expected json to contain state_class: measurement, got %s", jsonStr)
	}
}

func TestLastUpdateConfigOmitsStateClass(t *testing.T) {
	device := HADevice{
		Identifiers:  []string{"wall_calendar"},
		Name:         "Wall Calendar",
		Model:        "E-Ink Display",
	}

	updateCfg := newLastUpdateSensorConfig(device)
	data, err := json.Marshal(updateCfg)
	if err != nil {
		t.Fatalf("failed to marshal update config: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "state_class") {
		t.Errorf("expected json to omit state_class, got %s", jsonStr)
	}
}

func TestBuildHAStatePayload_PositiveBattery(t *testing.T) {
	now := time.Now()
	payload := buildHAStatePayload(85.5, now)
	if payload["battery"] != 85.5 {
		t.Errorf("expected battery to be 85.5, got %v", payload["battery"])
	}
	if payload["last_update"] != now.Format(time.RFC3339) {
		t.Errorf("expected last_update to be formatted RFC3339, got %v", payload["last_update"])
	}
}

func TestBuildHAStatePayload_ZeroOrNegativeBatteryOmitsBattery(t *testing.T) {
	now := time.Now()
	for _, b := range []float64{0.0, -1.0} {
		payload := buildHAStatePayload(b, now)
		if _, exists := payload["battery"]; exists {
			t.Errorf("expected battery to be omitted when <= 0, got %v", payload["battery"])
		}
		if payload["last_update"] != now.Format(time.RFC3339) {
			t.Errorf("expected last_update to be present, got %v", payload["last_update"])
		}
	}
}

func TestBatteryConfig_ValueTemplateFallback(t *testing.T) {
	device := HADevice{
		Identifiers: []string{"wall_calendar"},
		Name:        "Wall Calendar",
	}
	cfg := newBatterySensorConfig(device)
	if !strings.Contains(cfg.ValueTemplate, "this.state") {
		t.Errorf("expected value_template to contain fallback to this.state, got %q", cfg.ValueTemplate)
	}
}

func TestParseBatteryOverride(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"80", 80.0, false},
		{"80%", 80.0, false},
		{"battery: 85.5", 85.5, false},
		{"battery: 85.5%", 85.5, false},
		{"  92%  ", 92.0, false},
		{"", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		got, err := parseBatteryOverride(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseBatteryOverride(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("parseBatteryOverride(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

