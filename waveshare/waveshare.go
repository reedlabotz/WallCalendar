package waveshare

import (
	"image"
	"log"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

const (
	width  = 1304
	height = 984

	s2Height = 492
	s2Width  = 648
	m1Height = 492
	m1Width  = 648
	s1Height = 492
	s1Width  = 656
	m2Height = 492
	m2Width  = 656

	m1CsPin = rpio.Pin(8)
	s1CsPin = rpio.Pin(7)
	m2CsPin = rpio.Pin(17)
	s2CsPin = rpio.Pin(18)

	m1s1DcPin = rpio.Pin(13)
	m2s2DcPin = rpio.Pin(22)

	m1s1ResetPin = rpio.Pin(6)
	m2s2ResetPin = rpio.Pin(23)

	m1BusyPin = rpio.Pin(5)
	s1BusyPin = rpio.Pin(19)
	m2BusyPin = rpio.Pin(27)
	s2BusyPin = rpio.Pin(24)
)

type Screen int

const (
	M1 Screen = iota + 1
	S1
	M2
	S2
)

// Exit releases SPI/GPIO control
func Close() {
	rpio.SpiEnd(rpio.Spi0)
	rpio.Close()
}

func Initialize() {
	log.Print("Initialize()")
	if err := rpio.Open(); err != nil {
		panic(err)
	}

	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		panic(err)
	}

	// configure SPI settings
	rpio.SpiSpeed(4_000_000)
	rpio.SpiMode(0, 0)

	m1s1ResetPin.Mode(rpio.Output)
	m2s2ResetPin.Mode(rpio.Output)

	m1s1DcPin.Mode(rpio.Output)
	m2s2DcPin.Mode(rpio.Output)

	m1CsPin.Mode(rpio.Output)
	s1CsPin.Mode(rpio.Output)
	m2CsPin.Mode(rpio.Output)
	s2CsPin.Mode(rpio.Output)

	m1BusyPin.Mode(rpio.Input)
	s1BusyPin.Mode(rpio.Input)
	m2BusyPin.Mode(rpio.Input)
	s2BusyPin.Mode(rpio.Input)

	m1CsPin.Write(rpio.High)
	s1CsPin.Write(rpio.High)
	m2CsPin.Write(rpio.High)
	s2CsPin.Write(rpio.High)

	Reset()

	// panel setting
	SendCommand(M1, 0x00)
	SendData(M1, 0x2f) // KW-3f   KWR-2F	BWROTP 0f	BWOTP 1f
	SendCommand(S1, 0x00)
	SendData(S1, 0x2f)
	SendCommand(M2, 0x00)
	SendData(M2, 0x23)
	SendCommand(S2, 0x00)
	SendData(S2, 0x23)

	// POWER SETTING
	SendCommand(M1, 0x01)
	SendData(M1, 0x07)
	SendData(M1, 0x17) // VGH=20V,VGL=-20V
	SendData(M1, 0x3F) // VDH=15V
	SendData(M1, 0x3F) // VDL=-15V
	SendData(M1, 0x0d)
	SendCommand(M2, 0x01)
	SendData(M2, 0x07)
	SendData(M2, 0x17) // VGH=20V,VGL=-20V
	SendData(M2, 0x3F) // VDH=15V
	SendData(M2, 0x3F) // VDL=-15V
	SendData(M2, 0x0d)

	// booster soft start
	SendCommand(M1, 0x06)
	SendData(M1, 0x17) //A
	SendData(M1, 0x17) //B
	SendData(M1, 0x39) //C
	SendData(M1, 0x17)
	SendCommand(M2, 0x06)
	SendData(M2, 0x17)
	SendData(M2, 0x17)
	SendData(M2, 0x39)
	SendData(M2, 0x17)

	// resolution setting
	SendCommand(M1, 0x61)
	SendData(M1, 0x02)
	SendData(M1, 0x88) // source 648
	SendData(M1, 0x01) // gate 492
	SendData(M1, 0xEC)
	SendCommand(S1, 0x61)
	SendData(S1, 0x02)
	SendData(S1, 0x90) // source 656
	SendData(S1, 0x01) // gate 492
	SendData(S1, 0xEC)
	SendCommand(M2, 0x61)
	SendData(M2, 0x02)
	SendData(M2, 0x90) // source 656
	SendData(M2, 0x01) // gate 492
	SendData(M2, 0xEC)
	SendCommand(S2, 0x61)
	SendData(S2, 0x02)
	SendData(S2, 0x88) // source 648
	SendData(S2, 0x01) // gate 492
	SendData(S2, 0xEC)

	SendCommandAll(0x15) // DUSPI
	SendDataAll(0x20)

	SendCommandAll(0x30) // PLL
	SendDataAll(0x08)

	SendCommandAll(0x50) // Vcom and data interval setting
	SendDataAll(0x31)
	SendDataAll(0x07)

	SendCommandAll(0x60) // TCON
	SendDataAll(0x22)

	SendCommand(M1, 0xE0) // POWER SETTING
	SendData(M1, 0x01)
	SendCommand(M2, 0xE0) // POWER SETTING
	SendData(M2, 0x01)

	SendCommandAll(0xE3)
	SendDataAll(0x00)

	SendCommand(M1, 0x82)
	SendData(M1, 0x1c)
	SendCommand(M2, 0x82)
	SendData(M2, 0x1c)

	setLut()

	log.Print("Initialize() done")
}

