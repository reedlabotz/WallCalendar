package main

import (
	"fmt"
	"time"

	"github.com/tdewolff/canvas"
	"google.golang.org/api/calendar/v3"
)

type Calendar struct {
	HeaderFace    *canvas.FontFace
	EventFace     *canvas.FontFace
	EventTimeFace *canvas.FontFace
	Timezone      *time.Location
}

func (c Calendar) RenderHeader(ctx *canvas.Context, boxLeft float64, boxTop float64, day string) {
	ctx.DrawText(boxLeft, boxTop, canvas.NewTextBox(c.HeaderFace, day, 150, 40, canvas.Right, canvas.Top, 0.0, 0.0))
}

func (c Calendar) Render(ctx *canvas.Context, boxLeft float64, boxTop float64, date time.Time, events []*calendar.Event, isToday bool) {
	ctx.SetStrokeColor(canvas.Black)
	if isToday {
		ctx.SetStrokeColor(canvas.Red)
	}
	ctx.SetStrokeWidth(1)
	ctx.MoveTo(boxLeft, boxTop)
	ctx.LineTo(boxLeft+150, boxTop)
	ctx.Stroke()

	ctx.DrawText(boxLeft, boxTop-10, canvas.NewTextBox(c.HeaderFace, date.Format("2"), 150, 40, canvas.Right, canvas.Top, 0.0, 0.0))
	for k, item := range events {
		out := ""
		if item.Start.DateTime != "" {
			eventTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			if eventTime.Minute() != 0 {
				out = fmt.Sprintf("%s ", eventTime.In(c.Timezone).Format("3:04pm"))
			} else {
				out = fmt.Sprintf("%s ", eventTime.In(c.Timezone).Format("3pm"))
			}
		}
		out = fmt.Sprintf("%s%s", out, item.Summary)
		ctx.DrawText(boxLeft, boxTop-float64(40+30*k), canvas.NewTextBox(c.EventFace, out, 150, 40, canvas.Left, canvas.Top, 0.0, 0.0))
	}
}
