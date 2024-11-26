package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"
	"wallcalendar/waveshare"

	"github.com/furconz/freetype/truetype"
	"golang.org/x/image/font/gofont/gomono"
)

const (
	numWeeks = 4
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
	battery := flag.String("battery", "", "Battery output from pisugar")
	flag.Parse()

	img := waveshare.NewHorizontalLSB(image.Rect(0, 0, 1304, 984))

	newYork, _ := time.LoadLocation("America/New_York")

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

	dateMap, start, lastday := FetchEvents(today, newYork)

	calendar := &Calendar{
		HeaderFace:    loadFont(gomono.TTF),
		DateFace:      loadFont(gomono.TTF),
		EventFace:     loadFontFile("fonts/UnifontExMono.ttf"),
		EventTimeFace: loadFontFile("fonts/UnifontExMono.ttf"),
		Timezone:      newYork,
	}

	if start.Month() == lastday.Month() {
		calendar.RenderMonth(img, today.Format("January 2006"))
	} else if start.Year() == lastday.Year() {
		calendar.RenderMonth(img, fmt.Sprintf("%s/%s %s", start.Format("January"), lastday.Format("January"), start.Format("2006")))
	} else {
		calendar.RenderMonth(img, fmt.Sprintf("%s/%s", start.Format("January 2006"), lastday.Format("January 2006")))
	}

	calendar.RenderDayHeaders(img)

	for i := 0; i < numWeeks; i++ {
		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			calendar.Render(img, j, i, date, dateMap[date], date == today)
		}
	}

	calendar.RenderBatteryAndTime(img, *battery)

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
}
