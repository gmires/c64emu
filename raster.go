package c64

type VideoMode int

const (
	ModeStdText VideoMode = iota
	ModeMCMText
	ModeStdBitmap
	ModeMCMBitmap
	ModeECMText
	ModeInvalid
)

var PALColors = [16]uint32{
	0x000000, 0xFFFFFF, 0x880000, 0xAAFF66,
	0xCC44CC, 0x44AA44, 0x0000AA, 0xEEEE77,
	0xDD8855, 0x0066DD, 0xCC6688, 0x333333,
	0x777777, 0xAAFF00, 0x0088FF, 0xBBBBBB,
}

type RasterRenderer struct {
	bus      *Bus
	cfg      *SystemConfig
	scrBase  uint16
	charBase uint16
}

func NewRasterRenderer(bus *Bus, cfg *SystemConfig) *RasterRenderer {
	return &RasterRenderer{
		bus:     bus,
		cfg:     cfg,
		scrBase: 0x0400,
	}
}

func (r *RasterRenderer) SetScreenBase(addr uint16) {
	r.scrBase = addr & 0x3FFF
}

func (r *RasterRenderer) SetCharBase(addr uint16) {
	r.charBase = addr & 0x3FFF
}

func (r *RasterRenderer) ScreenBase() uint16 {
	return r.scrBase
}

func (r *RasterRenderer) CharBase() uint16 {
	return r.charBase
}

func (r *RasterRenderer) IsBadLine(line int) bool {
	return line >= 50 && line < 50+25
}

func (r *RasterRenderer) GetVideoMode(ctrl1, ctrl2 uint8) VideoMode {
	ecm := (ctrl1 >> 6) & 1
	bmm := (ctrl1 >> 5) & 1
	mcm := (ctrl2 >> 4) & 1
	switch {
	case ecm == 0 && bmm == 0 && mcm == 0:
		return ModeStdText
	case ecm == 0 && bmm == 0 && mcm == 1:
		return ModeMCMText
	case ecm == 0 && bmm == 1 && mcm == 0:
		return ModeStdBitmap
	case ecm == 0 && bmm == 1 && mcm == 1:
		return ModeMCMBitmap
	case ecm == 1 && bmm == 0:
		return ModeECMText
	default:
		return ModeStdText
	}
}

func (r *RasterRenderer) RenderToBuffer(buf []uint32, vic *VIC2) {
	mode := r.GetVideoMode(vic.ctrl1, vic.ctrl2)
	borderColor := PALColors[vic.borderColor]
	bgColor := PALColors[vic.bgColor[0]]
	w, h := VIC_SCREEN_WIDTH, VIC_SCREEN_HEIGHT

	// Fill border
	for i := range buf {
		buf[i] = borderColor
	}

	// Visible screen area (with scrolling offset)
	scrollX := int(vic.scrollX)
	scrollY := int(vic.scrollY)

	switch mode {
	case ModeStdText:
		r.renderStdText(buf, vic, scrollX, scrollY)
	case ModeMCMText:
		r.renderMCMText(buf, vic, scrollX, scrollY)
	case ModeStdBitmap:
		r.renderStdBitmap(buf, vic, scrollX, scrollY)
	case ModeMCMBitmap:
		r.renderMCMBitmap(buf, vic, scrollX, scrollY)
	case ModeECMText:
		r.renderECMText(buf, vic, scrollX, scrollY)
	default:
		// Fill visible area with background for unimplemented/invalid modes
		for y := VIC_BORDER_TOP; y < VIC_BORDER_TOP+200; y++ {
			for x := VIC_BORDER_LEFT; x < VIC_BORDER_LEFT+320; x++ {
				buf[y*w+x] = bgColor
			}
		}
	}

	// Render sprites on top
	r.renderSprites(buf, vic)

	// Debug overlay
	r.drawDebugOverlay(buf, vic, w, h)
}

func (r *RasterRenderer) renderStdText(buf []uint32, vic *VIC2, scrollX, scrollY int) {
	bgColor := PALColors[vic.bgColor[0]]
	w := VIC_SCREEN_WIDTH

	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			charIdx := uint16(row*40 + col)
			charCode := r.bus.Read(r.scrBase + charIdx)
			color := r.bus.Read(0xD800 + charIdx) & 0x0F

			for rowLine := 0; rowLine < 8; rowLine++ {
				pixelRow := r.readCharData(charCode, rowLine)
				for bit := 0; bit < 8; bit++ {
					x := VIC_BORDER_LEFT + col*8 + bit + scrollX
					y := VIC_BORDER_TOP + row*8 + rowLine + scrollY
				if x >= 0 && x < w && y >= 0 && y < VIC_SCREEN_HEIGHT {
					if pixelRow&(0x80>>bit) != 0 {
						buf[y*w+x] = PALColors[color]
					} else {
						buf[y*w+x] = bgColor
					}
				}
				}
			}
		}
	}
}

