package firmata

import (
	"fmt"
	"reflect"
	"testing"
)

func TestByteConversion(t *testing.T) {
	for i := uint16(0x00); i <= 0xFF; i++ {
		t.Run(fmt.Sprintf("0x%02X", i), func(t *testing.T) {
			a, b := ByteToTwoByte(byte(i))
			o := TwoByteToByte(a, b)
			if byte(i) != o {
				t.Errorf("ByteToTwoByte(0x%02X) = 0x%02X, 0x%02X => TwoByteToByte() = 0x%02X", i, a, b, o)
			}
		})
	}
}

func TestTwoByteString(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{
			name:  "nil",
			bytes: nil,
			want:  "",
		},
		{
			name:  "empty",
			bytes: []byte{},
			want:  "",
		},
		{
			name: "test string",
			bytes: ByteSliceToTwoByteRepresentation([]byte{
				0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6E, 0x67,
			}),
			want: "test string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TwoByteString(tt.bytes); got != tt.want {
				t.Errorf("TwoByteString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestByteSliceTo2ByteRepresentation(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "nil",
			input:    nil,
			expected: []byte{},
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "7 lsb set",
			input:    []byte{0b01111111, 0b11111111},
			expected: []byte{0b01111111, 0b00000000, 0b01111111, 0b00000001},
		},
		{
			name:     "7 lsb not set",
			input:    []byte{0b00000000, 0b10000000},
			expected: []byte{0b00000000, 0b00000000, 0b00000000, 0b00000001},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ByteSliceToTwoByteRepresentation(tt.input); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ByteSliceToTwoByteRepresentation() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTwoByteRepresentationToByteSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "nil",
			input:    nil,
			expected: []byte{},
		},
		{
			name:     "empty",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "7 lsb set",
			input:    []byte{0b01111111, 0b00000000, 0b01111111, 0b00000001},
			expected: []byte{0b01111111, 0b11111111},
		},
		{
			name:     "7 lsb not set",
			input:    []byte{0b00000000, 0b00000000, 0b00000000, 0b00000001},
			expected: []byte{0b00000000, 0b10000000},
		},
		{
			name:     "only 1 byte",
			input:    []byte{0b01000000},
			expected: []byte{0b01000000},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TwoByteRepresentationToByteSlice(tt.input); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("TwoByteRepresentationToByteSlice() = %v, want %v", got, tt.expected)
			}
		})
	}
}
