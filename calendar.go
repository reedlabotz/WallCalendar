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
	tz          *time.Location
}

func NewCalendar(
	canv canvas.Canvas,
	monthFace font.Face,
	dateFace font.Face,
	eventFace font.Face,
	batteryFont font.Face,
	tz *time.Location) Calendar {
	return Calendar{
		canv:        canv,
		monthFace:   monthFace,
		dateFace:    dateFace,
		eventFace:   eventFace,
		batteryFont: batteryFont,
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

func (c Calendar) drawCarryoverLine(e *Event, x int, y int, columnWidth int, date time.Time, column int, dropLeftMargin bool) {
	w := columnWidth
	if e.EndsOnDate(date, c.tz) {
		w -= cellPadding
		c.canv.DrawHorizontalArrow(x, y-c.eventFace.Metrics().Ascent.Ceil()/2, w, canvas.Red, canvas.ArrowRight)
	} else {
		left := x
		if column == 0 && dropLeftMargin {
			left -= margin
			w += margin
		}
		if column == 6 {
			w += margin
		}
		c.canv.DrawHorizontalLine(left, y-c.eventFace.Metrics().Ascent.Ceil()/2, w, canvas.Red)
	}
}

func (c Calendar) Render(col int, row int, date time.Time, events []*Event, isToday bool, slotHeights map[int]int, rowY int, rowHeight int) {
	columnWidth := c.ColumnWidth()

	boxLeft := margin + columnWidth*col
	boxTop := rowY

	c.canv.DrawHorizontalLine(boxLeft+cellPadding, boxTop, columnWidth-cellPadding*2, canvas.Black)

	if isToday {
		size := font.MeasureString(c.dateFace, date.Format("2"))
		left := boxLeft + cellPadding + size.Round()/2
		top := boxTop - cellPadding + c.dateFace.Metrics().Height.Ceil()
		c.canv.DrawCircle(left, top, 18, canvas.Red)
		c.canv.DrawCircle(left, top, 15, canvas.White)
	}
	c.canv.DrawString(date.Format("2"), boxLeft+cellPadding, boxTop+c.dateFace.Metrics().Height.Ceil()+cellPadding, columnWidth, c.dateFace, canvas.Black, canvas.Left)

	baseY := boxTop + 55
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
		if !startsToday && !isFirstCalendarDay {
			c.drawCarryoverLine(e, boxLeft, y, columnWidth, date, col, true)
		} else {
			timePart := ""
			redSpan := 0
			if !startsToday {
				// Add spacing for continuation arrow
				timePart += "  "
				c.canv.DrawHorizontalArrow(boxLeft+cellPadding, y-c.eventFace.Metrics().Ascent.Ceil()/2, int(1.5*float64(font.MeasureString(c.eventFace, " ").Ceil())), canvas.Red, canvas.ArrowLeft)
			}
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
			_, widths := c.canv.DrawMultiColorString(timePart+e.Summary, boxLeft+cellPadding, y, columnWidth-cellPadding*2, c.eventFace, cols, canvas.Left)
			if !e.EndsOnDate(e.StartTime, c.tz) {
				c.drawCarryoverLine(e, boxLeft+cellPadding+widths[0], y, columnWidth-cellPadding-widths[0], date, col, false)
			}
		}
	}
}

func (c Calendar) RenderBatteryAndTime(battery float64) {
	color := canvas.Black
	if battery < 20 {
		color = canvas.Red
	}
	c.canv.DrawString(time.Now().In(c.tz).Format(time.Kitchen)+" | "+fmt.Sprintf("%.0f", battery)+"%", 2, 10, 100, c.batteryFont, color, canvas.Left)
}
