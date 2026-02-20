# Wall Calendar

An e-ink wall calendar display built for Raspberry Pi. It displays Google Calendar events, local weather, and reports device status to Home Assistant.

## Features
- **3-Week Calendar**: View upcoming events from Google Calendar.
- **Weather Integration**: Displays high/low temperatures and weather icons for each day.
- **Battery Status**: Shows PiSugar (or similar) battery level on screen and reports it to Home Assistant.
- **Home Assistant Integration**: Uses MQTT Discovery to report battery level and last update time. Merges automatically with the UniFi device for the host.
- **E-Ink Optimizations**: Designed for high contrast on Waveshare e-ink displays.

## Setup

### 1. Google Calendar API
- Create a project in the [Google Cloud Console](https://console.cloud.google.com/).
- Enable the Google Calendar API.
- Create OAuth 2.0 credentials and download them as `credentials.json` into this directory.
- Run the program in bootstrap mode to generate your access token:
  ```bash
  go run *.go --bootstrap
  ```
- Copy the resulting JSON token and paste it into `config.yaml` under `google: token:`.

### 2. Home Assistant (MQTT)
- Ensure you have an MQTT broker (like Mosquitto) running.
- In `config.yaml`, provide your broker's address, port, username, and password.
- The device will automatically appear in Home Assistant as "Wall Calendar" and will merge with the host reachable by its MAC address (e.g., as seen in the UniFi integration).

### 3. Configuration
- Edit `config.yaml` to set your:
    - Calendar ID
    - Latitude/Longitude (for weather)
    - Timezone
    - MQTT details
    - Google Token

## Build & Run

### Build for Raspberry Pi
```bash
env GOOS=linux GOARCH=arm GOARM=6 go build -o wallcalendar *.go
```

### Local Render (Preview)
To see a PNG representation of the display without having the screen connected:
```bash
./wallcalendar --only_render_image
```

### Options
- `--date_override 2024-11-21`: View the calendar as it would appear on a specific date.
- `--clear_screen`: Perform a full refresh/clear of the e-ink screen.
- `--battery "battery: 80"`: Simulate a battery level (useful for automation or testing).