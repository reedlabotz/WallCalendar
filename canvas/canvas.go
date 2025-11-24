package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

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

type ArrowDirection bool

const (
	ArrowLeft  ArrowDirection = true
	ArrowRight ArrowDirection = false
)

func (c Canvas) DrawHorizontalArrow(x int, y int, width int, color Color, direction ArrowDirection) {
	c.DrawHorizontalLine(x, y, width, color)
	for i := 1; i < 5; i++ {
		for j := 0; j < i; j++ {
			if direction == ArrowRight {
				c.dst.Set(x+width-1-i, y+i-j, color.ToColor())
				c.dst.Set(x+width-1-i, y-i+j, color.ToColor())
			} else {
				c.dst.Set(x+i, y+i-j, color.ToColor())
				c.dst.Set(x+i, y-i+j, color.ToColor())
			}
		}
	}
}

type Alignment int

const (
	Left Alignment = iota + 1
	Center
)

type ColorSpan struct {
	Start int
	Color Color
}

func (c Canvas) DrawMultiColorString(s string, x int, y int, w int, f font.Face, cols []ColorSpan, a Alignment) (int, []int) {
	d := &font.Drawer{
		Dst:  c.dst,
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
	colorIndex := 0
	stringIndex := 0
	for _, word := range words {
		if colorIndex < len(cols) {
			if cols[colorIndex].Start <= stringIndex {
				d.Src = cols[colorIndex].Color.ToImage()
				colorIndex += 1
			}
		}
		width := font.MeasureString(f, word)
		if d.Dot.X+width-fixed.I(x) > fixed.I(w) {
			widths = append(widths, d.Dot.X.Round()-x)
			d.Dot.X = fixed.I(x)
			d.Dot.Y += fixed.I(f.Metrics().Height.Ceil())
		}
		d.DrawString(word + " ")
		stringIndex += len(word + " ")
	}
	widths = append(widths, d.Dot.X.Round()-x)
	return d.Dot.Y.Round() - y, widths
}

func (c Canvas) MeasureMultiColorString(s string, w int, f font.Face) int {
	// We use a dummy drawer just to measure
	d := &font.Drawer{
		Face: f,
	}
	// Starting dot doesn't matter much for height relative to start, but let's keep it simple
	d.Dot = fixed.Point26_6{
		X: fixed.I(0),
		Y: fixed.I(0),
	}

	words := strings.Split(s, " ")
	for _, word := range words {
		width := font.MeasureString(f, word)
		if d.Dot.X+width > fixed.I(w) {
			d.Dot.X = fixed.I(0)
			d.Dot.Y += fixed.I(f.Metrics().Height.Ceil())
		}
		// Simulate drawing
		d.Dot.X += font.MeasureString(f, word + " ")
	}
	// The height is the final Y + one line height (since we start at 0 and add height for new lines)
	// Actually, DrawMultiColorString returns d.Dot.Y.Round() - y.
	// If we start at y=0, it returns d.Dot.Y.Round().
	// However, the first line sits at y=0 (baseline? no, usually top-left for this canvas lib?).
	// Let's look at DrawMultiColorString:
	// It adds Height.Ceil() when wrapping.
	// So if 1 line, Y is 0. Return is 0?
	// Wait, DrawMultiColorString returns `d.Dot.Y.Round() - y`.
	// If no wrap, Y stays at y. Return is 0.
	// But the caller usually adds `height` to `y`.
	// If return is 0, `y += 0`. That seems wrong for the next event.
	// Ah, `DrawMultiColorString` logic in `canvas.go`:
	// `d.Dot.Y += fixed.I(f.Metrics().Height.Ceil())` on wrap.
	// So if 1 line, Y is unchanged.
	// But `Render` does `y += height`.
	// If height is 0, lines overlap.
	// Let's check `Render` in `calendar.go` before this change.
	// `height, widths := c.canv.DrawMultiColorString(...)`
	// `y += height`
	// `y += int(float64(c.eventFace.Metrics().Height.Round()) * 1.5)`
	// So `height` is the *additional* height from wrapping?
	// If `DrawMultiColorString` returns 0 for single line, then `y` increments by the fixed amount (1.5 * line height).
	// So `MeasureMultiColorString` should return the *additional* height too?
	// Or should it return the *total* height?
	// The plan says "calculate the maximum height required".
	// If I want total height, I should probably return (lines * line_height).
	// Let's stick to what DrawMultiColorString returns for consistency, but we need to know the *total* visual height to reserve space.
	// Actually, `DrawMultiColorString` returns the Y delta.
	// If I have 2 lines, Y increases by 1 line height.
	// So return value is `(lines - 1) * lineHeight`.
	// We probably want the full height: `lines * lineHeight`.
	// Let's adjust `MeasureMultiColorString` to return the full height in pixels.

	lines := 1
	currentX := fixed.I(0)
	for _, word := range words {
		width := font.MeasureString(f, word)
		if currentX+width > fixed.I(w) {
			lines++
			currentX = fixed.I(0)
		}
		currentX += font.MeasureString(f, word + " ")
	}
	return lines * f.Metrics().Height.Ceil()
}

func (c Canvas) DrawString(s string, x int, y int, w int, f font.Face, col Color, a Alignment) (int, []int) {
	cols := []ColorSpan{
		{
			Start: 0,
			Color: col,
		},
	}
	return c.DrawMultiColorString(s, x, y, w, f, cols, a)
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
