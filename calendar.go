package main

import (
	"fmt"
	"strings"
	"time"
	"wallcalendar/canvas"

	"github.com/lovelydeng/gomoji"
	"golang.org/x/image/font"
)

const (
	margin       = 8
	cellPadding  = 5
	headerHeight = 140
)

type Calendar struct {
	canv        canvas.Canvas
	monthFace   font.Face
	dateFace    font.Face
	eventFace   font.Face
	batteryFont font.Face
	weatherFace font.Face
	tz          *time.Location
}

func NewCalendar(
	canv canvas.Canvas,
	monthFace font.Face,
	dateFace font.Face,
	eventFace font.Face,
	batteryFont font.Face,
	weatherFace font.Face,
	tz *time.Location) Calendar {
	return Calendar{
		canv:        canv,
		monthFace:   monthFace,
		dateFace:    dateFace,
		eventFace:   eventFace,
		batteryFont: batteryFont,
		weatherFace: weatherFace,
		tz:          tz,
	}
}

func (c Calendar) RenderMonth(m string) {
	c.canv.DrawString(m, 0, 80, c.canv.Width(), c.monthFace, canvas.Black, canvas.Center)

}

func (c Calendar) ColumnWidth() int {
	return (c.canv.Width() - margin*2) / 7
}

func (c Calendar) RowHeight() int {
	return (c.canv.Height() - margin*2 - headerHeight) / 4
}

func (c Calendar) RenderDayHeaders() {
	columnWidth := c.ColumnWidth()

	daysOfWeek := []string{
		"SUNDAY",
		"MONDAY",
		"TUESDAY",
		"WEDNESDAY",
		"THURSDAY",
		"FRIDAY",
		"SATURDAY",
	}

	for i, day := range daysOfWeek {
		c.canv.DrawString(day, margin+i*columnWidth, headerHeight, columnWidth, c.dateFace, canvas.Black, canvas.Center)
	}
}

func (c Calendar) drawCarryoverBox(e *Event, x int, y int, columnWidth int, date time.Time, column int, dropLeftMargin bool, isEnd bool) {
	w := columnWidth
	h := c.eventFace.Metrics().Height.Ceil() + 8        // Increased padding for the box
	boxY := y - c.eventFace.Metrics().Ascent.Ceil() - 3 // Padding: 3px top, 5px bottom (approx)

	leftRounded := e.StartsOnDate(date, c.tz) || (column == 0 && !dropLeftMargin)
	rightRounded := isEnd

	// If it's a continuation from previous row, don't round left
	if column == 0 && !e.StartsOnDate(date, c.tz) && dropLeftMargin {
		leftRounded = false
	}

	// Adjust width and start for margins
	startX := x
	if column == 0 && dropLeftMargin {
		startX -= margin
		w += margin
	}
	if column == 6 && !isEnd {
		w += margin
	}

	// Multi-day events get a red box
	c.canv.DrawPartialRoundedRectangle(startX, boxY, w, h, 8, canvas.Red, leftRounded, rightRounded)
}

