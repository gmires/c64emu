package c64

// DebugRenderOverlay abilita un overlay visivo per debug del rendering
var DebugRenderOverlay = false

func (r *RasterRenderer) drawDebugOverlay(buf []uint32, vic *VIC2, width, height int) {
	if !DebugRenderOverlay {
		return
	}

	scrollX := int(vic.scrollX)
	scrollY := int(vic.scrollY)

	// 1. Griglia grigia scura ogni 8 pixel (dimensione carattere)
	for x := 0; x < width; x += 8 {
		for y := 0; y < height; y++ {
			if buf[y*width+x] == PALColors[vic.borderColor] || buf[y*width+x] == PALColors[vic.bgColor[0]] {
				buf[y*width+x] = 0x222222 // Dark grey grid
			}
		}
	}
	for y := 0; y < height; y += 8 {
		for x := 0; x < width; x++ {
			if buf[y*width+x] == PALColors[vic.borderColor] || buf[y*width+x] == PALColors[vic.bgColor[0]] {
				buf[y*width+x] = 0x222222
			}
		}
	}

	// 2. Linee ROSSE per i bordi del framebuffer
	for x := 0; x < width; x++ {
		buf[0*width+x] = 0xFF0000       // Top edge = RED
		buf[(height-1)*width+x] = 0xFF0000 // Bottom edge = RED
	}
	for y := 0; y < height; y++ {
		buf[y*width+0] = 0xFF0000           // Left edge = RED
		buf[y*width+(width-1)] = 0xFF0000   // Right edge = RED
	}

	// 3. Linee VERDI per l'area testo effettiva (centro del framebuffer + scroll)
	textTop := VIC_BORDER_TOP + scrollY
	if textTop >= 0 && textTop < height {
		for x := 0; x < width; x++ {
			buf[textTop*width+x] = 0x00FF00
		}
	}
	textBottom := VIC_BORDER_TOP + 25*8 + scrollY
	if textBottom >= 0 && textBottom < height {
		for x := 0; x < width; x++ {
			buf[textBottom*width+x] = 0x00FF00
		}
	}
	textLeft := VIC_BORDER_LEFT + scrollX
	if textLeft >= 0 && textLeft < width {
		for y := 0; y < height; y++ {
			buf[y*width+textLeft] = 0x00FF00
		}
	}
	textRight := VIC_BORDER_LEFT + 40*8 + scrollX
	if textRight >= 0 && textRight < width {
		for y := 0; y < height; y++ {
			buf[y*width+textRight] = 0x00FF00
		}
	}

	// 4. Croce BLU al centro del framebuffer
	cx, cy := width/2, height/2
	for x := 0; x < width; x++ {
		buf[cy*width+x] = 0x0000FF
	}
	for y := 0; y < height; y++ {
		buf[y*width+cx] = 0x0000FF
	}

	// 5. Pixel BIANCO agli angoli per confermare dimensione
	buf[0] = 0xFFFFFF
	buf[width-1] = 0xFFFFFF
	buf[(height-1)*width] = 0xFFFFFF
	buf[(height-1)*width+(width-1)] = 0xFFFFFF
}
