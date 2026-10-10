package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"wallcalendar/canvas"
	"wallcalendar/lifecycle"
	"wallcalendar/waveshare"

	"github.com/furconz/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/oauth2/google"
)

const (
	numWeeks = 3
)

func loadFont(ttf []byte) *truetype.Font {
	font, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return font
}

func loadFontFile(path string) *truetype.Font {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return loadFont(b)
}

func parseBatteryOverride(raw string) (float64, error) {
	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		return 0, fmt.Errorf("empty battery override")
	}
	return strconv.ParseFloat(parts[len(parts)-1], 64)
}

func main() {
	onlyRenderImage := flag.Bool("only_render_image", false, "Only render the image, no screen")
	clearScreen := flag.Bool("clear_screen", false, "Force clear the screen")
	clearWeekly := flag.Bool("clear_weekly", true, "Automatically clear the screen once a week on Sundays")
	dateOverride := flag.String("date_override", "", "Date to use as today, e.g. 2024-11-21")
	battery := flag.String("battery", "", "Battery override from pisugar, e.g. '80' or 'battery: 80'")
	bootstrap := flag.Bool("bootstrap", false, "Run Google OAuth bootstrap flow")
	shutdown := flag.Bool("shutdown", false, "Enable automatic shutdown when scheduled wake finishes")
	piSugarAddr := flag.String("pisugar_addr", "127.0.0.1:8423", "PiSugar TCP server address")
	networkTimeout := flag.Duration("network_timeout", 60*time.Second, "Timeout waiting for network connection")
	flag.Parse()

	if *bootstrap {
		runBootstrap()
		return
	}

	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}

	newYork, err := time.LoadLocation(cfg.Location.Timezone)
	if err != nil {
		fmt.Printf("Error loading timezone: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	var goMono, unifontMono *truetype.Font

	// Task 1: Font Setup (local I/O and TTF parsing)
	wg.Add(1)
	go func() {
		defer wg.Done()
		goMono = loadFont(gomono.TTF)
		unifontMono = loadFontFile("fonts/UnifontExMono.ttf")
	}()

	// Task 2: PiSugar status & Network probe + remote data fetching
	var today time.Time
	var batteryPercent float64
	var isRTCWake bool
	var dateMap map[time.Time][]*Event
	var start, lastday time.Time
	var weatherMap map[string]DailyWeather
	var networkErr error

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Query PiSugar
		sugarClient := lifecycle.NewPiSugarClient(*piSugarAddr, 500*time.Millisecond)
		if *battery != "" {
			if b, err := parseBatteryOverride(*battery); err == nil {
				batteryPercent = b
			} else {
				log.Printf("Failed to parse --battery override %q: %v", *battery, err)
			}
		} else {
			if b, err := sugarClient.GetBattery(); err == nil {
				batteryPercent = b
			} else {
				log.Printf("PiSugar battery query failed: %v", err)
			}
		}

		if isWake, err := sugarClient.IsRTCWake(); err == nil {
			isRTCWake = isWake
		} else {
			log.Printf("PiSugar RTC alarm flag query failed: %v", err)
		}

		// Sync hardware RTC from PiSugar into the Linux system clock immediately
		if err := sugarClient.SyncRTCToPi(); err != nil {
			log.Printf("PiSugar RTC to Pi sync failed: %v", err)
		}

		// Probe network with sub-second polling and fast timeout
		ctx, cancel := context.WithTimeout(context.Background(), *networkTimeout)
		defer cancel()

		if err := lifecycle.WaitForNetwork(ctx, "1.1.1.1:53", 250*time.Millisecond); err != nil {
			log.Printf("Network unreachable within %v: %v; skipping render.", *networkTimeout, err)
			networkErr = err
			return
		}

		// Compute calendar date now that hardware RTC and network are available
		if len(*dateOverride) > 0 {
			var err error
			today, err = time.ParseInLocation("2006-01-02", *dateOverride, newYork)
			if err != nil {
				log.Printf("Error parsing date override: %v", err)
				networkErr = err
				return
			}
			today = midnight(today, newYork)
		} else {
			today = midnight(time.Now().In(newYork), newYork)
		}
		log.Printf("Calendar active date: %s (%s)", today.Format("2006-01-02"), today.Weekday())

		// Fetch calendar events
		dMap, s, l, err := FetchEvents(today, cfg.Calendar.ID, newYork, cfg.Calendar.NumWeeks, cfg.Google.Token)
		if err != nil {
			log.Printf("Error fetching calendar events: %v; skipping render.", err)
			networkErr = err
			return
		}
		dateMap = dMap
		start = s
		lastday = l

		// Fetch weather (non-fatal)
		wMap, err := FetchWeather(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone, cfg.Weather.TempUnit)
		if err != nil {
			log.Printf("Error fetching weather: %v", err)
		}
		weatherMap = wMap
	}()

	wg.Wait()

	// If network timed out or data fetching failed: skip rendering and evaluate shutdown
	if networkErr != nil {
		log.Printf("Data fetch failed (%v). Skipping render to conserve battery.", networkErr)
		handleShutdownEvaluation(*shutdown, isRTCWake)
		return
	}

	// Network succeeded: render calendar image
	img := waveshare.NewHorizontalLSB(image.Rect(0, 0, 1304, 984))
	renderCalendar(img, cfg, today, start, lastday, dateMap, weatherMap, goMono, unifontMono, batteryPercent, newYork)

	if *onlyRenderImage {
		f, err := os.Create("processed.png")
		if err == nil {
			_ = png.Encode(f, img)
			_ = f.Close()
			log.Println("Successfully saved rendered image to processed.png")
		} else {
			log.Printf("Error creating processed.png: %v", err)
		}
	} else {
		waveshare.Initialize()
		defer waveshare.Close()

		shouldClear := *clearScreen || (*clearWeekly && today.Weekday() == time.Sunday)
		if shouldClear {
			log.Println("Running full clear screen (weekly Sunday reset)...")
			waveshare.Clear()
			time.Sleep(300 * time.Millisecond)
		} else {
			log.Println("Skipping clear screen (fast daily update)")
		}
		waveshare.Display(img)
		waveshare.Sleep()
	}

	if err := PublishToHomeAssistant(batteryPercent, *cfg); err != nil {
		log.Printf("Error publishing to Home Assistant: %v", err)
	}

	handleShutdownEvaluation(*shutdown, isRTCWake)
}

