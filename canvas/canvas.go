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

func (c Canvas) DrawThickVerticalLine(x int, y int, height int, thickness int, color Color) {
	for t := 0; t < thickness; t++ {
		for i := 0; i < height; i++ {
			c.dst.Set(x+t-thickness/2, y+i, color.ToColor())
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

func (c Canvas) DrawMultiColorString(s string, x int, y int, w int, f font.Face, cols []ColorSpan, a Alignment, maxHeight int) (int, []int) {
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
			// Check if adding another line would exceed maxHeight
			if maxHeight > 0 && currentDot.Y.Round()+lineHeight > maxHeight {
				// Current line is the last one allowed. Truncate it if it's the last word but doesn't fit?
				// Actually, if we're here, we need to wrap. If wrapping exceeds maxHeight, truncate currentLine.
				lastWordIdx := len(currentLine.words) - 1
				if lastWordIdx >= 0 {
					ellipsis := "..."
					eWidth := font.MeasureString(f, ellipsis)
					for lastWordIdx >= 0 {
						wInfo := currentLine.words[lastWordIdx]
						if fixed.I(wInfo.x)+font.MeasureString(f, wInfo.text)+eWidth <= fixed.I(w) {
							currentLine.words[lastWordIdx].text += ellipsis
							currentLine.width = fixed.I(wInfo.x) + font.MeasureString(f, currentLine.words[lastWordIdx].text)
							break
						}
						currentLine.words = currentLine.words[:lastWordIdx]
						lastWordIdx--
					}
				}
				break
			}

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
	// Check if the final set of words exceeded maxHeight (though it shouldn't if loop break worked)
	if maxHeight == 0 || currentDot.Y.Round() <= maxHeight {
		widths = append(widths, currentDot.X.Round())
		layoutLines = append(layoutLines, currentLine)
	}

	calculatedHeight := (lines-1)*lineHeight + ascent + f.Metrics().Descent.Ceil()
	renderHeight := calculatedHeight
	if maxHeight > 0 && renderHeight > maxHeight {
		renderHeight = maxHeight
	}

	mask := image.NewAlpha(image.Rect(0, 0, w, renderHeight))

	// Second pass: Draw each line with alignment
	for i, l := range layoutLines {
		var xOffset fixed.Int26_6
		if a == Center {
			xOffset = (fixed.I(w) - l.width) / 2
		} else if a == Right {
			xOffset = fixed.I(w) - l.width
		}

		for _, wi := range l.words {
			if wi.y > renderHeight {
				continue
			}
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
	for my := 0; my < renderHeight; my++ {
		for mx := 0; mx < w; mx++ {
			alpha := mask.AlphaAt(mx, my).A
			if alpha > 0 {
				targetX := x + mx
				targetY := y - ascent + my

				// Determine color for this pixel
				pixelColor := Black
				isRed := false
				for _, l := range layoutLines {
					for _, wi := range l.words {
						if wi.color == Red {
							if my >= wi.y-ascent && my < wi.y+(lineHeight-ascent) {
								var xOffset fixed.Int26_6
								if a == Center {
									xOffset = (fixed.I(w) - l.width) / 2
								} else if a == Right {
									xOffset = fixed.I(w) - l.width
								}
								wordXStart := xOffset.Round() + wi.x
								if mx >= wordXStart && mx < wordXStart+wi.width+1 {
									isRed = true
									break
								}
							}
						}
					}
					if isRed {
						break
					}
				}

				threshold := uint8(128)
				if isRed {
					threshold = 96 // Lower threshold for red to make it fuller
					pixelColor = Red
				}

				if alpha > threshold {
					if targetX >= 0 && targetX < c.Width() && targetY >= 0 && targetY < c.Height() {
						c.dst.Set(targetX, targetY, pixelColor.ToColor())
						// Fake bold for red: double pixels horizontally
						if isRed && targetX+1 < c.Width() {
							c.dst.Set(targetX+1, targetY, pixelColor.ToColor())
						}
					}
				}
			}
		}
	}

	return renderHeight, widths
}

func (c Canvas) MeasureMultiColorString(s string, w int, f font.Face) int {
	lineHeight := f.Metrics().Height.Ceil()
	ascent := f.Metrics().Ascent.Ceil()
	descent := f.Metrics().Descent.Ceil()

	d := &font.Drawer{
		Face: f,
	}
	d.Dot = fixed.Point26_6{
		X: fixed.I(0),
		Y: fixed.I(ascent),
	}

	lines := 1
	words := strings.Split(s, " ")
	for _, word := range words {
		width := font.MeasureString(f, word)
		if d.Dot.X+width > fixed.I(w) {
			d.Dot.X = fixed.I(0)
			d.Dot.Y += fixed.I(lineHeight)
			lines++
		}
		d.Dot.X += font.MeasureString(f, word+" ")
	}
	return (lines-1)*lineHeight + ascent + descent
}

func (c Canvas) DrawString(s string, x int, y int, w int, f font.Face, col Color, a Alignment, maxHeight int) (int, []int) {
	cols := []ColorSpan{
		{
			Start: 0,
			Color: col,
		},
	}
	return c.DrawMultiColorString(s, x, y, w, f, cols, a, maxHeight)
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

	if endX > startX {
		// Manual loops instead of draw.Draw to be sure it works with HorizontalLSB
		for x := startX; x < endX; x++ {
			c.dst.Set(x, y, col.ToColor())
			c.dst.Set(x, y+1, col.ToColor()) // 2px thick
			c.dst.Set(x, y+h-1, col.ToColor())
			c.dst.Set(x, y+h-2, col.ToColor()) // 2px thick
		}
	}

	// Left and right lines - only if rounded (otherwise they catch the edge)
	startY := y + radius
	endY := y + h - radius
	if endY > startY {
		if leftRounded {
			for yP := startY; yP < endY; yP++ {
				c.dst.Set(x, yP, col.ToColor())
				c.dst.Set(x+1, yP, col.ToColor()) // 2px thick
			}
		}
		if rightRounded {
			for yP := startY; yP < endY; yP++ {
				c.dst.Set(x+w-1, yP, col.ToColor())
				c.dst.Set(x+w-2, yP, col.ToColor()) // 2px thick
			}
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