func (r *RasterRenderer) renderMCMText(buf []uint32, vic *VIC2, scrollX, scrollY int) {
	bgColor0 := PALColors[vic.bgColor[0]]
	bgColor1 := PALColors[vic.bgColor[1]]
	bgColor2 := PALColors[vic.bgColor[2]]
	w := VIC_SCREEN_WIDTH

	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			charIdx := uint16(row*40 + col)
			charCode := r.bus.Read(r.scrBase + charIdx)
			colorVal := r.bus.Read(0xD800 + charIdx) & 0x0F

			for rowLine := 0; rowLine < 8; rowLine++ {
				pixelRow := r.readCharData(charCode, rowLine)
				for pair := 0; pair < 4; pair++ {
					bits := (pixelRow >> (6 - pair*2)) & 0x03
					x := VIC_BORDER_LEFT + col*8 + pair*2 + scrollX
					y := VIC_BORDER_TOP + row*8 + rowLine + scrollY
					if x < 0 || x+1 >= w || y < 0 || y >= VIC_SCREEN_HEIGHT {
						continue
					}

					var c uint32
					if colorVal&0x08 == 0 {
						// Color < 8: standard text rendering
						for bit := 0; bit < 2; bit++ {
							if pixelRow&(0x80>>(pair*2+bit)) != 0 {
								c = PALColors[colorVal]
							} else {
								c = bgColor0
							}
							buf[y*w+x+bit] = c
						}
						continue
					}

					// Multicolor text
					switch bits {
					case 0:
						c = bgColor0
					case 1:
						c = bgColor1
					case 2:
						c = bgColor2
					case 3:
						c = PALColors[colorVal&0x07]
					}
					buf[y*w+x] = c
					buf[y*w+x+1] = c
				}
			}
		}
	}
}

func (r *RasterRenderer) renderStdBitmap(buf []uint32, vic *VIC2, scrollX, scrollY int) {
	// In bitmap mode, only bit 3 of $D018 selects the bitmap base:
	// bit 3 = 0 → $0000, bit 3 = 1 → $2000
	bitmapBase := r.charBase & 0x2000
	w := VIC_SCREEN_WIDTH

	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			charIdx := uint16(row*40 + col)
			colorByte := r.bus.Read(r.scrBase + charIdx)
			fg := colorByte & 0x0F
			bg := (colorByte >> 4) & 0x0F

			bitmapOffset := bitmapBase + charIdx*8
			for rowLine := 0; rowLine < 8; rowLine++ {
				pixelRow := r.bus.Read(bitmapOffset + uint16(rowLine))
				for bit := 0; bit < 8; bit++ {
					x := VIC_BORDER_LEFT + col*8 + bit + scrollX
					y := VIC_BORDER_TOP + row*8 + rowLine + scrollY
					if x < 0 || x >= w || y < 0 || y >= VIC_SCREEN_HEIGHT {
						continue
					}
					if pixelRow&(0x80>>bit) != 0 {
						buf[y*w+x] = PALColors[fg]
					} else {
						buf[y*w+x] = PALColors[bg]
					}
				}
			}
		}
	}
}

func (r *RasterRenderer) renderMCMBitmap(buf []uint32, vic *VIC2, scrollX, scrollY int) {
	bitmapBase := r.charBase & 0x2000
	bgColor0 := PALColors[vic.bgColor[0]]
	w := VIC_SCREEN_WIDTH

	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			charIdx := uint16(row*40 + col)
			colorByte := r.bus.Read(r.scrBase + charIdx)
			colorRAM := r.bus.Read(0xD800 + charIdx) & 0x0F

			bitmapOffset := bitmapBase + charIdx*8
			for rowLine := 0; rowLine < 8; rowLine++ {
				pixelRow := r.bus.Read(bitmapOffset + uint16(rowLine))
				for pair := 0; pair < 4; pair++ {
					bits := (pixelRow >> (6 - pair*2)) & 0x03
					x := VIC_BORDER_LEFT + col*8 + pair*2 + scrollX
					y := VIC_BORDER_TOP + row*8 + rowLine + scrollY
					if x < 0 || x+1 >= w || y < 0 || y >= VIC_SCREEN_HEIGHT {
						continue
					}
					var c uint32
					switch bits {
					case 0:
						c = bgColor0
					case 1:
						c = PALColors[(colorByte>>4)&0x0F]
					case 2:
						c = PALColors[colorByte&0x0F]
					case 3:
						c = PALColors[colorRAM]
					}
					buf[y*w+x] = c
					buf[y*w+x+1] = c
				}
			}
		}
	}
}

