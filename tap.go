package c64

import (
	"fmt"
	"os"
)

// TAPReader reads Commodore 64 tape images (.tap)
type TAPReader struct {
	version uint8
	data    []byte
}

func NewTAPReader(path string) (*TAPReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 20 {
		return nil, fmt.Errorf("TAP file too short")
	}
	// Check header
	if string(data[0:12]) != "C64-TAPE-RAW" {
		return nil, fmt.Errorf("invalid TAP header")
	}
	version := data[12]
	// Platform (13-15) usually $00 $00 $00
	// Length at 16-19 (little endian, excluding header)
	length := uint32(data[16]) | uint32(data[17])<<8 | uint32(data[18])<<16 | uint32(data[19])<<24
	if uint32(len(data)-20) != length {
		return nil, fmt.Errorf("TAP length mismatch: expected %d, got %d", length, len(data)-20)
	}
	return &TAPReader{version: version, data: data[20:]}, nil
}

// ExtractPRG attempts to extract the first PRG from the TAP data
// This is a simplified decoder that looks for standard C64 tape headers
func (t *TAPReader) ExtractPRG() ([]byte, error) {
	// Simplified: search for data pulses that decode to a PRG header
	// A real decoder would process pulse lengths (version 0: $00=overflow, else length in microseconds)
	// For now, we look for common PRG signatures in the raw pulse data
	// This is a heuristic and won't work for all TAP files
	for i := 0; i < len(t.data)-2; i++ {
		if t.data[i] == 0x01 && t.data[i+1] == 0x08 {
			// Potential load address $0801
			// Heuristic: extract up to 65535 bytes
			start := i
			end := i + 65536
			if end > len(t.data) {
				end = len(t.data)
			}
			return t.data[start:end], nil
		}
	}
	return nil, fmt.Errorf("could not find PRG data in TAP")
}
