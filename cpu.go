package c64

import "fmt"

const (
	FlagC = 1 << iota
	FlagZ
	FlagI
	FlagD
	FlagB
	FlagR = 1 << 5
	FlagV
	FlagN
)

type TraceEntry struct {
	PC      uint16
	Opcode  uint8
	A, X, Y uint8
	SP      uint8
	SR      uint8
}

type CPU6510 struct {
	A, X, Y uint8
	SP      uint8
	SR      uint8
	PC      uint16

	clk      uint64
	bus      *Bus
	pulseNMI bool
	pulseIRQ bool
	nmiEdge  bool
}

func NewCPU6510(bus *Bus) *CPU6510 {
	return &CPU6510{
		SP: 0xFF,
		SR: FlagR,
		bus: bus,
	}
}

func (c *CPU6510) Reset() {
	c.PC = uint16(c.bus.Read(0xFFFC)) | uint16(c.bus.Read(0xFFFD))<<8
	c.SP = 0xFF
	c.SR = FlagR | FlagI
	c.clk = 0
}

func (c *CPU6510) step() uint8 {
	opcode := c.fetch()
	return c.execute(opcode)
}

func (c *CPU6510) fetch() uint8 {
	opcode := c.bus.Read(c.PC)
	c.PC++
	return opcode
}



func (c *CPU6510) read(addr uint16) uint8 {
	return c.bus.Read(addr)
}

func (c *CPU6510) write(addr uint16, val uint8) {
	c.bus.Write(addr, val)
}

func (c *CPU6510) push(val uint8) {
	c.write(0x0100|uint16(c.SP), val)
	c.SP--
}

func (c *CPU6510) pop() uint8 {
	c.SP++
	return c.read(0x0100 | uint16(c.SP))
}

func (c *CPU6510) setFlag(flag uint8, val bool) {
	if val {
		c.SR |= flag
	} else {
		c.SR &^= flag
	}
}

func (c *CPU6510) getFlag(flag uint8) bool {
	return c.SR&flag != 0
}

func (c *CPU6510) Clock() uint64 {
	return c.clk
}

func (c *CPU6510) ClockAdd(cycles uint64) {
	c.clk += cycles
}

func (c *CPU6510) NMI() {
	c.push(uint8(c.PC >> 8))
	c.push(uint8(c.PC))
	c.push(c.SR &^ FlagB)
	c.setFlag(FlagI, true)
	c.PC = uint16(c.bus.Read(0xFFFA)) | uint16(c.bus.Read(0xFFFB))<<8
	c.clk += 7
}

func (c *CPU6510) IRQ() {
	if c.getFlag(FlagI) {
		return
	}
	c.push(uint8(c.PC >> 8))
	c.push(uint8(c.PC))
	c.push(c.SR &^ FlagB)
	c.setFlag(FlagI, true)
	c.PC = uint16(c.bus.Read(0xFFFE)) | uint16(c.bus.Read(0xFFFF))<<8
	c.clk += 7
}

// Addressing modes
func (c *CPU6510) zp() uint16     { return uint16(c.fetch()) }
func (c *CPU6510) zpX() uint16    { return uint16((c.fetch() + c.X) & 0xFF) }
func (c *CPU6510) zpY() uint16    { return uint16((c.fetch() + c.Y) & 0xFF) }
func (c *CPU6510) abs() uint16    { lo := c.fetch(); hi := c.fetch(); return uint16(lo) | uint16(hi)<<8 }
func (c *CPU6510) absX() uint16   { lo := c.fetch(); hi := c.fetch(); return (uint16(lo) | uint16(hi)<<8) + uint16(c.X) }
func (c *CPU6510) absY() uint16   { lo := c.fetch(); hi := c.fetch(); return (uint16(lo) | uint16(hi)<<8) + uint16(c.Y) }
func (c *CPU6510) indX() uint16   { zp := uint16((c.fetch() + c.X) & 0xFF); lo := c.read(zp); hi := c.read((zp + 1) & 0xFF); return uint16(lo) | uint16(hi)<<8 }
func (c *CPU6510) indY() uint16   { zp := uint16(c.fetch()); lo := c.read(zp); hi := c.read((zp + 1) & 0xFF); return (uint16(lo) | uint16(hi)<<8) + uint16(c.Y) }
func (c *CPU6510) ind() uint16 {
	lo := c.fetch()
	hi := c.fetch()
	ptr := uint16(lo) | uint16(hi)<<8
	// Replicate the 6502 page-boundary bug: if ptr low byte is $FF, the high
	// byte of the target address wraps to $xx00 instead of advancing to $xx+1,00.
	hiPtr := (ptr & 0xFF00) | uint16(uint8(lo+1))
	return uint16(c.read(ptr)) | uint16(c.read(hiPtr))<<8
}