func Reset() {
	log.Print("Reset()")
	m1s1ResetPin.Write(rpio.High)
	m2s2ResetPin.Write(rpio.High)
	time.Sleep(200 * time.Millisecond)
	m1s1ResetPin.Write(rpio.Low)
	m2s2ResetPin.Write(rpio.Low)
	time.Sleep(10 * time.Millisecond)
	m1s1ResetPin.Write(rpio.High)
	m2s2ResetPin.Write(rpio.High)
	time.Sleep(200 * time.Millisecond)
	log.Print("Reset() done")
}

func SendCommand(screen Screen, b byte) {
	Send(screen, true, b)
}
func SendData(screen Screen, b byte) {
	Send(screen, false, b)
}
func Send(screen Screen, isCommand bool, b byte) {
	var dcPin rpio.Pin
	var csPin rpio.Pin
	switch screen {
	case M1:
		dcPin = m1s1DcPin
		csPin = m1CsPin
	case M2:
		dcPin = m2s2DcPin
		csPin = m2CsPin
	case S1:
		dcPin = m1s1DcPin
		csPin = s1CsPin
	case S2:
		dcPin = m2s2DcPin
		csPin = s2CsPin
	}

	if isCommand {
		dcPin.Write(rpio.Low)
	} else {
		dcPin.Write(rpio.High)
	}

	csPin.Write(rpio.Low)
	rpio.SpiTransmit(b)
	csPin.Write(rpio.High)
}

func SendCommandAll(b byte) {
	SendAll(true, b)
}
func SendDataAll(b byte) {
	SendAll(false, b)
}
func SendAll(isCommand bool, b byte) {
	if isCommand {
		m1s1DcPin.Write(rpio.Low)
		m2s2DcPin.Write(rpio.Low)
	} else {
		m1s1DcPin.Write(rpio.High)
		m2s2DcPin.Write(rpio.High)
	}
	m1CsPin.Write(rpio.Low)
	s1CsPin.Write(rpio.Low)
	m2CsPin.Write(rpio.Low)
	s2CsPin.Write(rpio.Low)
	rpio.SpiTransmit(b)
	m1CsPin.Write(rpio.High)
	s1CsPin.Write(rpio.High)
	m2CsPin.Write(rpio.High)
	s2CsPin.Write(rpio.High)
}

func SendCommandM1M2(b byte) {
	m1s1DcPin.Write(rpio.Low)
	m2s2DcPin.Write(rpio.Low)
	m1CsPin.Write(rpio.Low)
	m2CsPin.Write(rpio.Low)
	rpio.SpiTransmit(b)
	m1CsPin.Write(rpio.High)
	m2CsPin.Write(rpio.High)
}

func WriteRepeatedBytes(screen Screen, pixels int, command byte, value byte) {
	SendCommand(screen, command)
	for i := 0; i < pixels/8; i++ {
		SendData(screen, value)
	}
}

func Clear() {
	log.Print("Clear()")
	WriteRepeatedBytes(M1, m1Width*m1Height, 0x10, 0xff)
	WriteRepeatedBytes(M1, m1Width*m1Height, 0x13, 0x00)
	WriteRepeatedBytes(S1, s1Width*s1Height, 0x10, 0xff)
	WriteRepeatedBytes(S1, s1Width*s1Height, 0x13, 0x00)
	WriteRepeatedBytes(M2, m2Width*m2Height, 0x10, 0xff)
	WriteRepeatedBytes(M2, m2Width*m2Height, 0x13, 0x00)
	WriteRepeatedBytes(S2, s2Width*s2Height, 0x10, 0xff)
	WriteRepeatedBytes(S2, s2Width*s2Height, 0x13, 0x00)

	TurnDisplayOn()

	log.Print("Clear() done")
}

type Color int

const (
	Black Color = iota + 1
	Red
)

func DisplayInner(screen Screen, xOffset int, yOffset int, width int, height int, img image.Image) {
	imgOut := NewHorizontalLSB(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			imgOut.Set(x, y, img.At(xOffset+x, yOffset+y))
		}
	}

	SendCommand(screen, 0x10)
	for _, b := range imgOut.BlackPix {
		SendData(screen, b)
	}
	SendCommand(screen, 0x13)
	for _, b := range imgOut.RedPix {
		SendData(screen, b)
	}
}