func (r *RasterRenderer) renderECMText(buf []uint32, vic *VIC2, scrollX, scrollY int) {
	bgColors := [4]uint32{
		PALColors[vic.bgColor[0]],
		PALColors[vic.bgColor[1]],
		PALColors[vic.bgColor[2]],
		PALColors[vic.bgColor[3]],
	}
	w := VIC_SCREEN_WIDTH

	for row := 0; row < 25; row++ {
		for col := 0; col < 40; col++ {
			charIdx := uint16(row*40 + col)
			charCode := r.bus.Read(r.scrBase + charIdx)
			color := r.bus.Read(0xD800 + charIdx) & 0x0F
			bgIdx := (charCode >> 6) & 0x03
			charCode &= 0x3F // Only 6 bits for character

			for rowLine := 0; rowLine < 8; rowLine++ {
				pixelRow := r.readCharData(charCode, rowLine)
				for bit := 0; bit < 8; bit++ {
					x := VIC_BORDER_LEFT + col*8 + bit + scrollX
					y := VIC_BORDER_TOP + row*8 + rowLine + scrollY
					if x < 0 || x >= w || y < 0 || y >= VIC_SCREEN_HEIGHT {
						continue
					}
					if pixelRow&(0x80>>bit) != 0 {
						buf[y*w+x] = PALColors[color]
					} else {
						buf[y*w+x] = bgColors[bgIdx]
					}
				}
			}
		}
	}
}

func (r *RasterRenderer) renderSprites(buf []uint32, vic *VIC2) {
	w := VIC_SCREEN_WIDTH
	mc1 := r.bus.Read(0xD025) & 0x0F
	mc2 := r.bus.Read(0xD026) & 0x0F

	for sprite := 0; sprite < 8; sprite++ {
		if vic.spriteEn&(1<<sprite) == 0 {
			continue
		}

		sx := int(vic.mx[sprite])
		sy := int(vic.my[sprite])
		mc := vic.spriteMC&(1<<sprite) != 0
		prior := vic.spritePrior&(1<<sprite) != 0
		xExp := vic.mxExp&(1<<sprite) != 0
		yExp := vic.myExp&(1<<sprite) != 0
		spriteColor := r.bus.Read(0xD027+uint16(sprite)) & 0x0F

		// Read sprite data pointer from screen RAM + $3F8
		spritePtr := r.bus.Read(r.scrBase + 0x03F8 + uint16(sprite))
		spriteAddr := uint16(spritePtr) * 64

		// Sprite data: 21 rows x 3 bytes = 63 bytes
		for row := 0; row < 21; row++ {
			dataRow := row
			if yExp {
				dataRow = row / 2
			}
			drawY := sy + row
			if yExp {
				drawY = sy + row*2
			}
			if drawY < 0 || drawY >= VIC_SCREEN_HEIGHT {
				continue
			}
			data := uint32(r.bus.Read(spriteAddr+uint16(dataRow*3))) |
				(uint32(r.bus.Read(spriteAddr+uint16(dataRow*3+1))) << 8) |
				(uint32(r.bus.Read(spriteAddr+uint16(dataRow*3+2))) << 16)

			if mc {
				// Multicolor sprite: 24 bits = 12 pixel pairs
				for pair := 0; pair < 12; pair++ {
					bits := (data >> (22 - pair*2)) & 0x03
					if bits == 0 {
						continue
					}
					var color uint8
					switch bits {
					case 1:
						color = mc1
					case 2:
						color = spriteColor
					case 3:
						color = mc2
					}
					for dx := 0; dx < 2; dx++ {
						x := sx + pair*2 + dx
						if xExp {
							x = sx + pair*4 + dx*2
							if x+1 >= w {
								continue
							}
							// Draw expanded pixel as 2 horizontal pixels
							r.drawSpritePixel(buf, w, x, drawY, color, prior, vic, sprite)
							r.drawSpritePixel(buf, w, x+1, drawY, color, prior, vic, sprite)
						} else {
							if x >= w {
								continue
							}
							r.drawSpritePixel(buf, w, x, drawY, color, prior, vic, sprite)
						}
					}
				}
				if yExp && drawY+1 < VIC_SCREEN_HEIGHT {
					// Duplicate row for Y expansion
					for pair := 0; pair < 12; pair++ {
						bits := (data >> (22 - pair*2)) & 0x03
						if bits == 0 {
							continue
						}
						var color uint8
						switch bits {
						case 1:
							color = mc1
						case 2:
							color = spriteColor
						case 3:
							color = mc2
						}
						for dx := 0; dx < 2; dx++ {
							x := sx + pair*2 + dx
							if xExp {
								x = sx + pair*4 + dx*2
								if x+1 >= w {
									continue
								}
								r.drawSpritePixel(buf, w, x, drawY+1, color, prior, vic, sprite)
								r.drawSpritePixel(buf, w, x+1, drawY+1, color, prior, vic, sprite)
							} else {
								if x >= w {
									continue
								}
								r.drawSpritePixel(buf, w, x, drawY+1, color, prior, vic, sprite)
							}
						}
					}
				}
			} else {
				// Standard sprite: 24 bits = 24 pixels
				for bit := 0; bit < 24; bit++ {
					if data&(1<<(23-bit)) == 0 {
						continue
					}
					x := sx + bit
					if xExp {
						x = sx + bit*2
						if x+1 >= w {
							continue
						}
						r.drawSpritePixel(buf, w, x, drawY, spriteColor, prior, vic, sprite)
						r.drawSpritePixel(buf, w, x+1, drawY, spriteColor, prior, vic, sprite)
					} else {
						if x < 0 || x >= w {
							continue
						}
						r.drawSpritePixel(buf, w, x, drawY, spriteColor, prior, vic, sprite)
					}
				}
				if yExp && drawY+1 < VIC_SCREEN_HEIGHT {
					// Duplicate row for Y expansion
					for bit := 0; bit < 24; bit++ {
						if data&(1<<(23-bit)) == 0 {
							continue
						}
						x := sx + bit
						if xExp {
							x = sx + bit*2
							if x+1 >= w {
								continue
							}
							r.drawSpritePixel(buf, w, x, drawY+1, spriteColor, prior, vic, sprite)
							r.drawSpritePixel(buf, w, x+1, drawY+1, spriteColor, prior, vic, sprite)
						} else {
							if x < 0 || x >= w {
								continue
							}
							r.drawSpritePixel(buf, w, x, drawY+1, spriteColor, prior, vic, sprite)
						}
					}
				}
			}
		}
	}
}

