package c64

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Display struct {
	machine     *Machine
	scale       float64
	fullscreen  bool
	img         *ebiten.Image
	pixels      []byte
	width       int
	height      int
	frameCount  int
	audio       *AudioSystem
}

func NewDisplay(machine *Machine, scale float64, fullscreen bool) (*Display, error) {
	w, h := VIC_SCREEN_WIDTH, VIC_SCREEN_HEIGHT
	d := &Display{
		machine:    machine,
		scale:      scale,
		fullscreen: fullscreen,
		width:      w,
		height:     h,
		img:        ebiten.NewImage(w, h),
		pixels:     make([]byte, w*h*4),
	}
	// Initialize audio if possible
	if audioSys, err := NewAudioSystem(machine); err == nil {
		d.audio = audioSys
		d.audio.Start()
	}
	return d, nil
}

func (d *Display) Update() error {
	d.handleInput()
	d.machine.runFrame()
	d.frameCount++
	return nil
}

func (d *Display) Draw(screen *ebiten.Image) {
	fb := d.machine.FrameBuffer()

	// Write into the pre-allocated pixel buffer (avoids 418 KB allocation/GC per frame)
	for i := 0; i < len(fb) && i < d.width*d.height; i++ {
		c := fb[i]
		d.pixels[i*4+0] = uint8((c >> 16) & 0xFF) // R
		d.pixels[i*4+1] = uint8((c >> 8) & 0xFF)  // G
		d.pixels[i*4+2] = uint8(c & 0xFF)         // B
		d.pixels[i*4+3] = 255                     // A
	}
	d.img.ReplacePixels(d.pixels)

	// Draw - Ebiten handles scaling automatically between Layout size and window
	op := &ebiten.DrawImageOptions{}
	screen.DrawImage(d.img, op)
}

func (d *Display) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Return the logical game screen size
	// Ebiten automatically scales this to the window size
	return d.width, d.height
}

func (d *Display) Run() error {
	w := int(float64(d.width) * d.scale)
	h := int(float64(d.height) * d.scale)
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("C64 Emulator")
	ebiten.SetTPS(60)
	if d.fullscreen {
		ebiten.SetFullscreen(true)
	}
	return ebiten.RunGame(d)
}

func (d *Display) FrameCount() int {
	return d.frameCount
}
