package c64

import "fmt"

type Bus struct {
	cfg    *SystemConfig
	RAM    []uint8
	Basic  []uint8
	Kernal []uint8
	Charen []uint8
	IO     []uint8
	Color  [COLOR_MEM]uint8

	vic *VIC2
	cia [2]*CIA
	sid *SID
	
	cpuPortDDR  uint8 // 6510 CPU port $00 (Data Direction)
	cpuPortData uint8 // 6510 CPU port $01 (Data)
}

func NewBus(cfg *SystemConfig) *Bus {
	return &Bus{
		cfg:         cfg,
		RAM:         cfg.RAM,
		Basic:       cfg.ROMBasic,
		Kernal:      cfg.ROMKernal,
		Charen:      cfg.ROMChar,
		IO:          make([]uint8, 0x1000),
		cpuPortDDR:  0x00,
		cpuPortData: 0x00,
	}
}

func (b *Bus) AttachVIC(vic *VIC2)     { b.vic = vic }
func (b *Bus) AttachCIA(idx int, cia *CIA) {
	if idx >= 0 && idx < 2 {
		b.cia[idx] = cia
	}
}
func (b *Bus) AttachSID(sid *SID)      { b.sid = sid }

func (b *Bus) Read(addr uint16) uint8 {
	switch {
	case addr == 0x0000:
		return b.cpuPortDDR | 0xC0 // Bits 6-7 always 1
	case addr == 0x0001:
		// For input pins (DDR=0), return 1 (pull-up resistor)
		return (b.cpuPortData & b.cpuPortDDR) | (^b.cpuPortDDR & 0xFF)
	case addr < 0x1000:
		return b.RAM[addr]
	case addr >= 0x1000 && addr < 0x8000:
		return b.RAM[addr]
	case addr >= 0x8000 && addr < 0xA000:
		return b.RAM[addr]
	case addr >= 0xA000 && addr < 0xC000:
		if b.loram() && len(b.Basic) > 0 {
			return b.Basic[addr-0xA000]
		}
		return b.RAM[addr]
	case addr >= 0xC000 && addr < 0xD000:
		return b.RAM[addr]
	case addr >= 0xD000 && addr < 0xE000:
		// Color RAM ($D800-$DBFF) is always accessible, regardless of CHAREN
		if addr >= 0xD800 && addr < 0xDC00 {
			return b.Color[addr-0xD800] & 0x0F
		}
		if b.charon() {
			return b.readIO(addr)
		}
		if !b.charon() && !b.hiram() && !b.loram() {
			return b.RAM[addr]
		}
		if !b.charon() && len(b.Charen) > 0 {
			return b.Charen[addr-0xD000]
		}
		return b.RAM[addr]
	case addr >= 0xE000:
		if b.hiram() && len(b.Kernal) > 0 {
			return b.Kernal[addr-0xE000]
		}
		return b.RAM[addr]
	default:
		return 0
	}
}

func (b *Bus) effectivePort() uint8 {
	// Input pins (DDR=0) float high due to pull-up resistors
	return (b.cpuPortData & b.cpuPortDDR) | (^b.cpuPortDDR & 0xFF)
}

func (b *Bus) loram() bool  { return b.effectivePort()&0x01 != 0 }
func (b *Bus) hiram() bool  { return b.effectivePort()&0x02 != 0 }
func (b *Bus) charon() bool { return b.effectivePort()&0x04 != 0 }

func (b *Bus) readIO(addr uint16) uint8 {
	offset := addr & 0x0FFF
	switch {
	case offset >= 0x000 && offset <= 0x3FF:
		if b.vic != nil && offset < 0x040 {
			return b.vic.readReg(uint8(offset & 0x3F))
		}
	case offset >= 0x400 && offset <= 0x7FF:
		if b.sid != nil && offset < 0x420 {
			return b.sid.readReg(uint8(offset & 0x1F))
		}
	case offset >= 0x800 && offset <= 0xBFF:
		return b.Color[offset-0x800] & 0x0F
	case offset >= 0xC00 && offset <= 0xCFF:
		if b.cia[0] != nil && offset < 0xC10 {
			return b.cia[0].readReg(uint8(offset & 0x0F))
		}
	case offset >= 0xD00 && offset <= 0xDFF:
		if b.cia[1] != nil && offset < 0xD10 {
			return b.cia[1].readReg(uint8(offset & 0x0F))
		}
	}
	return b.IO[addr-0xD000]
}

func (b *Bus) Write(addr uint16, val uint8) {
	switch {
	case addr == 0x0000:
		b.cpuPortDDR = val
		return
	case addr == 0x0001:
		b.cpuPortData = val
		return
	case addr < 0x1000:
		b.RAM[addr] = val
	case addr >= 0x1000 && addr < 0x8000:
		b.RAM[addr] = val
	case addr >= 0x8000 && addr < 0xA000:
		b.RAM[addr] = val
	case addr >= 0xA000 && addr < 0xC000:
		b.RAM[addr] = val
	case addr >= 0xC000 && addr < 0xD000:
		b.RAM[addr] = val
	case addr >= 0xD000 && addr < 0xE000:
		// Color RAM ($D800-$DBFF) is always writable, regardless of CHAREN
		if addr >= 0xD800 && addr < 0xDC00 {
			b.Color[addr-0xD800] = val & 0x0F
		}
		// Always write to RAM (even in I/O area, for write-through behavior)
		b.RAM[addr] = val
		b.handleIO(addr, val)
	case addr >= 0xE000:
		b.RAM[addr] = val
	}
}

func (b *Bus) handleIO(addr uint16, val uint8) {
	offset := addr & 0x0FFF
	switch {
	case offset >= 0x000 && offset <= 0x3FF:
		if b.vic != nil {
			b.vic.handleReg(uint8(offset&0x3F), val)
		}
	case offset >= 0x400 && offset <= 0x7FF:
		if b.sid != nil {
			b.sid.handleReg(uint8(offset&0x1F), val)
		}
	case offset >= 0xC00 && offset <= 0xCFF:
		if b.cia[0] != nil {
			b.cia[0].handleReg(uint8(offset&0x0F), val)
		}
	case offset >= 0xD00 && offset <= 0xDFF:
		if b.cia[1] != nil {
			b.cia[1].handleReg(uint8(offset&0x0F), val)
		}
	}
}

func (b *Bus) LoadROM(path string) error {
	return fmt.Errorf("ROM loading not yet implemented")
}