func handleShutdownEvaluation(enableShutdown bool, isRTCWake bool) {
	shouldPoweroff, reason, err := lifecycle.EvaluateShutdown(lifecycle.ShutdownOptions{
		EnableShutdown: enableShutdown,
		IsRTCWake:      isRTCWake,
	})
	if err != nil {
		log.Printf("Error during shutdown evaluation: %v", err)
	}
	log.Printf("Shutdown evaluation result: %s (will poweroff: %t)", reason, shouldPoweroff)
}

func renderCalendar(
	img *waveshare.HorizontalLSB,
	cfg *Config,
	today, start, lastday time.Time,
	dateMap map[time.Time][]*Event,
	weatherMap map[string]DailyWeather,
	goMono, unifontMono *truetype.Font,
	batteryPercent float64,
	newYork *time.Location,
) {
	canv := canvas.NewCanvas(img)
	c := NewCalendar(
		canv,
		truetype.NewFace(goMono, &truetype.Options{
			Size:    70,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
		truetype.NewFace(goMono, &truetype.Options{
			Size:    24,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
		truetype.NewFace(unifontMono, &truetype.Options{
			Size:    16, // Event font, native Unifont size is 16
			DPI:     72,
			Hinting: font.HintingFull,
		}),
		truetype.NewFace(unifontMono, &truetype.Options{
			Size:    16, // Battery percentage font, native Unifont size is 16
			DPI:     72,
			Hinting: font.HintingFull,
		}),
		truetype.NewFace(unifontMono, &truetype.Options{
			Size:    16, // Weather temperature font, slightly larger as requested
			DPI:     72,
			Hinting: font.HintingFull,
		}),
		newYork)

	if start.Month() == lastday.Month() {
		c.RenderMonth(today.Format("January 2006"))
	} else if start.Year() == lastday.Year() {
		c.RenderMonth(fmt.Sprintf("%s/%s %s", start.Format("January"), lastday.Format("January"), start.Format("2006")))
	} else {
		c.RenderMonth(fmt.Sprintf("%s/%s", start.Format("January 2006"), lastday.Format("January 2006")))
	}

	c.RenderDayHeaders()

	// Calculate required height for each week
	slotHeights := make([]map[int]int, numWeeks)
	requiredHeights := make([]int, numWeeks)
	totalRequiredHeight := 0

	// Standard face for measurement - MUST MATCH NewCalendar eventFace (unifontMono, 16)
	face := truetype.NewFace(unifontMono, &truetype.Options{
		Size:    16, // MATCH NewCalendar
		DPI:     72,
		Hinting: font.HintingFull,
	})

	canonicalHeights := make(map[string]int)

	for i := 0; i < cfg.Calendar.NumWeeks; i++ {
		slotHeights[i] = make(map[int]int)
		maxSlot := -1
		measuredInWeek := make(map[string]bool)

		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			events := dateMap[date]
			for _, e := range events {
				if e.Slot > maxSlot {
					maxSlot = e.Slot
				}

				if measuredInWeek[e.ID] {
					continue
				}
				measuredInWeek[e.ID] = true

				text := e.Summary
				startsToday := e.StartsOnDate(date, newYork)
				if !e.IsAllDayEvent && startsToday {
					text = e.StartTimeShort(newYork) + " " + text
				}

				// Calculate width: if multi-day, it might span multiple columns in this week
				w := c.ColumnWidth() - 10
				endsOnDifferentDay := !e.EndsOnDate(e.StartTime, newYork)
				if endsOnDifferentDay {
					daysToWeekEnd := 7 - j
					daysToEventEnd := int(e.EndTime.Sub(date).Hours()/24) + 1
					spanDays := daysToWeekEnd
					if daysToEventEnd < spanDays {
						spanDays = daysToEventEnd
					}
					w = spanDays*c.ColumnWidth() - 10
				} else if !startsToday {
					text = "  " + text
				}

				var h int
				if ch, ok := canonicalHeights[e.ID]; ok {
					h = ch
				} else {
					h = canv.MeasureMultiColorString(text, w, face)
					if endsOnDifferentDay {
						canonicalHeights[e.ID] = h
					}
				}
				e.RowHeight = h

				if h > slotHeights[i][e.Slot] {
					slotHeights[i][e.Slot] = h
				}
			}
		}

		weekHeight := 85
		eventPadding := int(float64(face.Metrics().Height.Round()) * 0.5)

		for s := 0; s <= maxSlot; s++ {
			if h, ok := slotHeights[i][s]; ok {
				weekHeight += h + eventPadding
			} else {
				weekHeight += face.Metrics().Height.Ceil() + eventPadding
			}
		}
		weekHeight += 10

		requiredHeights[i] = weekHeight
		totalRequiredHeight += weekHeight
	}

	availableHeight := 984 - 140 - 8
	finalHeights := make([]int, cfg.Calendar.NumWeeks)

	if totalRequiredHeight <= availableHeight {
		extra := availableHeight - totalRequiredHeight
		for i := 0; i < cfg.Calendar.NumWeeks; i++ {
			finalHeights[i] = requiredHeights[i] + extra/cfg.Calendar.NumWeeks
		}
	} else {
		currentY := 0
		for i := 0; i < cfg.Calendar.NumWeeks; i++ {
			h := int(float64(requiredHeights[i]) / float64(totalRequiredHeight) * float64(availableHeight))
			finalHeights[i] = h
			if i == cfg.Calendar.NumWeeks-1 {
				finalHeights[i] = availableHeight - currentY
			}
			currentY += h
		}
	}

	currentY := 140 + 8 // headerHeight + margin
	for i := 0; i < cfg.Calendar.NumWeeks; i++ {
		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			dateKey := date.Format("2006-01-02")
			weather, ok := weatherMap[dateKey]
			if !ok || i >= 2 {
				weather = DailyWeather{}
			}
			c.Render(j, i, date, dateMap[date], date == today, slotHeights[i], currentY, finalHeights[i], weather)
		}
		currentY += finalHeights[i]
	}

	c.RenderBatteryAndTime(batteryPercent)
}

func runBootstrap() {
	b, err := os.ReadFile("credentials.json")
	if err != nil {
		fmt.Printf("Unable to read client secret file: %v\n", err)
		fmt.Println("Please make sure credentials.json is present in the current directory.")
		return
	}

	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/calendar.readonly")
	if err != nil {
		fmt.Printf("Unable to parse client secret file to config: %v\n", err)
		return
	}
	config.RedirectURL = "http://localhost:8080"

	tok := getTokenFromWeb(config)
	tokBytes, err := json.Marshal(tok)
	if err != nil {
		fmt.Printf("Unable to marshal token: %v\n", err)
		return
	}

	fmt.Println("\n--- GOOGLE TOKEN JSON ---")
	fmt.Println(string(tokBytes))
	fmt.Println("-------------------------")
	fmt.Println("\nPlease copy the JSON string above and paste it into the 'google: token:' field in your config.yaml file.")
	fmt.Println("Make sure it's inside quotes if it contains special characters, or use the literal block style in YAML.")
}
