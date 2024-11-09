package waveshare

import (
	"image"
	"image/color"
	"image/draw"
)

// Bit implements a 1 bit color.
type Bit int8

const (
	WhiteBit Bit = iota + 1
	BlackBit
	RedBit
)

// RGBA returns either all white or all black.
//
// Technically the monochrome display could be colored but this information is
// unavailable here. To use a colored display, use the 1 bit image as a mask
// for a color.
func (b Bit) RGBA() (uint32, uint32, uint32, uint32) {
	switch b {
	case WhiteBit:
		return 0xc000, 0xc000, 0xc000, 65535
	case BlackBit:
		return 0, 0, 0, 65535
	case RedBit:
		return 65535, 0, 0, 65535
	}
	panic("unknown bit")
}

func (b Bit) String() string {
	switch b {
	case WhiteBit:
		return "White"
	case BlackBit:
		return "Black"
	case RedBit:
		return "Red"
	}
	panic("unknown bit")
}

// BitModel is the color Model for 1 bit color.
var BitModel = color.ModelFunc(convert)

type HorizontalLSB struct {
	// Pix holds the image's pixels, as horizontally LSB-first packed bitmap.
	BlackPix []byte
	RedPix   []byte
	// Rect is the image's bounds.
	Rect image.Rectangle
}

func NewHorizontalLSB(r image.Rectangle) *HorizontalLSB {
	w := r.Dx()
	h := r.Dy()
	bands := w * h / 8

	blackPix := make([]byte, bands)
	for i := range bands {
		blackPix[i] = 0xff
	}
	return &HorizontalLSB{BlackPix: blackPix, RedPix: make([]byte, bands), Rect: r}
}

// ColorModel implements image.Image.
func (i *HorizontalLSB) ColorModel() color.Model {
	return BitModel
}

// Bounds implements image.Image.
func (i *HorizontalLSB) Bounds() image.Rectangle {
	return i.Rect
}

// At implements image.Image.
func (i *HorizontalLSB) At(x, y int) color.Color {
	return i.BitAt(x, y)
}

// BitAt is the optimized version of At().
func (i *HorizontalLSB) BitAt(x, y int) Bit {
	if !(image.Point{x, y}.In(i.Rect)) {
		return WhiteBit
	}
	offset, mask := i.PixOffset(x, y)

	if i.BlackPix[offset]&mask == 0 {
		return BlackBit
	}
	if i.RedPix[offset]&mask > 1 {
		return RedBit
	}
	return WhiteBit
}

// Opaque scans the entire image and reports whether it is fully opaque.
func (i *HorizontalLSB) Opaque() bool {
	return true
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y) and the corresponding mask.
func (i *HorizontalLSB) PixOffset(x, y int) (int, byte) {
	perRow := i.Rect.Dx() / 8

	xOffset := x / 8

	offset := y*perRow + xOffset
	pX := x - xOffset*8
	bit := uint(pX & 7)
	return offset, 1 << (7 - bit)
}

// Set implements draw.Image
func (i *HorizontalLSB) Set(x, y int, c color.Color) {
	i.SetBit(x, y, convertBit(c))
}

// SetBit is the optimized version of Set().
func (i *HorizontalLSB) SetBit(x, y int, b Bit) {
	if !(image.Point{x, y}.In(i.Rect)) {
		return
	}
	offset, mask := i.PixOffset(x, y)
	notMask := ^mask
	switch b {
	case WhiteBit:
		i.BlackPix[offset] |= mask
		i.RedPix[offset] &= notMask
	case BlackBit:
		i.BlackPix[offset] &= notMask
		i.RedPix[offset] &= notMask
	case RedBit:
		i.BlackPix[offset] |= mask
		i.RedPix[offset] |= mask
	}
}

var _ draw.Image = &HorizontalLSB{}

// Anything not transparent and not pure black is white.
func convert(c color.Color) color.Color {
	return convertBit(c)
}

// Anything not transparent and not pure black is white.
func convertBit(c color.Color) Bit {
	switch t := c.(type) {
	case Bit:
		return t
	default:
		r, g, b, _ := c.RGBA()
		if r > g {
			return RedBit
		}
		if (r | g | b) >= 0x8000 {
			return WhiteBit
		}
		return BlackBit

	}
}
