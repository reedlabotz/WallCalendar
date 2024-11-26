package main

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"github.com/furconz/freetype/truetype"
	"github.com/lovelydeng/gomoji"
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

	d := &font.Drawer{
		Dst: dst,
		Src: image.Black,
		Face: truetype.NewFace(c.DateFace, &truetype.Options{
			Size:    24,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
	}
	if isToday {
		size := font.MeasureString(d.Face, date.Format("2"))
		left := boxLeft + cellPadding + size.Round()/2
		top := boxTop - cellPadding + 24
		c.drawCircle(dst, left, top, 18, color.RGBA{0xff, 0, 0, 0xff})
		c.drawCircle(dst, left, top, 15, color.White)
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
	newYork, _ := time.LoadLocation("America/New_York")
	for _, item := range events {
		d2.Dot.X = fixed.I(x)
		if IsAllDayEvent(item) {
			d2.Face = truetype.NewFace(c.EventTimeFace, &truetype.Options{
				Size:    16,
				DPI:     72,
				Hinting: font.HintingFull,
			})
			d2.Src = image.NewUniform(color.RGBA{0xff, 0, 0, 0xff})
			d2.DrawString(StartTimeShort(item, newYork))
		}
		d2.Face = truetype.NewFace(c.EventFace, &truetype.Options{
			Size:    16,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		d2.Src = image.Black
		words := strings.Split(item.Summary, " ")
		for _, w := range words {
			if gomoji.ContainsEmoji(w) {
				d2.Src = image.NewUniform(color.RGBA{0xff, 0, 0, 0xff})
			} else {
				d2.Src = image.Black
			}
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

func (c Calendar) RenderBatteryAndTime(dst draw.Image, battery string) {
	d2 := &font.Drawer{
		Dst: dst,
		Src: image.Black,
		Face: truetype.NewFace(c.EventFace, &truetype.Options{
			Size:    11,
			DPI:     72,
			Hinting: font.HintingFull,
		}),
	}
	d2.Dot = fixed.Point26_6{
		X: fixed.I(2),
		Y: fixed.I(10),
	}
	newYork, _ := time.LoadLocation("America/New_York")
	d2.DrawString(time.Now().In(newYork).Format(time.Kitchen) + " | " + battery)
}