// pageCross returns 1 when indexing crosses a page boundary.
// finalAddr is the already-indexed address (base + reg); base = finalAddr - reg.
func (c *CPU6510) pageCross(finalAddr, reg uint16) uint8 {
	if ((finalAddr-reg) & 0xFF00) != (finalAddr & 0xFF00) {
		return 1
	}
	return 0
}

func (c *CPU6510) setNZ(val uint8) {
	c.setFlag(FlagZ, val == 0)
	c.setFlag(FlagN, val&0x80 != 0)
}

func (c *CPU6510) execute(op uint8) uint8 {
	var val uint8
	var carry uint8
	switch op {
	// ADC
	case 0x69: // #imm
		c.adc(c.fetch())
		return 2
	case 0x65: // zp
		c.adc(c.read(c.zp()))
		return 3
	case 0x75: // zp,X
		c.adc(c.read(c.zpX()))
		return 4
	case 0x6D: // abs
		c.adc(c.read(c.abs()))
		return 4
	case 0x7D: // abs,X
		addr := c.absX()
		c.adc(c.read(addr))
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0x79: // abs,Y
		addr := c.absY()
		c.adc(c.read(addr))
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0x61: // (zp,X)
		c.adc(c.read(c.indX()))
		return 6
	case 0x71: // (zp),Y
		addr := c.indY()
		c.adc(c.read(addr))
		return 5 + c.pageCross(addr, uint16(c.Y))

	// SBC
	case 0xE9: // #imm
		c.sbc(c.fetch())
		return 2
	case 0xE5: // zp
		c.sbc(c.read(c.zp()))
		return 3
	case 0xF5: // zp,X
		c.sbc(c.read(c.zpX()))
		return 4
	case 0xED: // abs
		c.sbc(c.read(c.abs()))
		return 4
	case 0xFD: // abs,X
		addr := c.absX()
		c.sbc(c.read(addr))
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0xF9: // abs,Y
		addr := c.absY()
		c.sbc(c.read(addr))
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0xE1: // (zp,X)
		c.sbc(c.read(c.indX()))
		return 6
	case 0xF1: // (zp),Y
		addr := c.indY()
		c.sbc(c.read(addr))
		return 5 + c.pageCross(addr, uint16(c.Y))

	// AND
	case 0x29:
		c.A &= c.fetch()
		c.setNZ(c.A)
		return 2
	case 0x25:
		c.A &= c.read(c.zp())
		c.setNZ(c.A)
		return 3
	case 0x35:
		c.A &= c.read(c.zpX())
		c.setNZ(c.A)
		return 4
	case 0x2D:
		c.A &= c.read(c.abs())
		c.setNZ(c.A)
		return 4
	case 0x3D:
		addr := c.absX()
		c.A &= c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0x39:
		addr := c.absY()
		c.A &= c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0x21:
		c.A &= c.read(c.indX())
		c.setNZ(c.A)
		return 6
	case 0x31:
		addr := c.indY()
		c.A &= c.read(addr)
		c.setNZ(c.A)
		return 5 + c.pageCross(addr, uint16(c.Y))

	// ORA
	case 0x09:
		c.A |= c.fetch()
		c.setNZ(c.A)
		return 2
	case 0x05:
		c.A |= c.read(c.zp())
		c.setNZ(c.A)
		return 3
	case 0x15:
		c.A |= c.read(c.zpX())
		c.setNZ(c.A)
		return 4
	case 0x0D:
		c.A |= c.read(c.abs())
		c.setNZ(c.A)
		return 4
	case 0x1D:
		addr := c.absX()
		c.A |= c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0x19:
		addr := c.absY()
		c.A |= c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0x01:
		c.A |= c.read(c.indX())
		c.setNZ(c.A)
		return 6
	case 0x11:
		addr := c.indY()
		c.A |= c.read(addr)
		c.setNZ(c.A)
		return 5 + c.pageCross(addr, uint16(c.Y))

	// EOR
	case 0x49:
		c.A ^= c.fetch()
		c.setNZ(c.A)
		return 2
	case 0x45:
		c.A ^= c.read(c.zp())
		c.setNZ(c.A)
		return 3
	case 0x55:
		c.A ^= c.read(c.zpX())
		c.setNZ(c.A)
		return 4
	case 0x4D:
		c.A ^= c.read(c.abs())
		c.setNZ(c.A)
		return 4
	case 0x5D:
		addr := c.absX()
		c.A ^= c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0x59:
		addr := c.absY()
		c.A ^= c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0x41:
		c.A ^= c.read(c.indX())
		c.setNZ(c.A)
		return 6
	case 0x51:
		addr := c.indY()
		c.A ^= c.read(addr)
		c.setNZ(c.A)
		return 5 + c.pageCross(addr, uint16(c.Y))

	// CMP
	case 0xC9:
		c.cmp(c.A, c.fetch())
		return 2
	case 0xC5:
		c.cmp(c.A, c.read(c.zp()))
		return 3
	case 0xD5:
		c.cmp(c.A, c.read(c.zpX()))
		return 4
	case 0xCD:
		c.cmp(c.A, c.read(c.abs()))
		return 4
	case 0xDD:
		addr := c.absX()
		c.cmp(c.A, c.read(addr))
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0xD9:
		addr := c.absY()
		c.cmp(c.A, c.read(addr))
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0xC1:
		c.cmp(c.A, c.read(c.indX()))
		return 6
	case 0xD1:
		addr := c.indY()
		c.cmp(c.A, c.read(addr))
		return 5 + c.pageCross(addr, uint16(c.Y))

	// CPX
	case 0xE0:
		c.cmp(c.X, c.fetch())
		return 2
	case 0xE4:
		c.cmp(c.X, c.read(c.zp()))
		return 3
	case 0xEC:
		c.cmp(c.X, c.read(c.abs()))
		return 4

	// CPY
	case 0xC0:
		c.cmp(c.Y, c.fetch())
		return 2
	case 0xC4:
		c.cmp(c.Y, c.read(c.zp()))
		return 3
	case 0xCC:
		c.cmp(c.Y, c.read(c.abs()))
		return 4

	// BIT
	case 0x24:
		c.bit(c.read(c.zp()))
		return 3
	case 0x2C:
		c.bit(c.read(c.abs()))
		return 4

	// Branch
	case 0x10: // BPL
		return c.branch(!c.getFlag(FlagN))
	case 0x30: // BMI
		return c.branch(c.getFlag(FlagN))
	case 0x50: // BVC
		return c.branch(!c.getFlag(FlagV))
	case 0x70: // BVS
		return c.branch(c.getFlag(FlagV))
	case 0x90: // BCC
		return c.branch(!c.getFlag(FlagC))
	case 0xB0: // BCS
		return c.branch(c.getFlag(FlagC))
	case 0xD0: // BNE
		return c.branch(!c.getFlag(FlagZ))
	case 0xF0: // BEQ
		return c.branch(c.getFlag(FlagZ))

	// JMP, JSR, RTS, RTI, BRK
	case 0x4C: // JMP abs
		c.PC = c.abs()
		return 3
	case 0x6C: // JMP ind
		c.PC = c.ind()
		return 5
	case 0x20: // JSR
		addr := c.abs()
		c.PC--
		c.push(uint8(c.PC >> 8))
		c.push(uint8(c.PC))
		c.PC = addr
		return 6
	case 0x60: // RTS
		lo := uint16(c.pop())
		hi := uint16(c.pop()) << 8
		c.PC = hi | lo
		c.PC++
		return 6
	case 0x40: // RTI
		c.SR = c.pop() | FlagR
		lo := uint16(c.pop())
		hi := uint16(c.pop()) << 8
		c.PC = hi | lo
		return 6
	case 0x00: // BRK
		c.PC++
		c.push(uint8(c.PC >> 8))
		c.push(uint8(c.PC))
		c.push(c.SR | FlagB)
		c.setFlag(FlagI, true)
		c.PC = uint16(c.read(0xFFFE)) | uint16(c.read(0xFFFF))<<8
		return 7

	// Transfer
	case 0xAA: // TAX
		c.X = c.A
		c.setNZ(c.X)
		return 2
	case 0xA8: // TAY
		c.Y = c.A
		c.setNZ(c.Y)
		return 2
	case 0x8A: // TXA
		c.A = c.X
		c.setNZ(c.A)
		return 2
	case 0x98: // TYA
		c.A = c.Y
		c.setNZ(c.A)
		return 2
	case 0x9A: // TXS
		c.SP = c.X
		return 2
	case 0xBA: // TSX
		c.X = c.SP
		c.setNZ(c.X)
		return 2

	// Stack
	case 0x48: // PHA
		c.push(c.A)
		return 3
	case 0x68: // PLA
		c.A = c.pop()
		c.setNZ(c.A)
		return 4
	case 0x08: // PHP
		c.push(c.SR | FlagB | FlagR)
		return 3
	case 0x28: // PLP
		c.SR = c.pop() | FlagR
		return 4

	// Flag ops
	case 0x38: // SEC
		c.setFlag(FlagC, true)
		return 2
	case 0x18: // CLC
		c.setFlag(FlagC, false)
		return 2
	case 0xF8: // SED
		c.setFlag(FlagD, true)
		return 2
	case 0xD8: // CLD
		c.setFlag(FlagD, false)
		return 2
	case 0x78: // SEI
		c.setFlag(FlagI, true)
		return 2
	case 0x58: // CLI
		c.setFlag(FlagI, false)
		return 2
	case 0xB8: // CLV
		c.setFlag(FlagV, false)
		return 2

	// NOP
	case 0xEA:
		return 2
	case 0x1A, 0x3A, 0x5A, 0x7A, 0xDA, 0xFA: // NOP variants
		return 2
	case 0x3C: // NOP abs,X (undocumented)
		_ = c.absX()
		return 4

	// LDA
	case 0xA9:
		c.A = c.fetch()
		c.setNZ(c.A)
		return 2
	case 0xA5:
		c.A = c.read(c.zp())
		c.setNZ(c.A)
		return 3
	case 0xB5:
		c.A = c.read(c.zpX())
		c.setNZ(c.A)
		return 4
	case 0xAD:
		c.A = c.read(c.abs())
		c.setNZ(c.A)
		return 4
	case 0xBD:
		addr := c.absX()
		c.A = c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.X))
	case 0xB9:
		addr := c.absY()
		c.A = c.read(addr)
		c.setNZ(c.A)
		return 4 + c.pageCross(addr, uint16(c.Y))
	case 0xA1:
		c.A = c.read(c.indX())
		c.setNZ(c.A)
		return 6
	case 0xB1:
		addr := c.indY()
		c.A = c.read(addr)
		c.setNZ(c.A)
		return 5 + c.pageCross(addr, uint16(c.Y))

	// LDX
	case 0xA2:
		c.X = c.fetch()
		c.setNZ(c.X)
		return 2
	case 0xA6:
		c.X = c.read(c.zp())
		c.setNZ(c.X)
		return 3
	case 0xB6:
		c.X = c.read(c.zpY())
		c.setNZ(c.X)
		return 4
	case 0xAE:
		c.X = c.read(c.abs())
		c.setNZ(c.X)
		return 4
	case 0xBE:
		addr := c.absY()
		c.X = c.read(addr)
		c.setNZ(c.X)
		return 4 + c.pageCross(addr, uint16(c.Y))

	// LDY
	case 0xA0:
		c.Y = c.fetch()
		c.setNZ(c.Y)
		return 2
	case 0xA4:
		c.Y = c.read(c.zp())
		c.setNZ(c.Y)
		return 3
	case 0xB4:
		c.Y = c.read(c.zpX())
		c.setNZ(c.Y)
		return 4
	case 0xAC:
		c.Y = c.read(c.abs())
		c.setNZ(c.Y)
		return 4
	case 0xBC:
		addr := c.absX()
		c.Y = c.read(addr)
		c.setNZ(c.Y)
		return 4 + c.pageCross(addr, uint16(c.X))

	// STA
	case 0x85:
		c.write(c.zp(), c.A)
		return 3
	case 0x95:
		c.write(c.zpX(), c.A)
		return 4
	case 0x8D:
		c.write(c.abs(), c.A)
		return 4
	case 0x9D:
		c.write(c.absX(), c.A)
		return 5
	case 0x99:
		c.write(c.absY(), c.A)
		return 5
	case 0x81:
		c.write(c.indX(), c.A)
		return 6
	case 0x91:
		c.write(c.indY(), c.A)
		return 6

	// STX
	case 0x86:
		c.write(c.zp(), c.X)
		return 3
	case 0x96:
		c.write(c.zpY(), c.X)
		return 4
	case 0x8E:
		c.write(c.abs(), c.X)
		return 4

	// STY
	case 0x84:
		c.write(c.zp(), c.Y)
		return 3
	case 0x94:
		c.write(c.zpX(), c.Y)
		return 4
	case 0x8C:
		c.write(c.abs(), c.Y)
		return 4

	// INC
	case 0xE6:
		addr := c.zp()
		val := c.read(addr) + 1
		c.write(addr, val)
		c.setNZ(val)
		return 5
	case 0xF6:
		addr := c.zpX()
		val := c.read(addr) + 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0xEE:
		addr := c.abs()
		val := c.read(addr) + 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0xFE:
		addr := c.absX()
		val := c.read(addr) + 1
		c.write(addr, val)
		c.setNZ(val)
		return 7

	// DEC
	case 0xC6:
		addr := c.zp()
		val := c.read(addr) - 1
		c.write(addr, val)
		c.setNZ(val)
		return 5
	case 0xD6:
		addr := c.zpX()
		val := c.read(addr) - 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0xCE:
		addr := c.abs()
		val := c.read(addr) - 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0xDE:
		addr := c.absX()
		val := c.read(addr) - 1
		c.write(addr, val)
		c.setNZ(val)
		return 7

	// INX, INY, DEX, DEY
	case 0xE8:
		c.X++
		c.setNZ(c.X)
		return 2
	case 0xC8:
		c.Y++
		c.setNZ(c.Y)
		return 2
	case 0xCA:
		c.X--
		c.setNZ(c.X)
		return 2
	case 0x88:
		c.Y--
		c.setNZ(c.Y)
		return 2

	// ASL
	case 0x0A: // A
		c.setFlag(FlagC, c.A&0x80 != 0)
		c.A <<= 1
		c.setNZ(c.A)
		return 2
	case 0x06: // zp
		addr := c.zp()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.setNZ(val)
		return 5
	case 0x16: // zp,X
		addr := c.zpX()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x0E: // abs
		addr := c.abs()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x1E: // abs,X
		addr := c.absX()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.setNZ(val)
		return 7

	// LSR
	case 0x4A: // A
		c.setFlag(FlagC, c.A&0x01 != 0)
		c.A >>= 1
		c.setNZ(c.A)
		return 2
	case 0x46: // zp
		addr := c.zp()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.setNZ(val)
		return 5
	case 0x56: // zp,X
		addr := c.zpX()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x4E: // abs
		addr := c.abs()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x5E: // abs,X
		addr := c.absX()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.setNZ(val)
		return 7

	// ROL
	case 0x2A: // A
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, c.A&0x80 != 0)
		c.A = (c.A << 1) | carry
		c.setNZ(c.A)
		return 2
	case 0x26: // zp
		addr := c.zp()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.setNZ(val)
		return 5
	case 0x36: // zp,X
		addr := c.zpX()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x2E: // abs
		addr := c.abs()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x3E: // abs,X
		addr := c.absX()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.setNZ(val)
		return 7

	// ROR
	case 0x6A: // A
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, c.A&0x01 != 0)
		c.A = (c.A >> 1) | (carry << 7)
		c.setNZ(c.A)
		return 2
	case 0x66: // zp
		addr := c.zp()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.setNZ(val)
		return 5
	case 0x76: // zp,X
		addr := c.zpX()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x6E: // abs
		addr := c.abs()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.setNZ(val)
		return 6
	case 0x7E: // abs,X
		addr := c.absX()
		val := c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.setNZ(val)
		return 7

	// Undocumented opcodes used by C64 KERNAL
	// $03: SLO (zp,X) - ASL + ORA
	case 0x03:
		addr := c.indX()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 8

	// $0F: SLO abs - ASL + ORA
	case 0x0F:
		addr := c.abs()
		val := c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 6

	// $E7: ISC zp - INC + SBC
	case 0xE7:
		addr := c.zp()
		val := c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 5

	// $F4: NOP zp,X (undocumented NOP)
	case 0xF4:
		_ = c.zpX()
		return 4

	// ============================================
	// OPCODE UNDOCUMENTATI COMPLETI (dal codice VICE)
	// ============================================

	// --- SLO (ASL + ORA) ---
	case 0x07: // SLO zp
		addr := c.zp()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 5
	case 0x13: // SLO (zp),Y
		addr := c.indY()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 8
	case 0x17: // SLO zp,X
		addr := c.zpX()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 6
	case 0x1B: // SLO abs,Y
		addr := c.absY()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 7
	case 0x1F: // SLO abs,X
		addr := c.absX()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x80 != 0)
		val <<= 1
		c.write(addr, val)
		c.A |= val
		c.setNZ(c.A)
		return 7

	// --- RLA (ROL + AND) ---
	case 0x23: // RLA (zp,X)
		addr := c.indX()
		val = c.read(addr)
		carry := uint8(0)
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 8
	case 0x27: // RLA zp
		addr := c.zp()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 5
	case 0x2F: // RLA abs
		addr := c.abs()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 6
	case 0x33: // RLA (zp),Y
		addr := c.indY()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 8
	case 0x37: // RLA zp,X
		addr := c.zpX()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 6
	case 0x3B: // RLA abs,Y
		addr := c.absY()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 7
	case 0x3F: // RLA abs,X
		addr := c.absX()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x80 != 0)
		val = (val << 1) | carry
		c.write(addr, val)
		c.A &= val
		c.setNZ(c.A)
		return 7

	// --- SRE (LSR + EOR) ---
	case 0x43: // SRE (zp,X)
		addr := c.indX()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 8
	case 0x47: // SRE zp
		addr := c.zp()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 5
	case 0x4F: // SRE abs
		addr := c.abs()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 6
	case 0x53: // SRE (zp),Y
		addr := c.indY()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 8
	case 0x57: // SRE zp,X
		addr := c.zpX()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 6
	case 0x5B: // SRE abs,Y
		addr := c.absY()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 7
	case 0x5F: // SRE abs,X
		addr := c.absX()
		val = c.read(addr)
		c.setFlag(FlagC, val&0x01 != 0)
		val >>= 1
		c.write(addr, val)
		c.A ^= val
		c.setNZ(c.A)
		return 7

	// --- RRA (ROR + ADC) ---
	case 0x63: // RRA (zp,X)
		addr := c.indX()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 8
	case 0x67: // RRA zp
		addr := c.zp()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 5
	case 0x6F: // RRA abs
		addr := c.abs()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 6
	case 0x73: // RRA (zp),Y
		addr := c.indY()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 8
	case 0x77: // RRA zp,X
		addr := c.zpX()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 6
	case 0x7B: // RRA abs,Y
		addr := c.absY()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 7
	case 0x7F: // RRA abs,X
		addr := c.absX()
		val = c.read(addr)
		carry = 0
		if c.getFlag(FlagC) {
			carry = 1
		}
		c.setFlag(FlagC, val&0x01 != 0)
		val = (val >> 1) | (carry << 7)
		c.write(addr, val)
		c.adc(val)
		return 7

	// --- SAX (Store A & X) ---
	case 0x83: // SAX (zp,X)
		addr := c.indX()
		c.write(addr, c.A&c.X)
		return 6
	case 0x87: // SAX zp
		addr := c.zp()
		c.write(addr, c.A&c.X)
		return 3
	case 0x8F: // SAX abs
		addr := c.abs()
		c.write(addr, c.A&c.X)
		return 4
	case 0x97: // SAX zp,Y
		addr := c.zpY()
		c.write(addr, c.A&c.X)
		return 4

	// --- LAX (Load A & X) ---
	case 0xA3: // LAX (zp,X)
		addr := c.indX()
		c.A = c.read(addr)
		c.X = c.A
		c.setNZ(c.A)
		return 6
	case 0xA7: // LAX zp
		addr := c.zp()
		c.A = c.read(addr)
		c.X = c.A
		c.setNZ(c.A)
		return 3
	case 0xAB: // LAX imm
		c.A = c.fetch()
		c.X = c.A
		c.setNZ(c.A)
		return 2
	case 0xAF: // LAX abs
		addr := c.abs()
		c.A = c.read(addr)
		c.X = c.A
		c.setNZ(c.A)
		return 4
	case 0xB3: // LAX (zp),Y
		addr := c.indY()
		c.A = c.read(addr)
		c.X = c.A
		c.setNZ(c.A)
		return 5
	case 0xB7: // LAX zp,Y
		addr := c.zpY()
		c.A = c.read(addr)
		c.X = c.A
		c.setNZ(c.A)
		return 4
	case 0xBF: // LAX abs,Y
		addr := c.absY()
		c.A = c.read(addr)
		c.X = c.A
		c.setNZ(c.A)
		return 4

	// --- DCP (DEC + CMP) ---
	case 0xC3: // DCP (zp,X)
		addr := c.indX()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 8
	case 0xC7: // DCP zp
		addr := c.zp()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 5
	case 0xCF: // DCP abs
		addr := c.abs()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 6
	case 0xD3: // DCP (zp),Y
		addr := c.indY()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 8
	case 0xD7: // DCP zp,X
		addr := c.zpX()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 6
	case 0xDB: // DCP abs,Y
		addr := c.absY()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 7
	case 0xDF: // DCP abs,X
		addr := c.absX()
		val = c.read(addr) - 1
		c.write(addr, val)
		c.cmp(c.A, val)
		return 7

	// --- ISB/ISC (INC + SBC) ---
	case 0xE3: // ISB (zp,X)
		addr := c.indX()
		val = c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 8
	case 0xEF: // ISB abs
		addr := c.abs()
		val = c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 6
	case 0xF3: // ISB (zp),Y
		addr := c.indY()
		val = c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 8
	case 0xF7: // ISB zp,X
		addr := c.zpX()
		val = c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 6
	case 0xFB: // ISB abs,Y
		addr := c.absY()
		val = c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 7
	case 0xFF: // ISB abs,X
		addr := c.absX()
		val = c.read(addr) + 1
		c.write(addr, val)
		c.sbc(val)
		return 7

	// --- NOP undocumented ---
	case 0x04, 0x44, 0x64: // NOP zp
		_ = c.zp()
		return 3
	case 0x14, 0x34, 0x54, 0x74, 0xD4: // NOP zp,X
		_ = c.zpX()
		return 4
	case 0x80, 0x82, 0x89, 0xC2: // NOP imm
		c.fetch()
		return 2
	case 0x5C, 0x7C, 0xDC, 0xFC: // NOP abs,X
		_ = c.absX()
		return 4

	// --- JAM (halt CPU) ---
	case 0x12, 0x32, 0x42, 0x52, 0x62, 0x72, 0x92, 0xB2, 0xD2, 0xF2:
		// Halt CPU - infinite loop
		c.PC--
		return 2

	default:
		return 2
	}
}

