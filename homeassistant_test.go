package main

import (
	"encoding/json"
	"strings"
	"testing"
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
