package main

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type HAEntityConfig struct {
	Name              string   `json:"name"`
	StateTopic        string   `json:"state_topic"`
	UnitOfMeasurement string   `json:"unit_of_measurement,omitempty"`
	ValueTemplate     string   `json:"value_template"`
	UniqueID          string   `json:"unique_id"`
	Device            HADevice `json:"device"`
	DeviceClass       string   `json:"device_class,omitempty"`
}

type HADevice struct {
	Identifiers  []string    `json:"identifiers"`
	Name         string      `json:"name"`
	Model        string      `json:"model"`
	Manufacturer string      `json:"manufacturer"`
	Connections  [][2]string `json:"connections"`
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
	batteryConfig := HAEntityConfig{
		Name:              "Wall Calendar Battery",
		StateTopic:        "wall_calendar/state",
		UnitOfMeasurement: "%",
		ValueTemplate:     "{{ value_json.battery }}",
		UniqueID:          "wall_calendar_battery",
		Device:            device,
		DeviceClass:       "battery",
	}
	publishJSON(client, batteryDiscoveryTopic, batteryConfig)

	// Last Update Sensor
	updateDiscoveryTopic := "homeassistant/sensor/wall_calendar/last_update/config"
	updateConfig := HAEntityConfig{
		Name:          "Wall Calendar Last Update",
		StateTopic:    "wall_calendar/state",
		ValueTemplate: "{{ value_json.last_update }}",
		UniqueID:      "wall_calendar_last_update",
		Device:        device,
		DeviceClass:   "timestamp",
	}
	publishJSON(client, updateDiscoveryTopic, updateConfig)

	// Publish State
	stateTopic := "wall_calendar/state"
	state := map[string]interface{}{
		"battery":     battery,
		"last_update": time.Now().Format(time.RFC3339),
	}
	publishJSON(client, stateTopic, state)

	return nil
}

func publishJSON(client mqtt.Client, topic string, data interface{}) {
	payload, _ := json.Marshal(data)
	client.Publish(topic, 0, true, payload)
}
