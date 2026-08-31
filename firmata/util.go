package firmata

import (
	"fmt"
)

const SevenBitMask byte = 0b01111111

func TwoByteToByte(a, b byte) byte {
	return (a & SevenBitMask) | ((b & SevenBitMask) << 7)
}

func TwoByteToWord(a, b byte) uint16 {
	return uint16(a&SevenBitMask) | (uint16(b&SevenBitMask) << 7)
}

func TwoByteString(bytes []byte) string {
	if len(bytes)%2 == 1 {
		bytes = append(bytes, 0)
	}

	var s string
	for i := 0; i < len(bytes); i += 2 {
		s += string(TwoByteToByte(bytes[i], bytes[i+1]))
	}
	return s
}

func TwoByteRepresentationToByteSlice(bytes []byte) []byte {
	if len(bytes)%2 == 1 {
		bytes = append(bytes, 0)
	}

	d := make([]byte, len(bytes)/2)
	i := 0
	for di := range d {
		d[di] = TwoByteToByte(bytes[i], bytes[i+1])
		i += 2
	}
	return d
}

func ByteToTwoByte(b byte) (lsb, msb byte) {
	return b & SevenBitMask, (b >> 7) & SevenBitMask
}

func WordToTwoByte(b uint16) (lsb, msb byte) {
	return byte(b) & SevenBitMask, byte(b>>7) & SevenBitMask
}

func ByteSliceToTwoByteRepresentation(bytes []byte) []byte {
	d := make([]byte, len(bytes)*2)
	i := 0
	for _, b := range bytes {
		d[i], d[i+1] = ByteToTwoByte(b)
		i += 2
	}
	return d
}

func SprintHexArray(data []byte) string {
	s := ""
	if len(data) == 0 {
		return s
	}
	for _, b := range data {
		s += fmt.Sprintf("0x%02X ", b)
	}
	return s[:len(s)-1]
}

// Encoder7Bit logic determined from here:
//  - ConfigurableFirmata@2.10.1/src/Encoder7Bit.cpp:34
func Encoder7Bit(inData []byte) []byte {
	var outData []byte
	var previous byte
	var shift = 0
	for _, data := range inData {
		if shift == 0 {
			outData = append(outData, data&0x7f)
			shift++
			previous = data >> 7
		} else {
			outData = append(outData, ((data<<shift)&0x7f)|previous)
			if shift == 6 {
				outData = append(outData, data>>1)
				shift = 0
			} else {
				shift++
				previous = data >> (8 - shift)
			}
		}
	}
	if shift > 0 {
		outData = append(outData, previous)
	}
	return outData
}

// Decoder7Bit logic determined from here:
//  - ConfigurableFirmata@2.10.1/src/Encoder7Bit.h:17
//  - ConfigurableFirmata@2.10.1/src/Encoder7Bit.cpp:54
func Decoder7Bit(inData []byte) []byte {
	var outBytes = ((len(inData)) * 7) >> 3

	var outData = make([]byte, outBytes)
	for i := 0; i < outBytes; i++ {
		var j = i << 3
		var pos = j / 7
		var shift = byte(j % 7)
		outData[i] = (inData[pos] >> shift) | ((inData[pos+1] << (7 - shift)) & 0xFF)
	}
	return outData
}
