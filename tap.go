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
	if string(data[0:12]) != "C64-TAPE-RAW" {
		return nil, fmt.Errorf("invalid TAP header")
	}
	version := data[12]
	length := uint32(data[16]) | uint32(data[17])<<8 | uint32(data[18])<<16 | uint32(data[19])<<24
	if uint32(len(data)-20) != length {
		return nil, fmt.Errorf("TAP length mismatch: expected %d, got %d", length, len(data)-20)
	}
	return &TAPReader{version: version, data: data[20:]}, nil
}

// getPulses converts raw TAP bytes to pulse durations in microseconds.
// TAP v0: each byte N gives a pulse of N*8 µs; byte 0 = overflow (~2048 µs).
// TAP v1: byte 0 is followed by a 3-byte little-endian exact duration in µs.
func (t *TAPReader) getPulses() []int {
	var pulses []int
	i := 0
	for i < len(t.data) {
		b := t.data[i]
		i++
		if b == 0 {
			if t.version >= 1 && i+2 < len(t.data) {
				dur := int(t.data[i]) | int(t.data[i+1])<<8 | int(t.data[i+2])<<16
				i += 3
				pulses = append(pulses, dur)
			} else {
				pulses = append(pulses, 256*8)
			}
		} else {
			pulses = append(pulses, int(b)*8)
		}
	}
	return pulses
}

// classifyPulse returns 'S' (short), 'M' (medium), 'L' (long), or 0.
// Thresholds are based on the PAL C64 1530 Datassette standard:
//
//	Short  ≈ 180 µs  (144–240)
//	Medium ≈ 264 µs  (240–320)
//	Long   ≈ 360 µs  (320–432)
func (t *TAPReader) classifyPulse(micros int) byte {
	switch {
	case micros >= 144 && micros < 240:
		return 'S'
	case micros >= 240 && micros < 320:
		return 'M'
	case micros >= 320 && micros < 432:
		return 'L'
	default:
		return 0
	}
}

// decodeBits converts a pulse stream to bits using the CBM tape encoding:
//
//	S + M  →  bit 1
//	S + L  →  bit 0
//
// Unrecognised pulse pairs (pilot, gaps, etc.) advance by one pulse and are skipped.
func (t *TAPReader) decodeBits(pulses []int) []byte {
	bits := make([]byte, 0, len(pulses)/2)
	i := 0
	for i+1 < len(pulses) {
		p1 := t.classifyPulse(pulses[i])
		p2 := t.classifyPulse(pulses[i+1])
		switch {
		case p1 == 'S' && p2 == 'M':
			bits = append(bits, 1)
			i += 2
		case p1 == 'S' && p2 == 'L':
			bits = append(bits, 0)
			i += 2
		default:
			i++
		}
	}
	return bits
}

// bitsToBytes packs bits into bytes, bitsPerByte bits each (LSB first).
// The standard CBM format uses 9 bits per byte (8 data + 1 XOR checksum);
// we extract only the 8 data bits and ignore the checksum bit.
func bitsToBytes(bits []byte, bitsPerByte int) []byte {
	var result []byte
	for i := 0; i+bitsPerByte-1 < len(bits); i += bitsPerByte {
		var b byte
		for j := 0; j < 8; j++ {
			if bits[i+j] != 0 {
				b |= 1 << uint(j)
			}
		}
		result = append(result, b)
	}
	return result
}

// findSync searches for the CBM sync sequence $89 $88 … $81 starting at from.
// Returns the index of the first data byte after the sync, or -1.
func findSync(rawBytes []byte, from int) int {
	for i := from; i+8 < len(rawBytes); i++ {
		if rawBytes[i] != 0x89 {
			continue
		}
		ok := true
		for j := 1; j <= 8; j++ {
			if i+j >= len(rawBytes) || rawBytes[i+j] != byte(0x89-j) {
				ok = false
				break
			}
		}
		if ok {
			return i + 9
		}
	}
	return -1
}

// ExtractPRG decodes the first PRG from the TAP data using the standard
// Commodore 64 CBM tape protocol.  It tries both 9-bit (data + checksum)
// and 8-bit groupings so it handles most real-world TAP files.
func (t *TAPReader) ExtractPRG() ([]byte, error) {
	pulses := t.getPulses()
	if len(pulses) < 100 {
		return nil, fmt.Errorf("TAP: insufficient pulse data (%d pulses)", len(pulses))
	}

	bits := t.decodeBits(pulses)
	if len(bits) < 200 {
		return nil, fmt.Errorf("TAP: too few decodable bits (%d)", len(bits))
	}

	// Try 9-bit grouping first (standard CBM: 8 data + 1 checksum), then 8-bit.
	for _, bpb := range []int{9, 8} {
		rawBytes := bitsToBytes(bits, bpb)

		sync1 := findSync(rawBytes, 0)
		if sync1 < 0 || sync1+192 > len(rawBytes) {
			continue
		}

		header := rawBytes[sync1 : sync1+192]
		// File type: 0x01 = relocatable PRG, 0x03 = absolute ML
		if header[0] != 0x01 && header[0] != 0x03 {
			continue
		}

		startAddr := uint16(header[1]) | uint16(header[2])<<8
		endAddr := uint16(header[3]) | uint16(header[4])<<8
		if endAddr <= startAddr {
			continue
		}
		dataLen := int(endAddr - startAddr)

		sync2 := findSync(rawBytes, sync1+192)
		if sync2 < 0 || sync2+dataLen > len(rawBytes) {
			continue
		}

		// Build a standard PRG: 2-byte load address followed by program data.
		prg := make([]byte, 2+dataLen)
		prg[0] = byte(startAddr)
		prg[1] = byte(startAddr >> 8)
		copy(prg[2:], rawBytes[sync2:sync2+dataLen])
		return prg, nil
	}

	return nil, fmt.Errorf("TAP: could not find valid CBM program block")
}