func (c Calendar) Render(col int, row int, date time.Time, events []*Event, isToday bool, slotHeights map[int]int, rowY int, rowHeight int, weather DailyWeather) {
	columnWidth := c.ColumnWidth()

	boxLeft := margin + columnWidth*col
	boxTop := rowY

	c.canv.DrawThickHorizontalLine(boxLeft+cellPadding, boxTop, columnWidth-cellPadding*2, 2, canvas.Black)

	if isToday {
		size := font.MeasureString(c.dateFace, date.Format("2"))
		left := boxLeft + cellPadding + size.Round()/2
		top := boxTop - cellPadding + c.dateFace.Metrics().Height.Ceil()
		// Thicker highlight for e-ink
		c.canv.DrawCircle(left, top, 22, canvas.Red)
		c.canv.DrawCircle(left, top, 18, canvas.White)
	}
	c.canv.DrawString(date.Format("2"), boxLeft+cellPadding, boxTop+c.dateFace.Metrics().Height.Ceil()+cellPadding, columnWidth, c.dateFace, canvas.Black, canvas.Left)

	// Render Weather
	if weather.TempMax != 0 || weather.TempMin != 0 {
		symbol := c.getWeatherSymbol(weather.Code)
		weatherStr := fmt.Sprintf("%s %.0f°/%.0f°", symbol, weather.TempMax, weather.TempMin)
		// Total width for weather part
		weatherWidth := 150 // Plenty of room for combined string
		// Move to the right, but leave room for padding
		weatherX := boxLeft + columnWidth - weatherWidth - cellPadding

		// Draw combined icon and text using dedicated weatherFace
		// Align baseline to date number baseline (boxTop + cellPadding + height)
		baselineY := boxTop + c.dateFace.Metrics().Height.Ceil() + cellPadding
		c.canv.DrawString(weatherStr, weatherX, baselineY, weatherWidth, c.weatherFace, canvas.Black, canvas.Right)
	}

	baseY := boxTop + 85 // Space before first event
	// Padding between events
	eventPadding := int(float64(c.eventFace.Metrics().Height.Round()) * 0.5)

	isFirstCalendarDay := row == 0 && col == 0
	for _, e := range events {
		// Calculate y based on slot heights
		y := baseY
		for i := 0; i < e.Slot; i++ {
			if h, ok := slotHeights[i]; ok {
				y += h + eventPadding
			} else {
				// Fallback if slot height not found (shouldn't happen if logic is correct)
				y += c.eventFace.Metrics().Height.Ceil() + eventPadding
			}
		}

		// Check if y is out of bounds for the cell
		if y+slotHeights[e.Slot] > boxTop+rowHeight {
			continue // Skip rendering if it overflows the cell
		}

		startsToday := e.StartsOnDate(date, c.tz)
		endsOnDifferentDay := !e.EndsOnDate(e.StartTime, c.tz)

		if endsOnDifferentDay {
			isEnd := e.EndsOnDate(date, c.tz)
			// Draw the multi-day box
			c.drawCarryoverBox(e, boxLeft, y, columnWidth, date, col, !startsToday && !isFirstCalendarDay, isEnd)

			// If it's the start, or it's the first day we see in the calendar view, draw text
			if startsToday || isFirstCalendarDay {
				text := e.Summary
				if !e.IsAllDayEvent && startsToday {
					text = e.StartTimeShort(c.tz) + " " + text
				}
				// Always LEFT align multi-day text inside the box.
				// We add a bit of padding-left.
				c.canv.DrawString(text, boxLeft+cellPadding+2, y, columnWidth-cellPadding*2-4, c.eventFace, canvas.Black, canvas.Left)
			}
		} else {
			// Normal single-day event
			timePart := ""
			redSpan := 0
			if !e.IsAllDayEvent && startsToday {
				timePart += e.StartTimeShort(c.tz) + " "
			}
			redSpan = len(timePart)
			firstWord := strings.Split(e.Summary, " ")[0]
			if gomoji.ContainsEmoji(firstWord) {
				redSpan += len(firstWord)
			}
			var cols []canvas.ColorSpan
			if redSpan == 0 {
				cols = []canvas.ColorSpan{
					{
						Start: 0,
						Color: canvas.Black,
					},
				}
			} else {
				cols = []canvas.ColorSpan{
					{
						Start: 0,
						Color: canvas.Red,
					},
					{
						Start: redSpan,
						Color: canvas.Black,
					},
				}
			}
			c.canv.DrawMultiColorString(timePart+e.Summary, boxLeft+cellPadding, y, columnWidth-cellPadding*2, c.eventFace, cols, canvas.Left)
		}
	}
}

func (c Calendar) RenderBatteryAndTime(battery float64) {
	c.RenderTime()
	c.RenderBattery(battery)
}

func (c Calendar) RenderTime() {
	// Offset from the corner to match margin and align with battery text
	c.canv.DrawString(time.Now().In(c.tz).Format(time.Kitchen), margin, 28, 200, c.batteryFont, canvas.Black, canvas.Left)
}

func (c Calendar) RenderBattery(battery float64) {
	color := canvas.Black
	if battery < 25 {
		color = canvas.Red
	}

	batteryText := fmt.Sprintf("%.0f%%", battery)
	iconWidth := 60
	iconHeight := 25
	x := c.canv.Width() - iconWidth - margin
	y := 10

	// Draw battery body outline (thick)
	c.canv.DrawThickHorizontalLine(x, y, iconWidth-4, 2, color)
	c.canv.DrawThickHorizontalLine(x, y+iconHeight, iconWidth-4, 2, color)
	for i := 0; i <= iconHeight; i++ {
		c.canv.SetPixel(x, y+i, color)
		c.canv.SetPixel(x+1, y+i, color)
		c.canv.SetPixel(x+iconWidth-4, y+i, color)
		c.canv.SetPixel(x+iconWidth-5, y+i, color)
	}

	// Battery tip
	tipHeight := 15
	tipY := y + (iconHeight-tipHeight)/2
	for i := 0; i < tipHeight; i++ {
		c.canv.SetPixel(x+iconWidth-3, tipY+i, color)
		c.canv.SetPixel(x+iconWidth-2, tipY+i, color)
		c.canv.SetPixel(x+iconWidth-1, tipY+i, color)
	}

	// Fill the battery
	fillWidth := int(float64(iconWidth-10) * (battery / 100.0))
	if fillWidth > 0 {
		for i := 3; i < iconHeight-2; i++ {
			for j := 3; j < fillWidth+3; j++ {
				c.canv.SetPixel(x+j, y+i, color)
			}
		}
	}

	// Draw smart inverted text on top of the battery icon
	c.canv.DrawInvertedString(batteryText, x, y+iconHeight/2+6, iconWidth-4, c.batteryFont, canvas.Center)
}

func (c Calendar) getWeatherSymbol(code int) string {
	// Unicode symbols for weather codes
	// https://open-meteo.com/en/docs
	switch {
	case code == 0: // Clear sky
		return "☀"
	case code <= 3: // Mainly clear, partly cloudy, and overcast
		return "⛅"
	case code >= 51 && code <= 67: // Drizzle and Rain
		return "🌧"
	case code >= 71 && code <= 77: // Snow fall
		return "❄"
	case code >= 80 && code <= 82: // Rain showers
		return "🌧"
	case code >= 85 && code <= 86: // Snow showers
		return "❄"
	case code >= 95: // Thunderstorm
		return "⛈"
	default: // Other
		return "☁"
	}
}
