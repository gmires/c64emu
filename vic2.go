package c64

const (
	// Full PAL frame including border (visible area is ~384x272)
	VIC_SCREEN_WIDTH  = 384
	VIC_SCREEN_HEIGHT = 272
	VIC_BORDER_LEFT   = 32
	VIC_BORDER_RIGHT  = 32
	VIC_BORDER_TOP    = 36
	VIC_BORDER_BOTTOM = 36
)

type VIC2 struct {
	clk           uint64
	bus           *Bus
	cfg           *SystemConfig
	raster        *RasterRenderer

	mx            [8]uint16
	my            [8]uint8
	mxExp         uint8
	myExp         uint8
	spritePrior   uint8
	spriteMC      uint8
	spriteColl    uint8
	spriteBkgColl uint8
	spriteEn      uint8
	ctrl1         uint8
	ctrl2         uint8
	irqMask       uint8
	irqFlags      uint8
	rasterIRQLine uint16
	scrollX       uint8
	scrollY       uint8
	borderColor   uint8
	bgColor       [4]uint8

	regs          [64]uint8
	frameBuf      []uint32
	line          int
	cycle         int
}

func NewVIC2(bus *Bus, cfg *SystemConfig) *VIC2 {
	return &VIC2{
		clk:           0,
		bus:           bus,
		cfg:           cfg,
		raster:        NewRasterRenderer(bus, cfg),
		frameBuf:      make([]uint32, VIC_SCREEN_WIDTH*VIC_SCREEN_HEIGHT),
		line:          0,
		cycle:         0,
		borderColor:   0x0E,
		bgColor:       [4]uint8{0x06, 0x00, 0x00, 0x00},
	}
}

func (v *VIC2) step() uint8 {
	v.clk++
	v.cycle++
	if v.cycle >= 63 {
		v.cycle = 0
		v.line++
		if v.line >= 312 {
			v.line = 0
		}
		if uint16(v.line) == v.rasterIRQLine && v.irqMask&0x01 != 0 {
			v.irqFlags |= 0x01
		}
	}
	return 1
}

func (v *VIC2) handleReg(addr uint8, val uint8) {
	reg := addr & 0x3F
	v.regs[reg] = val
	switch reg {
	case 0x00, 0x02, 0x04, 0x06, 0x08, 0x0A, 0x0C, 0x0E:
		idx := reg >> 1
		v.mx[idx] = (v.mx[idx] & 0xFF00) | uint16(val)
	case 0x10:
		for i := 0; i < 8; i++ {
			if val&(1<<i) != 0 {
				v.mx[i] |= 0x0100
			} else {
				v.mx[i] &^= 0x0100
			}
		}
	case 0x01, 0x03, 0x05, 0x07, 0x09, 0x0B, 0x0D, 0x0F:
		idx := reg >> 1
		v.my[idx] = val
	case 0x11: // Control register 1
		v.ctrl1 = val
		v.scrollY = val & 0x07
		v.rasterIRQLine = (v.rasterIRQLine & 0xFF) | (uint16(val&0x80) << 1)
	case 0x12: // Raster line
		v.rasterIRQLine = (v.rasterIRQLine & 0x100) | uint16(val)
	case 0x16: // Control register 2
		v.ctrl2 = val
		v.scrollX = val & 0x07
	case 0x18: // Memory pointers
		v.raster.SetScreenBase(uint16((val >> 4) & 0x0F) * 0x0400)
		v.raster.SetCharBase(uint16((val >> 1) & 0x07) * 0x0800)
	case 0x15: // Sprite enable
		v.spriteEn = val
	case 0x17: // Sprite Y expansion
		v.myExp = val
	case 0x1B: // Sprite priority
		v.spritePrior = val
	case 0x1C: // Sprite multicolor
		v.spriteMC = val
	case 0x1D: // Sprite X expansion
		v.mxExp = val
	case 0x19: // Interrupt flags
		v.irqFlags &^= val & 0x0F
	case 0x1A: // Interrupt mask
		v.irqMask = val & 0x0F
	case 0x20: // Border color
		v.borderColor = val & 0x0F
	case 0x21: // Background color 0
		v.bgColor[0] = val & 0x0F
	case 0x22: // Background color 1
		v.bgColor[1] = val & 0x0F
	case 0x23: // Background color 2
		v.bgColor[2] = val & 0x0F
	case 0x24: // Background color 3
		v.bgColor[3] = val & 0x0F
	}
}

func (v *VIC2) readReg(addr uint8) uint8 {
	reg := addr & 0x3F
	switch reg {
	case 0x11:
		return v.ctrl1 | uint8((uint16(v.line)&0x100)>>1)
	case 0x12:
		return uint8(v.line & 0xFF)
	case 0x19:
		flags := v.irqFlags | 0x70
		if v.irqFlags&v.irqMask&0x0F != 0 {
			flags |= 0x80
		}
		return flags
	case 0x1A:
		return v.irqMask | 0xF0
	case 0x1E: // Sprite-sprite collision
		c := v.spriteColl
		v.spriteColl = 0
		return c
	case 0x1F: // Sprite-background collision
		c := v.spriteBkgColl
		v.spriteBkgColl = 0
		return c
	default:
		if reg < 64 {
			return v.regs[reg]
		}
		return 0
	}
}

func (v *VIC2) getPixel(x, y int) uint32 {
	if x < 0 || x >= VIC_SCREEN_WIDTH || y < 0 || y >= VIC_SCREEN_HEIGHT {
		return 0
	}
	return v.frameBuf[y*VIC_SCREEN_WIDTH+x]
}

func (v *VIC2) FrameBuffer() []uint32 {
	return v.frameBuf
}

func (v *VIC2) IsBadLine() bool {
	return v.line >= 50 && v.line < 50+25
}

func (v *VIC2) BA() bool {
	return true
}

func (v *VIC2) Clock() uint64 {
	return v.clk
}

func (v *VIC2) LP() bool {
	return v.line == 0
}

func (v *VIC2) RenderFrame() {
	v.raster.RenderToBuffer(v.frameBuf, v)
}

func (v *VIC2) RasterRenderer() *RasterRenderer {
	return v.raster
}

func (v *VIC2) Ctrl1() uint8 {
	return v.ctrl1
}
