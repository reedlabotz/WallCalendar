package main

import (
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
			Size:    16,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
		truetype.NewFace(unifontMono, &truetype.Options{
			Size:    11,
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

	for i := 0; i < numWeeks; i++ {
		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			c.Render(j, i, date, dateMap[date], date == today)
		}
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
}
