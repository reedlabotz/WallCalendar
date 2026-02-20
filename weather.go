package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type WeatherData struct {
	Daily struct {
		Time             []string  `json:"time"`
		WeatherCode      []int     `json:"weather_code"`
		Temperature2mMax []float64 `json:"temperature_2m_max"`
		Temperature2mMin []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

type DailyWeather struct {
	Code    int
	TempMax float64
	TempMin float64
}

type WeatherMap map[string]DailyWeather

func FetchWeather(lat, lon float64, tz string, tempUnit string) (WeatherMap, error) {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&daily=weather_code,temperature_2m_max,temperature_2m_min&timezone=%s&temperature_unit=%s&forecast_days=14", lat, lon, tz, tempUnit)
	// fmt.Printf("Fetching weather from: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("HTTP error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API returned status: %s\n", resp.Status)
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var data WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Printf("JSON decode error: %v\n", err)
		return nil, err
	}
	// fmt.Printf("Fetched %d days of weather data\n", len(data.Daily.Time))

	weatherMap := make(WeatherMap)
	for i, tStr := range data.Daily.Time {
		weatherMap[tStr] = DailyWeather{
			Code:    data.Daily.WeatherCode[i],
			TempMax: data.Daily.Temperature2mMax[i],
			TempMin: data.Daily.Temperature2mMin[i],
		}
	}

	return weatherMap, nil
}
