package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func parseBatteryOverride(raw string) (float64, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty battery override")
	}
	rawVal := strings.TrimSuffix(parts[len(parts)-1], "%")
	return strconv.ParseFloat(rawVal, 64)
}

type HAEntityConfig struct {
	Name              string   `json:"name"`
	StateTopic        string   `json:"state_topic"`
	UnitOfMeasurement string   `json:"unit_of_measurement,omitempty"`
	ValueTemplate     string   `json:"value_template"`
	UniqueID          string   `json:"unique_id"`
	Device            HADevice `json:"device"`
	DeviceClass       string   `json:"device_class,omitempty"`
	StateClass        string   `json:"state_class,omitempty"`
}

type HADevice struct {
	Identifiers  []string    `json:"identifiers"`
	Name         string      `json:"name"`
	Model        string      `json:"model"`
	Manufacturer string      `json:"manufacturer"`
	Connections  [][2]string `json:"connections"`
}

func newBatterySensorConfig(device HADevice) HAEntityConfig {
	return HAEntityConfig{
		Name:              "Wall Calendar Battery",
		StateTopic:        "wall_calendar/state",
		UnitOfMeasurement: "%",
		ValueTemplate:     "{{ value_json.battery if value_json.battery is defined else this.state }}",
		UniqueID:          "wall_calendar_battery",
		Device:            device,
		DeviceClass:       "battery",
		StateClass:        "measurement",
	}
}

func newLastUpdateSensorConfig(device HADevice) HAEntityConfig {
	return HAEntityConfig{
		Name:          "Wall Calendar Last Update",
		StateTopic:    "wall_calendar/state",
		ValueTemplate: "{{ value_json.last_update }}",
		UniqueID:      "wall_calendar_last_update",
		Device:        device,
		DeviceClass:   "timestamp",
	}
}

func buildHAStatePayload(battery float64, now time.Time) map[string]interface{} {
	state := map[string]interface{}{
		"last_update": now.Format(time.RFC3339),
	}
	if battery > 0 {
		state["battery"] = battery
	}
	return state
}

func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.HardwareAddr != nil {
			return iface.HardwareAddr.String()
		}
	}
	return ""
}

func PublishToHomeAssistant(battery float64, cfg Config) error {
	mqttCfg := cfg.HomeAssistant.MQTT
	if mqttCfg.Broker == "" {
		return nil // HA integration not configured
	}

	mac := getMACAddress()
	if mac == "" {
		fmt.Println("Warning: Could not determine MAC address for HA device merging")
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", mqttCfg.Broker, mqttCfg.Port))
	opts.SetUsername(mqttCfg.User)
	opts.SetPassword(mqttCfg.Password)
	opts.SetClientID("wall_calendar_client")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	defer client.Disconnect(250)

	device := HADevice{
		Identifiers:  []string{"wall_calendar"},
		Name:         "Wall Calendar",
		Model:        "E-Ink Display",
		Manufacturer: "Custom",
	}
	if mac != "" {
		device.Connections = [][2]string{{"mac", mac}}
	}

	// Battery Sensor
	batteryDiscoveryTopic := "homeassistant/sensor/wall_calendar/battery/config"
	batteryConfig := newBatterySensorConfig(device)
	publishJSON(client, batteryDiscoveryTopic, batteryConfig)

	// Last Update Sensor
	updateDiscoveryTopic := "homeassistant/sensor/wall_calendar/last_update/config"
	updateConfig := newLastUpdateSensorConfig(device)
	publishJSON(client, updateDiscoveryTopic, updateConfig)

	// Publish State
	stateTopic := "wall_calendar/state"
	state := buildHAStatePayload(battery, time.Now())
	if battery <= 0 {
		log.Printf("Warning: battery value is non-positive (%.1f%%); omitting from Home Assistant state payload to preserve last known reading", battery)
	}
	publishJSON(client, stateTopic, state)

	return nil
}

func publishJSON(client mqtt.Client, topic string, data interface{}) {
	payload, _ := json.Marshal(data)
	client.Publish(topic, 0, true, payload)
}
