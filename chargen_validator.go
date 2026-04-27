package c64

// IsValidChargenROM checks if the chargen ROM looks like a real C64 character ROM
// by verifying known character patterns. This prevents garbage files from being used.
// Accepts both standard C64 ROM and VICE variants.
func IsValidChargenROM(rom []uint8) bool {
	if len(rom) != 4096 {
		return false
	}

	// Check space character (0x20) - should be all zeros in any valid ROM
	for i := 0; i < 8; i++ {
		if rom[0x20*8+i] != 0x00 {
			return false
		}
	}

	// Check 'A' character (0x41) - classic C64 uppercase pattern
	// Standard C64: {0x18, 0x3C, 0x66, 0x7E, 0x66, 0x66, 0x66, 0x00}
	// Some VICE ROMs have slightly different patterns, so we accept close variants
	aCharScore := 0
	expectedA := [8]uint8{0x18, 0x3C, 0x66, 0x7E, 0x66, 0x66, 0x66, 0x00}
	for i := 0; i < 8; i++ {
		if rom[0x41*8+i] == expectedA[i] {
			aCharScore++
		}
	}

	// Alternative VICE 'A' pattern: {0x08, 0x1C, 0x3E, 0x7F, 0x7F, 0x1C, 0x3E, 0x00}
	viceAScore := 0
	viceA := [8]uint8{0x08, 0x1C, 0x3E, 0x7F, 0x7F, 0x1C, 0x3E, 0x00}
	for i := 0; i < 8; i++ {
		if rom[0x41*8+i] == viceA[i] {
			viceAScore++
		}
	}

	// Accept if either pattern matches at least 6/8 bytes
	if aCharScore < 6 && viceAScore < 6 {
		return false
	}

	// Check '@' character (0x00) - classic C64 pattern
	// Standard C64: {0x3C, 0x66, 0x6E, 0x76, 0x66, 0x66, 0x3C, 0x00}
	// Some VICE variants: {0x3C, 0x66, 0x6E, 0x6E, 0x60, 0x62, 0x3C, 0x00}
	atCharScore := 0
	expectedAt := [8]uint8{0x3C, 0x66, 0x6E, 0x76, 0x66, 0x66, 0x3C, 0x00}
	for i := 0; i < 8; i++ {
		if rom[0x00*8+i] == expectedAt[i] {
			atCharScore++
		}
	}

	// Accept if space is valid and at least one of A/@ patterns is close enough
	return atCharScore >= 6 || aCharScore >= 7 || viceAScore >= 7
}
