package main

import (
	"fmt"
	"time"
	"wallcalendar/canvas"

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

func (c Calendar) Render(col int, row int, date time.Time, events []Event, isToday bool) {
	columnWidth := c.ColumnWidth()
	rowHeight := c.RowHeight()

	boxLeft := margin + columnWidth*col
	boxTop := margin + rowHeight*row + headerHeight

	c.canv.DrawHorizontalLine(boxLeft+cellPadding, boxTop, columnWidth-cellPadding*2, canvas.Black)

	if isToday {
		size := font.MeasureString(c.dateFace, date.Format("2"))
		left := boxLeft + cellPadding + size.Round()/2
		top := boxTop - cellPadding + c.dateFace.Metrics().Height.Ceil()
		c.canv.DrawCircle(left, top, 18, canvas.Red)
		c.canv.DrawCircle(left, top, 15, canvas.White)
	}
	c.canv.DrawString(date.Format("2"), boxLeft+cellPadding, boxTop+c.dateFace.Metrics().Height.Ceil()+cellPadding, columnWidth, c.dateFace, canvas.Black, canvas.Left)

	y := boxTop + 55
	for _, e := range events {
		if !e.StartsOnDate(date, c.tz) {
			w := columnWidth
			if e.EndsOnDate(date, c.tz) {
				w -= cellPadding
				c.canv.DrawHorizontalArrow(boxLeft, y-c.eventFace.Metrics().Ascent.Ceil()/2, w, canvas.Red)
			} else {
				left := boxLeft
				if col == 0 {
					left -= margin
					w += margin
				}
				if col == 6 {
					w += margin
				}
				c.canv.DrawHorizontalLine(left, y-c.eventFace.Metrics().Ascent.Ceil()/2, w, canvas.Red)
			}
		} else {
			timePart := ""
			if !e.IsAllDayEvent {
				timePart = e.StartTimeShort(c.tz) + " "
			}
			height, widths := c.canv.DrawString(timePart+e.Summary, boxLeft+cellPadding, y, columnWidth-cellPadding*2, c.eventFace, canvas.Black, canvas.Left)
			if !e.EndsOnDate(e.StartTime, c.tz) {
				w := columnWidth - cellPadding - widths[0]
				if col == 6 {
					w += margin
				}
				c.canv.DrawHorizontalLine(boxLeft+cellPadding+widths[0], y-c.eventFace.Metrics().Ascent.Ceil()/2, w, canvas.Red)
			}
			y += height
		}
		y += int(float64(c.eventFace.Metrics().Height.Round()) * 1.5)

	}
}

func (c Calendar) RenderBatteryAndTime(battery float64) {
	color := canvas.Black
	if battery < 20 {
		color = canvas.Red
	}
	c.canv.DrawString(time.Now().In(c.tz).Format(time.Kitchen)+" | "+fmt.Sprintf("%.0f", battery)+"%", 2, 10, 100, c.batteryFont, color, canvas.Left)
}
