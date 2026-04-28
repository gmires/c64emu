package c64

type CIA struct {
	clk         uint64
	bus         *Bus
	idx         int
	portA       uint8
	portB       uint8
	ddrA        uint8
	ddrB        uint8
	timerA      uint16
	timerB      uint16
	timerALatch uint16
	timerBLatch uint16
	ctrlA       uint8
	ctrlB       uint8
	intMask     uint8
	intFlags    uint8
	ser         uint8
	irqLine     bool
	portFlag    bool
	regs        [16]uint8
	keyboard    [8][8]bool
	joyA        uint8 // Joystick on PORT A: bit=0 when pressed (grounded)
	joyB        uint8 // Joystick on PORT B: bit=0 when pressed
}

func NewCIA(bus *Bus, idx int) *CIA {
	c := &CIA{
		clk:  0,
		bus:  bus,
		idx:  idx,
	}
	c.Reset()
	return c
}

func (c *CIA) Reset() {
	c.clk = 0
	c.portA = 0
	c.portB = 0
	c.ddrA = 0
	c.ddrB = 0
	c.timerA = 0
	c.timerB = 0
	c.timerALatch = 0
	c.timerBLatch = 0
	c.ctrlA = 0
	c.ctrlB = 0
	c.intMask = 0
	c.intFlags = 0
	c.ser = 0
	c.irqLine = false
	c.portFlag = false
	c.regs = [16]uint8{}
	c.keyboard = [8][8]bool{}
	c.joyA = 0xFF
	c.joyB = 0xFF
}

func (c *CIA) Step() {
	c.clk++
	if c.ctrlA&0x01 != 0 {
		c.tickTimerA()
	}
	if c.ctrlB&0x01 != 0 {
		c.tickTimerB()
	}
}

func (c *CIA) tickTimerA() {
	if c.timerA == 0 {
		c.timerA = c.timerALatch
		c.setIntFlag(0x01)
		if c.ctrlA&0x02 != 0 {
			c.ctrlA &^= 0x01 // One-shot: stop timer
		}
	} else {
		c.timerA--
	}
}

func (c *CIA) tickTimerB() {
	if c.timerB == 0 {
		c.timerB = c.timerBLatch
		c.setIntFlag(0x02)
		if c.ctrlB&0x02 != 0 {
			c.ctrlB &^= 0x01 // One-shot: stop timer
		}
	} else {
		c.timerB--
	}
}

func (c *CIA) handleReg(addr uint8, val uint8) {
	reg := addr & 0x0F
	c.regs[reg] = val
	switch reg {
	case 0x00:
		c.portA = val
	case 0x01:
		c.portB = val
	case 0x02:
		c.ddrA = val
	case 0x03:
		c.ddrB = val
	case 0x04:
		c.timerALatch = (c.timerALatch & 0xFF00) | uint16(val)
		if c.ctrlA&0x01 == 0 {
			c.timerA = c.timerALatch
		}
	case 0x05:
		c.timerALatch = (c.timerALatch & 0x00FF) | (uint16(val) << 8)
		if c.ctrlA&0x01 == 0 {
			c.timerA = c.timerALatch
		}
	case 0x06:
		c.timerBLatch = (c.timerBLatch & 0xFF00) | uint16(val)
		if c.ctrlB&0x01 == 0 {
			c.timerB = c.timerBLatch
		}
	case 0x07:
		c.timerBLatch = (c.timerBLatch & 0x00FF) | (uint16(val) << 8)
		if c.ctrlB&0x01 == 0 {
			c.timerB = c.timerBLatch
		}
	case 0x0E:
		c.ctrlA = val
		if val&0x10 != 0 {
			c.timerA = c.timerALatch
		}
	case 0x0F:
		c.ctrlB = val
		if val&0x10 != 0 {
			c.timerB = c.timerBLatch
		}
	case 0x0D:
		if val&0x80 != 0 {
			c.intMask |= val & 0x1F
		} else {
			c.intMask &^= val & 0x1F
		}
		c.irqLine = false
		if c.intMask&c.intFlags != 0 {
			c.irqLine = true
		}
	}
}

func (c *CIA) readReg(addr uint8) uint8 {
	reg := addr & 0x0F
	switch reg {
	case 0x00:
		pa := (c.portA & c.ddrA)
		inputMask := 0xFF &^ c.ddrA
		pa |= inputMask & c.joyA
		return pa
	case 0x01:
		pb := (c.portB & c.ddrB)
		inputMask := 0xFF &^ c.ddrB
		inputVal := inputMask & c.joyB
		for col := 0; col < 8; col++ {
			bit := uint8(1) << uint(col)
			if inputMask&bit == 0 {
				continue
			}
			for row := 0; row < 8; row++ {
				rowBit := uint8(1) << uint(row)
				if c.ddrA&rowBit != 0 && c.portA&rowBit == 0 {
					if c.keyboard[col][row] {
						inputVal &^= bit
					}
				}
			}
		}
		pb |= inputVal
		return pb
	case 0x04:
		return uint8(c.timerA & 0xFF)
	case 0x05:
		return uint8(c.timerA >> 8)
	case 0x06:
		return uint8(c.timerB & 0xFF)
	case 0x07:
		return uint8(c.timerB >> 8)
	case 0x0D:
		flags := c.intFlags | 0x70
		if c.irqLine {
			flags |= 0x80
		}
		c.intFlags = 0
		c.irqLine = false
		return flags
	default:
		return c.regs[reg]
	}
}

func (c *CIA) setIntFlag(f uint8) {
	c.intFlags |= f
	if c.intMask&f != 0 {
		c.irqLine = true
	}
}

func (c *CIA) IRQ() bool {
	return c.irqLine
}

func (c *CIA) AckIRQ() {
	c.irqLine = false
}

func (c *CIA) PortA() uint8 {
	return (c.portA & c.ddrA) | (0xFF &^ c.ddrA)
}

func (c *CIA) PortB() uint8 {
	return (c.portB & c.ddrB) | (0xFF &^ c.ddrB)
}

func (c *CIA) Clock() uint64 {
	return c.clk
}

func (c *CIA) CountA() uint16 {
	return c.timerA
}

func (c *CIA) LatchA() uint16 {
	return c.timerALatch
}

func (c *CIA) LatchB() uint16 {
	return c.timerBLatch
}

func (c *CIA) IntMask() uint8 {
	return c.intMask
}

func (c *CIA) IntFlags() uint8 {
	return c.intFlags
}

func (c *CIA) CtrlA() uint8 {
	return c.ctrlA
}

func (c *CIA) SetKey(col, row int, pressed bool) {
	if col >= 0 && col < 8 && row >= 0 && row < 8 {
		c.keyboard[col][row] = pressed
	}
}

func (c *CIA) ClearKeyboard() {
	for i := range c.keyboard {
		for j := range c.keyboard[i] {
			c.keyboard[i][j] = false
		}
	}
}

func (c *CIA) SetJoystickA(val uint8) {
	c.joyA = val
}

func (c *CIA) SetJoystickB(val uint8) {
	c.joyB = val
}
