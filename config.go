package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Calendar struct {
		ID       string `yaml:"id"`
		NumWeeks int    `yaml:"num_weeks"`
	} `yaml:"calendar"`
	Location struct {
		Latitude  float64 `yaml:"latitude"`
		Longitude float64 `yaml:"longitude"`
		Timezone  string  `yaml:"timezone"`
	} `yaml:"location"`
	Weather struct {
		TempUnit string `yaml:"temp_unit"`
	} `yaml:"weather"`
	HomeAssistant struct {
		MQTT struct {
			Broker   string `yaml:"broker"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
		} `yaml:"mqtt"`
	} `yaml:"home_assistant"`
	Google struct {
		Token string `yaml:"token"` // JSON string of the token
	} `yaml:"google"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
