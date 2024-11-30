package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"github.com/lovelydeng/gomoji"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

type Canvas struct {
	dst draw.Image
}

type Color int

const (
	Black Color = iota + 1
	Red
	White
)

func (c Color) ToColor() color.Color {
	if c == Red {
		return color.RGBA{0xff, 0, 0, 0xff}
	}
	if c == Black {
		return image.Black
	}
	return image.White
}

func (c Color) ToImage() image.Image {
	return image.NewUniform(c.ToColor())
}

func (c Canvas) DrawHorizontalLine(x int, y int, width int, color Color) {
	for i := 0; i < width; i++ {
		c.dst.Set(x+i, y, color.ToColor())
	}
}

func (c Canvas) DrawHorizontalArrow(x int, y int, width int, color Color) {
	c.DrawHorizontalLine(x, y, width, color)
	for i := 1; i < 5; i++ {
		for j := 0; j < i; j++ {
			c.dst.Set(x+width-1-i, y+i-j, color.ToColor())
			c.dst.Set(x+width-1-i, y-i+j, color.ToColor())
		}
	}
}

type Alignment int

const (
	Left Alignment = iota + 1
	Center
)

func (c Canvas) DrawString(s string, x int, y int, w int, f font.Face, col Color, a Alignment) (int, []int) {
	d := &font.Drawer{
		Dst:  c.dst,
		Src:  col.ToImage(),
		Face: f,
	}
	sw := font.MeasureString(f, s)
	if a == Center {
		d.Dot = fixed.Point26_6{
			X: fixed.I(x) + (fixed.I(w)-sw)/2,
			Y: fixed.I(y),
		}
	} else {
		d.Dot = fixed.Point26_6{
			X: fixed.I(x),
			Y: fixed.I(y),
		}
	}

	words := strings.Split(s, " ")
	widths := []int{}
	for _, word := range words {
		if gomoji.ContainsEmoji(word) {
			d.Src = Red.ToImage()
		} else {
			d.Src = col.ToImage()
		}
		width := font.MeasureString(f, word)
		if d.Dot.X+width-fixed.I(x) > fixed.I(w) {
			widths = append(widths, d.Dot.X.Round()-x)
			d.Dot.X = fixed.I(x)
			d.Dot.Y += fixed.I(f.Metrics().Height.Ceil())
		}
		d.DrawString(word + " ")
	}
	widths = append(widths, d.Dot.X.Round()-x)
	return d.Dot.Y.Round() - y, widths
}

func (c Canvas) DrawCircle(x int, y int, radius int, col Color) {
	for i := x - radius; i <= x+radius; i++ {
		for j := y - radius; j <= y+radius; j++ {
			if (i-x)*(i-x)+(j-y)*(j-y) <= radius*radius {
				c.dst.Set(i, j, col.ToColor())
			}
		}
	}
}

func (c Canvas) Width() int {
	return c.dst.Bounds().Dx()
}

func (c Canvas) Height() int {
	return c.dst.Bounds().Dy()
}

func NewCanvas(dst draw.Image) Canvas {
	return Canvas{
		dst: dst,
	}
}