func (r *RasterRenderer) drawSpritePixel(buf []uint32, w, x, y int, color uint8, prior bool, vic *VIC2, sprite int) {
	idx := y*w + x
	if idx < 0 || idx >= len(buf) {
		return
	}
	if !prior || buf[idx] == PALColors[vic.bgColor[0]] {
		buf[idx] = PALColors[color]
	}
	// Collision detection
	for other := 0; other < 8; other++ {
		if other != sprite && vic.spriteEn&(1<<other) != 0 {
			vic.spriteColl |= (1 << sprite) | (1 << other)
		}
	}
	vic.spriteBkgColl |= 1 << sprite
}

func (r *RasterRenderer) readCharData(charCode uint8, rowLine int) uint8 {
	// C64 character ROM is visible to VIC-II at offsets $1000-$1FFF and $9000-$9FFF
	// regardless of CPU memory configuration.
	useCharRom := r.charBase == 0 ||
		(r.charBase >= 0x1000 && r.charBase < 0x2000) ||
		(r.charBase >= 0x9000 && r.charBase < 0xA000)

	if useCharRom && len(r.bus.Charen) == 4096 && IsValidChargenROM(r.bus.Charen) {
		// $1000-$17FF / $9000-$97FF → first 2KB (uppercase/graphics)
		// $1800-$1FFF / $9800-$9FFF → second 2KB (lowercase/uppercase)
		offsetBase := uint16(0)
		if (r.charBase >= 0x1800 && r.charBase < 0x2000) ||
			(r.charBase >= 0x9800 && r.charBase < 0xA000) {
			offsetBase = 0x800
		}
			offset := offsetBase + uint16(charCode)*8 + uint16(rowLine)
		if int(offset) < len(r.bus.Charen) {
			return r.bus.Charen[offset]
		}
	}

	// If charBase points to actual RAM (not character ROM area), read from bus
	if !useCharRom && r.charBase != 0 {
		addr := r.charBase + uint16(charCode)*8 + uint16(rowLine)
		return r.bus.Read(addr)
	}

	// Fallback to embedded font (reliable C64 font)
	return GetCharData(charCode, rowLine)
}
