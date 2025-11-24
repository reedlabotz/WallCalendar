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

	dateMap, start, lastday := FetchEvents(today, "family01175849838019336469@group.calendar.google.com", newYork)
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
	
	// Calculate required height for each week
	slotHeights := make([]map[int]int, numWeeks)
	requiredHeights := make([]int, numWeeks)
	totalRequiredHeight := 0
	
	// Standard face for measurement - MUST MATCH NewCalendar eventFace (unifontMono, 16)
	face := truetype.NewFace(unifontMono, &truetype.Options{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	
	for i := 0; i < numWeeks; i++ {
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
		// Header offset (55) + padding
		weekHeight := 55
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
	
	fmt.Printf("Required Heights: %v\n", requiredHeights)
	fmt.Printf("Total Required: %d\n", totalRequiredHeight)
	
	// Distribute available height
	// Total available height = 984 - headerHeight (140) - margin (8) = 836
	availableHeight := 984 - 140 - 8
	finalHeights := make([]int, numWeeks)
	
	// If we have enough space, give everyone what they need + extra
	if totalRequiredHeight <= availableHeight {
		extra := availableHeight - totalRequiredHeight
		for i := 0; i < numWeeks; i++ {
			finalHeights[i] = requiredHeights[i] + extra/numWeeks
		}
	} else {
		// Not enough space, distribute proportionally
		currentY := 0
		for i := 0; i < numWeeks; i++ {
			// Use float math for better precision then round
			h := int(float64(requiredHeights[i]) / float64(totalRequiredHeight) * float64(availableHeight))
			finalHeights[i] = h
			// Adjust last one to fill exactly
			if i == numWeeks-1 {
				finalHeights[i] = availableHeight - currentY
			}
			currentY += h
		}
	}
	
	fmt.Printf("Final Heights: %v\n", finalHeights)

	currentY := 140 + 8 // headerHeight + margin
	for i := 0; i < numWeeks; i++ {
		for j := 0; j < 7; j++ {
			date := start.AddDate(0, 0, 7*i+j)
			c.Render(j, i, date, dateMap[date], date == today, slotHeights[i], currentY, finalHeights[i])
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
}
