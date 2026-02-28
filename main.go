package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"
	"wallcalendar/canvas"
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

func main() {
	onlyRenderImage := flag.Bool("only_render_image", false, "Only render the image, no screen")
	clearScreen := flag.Bool("clear_screen", false, "Clear the screen")
	dateOverride := flag.String("date_override", "", "Date to use as today, e.g. 2024-11-21")
	battery := flag.String("battery", "battery: 80", "Battery output from pisugar")
	bootstrap := flag.Bool("bootstrap", false, "Run Google OAuth bootstrap flow")
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

	img := waveshare.NewHorizontalLSB(image.Rect(0, 0, 1304, 984))

	newYork, err := time.LoadLocation(cfg.Location.Timezone)
	if err != nil {
		fmt.Printf("Error loading timezone: %v\n", err)
		return
	}

	today := midnight(time.Now().In(newYork), newYork)

	if len(*dateOverride) > 0 {
		var err error
		today, err = time.ParseInLocation("2006-01-02", *dateOverride, newYork)
		if err != nil {
			fmt.Println("Error parsing date:", err)
			return
		}
		today = midnight(today, newYork)
	}

	dateMap, start, lastday := FetchEvents(today, cfg.Calendar.ID, newYork, cfg.Calendar.NumWeeks, cfg.Google.Token)

	weatherMap, err := FetchWeather(cfg.Location.Latitude, cfg.Location.Longitude, cfg.Location.Timezone, cfg.Weather.TempUnit)
	if err != nil {
		fmt.Println("Error fetching weather:", err)
	}
	goMono := loadFont(gomono.TTF)
	unifontMono := loadFontFile("fonts/UnifontExMono.ttf")

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

	for i := 0; i < cfg.Calendar.NumWeeks; i++ {
		slotHeights[i] = make(map[int]int)
		maxSlot := -1

		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			events := dateMap[date]
			for _, e := range events {
				if e.Slot > maxSlot {
					maxSlot = e.Slot
				}

				text := e.Summary
				startsToday := e.StartsOnDate(date, newYork)
				if !e.IsAllDayEvent && startsToday {
					text = e.StartTimeShort(newYork) + " " + text
				}
				if !startsToday {
					text = "  " + text
				}

				w := c.ColumnWidth() - 10
				h := canv.MeasureMultiColorString(text, w, face)

				if h > slotHeights[i][e.Slot] {
					slotHeights[i][e.Slot] = h
				}
			}
		}

		// Calculate total height for this week
		// Header offset (85) + padding
		weekHeight := 85
		eventPadding := int(float64(face.Metrics().Height.Round()) * 0.5)

		for s := 0; s <= maxSlot; s++ {
			if h, ok := slotHeights[i][s]; ok {
				weekHeight += h + eventPadding
			} else {
				weekHeight += face.Metrics().Height.Ceil() + eventPadding
			}
		}
		// Add a bit of bottom padding
		weekHeight += 10

		requiredHeights[i] = weekHeight
		totalRequiredHeight += weekHeight
	}

	// fmt.Printf("Required Heights: %v\n", requiredHeights)
	// fmt.Printf("Total Required: %d\n", totalRequiredHeight)

	// Distribute available height
	// Total available height = 1304 - headerHeight (140) - margin (8) = 836
	availableHeight := 984 - 140 - 8
	finalHeights := make([]int, cfg.Calendar.NumWeeks)

	// If we have enough space, give everyone what they need + extra
	if totalRequiredHeight <= availableHeight {
		extra := availableHeight - totalRequiredHeight
		for i := 0; i < cfg.Calendar.NumWeeks; i++ {
			finalHeights[i] = requiredHeights[i] + extra/cfg.Calendar.NumWeeks
		}
	} else {
		// Not enough space, distribute proportionally
		currentY := 0
		for i := 0; i < cfg.Calendar.NumWeeks; i++ {
			// Use float math for better precision then round
			h := int(float64(requiredHeights[i]) / float64(totalRequiredHeight) * float64(availableHeight))
			finalHeights[i] = h
			// Adjust last one to fill exactly
			if i == cfg.Calendar.NumWeeks-1 {
				finalHeights[i] = availableHeight - currentY
			}
			currentY += h
		}
	}

	// fmt.Printf("Final Heights: %v\n", finalHeights)

	currentY := 140 + 8 // headerHeight + margin
	for i := 0; i < cfg.Calendar.NumWeeks; i++ {
		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			dateKey := date.Format("2006-01-02")
			weather, ok := weatherMap[dateKey]
			if !ok || i >= 2 {
				// if i < 2 {
				// 	fmt.Printf("No weather for %s\n", dateKey)
				// }
				weather = DailyWeather{}
			} else {
				// fmt.Printf("Found weather for %s: %.0f/%.0f\n", dateKey, weather.TempMax, weather.TempMin)
			}
			c.Render(j, i, date, dateMap[date], date == today, slotHeights[i], currentY, finalHeights[i], weather)
		}
		currentY += finalHeights[i]
	}

	batteryParts := strings.Split(*battery, " ")
	if len(batteryParts) != 2 {
		panic("Battery is wrong" + *battery)
	}
	num, err := strconv.ParseFloat(batteryParts[1], 64)
	if err != nil {
		panic(err)
	}

	c.RenderBatteryAndTime(num)

	if *onlyRenderImage {
		f, _ := os.Create("processed.png")
		png.Encode(f, img)
	} else {
		waveshare.Initialize()
		defer waveshare.Close()

		if *clearScreen {
			waveshare.Clear()
			time.Sleep(300 * time.Millisecond)
		}

		waveshare.Display(img)
		waveshare.Sleep()
	}

	if err := PublishToHomeAssistant(num, *cfg); err != nil {
		fmt.Printf("Error publishing to Home Assistant: %v\n", err)
	}
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
