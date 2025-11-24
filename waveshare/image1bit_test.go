package waveshare

import (
	"image"
	"testing"
)

func TestSetBit(t *testing.T) {
	rect := image.Rect(0, 0, 8, 1)
	img := NewHorizontalLSB(rect)

	// Verify initialization (all 1s = White/No Red)
	if img.BlackPix[0] != 0xff {
		t.Errorf("Expected BlackPix to be 0xff, got 0x%x", img.BlackPix[0])
	}
	if img.RedPix[0] != 0xff {
		t.Errorf("Expected RedPix to be 0xff, got 0x%x", img.RedPix[0])
	}

	// Test BlackBit
	// BlackBit should set BlackPix bit to 0, RedPix bit to 1
	img.SetBit(0, 0, BlackBit)
	// offset 0, mask 0x80 (10000000)
	// BlackPix: 0xff & ^0x80 = 0x7f (01111111)
	// RedPix: 0xff | 0x80 = 0xff
	if img.BlackPix[0] != 0x7f {
		t.Errorf("After setting BlackBit, expected BlackPix to be 0x7f, got 0x%x", img.BlackPix[0])
	}
	if img.RedPix[0] != 0xff {
		t.Errorf("After setting BlackBit, expected RedPix to be 0xff, got 0x%x", img.RedPix[0])
	}

	// Test RedBit
	// RedBit should set BlackPix bit to 1, RedPix bit to 0
	img.SetBit(1, 0, RedBit)
	// offset 0, mask 0x40 (01000000)
	// BlackPix: 0x7f | 0x40 = 0xbf (10111111)
	// RedPix: 0xff & ^0x40 = 0xbf (10111111)
	if img.BlackPix[0] != 0xbf {
		t.Errorf("After setting RedBit, expected BlackPix to be 0xbf, got 0x%x", img.BlackPix[0])
	}
	if img.RedPix[0] != 0xbf {
		t.Errorf("After setting RedBit, expected RedPix to be 0xbf, got 0x%x", img.RedPix[0])
	}

	// Test WhiteBit
	// WhiteBit should set BlackPix bit to 1, RedPix bit to 1
	img.SetBit(0, 0, WhiteBit)
	// offset 0, mask 0x80
	// BlackPix: 0xbf | 0x80 = 0xff
	// RedPix: 0xbf | 0x80 = 0xff
	if img.BlackPix[0] != 0xff {
		t.Errorf("After setting WhiteBit, expected BlackPix to be 0xff, got 0x%x", img.BlackPix[0])
	}
	if img.RedPix[0] != 0xff {
		t.Errorf("After setting WhiteBit, expected RedPix to be 0xff, got 0x%x", img.RedPix[0])
	}
}
