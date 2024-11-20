package main

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"google.golang.org/api/calendar/v3"
)

const (
	margin      = 8
	cellPadding = 5
)

type Calendar struct {
	HeaderFace    *truetype.Font
	DateFace      *truetype.Font
	EventFace     *truetype.Font
	EventTimeFace *truetype.Font
	Timezone      *time.Location
}

func (c Calendar) RenderMonth(dst draw.Image, month string) {
	d := &font.Drawer{
		Dst: dst,
		Src: image.Black,
		Face: truetype.NewFace(c.HeaderFace, &truetype.Options{
			Size:    70,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
	}
	width := font.MeasureString(d.Face, month)
	d.Dot = fixed.Point26_6{
		X: (fixed.I(dst.Bounds().Dx()) - width) / 2,
		Y: fixed.I(80),
	}
	d.DrawString(month)
}

func (c Calendar) RenderDayHeaders(dst draw.Image) {
	columnWidth := (dst.Bounds().Dx() - margin*2) / 7
	d := &font.Drawer{
		Dst: dst,
		Src: image.Black,
		Face: truetype.NewFace(c.HeaderFace, &truetype.Options{
			Size:    24,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
	}
	daysOfWeek := []string{
		"SUNDAY",
		"MONDAY",
		"TUESDAY",
		"WEDNESDAY",
		"THURUSDAY",
		"FRIDAY",
		"SATURDAY",
	}
	d.Dot = fixed.Point26_6{
		Y: fixed.I(140),
	}
	for i, day := range daysOfWeek {
		width := font.MeasureString(d.Face, day)
		d.Dot.X = fixed.I(margin+i*columnWidth) + (fixed.I(columnWidth)-width)/2
		d.DrawString(day)
	}

}

func (c Calendar) drawCircle(dst draw.Image, x int, y int, radius int, col color.Color) {
	for i := x - radius; i <= x+radius; i++ {
		for j := y - radius; j <= y+radius; j++ {
			if (i-x)*(i-x)+(j-y)*(j-y) <= radius*radius {
				dst.Set(i, j, col)
			}
		}
	}
}

func (c Calendar) Render(dst draw.Image, col int, row int, date time.Time, events []*calendar.Event, isToday bool) {
	columnWidth := (dst.Bounds().Dx() - margin*2) / 7
	rowHeight := (dst.Bounds().Dy() - margin*2 - 140) / 4

	boxLeft := margin + columnWidth*col
	boxTop := margin + rowHeight*row + 140

	for i := cellPadding; i < columnWidth-cellPadding*2; i++ {
		dst.Set(boxLeft+i, boxTop, image.Black)
	}

	if isToday {
		left := boxLeft + cellPadding + 15
		top := boxTop + 20
		c.drawCircle(dst, left, top, 18, color.RGBA{0xff, 0, 0, 0xff})
		c.drawCircle(dst, left, top, 15, color.White)
	}
	d := &font.Drawer{
		Dst: dst,
		Src: image.Black,
		Face: truetype.NewFace(c.DateFace, &truetype.Options{
			Size:    24,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
	}
	d.Dot = fixed.Point26_6{
		X: fixed.I(boxLeft + cellPadding),
		Y: fixed.I(boxTop + 24 + cellPadding),
	}
	d.DrawString(date.Format("2"))

	d2 := &font.Drawer{
		Dst: dst,
		Src: image.Black,
		Face: truetype.NewFace(c.EventFace, &truetype.Options{
			Size:    14,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
	}

	y := boxTop + 55
	x := boxLeft + cellPadding
	d2.Dot = fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}
	for _, item := range events {
		d2.Dot.X = fixed.I(x)
		if item.Start.DateTime != "" {
			eventTime, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			fmt := "3:04pm "
			if eventTime.Minute() == 0 {
				fmt = "3pm "
			}
			d2.Face = truetype.NewFace(c.EventTimeFace, &truetype.Options{
				Size:    12,
				DPI:     72,
				Hinting: font.HintingFull,
			})
			d2.Dot.Y -= fixed.I(4)
			d2.Src = image.NewUniform(color.RGBA{0xff, 0, 0, 0xff})
			d2.DrawString(eventTime.In(c.Timezone).Format(fmt))
			d2.Dot.Y += fixed.I(4)
		}
		d2.Face = truetype.NewFace(c.EventFace, &truetype.Options{
			Size:    16,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		d2.Src = image.Black
		words := strings.Split(item.Summary, " ")
		for _, w := range words {
			width := font.MeasureString(d2.Face, w)
			if d2.Dot.X+width-fixed.I(boxLeft) > fixed.I(columnWidth-cellPadding*2) {
				d2.Dot.X = fixed.I(boxLeft + cellPadding)
				d2.Dot.Y += fixed.I(18)
			}
			d2.DrawString(w + " ")
		}
		d2.Dot.Y += fixed.I(24)
	}
}