func (c *CPU6510) adc(val uint8) {
	if c.getFlag(FlagD) {
		c.adcDecimal(val)
		return
	}
	oldA := c.A
	carry := uint16(0)
	if c.getFlag(FlagC) {
		carry = 1
	}
	result := uint16(c.A) + uint16(val) + carry
	c.setFlag(FlagC, result > 0xFF)
	c.setFlag(FlagV, ((oldA^uint8(result))&(^oldA^val)&0x80) != 0)
	c.A = uint8(result)
	c.setNZ(c.A)
}

func (c *CPU6510) adcDecimal(val uint8) {
	carry := uint16(0)
	if c.getFlag(FlagC) {
		carry = 1
	}
	lo := (c.A & 0x0F) + (val & 0x0F) + uint8(carry)
	hi := (c.A >> 4) + (val >> 4)
	if lo > 9 {
		lo += 6
		hi++
	}
	if hi > 9 {
		hi += 6
	}
	result := uint16(c.A) + uint16(val) + carry
	c.setFlag(FlagC, hi > 9)
	c.setFlag(FlagV, ((c.A^val)&0x80) == 0 && ((c.A^uint8(result))&0x80) != 0)
	c.A = (hi << 4) | (lo & 0x0F)
	c.setNZ(c.A)
}

