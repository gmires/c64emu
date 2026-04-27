package c64

import (
	"fmt"
	"os"
)

// CRTReader reads C64 cartridge images
type CRTReader struct {
	name     string
	hwType   uint16
	exrom    bool
	game     bool
	romBanks []romBank
}

type romBank struct {
	bankNum uint16
	loadAddr uint16
	data     []byte
}

func NewCRTReader(path string) (*CRTReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 64 {
		return nil, fmt.Errorf("CRT file too short")
	}
	if string(data[0:16]) != "C64 CARTRIDGE   " {
		return nil, fmt.Errorf("invalid CRT header")
	}
	// Version at offset 16-17 (big endian)
	// hwType at 18-19 (big endian)
	hwType := uint16(data[18])<<8 | uint16(data[19])
	exrom := data[20] != 0
	game := data[21] != 0
	nameBytes := data[32:64]
	name := ""
	for _, b := range nameBytes {
		if b == 0 {
			break
		}
		name += string(b)
	}

	crt := &CRTReader{
		name:   name,
		hwType: hwType,
		exrom:  exrom,
		game:   game,
	}

	// Read chip packets
	offset := 64
	for offset+20 <= len(data) {
		if string(data[offset:offset+4]) != "CHIP" {
			break
		}
		pktLen := uint32(data[offset+4])<<24 | uint32(data[offset+5])<<16 | uint32(data[offset+6])<<8 | uint32(data[offset+7])
		// chipType := uint16(data[offset+8])<<8 | uint16(data[offset+9])
		bankNum := uint16(data[offset+10])<<8 | uint16(data[offset+11])
		loadAddr := uint16(data[offset+12])<<8 | uint16(data[offset+13])
		romSize := uint16(data[offset+14])<<8 | uint16(data[offset+15])
		if offset+20+int(romSize) > len(data) {
			break
		}
		crt.romBanks = append(crt.romBanks, romBank{
			bankNum:  bankNum,
			loadAddr: loadAddr,
			data:     append([]byte{}, data[offset+20:offset+20+int(romSize)]...),
		})
		offset += int(pktLen)
	}

	return crt, nil
}

func (c *CRTReader) Name() string     { return c.name }
func (c *CRTReader) HWType() uint16   { return c.hwType }
func (c *CRTReader) Banks() []romBank { return c.romBanks }

// LoadIntoMachine maps the cartridge ROMs into the machine memory
func (c *CRTReader) LoadIntoMachine(machine *Machine) error {
	for _, bank := range c.romBanks {
		addr := bank.loadAddr
		for i, b := range bank.data {
			machine.Bus().Write(addr+uint16(i), b)
		}
	}
	return nil
}