func Display(img image.Image) {
	log.Print("Display()")
	DisplayInner(S2, 0, 0, s2Width, s2Height, img)
	DisplayInner(M2, s2Width, 0, m2Width, m2Height, img)
	DisplayInner(S1, m1Width, s2Height, s1Width, s1Height, img)
	DisplayInner(M1, 0, s2Height, m1Width, m1Height, img)
	TurnDisplayOn()

	log.Print("Display() done")
}

func TurnDisplayOn() {
	log.Print("TurnDisplayOn()")
	SendCommandM1M2(0x04) // Power on.
	time.Sleep(300 * time.Millisecond)
	SendCommandAll(0x12) // Refresh.
	time.Sleep(100 * time.Millisecond)

	ReadBusy(M1)
	ReadBusy(S1)
	ReadBusy(M2)
	ReadBusy(S2)
	log.Print("TurnDisplayOn() done")
}

func Sleep() {
	log.Print("Sleep()")
	SendCommandAll(0x02) // Power off.
	time.Sleep(300 * time.Millisecond)
	SendCommandAll(0x07) // maybe should be 0X07??
	SendDataAll(0xA5)
	time.Sleep(300 * time.Millisecond)
	log.Print("Sleep() done")
}

func ReadBusy(screen Screen) {
	log.Printf("ReadBusy(%d)", screen)
	var pin rpio.Pin
	switch screen {
	case M1:
		pin = m1BusyPin
	case M2:
		pin = m2BusyPin
	case S1:
		pin = s1BusyPin
	case S2:
		pin = s2BusyPin
	}

	for pin.Read() == rpio.Low {
		time.Sleep(200 * time.Millisecond)
	}
	log.Print("ReadBusy() done")
}

func ReadTemperature() {
	log.Print("ReadTemperature()")
	SendCommand(M1, 0x40)
	ReadBusy(M1)
	time.Sleep(300 * time.Millisecond)
	m1CsPin.Write(rpio.High)
	s1CsPin.Write(rpio.High)
	m2CsPin.Write(rpio.High)
	s2CsPin.Write(rpio.High)

	m1s1DcPin.Write(rpio.High)
	time.Sleep(5 * time.Microsecond)

	temp := rpio.SpiReceive(1)
	m1CsPin.Write(rpio.High)
	log.Printf("Read Temperature Reg:%d", temp[0])

	SendCommandAll(0xe0) //Cascade setting
	SendDataAll(0x03)
	SendCommandAll(0xe5) //Force temperature
	SendDataAll(temp[0])
	log.Print("ReadTemperature() done")
}

func setLut() {
	lut_vcom1 := []byte{
		0x00, 0x10, 0x10, 0x01, 0x08, 0x01,
		0x00, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x00, 0x08, 0x01, 0x08, 0x01, 0x06,
		0x00, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x06,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x01,
		0x00, 0x04, 0x05, 0x08, 0x08, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	lut_ww1 := []byte{
		0x91, 0x10, 0x10, 0x01, 0x08, 0x01,
		0x04, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x84, 0x08, 0x01, 0x08, 0x01, 0x06,
		0x80, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x06,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x01,
		0x08, 0x04, 0x05, 0x08, 0x08, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	lut_bw1 := []byte{
		0xA8, 0x10, 0x10, 0x01, 0x08, 0x01,
		0x84, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x84, 0x08, 0x01, 0x08, 0x01, 0x06,
		0x86, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x8C, 0x05, 0x01, 0x1E, 0x0F, 0x06,
		0x8C, 0x05, 0x01, 0x1E, 0x0F, 0x01,
		0xF0, 0x04, 0x05, 0x08, 0x08, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	lut_wb1 := []byte{
		0x91, 0x10, 0x10, 0x01, 0x08, 0x01,
		0x04, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x84, 0x08, 0x01, 0x08, 0x01, 0x06,
		0x80, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x06,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x01,
		0x08, 0x04, 0x05, 0x08, 0x08, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	lut_bb1 := []byte{
		0x92, 0x10, 0x10, 0x01, 0x08, 0x01,
		0x80, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x84, 0x08, 0x01, 0x08, 0x01, 0x06,
		0x04, 0x06, 0x01, 0x06, 0x01, 0x05,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x06,
		0x00, 0x05, 0x01, 0x1E, 0x0F, 0x01,
		0x01, 0x04, 0x05, 0x08, 0x08, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	SendCommandAll(0x20) // vcom
	for _, b := range lut_vcom1 {
		SendDataAll(b)
	}

	SendCommandAll(0x21) // red not use
	for _, b := range lut_ww1 {
		SendDataAll(b)
	}

	SendCommandAll(0x22) // bw r
	for _, b := range lut_bw1 {
		SendDataAll(b)
	}

	SendCommandAll(0x23) // wb w
	for _, b := range lut_wb1 {
		SendDataAll(b)
	}

	SendCommandAll(0x24) // bb b
	for _, b := range lut_bb1 {
		SendDataAll(b)
	}

	SendCommandAll(0x25) // bb b
	for _, b := range lut_ww1 {
		SendDataAll(b)
	}
}
