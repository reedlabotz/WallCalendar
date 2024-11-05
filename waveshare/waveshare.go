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

	epdSckPin  = rpio.Pin(11)
	epdMosiPin = rpio.Pin(10)

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

func init() {
	log.Print("init()")
	if err := rpio.Open(); err != nil {
		panic(err)
	}

	if err := rpio.SpiBegin(rpio.Spi0); err != nil {
		panic(err)
	}
	log.Print("init() done")
}

// Exit releases SPI/GPIO control
func Exit() {
	rpio.SpiEnd(rpio.Spi0)
	rpio.Close()
}

func Initialize() {
	log.Print("Initialize()")
	m1s1ResetPin.Mode(rpio.Output)
	m2s2ResetPin.Mode(rpio.Output)

	m1s1DcPin.Mode(rpio.Output)
	m1s1DcPin.Mode(rpio.Output)

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

	// panel setting for Display
	SendCommand(M1, 0x00)
	SendData(M1, 0x0f) //KW-3f   KWR-2F    BWROTP 0f   BWOTP 1f
	SendCommand(S2, 0x00)
	SendData(S2, 0x0f)
	SendCommand(M2, 0x00)
	SendData(M2, 0x03)
	SendCommand(S2, 0x00)
	SendData(S2, 0x03)

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

	//resolution setting
	SendCommand(M1, 0x61)
	SendData(M1, 0x02)
	SendData(M1, 0x88) //source 648
	SendData(M1, 0x01) //gate 492
	SendData(M1, 0xEC)
	SendCommand(S1, 0x61)
	SendData(S1, 0x02)
	SendData(S1, 0x90) //source 656
	SendData(S1, 0x01) //gate 492
	SendData(S1, 0xEC)
	SendCommand(M2, 0x61)
	SendData(M2, 0x02)
	SendData(M2, 0x90) //source 656
	SendData(M2, 0x01) //gate 492
	SendData(M2, 0xEC)
	SendCommand(S2, 0x61)
	SendData(S2, 0x02)
	SendData(S2, 0x88) //source 648
	SendData(S2, 0x01) //gate 492
	SendData(S2, 0xEC)

	SendCommandAll(0x15) //DUSPI
	SendDataAll(0x20)

	SendCommandAll(0x50) //Vcom and data interval setting
	SendDataAll(0x11)
	SendDataAll(0x07)

	SendCommandAll(0x60) //TCON
	SendDataAll(0x22)

	SendCommandAll(0xE3)
	SendDataAll(0x00)

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

func DisplayInner(screen Screen, color Color, xOffset int, yOffset int, width int, height int, image image.Image) {
	log.Printf("DisplayInner(%d, %d)", screen, color)
	var command byte
	switch color {
	case Black:
		command = 0x10
	case Red:
		command = 0x13
	}
	count := 0
	SendCommand(screen, command)
	for y := 0; y < height; y++ {
		for x := 0; x < width/8; x++ {
			bitArray := []int{1, 1, 1, 1, 1, 1, 1, 1}
			for i := 0; i < 8; i++ {
				r, g, b, _ := image.At(xOffset+x*8+i, yOffset+y).RGBA()
				if r < 1000 && g < 1000 && b < 1000 {
					bitArray[i] = 0
				}
			}

			b := (bitArray[0] * 128) + (bitArray[1] * 64) + (bitArray[2] * 32) + (bitArray[3] * 16) + (bitArray[4] * 8) + (bitArray[5] * 4) + (bitArray[6] * 2) + (bitArray[7] * 1)
			if color == Red {
				SendData(screen, 0x00)
			} else {
				SendData(screen, byte(b))
			}
			count += 1
		}
	}
	log.Printf("count: %d", count)
}

func Display(image image.Image) {
	log.Print("Display()")
	DisplayInner(S2, Black, 0, 0, s2Width, s2Height, image)
	DisplayInner(S2, Red, 0, 0, s2Width, s2Height, image)
	DisplayInner(M2, Black, s2Width, 0, m2Width, m2Height, image)
	DisplayInner(M2, Red, s2Width, 0, m2Width, m2Height, image)
	DisplayInner(S1, Black, m1Width, s2Height, s1Width, s1Height, image)
	DisplayInner(S1, Red, m1Width, s2Height, s1Width, s1Height, image)
	DisplayInner(M1, Black, 0, s2Height, m1Width, m1Height, image)
	DisplayInner(M1, Red, 0, s2Height, m1Width, m1Height, image)

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

	for busy := pin.Read(); busy == rpio.Low; busy = pin.Read() {
		SendCommand(screen, 0x71)
		time.Sleep(200 * time.Millisecond)
	}
	log.Print("ReadBusy() done")
}