func (c *CPU6510) sbc(val uint8) {
	if c.getFlag(FlagD) {
		c.sbcDecimal(val)
		return
	}
	oldA := c.A
	carry := uint16(0)
	if c.getFlag(FlagC) {
		carry = 1
	}
	result := uint16(c.A) - uint16(val) - (1 - carry)
	c.setFlag(FlagC, result <= 0xFF)
	c.setFlag(FlagV, ((oldA^uint8(result))&(oldA^val)&0x80) != 0)
	c.A = uint8(result)
	c.setNZ(c.A)
}

func (c *CPU6510) sbcDecimal(val uint8) {
	carry := uint16(0)
	if c.getFlag(FlagC) {
		carry = 1
	}
	lo := int(c.A&0x0F) - int(val&0x0F) - int(1-carry)
	hi := int(c.A>>4) - int(val>>4)
	if lo < 0 {
		lo += 10
		hi--
	}
	if hi < 0 {
		hi += 10
	}
	result := uint16(c.A) - uint16(val) - (1 - carry)
	c.setFlag(FlagC, result <= 0xFF)
	c.setFlag(FlagV, ((c.A^val)&0x80) != 0 && ((c.A^uint8(result))&0x80) != 0)
	c.A = (uint8(hi) << 4) | (uint8(lo) & 0x0F)
	c.setNZ(c.A)
}

func (c *CPU6510) cmp(reg, val uint8) {
	result := reg - val
	c.setFlag(FlagC, reg >= val)
	c.setFlag(FlagZ, reg == val)
	c.setFlag(FlagN, result&0x80 != 0)
}

func (c *CPU6510) bit(val uint8) {
	c.setFlag(FlagZ, (c.A&val) == 0)
	c.setFlag(FlagN, val&0x80 != 0)
	c.setFlag(FlagV, val&0x40 != 0)
}

func (c *CPU6510) branch(take bool) uint8 {
	off := int8(c.fetch())
	if !take {
		return 2
	}
	oldPC := c.PC
	c.PC = uint16(int16(c.PC) + int16(off))
	cycles := uint8(3)
	if (oldPC & 0xFF00) != (c.PC & 0xFF00) {
		cycles = 4
	}
	return cycles
}

func (c *CPU6510) Disassemble(addr uint16) string {
	return fmt.Sprintf("$%04X: ???", addr)
}
