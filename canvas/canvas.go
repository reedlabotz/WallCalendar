package canvas

import (
	"image"
	"image/color"
	"image/draw"
	"math"
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

func (c Canvas) SetPixel(x int, y int, col Color) {
	c.dst.Set(x, y, col.ToColor())
}

func (c Canvas) DrawHorizontalLine(x int, y int, width int, color Color) {
	c.DrawThickHorizontalLine(x, y, width, 1, color)
}

func (c Canvas) DrawThickHorizontalLine(x int, y int, width int, thickness int, color Color) {
	for t := 0; t < thickness; t++ {
		for i := 0; i < width; i++ {
			c.dst.Set(x+i, y+t-thickness/2, color.ToColor())
		}
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
	Right
)

type ColorSpan struct {
	Start int
	Color Color
}

func (c Canvas) DrawMultiColorString(s string, x int, y int, w int, f font.Face, cols []ColorSpan, a Alignment) (int, []int) {
	// 1. Calculate height and wrap words to determine mask size
	words := strings.Split(s, " ")
	lineHeight := f.Metrics().Height.Ceil()
	ascent := f.Metrics().Ascent.Ceil()

	type wordInfo struct {
		text  string
		x, y  int
		width int
		color Color
	}
	var widths []int

	currentDot := fixed.Point26_6{X: fixed.I(0), Y: fixed.I(ascent)}
	stringIndex := 0

	lines := 1

	// First pass: layout and group into lines for alignment
	type line struct {
		words []wordInfo
		width fixed.Int26_6
	}
	var layoutLines []line
	var currentLine line

	for _, word := range words {
		// Calculate active color based on stringIndex and cols
		activeColor := Black
		for i := len(cols) - 1; i >= 0; i-- {
			if stringIndex >= cols[i].Start {
				activeColor = cols[i].Color
				break
			}
		}

		wordWidth固定 := font.MeasureString(f, word)
		spaceWidth固定 := font.MeasureString(f, " ")

		if currentDot.X+wordWidth固定 > fixed.I(w) && len(currentLine.words) > 0 {
			widths = append(widths, currentDot.X.Round())
			layoutLines = append(layoutLines, currentLine)
			currentLine = line{}
			currentDot.X = 0
			currentDot.Y += fixed.I(lineHeight)
			lines++
		}

		info := wordInfo{
			text:  word,
			x:     currentDot.X.Round(),
			y:     currentDot.Y.Round(),
			width: wordWidth固定.Round(),
			color: activeColor,
		}
		currentLine.words = append(currentLine.words, info)
		currentLine.width = currentDot.X + wordWidth固定
		currentDot.X += wordWidth固定 + spaceWidth固定
		stringIndex += len(word + " ")
	}
	widths = append(widths, currentDot.X.Round())
	layoutLines = append(layoutLines, currentLine)

	totalHeight := lines * lineHeight
	mask := image.NewAlpha(image.Rect(0, 0, w, totalHeight))

	// Second pass: Draw each line with alignment
	for i, l := range layoutLines {
		var xOffset fixed.Int26_6
		if a == Center {
			xOffset = (fixed.I(w) - l.width) / 2
		} else if a == Right {
			xOffset = fixed.I(w) - l.width
		}

		for _, wi := range l.words {
			d := &font.Drawer{
				Dst:  mask,
				Src:  image.Black, // Alpha mask
				Face: f,
				Dot:  fixed.Point26_6{X: xOffset + fixed.I(wi.x), Y: fixed.I(wi.y)},
			}
			d.DrawString(wi.text)
		}
		// Adjust widths to include alignment offset for the return value
		widths[i] = (l.width + xOffset).Round()
	}

	// Apply thresholded mask to destination
	for my := 0; my < totalHeight; my++ {
		for mx := 0; mx < w; mx++ {
			if mask.AlphaAt(mx, my).A > 128 {
				// Find color for this pixel
				// For multi-color strings, we need to know which word this pixel belongs to.
				// This is getting complex. Let's simplify:
				// Since we only have Black and Red, and Red usually comes first (time/emoji),
				// let's just track which words were red and check bounds.

				targetX := x + mx
				targetY := y - ascent + my

				// Default to Black, check if it falls within any Red word's bounds
				pixelColor := Black
				for _, l := range layoutLines {
					for _, wi := range l.words {
						if wi.color == Red {
							// Rough check: is mx, my within this word?
							// Ascent is relative to wi.y
							if my >= wi.y-ascent && my < wi.y+(lineHeight-ascent) {
								var xOffset fixed.Int26_6
								if a == Center {
									xOffset = (fixed.I(w) - l.width) / 2
								} else if a == Right {
									xOffset = fixed.I(w) - l.width
								}
								wordXStart := xOffset.Round() + wi.x
								if mx >= wordXStart && mx < wordXStart+wi.width+1 { // +1 for safety
									pixelColor = Red
								}
							}
						}
					}
				}

				if targetX >= 0 && targetX < c.Width() && targetY >= 0 && targetY < c.Height() {
					c.dst.Set(targetX, targetY, pixelColor.ToColor())
				}
			}
		}
	}

	return totalHeight - lineHeight, widths
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
		d.Dot.X += font.MeasureString(f, word+" ")
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
		currentX += font.MeasureString(f, word+" ")
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

func (c Canvas) DrawInvertedString(s string, x int, y int, w int, f font.Face, a Alignment) (int, []int) {
	// Create a temporary mask to draw the text into
	h := c.MeasureMultiColorString(s, w, f)
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	d := &font.Drawer{
		Dst:  mask,
		Src:  image.Black, // Full alpha
		Face: f,
	}

	ascent := f.Metrics().Ascent.Ceil()
	sw := font.MeasureString(f, s)
	if a == Center {
		d.Dot = fixed.Point26_6{
			X: (fixed.I(w) - sw) / 2,
			Y: fixed.I(ascent),
		}
	} else if a == Right {
		d.Dot = fixed.Point26_6{
			X: fixed.I(w) - sw,
			Y: fixed.I(ascent),
		}
	} else {
		d.Dot = fixed.Point26_6{
			X: fixed.I(0),
			Y: fixed.I(ascent),
		}
	}

	words := strings.Split(s, " ")
	for _, word := range words {
		width := font.MeasureString(f, word)
		if d.Dot.X+width > fixed.I(w) {
			d.Dot.X = fixed.I(0)
			d.Dot.Y += fixed.I(f.Metrics().Height.Ceil())
		}
		d.DrawString(word + " ")
	}

	// Apply inverted mask to dst
	for my := 0; my < h; my++ {
		for mx := 0; mx < w; mx++ {
			// Use a threshold for alpha to avoid blurry edges on e-ink
			if mask.AlphaAt(mx, my).A > 128 {
				targetX := x + mx
				targetY := y - ascent + my
				if targetX >= 0 && targetX < c.Width() && targetY >= 0 && targetY < c.Height() {
					c.invertAt(targetX, targetY)
				}
			}
		}
	}

	return h, nil
}

func (c Canvas) invertAt(x, y int) {
	curr := c.dst.At(x, y)
	r, g, b, _ := curr.RGBA()
	// Logic matches waveshare.convertBit
	if r > g { // Red
		c.dst.Set(x, y, White.ToColor())
	} else if (r | g | b) >= 0x8000 { // White
		c.dst.Set(x, y, Black.ToColor())
	} else { // Black
		c.dst.Set(x, y, White.ToColor())
	}
}

func (c Canvas) DrawRoundedRectangle(x, y, w, h, radius int, col Color) {
	c.DrawPartialRoundedRectangle(x, y, w, h, radius, col, true, true)
}

func (c Canvas) DrawPartialRoundedRectangle(x, y, w, h, radius int, col Color, leftRounded, rightRounded bool) {
	// Top and bottom lines
	startX := x
	endX := x + w
	if leftRounded {
		startX += radius
	}
	if rightRounded {
		endX -= radius
	}

	for i := startX; i < endX; i++ {
		c.dst.Set(i, y, col.ToColor())
		c.dst.Set(i, y+h-1, col.ToColor())
	}

	// Left and right lines
	for j := y + radius; j < y+h-radius; j++ {
		if leftRounded {
			c.dst.Set(x, j, col.ToColor())
		}
		if rightRounded {
			c.dst.Set(x+w-1, j, col.ToColor())
		}
	}

	// Corners
	if leftRounded {
		c.drawCorner(x+radius, y+radius, radius, 180, col)    // Top-left
		c.drawCorner(x+radius, y+h-1-radius, radius, 90, col) // Bottom-left
	}
	if rightRounded {
		c.drawCorner(x+w-1-radius, y+radius, radius, 270, col)   // Top-right
		c.drawCorner(x+w-1-radius, y+h-1-radius, radius, 0, col) // Bottom-right
	}
}

func (c Canvas) drawCorner(cx, cy, r, startAngle int, col Color) {
	// Simple arc drawing
	for i := 0; i <= 90; i++ {
		angle := float64(startAngle+i) * 3.14159 / 180.0
		px := cx + int(float64(r)*math.Cos(angle))
		py := cy + int(float64(r)*math.Sin(angle))
		c.dst.Set(px, py, col.ToColor())
	}
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
