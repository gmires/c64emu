package c64

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var keyMap = map[ebiten.Key]rune{
	ebiten.Key1:         '1',
	ebiten.Key2:         '2',
	ebiten.Key3:         '3',
	ebiten.Key4:         '4',
	ebiten.Key5:         '5',
	ebiten.Key6:         '6',
	ebiten.Key7:         '7',
	ebiten.Key8:         '8',
	ebiten.Key9:         '9',
	ebiten.Key0:         '0',
	ebiten.KeyQ:         'q',
	ebiten.KeyW:         'w',
	ebiten.KeyE:         'e',
	ebiten.KeyR:         'r',
	ebiten.KeyT:         't',
	ebiten.KeyY:         'y',
	ebiten.KeyU:         'u',
	ebiten.KeyI:         'i',
	ebiten.KeyO:         'o',
	ebiten.KeyP:         'p',
	ebiten.KeyA:         'a',
	ebiten.KeyS:         's',
	ebiten.KeyD:         'd',
	ebiten.KeyF:         'f',
	ebiten.KeyG:         'g',
	ebiten.KeyH:         'h',
	ebiten.KeyJ:         'j',
	ebiten.KeyK:         'k',
	ebiten.KeyL:         'l',
	ebiten.KeyZ:         'z',
	ebiten.KeyX:         'x',
	ebiten.KeyC:         'c',
	ebiten.KeyV:         'v',
	ebiten.KeyB:         'b',
	ebiten.KeyN:         'n',
	ebiten.KeyM:         'm',
	ebiten.KeySpace:     ' ',
	ebiten.KeyEnter:     '\r',
	ebiten.KeyBackspace: '\b',
	ebiten.KeyComma:     ',',
	ebiten.KeyPeriod:    '.',
	ebiten.KeySlash:     '/',
	ebiten.KeySemicolon: ';',
	ebiten.KeyEqual:     '=',
	ebiten.KeyMinus:     '-',
	// C64 function keys
	ebiten.KeyF1:        KeyF1,
	ebiten.KeyF2:        KeyF2,
	ebiten.KeyF3:        KeyF3,
	ebiten.KeyF4:        KeyF4,
	ebiten.KeyF5:        KeyF5,
	ebiten.KeyF6:        KeyF6,
	ebiten.KeyF7:        KeyF7,
	ebiten.KeyF8:        KeyF8,
	ebiten.KeyEscape:    KeyRunStop,
}

func (d *Display) handleInput() {
	for ebitenKey, char := range keyMap {
		if inpututil.IsKeyJustPressed(ebitenKey) {
			// Use direct KERNAL buffer insert to avoid conflict with hardware matrix
			d.machine.TypeKey(char)
		}
		// NOTE: we intentionally do NOT call KeyPress/KeyRelease here,
		// because TypeKey already inserts into the KERNAL keyboard buffer.
		// Hardware matrix scanning is kept for joystick only.
	}

	// Joystick 2 emulation with arrow keys + right-control
	cia1 := d.machine.CIA(0)
	if cia1 != nil {
		joy := uint8(0xFF)
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			joy &^= 0x01 // UP
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			joy &^= 0x02 // DOWN
		}
		if ebiten.IsKeyPressed(ebiten.KeyLeft) {
			joy &^= 0x04 // LEFT
		}
		if ebiten.IsKeyPressed(ebiten.KeyRight) {
			joy &^= 0x08 // RIGHT
		}
		if ebiten.IsKeyPressed(ebiten.KeyControlRight) {
			joy &^= 0x10 // FIRE
		}
		cia1.SetJoystickA(joy)
	}
}
