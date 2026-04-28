package c64

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

type Machine struct {
	cfg  *SystemConfig
	bus  *Bus
	cpu  *CPU6510
	vic  *VIC2
	cia  [2]*CIA
	sid  *SID

	// Mounted disk image for KERNAL I/O interception
	d64Reader *D64Reader

	// KERNAL hook state (SETLFS / SETNAM)
	hookLfsDevice    uint8
	hookLfsSecondary uint8
	hookFileName     string

	// Serial I/O emulation for multi-file games (OPEN/CHKIN/CHRIN)
	serialOpen       bool
	serialData       []byte
	serialPos        int
}

func NewMachine(cfg *SystemConfig) *Machine {
	bus := NewBus(cfg)
	m := &Machine{
		cfg: cfg,
		bus: bus,
		cpu: NewCPU6510(bus),
		vic: NewVIC2(bus, cfg),
		sid: NewSID(bus),
	}
	m.cia[0] = NewCIA(bus, 0)
	m.cia[1] = NewCIA(bus, 1)

	bus.AttachVIC(m.vic)
	bus.AttachCIA(0, m.cia[0])
	bus.AttachCIA(1, m.cia[1])
	bus.AttachSID(m.sid)
	m.cpu.bus = bus

	return m
}

func (m *Machine) HasValidROMs() bool {
	if len(m.bus.Kernal) < 0x1FFE {
		return false
	}
	// C64 KERNAL ROM is mapped at $E000-$FFFF
	// Reset vector is at CPU address $FFFC, which is offset $1FFC in the ROM file
	resetVec := uint16(m.bus.Kernal[0x1FFC]) | uint16(m.bus.Kernal[0x1FFD])<<8
	// C64 standard reset vector is $FCE2
	if resetVec != 0xFCE2 {
		fmt.Printf("WARNING: KERNAL reset vector is $%04X (expected $FCE2 for C64)\n", resetVec)
		return false
	}
	return true
}

func (m *Machine) LoadROM(path string, slot string) error {
	if path == "" {
		switch slot {
		case "basic":
			m.bus.Basic = nil
		case "kernal":
			m.bus.Kernal = nil
		case "chargen":
			m.bus.Charen = nil
		}
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("loading ROM %s: %w", slot, err)
	}

	switch slot {
	case "basic":
		if len(data) >= 8192 {
			m.bus.Basic = data[:8192]
		}
	case "kernal":
		if len(data) >= 8192 {
			m.bus.Kernal = data[:8192]
		}
	case "chargen":
		if len(data) >= 4096 {
			m.bus.Charen = data[:4096]
		}
	}

	return nil
}

func (m *Machine) LoadPRG(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 2 {
		return fmt.Errorf("PRG file too short")
	}
	addr := uint16(data[0]) | uint16(data[1])<<8
	for i := 2; i < len(data); i++ {
		m.bus.Write(addr+uint16(i-2), data[i])
	}

	// Set BASIC pointers if loaded to standard BASIC area ($0801)
	if addr == 0x0801 {
		endAddr := addr + uint16(len(data)-2)
		m.bus.Write(0x2D, uint8(endAddr))
		m.bus.Write(0x2E, uint8(endAddr>>8))
		m.bus.Write(0x2F, uint8(endAddr))
		m.bus.Write(0x30, uint8(endAddr>>8))
		m.bus.Write(0x31, uint8(endAddr))
		m.bus.Write(0x32, uint8(endAddr>>8))
		m.bus.Write(0xAE, uint8(endAddr))
		m.bus.Write(0xAF, uint8(endAddr>>8))
	}
	return nil
}

func (m *Machine) AutoRun(prgStartAddr uint16) {
	if prgStartAddr == 0x0801 {
		// Simulate typing RUN\r in BASIC
		m.TypeKey('r')
		m.TypeKey('u')
		m.TypeKey('n')
		m.TypeKey('\r')
	} else {
		// Direct jump for machine language programs
		m.cpu.PC = prgStartAddr
	}
}

func (m *Machine) InjectTestPattern() {
	// Write "CIAO" to screen RAM $0400 with colors
	message := []uint8{'C', 'I', 'A', 'O'}
	colors := []uint8{0x01, 0x02, 0x05, 0x07} // White, Red, Green, Yellow
	for i := 0; i < 4; i++ {
		m.bus.Write(0x0400+uint16(i), message[i])
		m.bus.Write(0xD800+uint16(i), colors[i])
	}
	// Fill rest of screen with spaces and light blue
	for i := 4; i < 1000; i++ {
		m.bus.Write(0x0400+uint16(i), 0x20) // space
		m.bus.Write(0xD800+uint16(i), 0x0E) // light blue
	}
}

func (m *Machine) ShowBootScreen() {
	// Set VIC-II to standard C64 mode
	m.bus.Write(0xD011, 0x1B) // Screen on, 25 rows
	m.bus.Write(0xD016, 0x08) // 40 cols, no multicolor
	m.bus.Write(0xD018, 0x14) // Screen=$0400, chars=$0000
	m.bus.Write(0xD020, 0x0E) // Border = light blue
	m.bus.Write(0xD021, 0x06) // Background = blue

	// Clear screen (spaces)
	for i := uint16(0); i < 1000; i++ {
		m.bus.Write(0x0400+i, 0x20)
		m.bus.Write(0xD800+i, 0x0E) // light blue color
	}

	// Write boot messages with WHITE color
	msg1 := "**** COMMODORE 64 BASIC V2 ****"
	msg2 := "64K RAM SYSTEM  38911 BASIC BYTES FREE"
	msg3 := "READY."

	for i, c := range msg1 {
		addr := 0x0400 + 40 + uint16(i)
		m.bus.Write(addr, uint8(c))
		m.bus.Write(0xD800+(addr-0x0400), 0x01) // white
	}
	for i, c := range msg2 {
		addr := 0x0400 + 120 + uint16(i)
		m.bus.Write(addr, uint8(c))
		m.bus.Write(0xD800+(addr-0x0400), 0x01) // white
	}
	for i, c := range msg3 {
		addr := 0x0400 + 200 + uint16(i)
		m.bus.Write(addr, uint8(c))
		m.bus.Write(0xD800+(addr-0x0400), 0x01) // white
	}

	// Set cursor at end of READY. line
	m.bus.Write(0x0400+206, 0xA0) // cursor character (inverted space)
	m.bus.Write(0xD800+206, 0x01) // white color for cursor
}

func (m *Machine) Reset() {
	m.cpu.Reset()
	// Initialize RAM with VICE-like pattern instead of all zeros
	// Pattern: alternating 0,0,255,255 with 16KB phase inversion
	for i := range m.cfg.RAM {
		val := 0
		if ((i+2)/4)%2 == 1 {
			val = 255
		}
		if (i/16384)%2 == 1 {
			val ^= 255
		}
		m.cfg.RAM[i] = uint8(val)
	}
	// Reset CPU port to power-on state (DDR=0, Data=0)
	// With DDR=0, all pins are inputs and pull-ups make them read as 1
	m.bus.Write(0x0000, 0x00) // DDR = all input
	m.bus.Write(0x0001, 0x00) // Data = 0
}

func (m *Machine) Run() error {
	for {
		m.runFrame()
	}
}

func (m *Machine) RunFrames(n int) {
	for i := 0; i < n; i++ {
		m.runFrame()
	}
}

func (m *Machine) runFrame() {
	cyclesPerLine := 63
	linesPerFrame := 312
	if !m.cfg.PAL {
		cyclesPerLine = 65
		linesPerFrame = 262
	}
	totalCycles := linesPerFrame * cyclesPerLine
	cyclesDone := 0
	for cyclesDone < totalCycles {
		cycles := m.Step()
		cyclesDone += cycles
	}

	m.vic.RenderFrame()
}

func (m *Machine) Step() int {
	// Intercept KERNAL I/O routines before executing the opcode.
	// The Bus returns RTS ($60) for these addresses so the CPU
	// immediately returns to the caller after our hook runs.
	pcBefore := m.cpu.PC
	switch pcBefore {
	case 0xFFBA:
		m.handleSetLfs()
	case 0xFFBD:
		m.handleSetNam()
	case 0xFFC0:
		m.handleOpen()
	case 0xFFC3:
		m.handleClose()
	case 0xFFC6:
		m.handleChkin()
	case 0xFFCF:
		m.handleChrin()
	case 0xFFD5:
		m.handleLoad()
	}

	cycles := m.cpu.step()
	m.cpu.ClockAdd(uint64(cycles))

	for i := 0; i < int(cycles); i++ {
		m.vic.step()
		m.cia[0].Step()
		m.cia[1].Step()
		m.sid.step()
	}

	if m.cia[0].IRQ() || m.cia[1].IRQ() {
		m.cpu.IRQ()
		if m.cia[0].IRQ() {
			m.cia[0].AckIRQ()
		}
		if m.cia[1].IRQ() {
			m.cia[1].AckIRQ()
		}
	}
	return int(cycles)
}

// MountD64 attaches a D64 disk image for KERNAL LOAD interception.
func (m *Machine) MountD64(reader *D64Reader) {
	m.d64Reader = reader
}

// handleSetLfs implements the KERNAL SETLFS routine ($FFBA).
// Input: A = device, X = secondary address, Y = not used
func (m *Machine) handleSetLfs() {
	m.hookLfsDevice = m.cpu.A
	m.hookLfsSecondary = m.cpu.X
}

// handleSetNam implements the KERNAL SETNAM routine ($FFBD).
// Input: A = name length, X/Y = pointer to name
func (m *Machine) handleSetNam() {
	length := int(m.cpu.A)
	if length > 16 {
		length = 16
	}
	ptr := uint16(m.cpu.X) | uint16(m.cpu.Y)<<8
	name := make([]byte, length)
	for i := 0; i < length; i++ {
		name[i] = m.bus.Read(ptr + uint16(i))
	}
	m.hookFileName = string(name)
}

// handleOpen implements the KERNAL OPEN routine ($FFC0).
func (m *Machine) handleOpen() {
	if m.d64Reader == nil {
		m.cpu.SR |= 0x01 // Carry = error
		return
	}
	// The filename was already set by SETNAM; the device/secondary by SETLFS.
	// Try to open the file from the D64.
	data, err := m.d64Reader.ReadFile(m.hookFileName)
	if err != nil {
		m.cpu.SR |= 0x01 // Carry = error
		return
	}
	m.serialData = data
	m.serialPos = 0
	m.serialOpen = true
	m.cpu.SR &^= 0x01 // Clear Carry = success
}

// handleClose implements the KERNAL CLOSE routine ($FFC3).
func (m *Machine) handleClose() {
	m.serialOpen = false
	m.serialData = nil
	m.serialPos = 0
}

// handleChkin implements the KERNAL CHKIN routine ($FFC6).
// Input: X = logical file number.
func (m *Machine) handleChkin() {
	// Nothing special needed; CHRIN will read from serialData if open.
	m.cpu.SR &^= 0x01 // Clear Carry = success
}

// handleChrin implements the KERNAL CHRIN routine ($FFCF).
// Returns next byte in A. Sets Carry on error/EOF.
func (m *Machine) handleChrin() {
	if !m.serialOpen || m.serialData == nil {
		m.cpu.A = 0x0D // Return CR as EOF marker
		m.cpu.SR |= 0x01
		return
	}
	if m.serialPos >= len(m.serialData) {
		m.cpu.A = 0x0D // EOF
		m.cpu.SR |= 0x01
		return
	}
	m.cpu.A = m.serialData[m.serialPos]
	m.serialPos++
	m.cpu.SR &^= 0x01 // Clear Carry = success
}

// handleLoad implements the KERNAL LOAD routine ($FFD5).
// Input: A = 0 (load to address in file), A != 0 (load to X/Y)
// Output: C = 0 success, C = 1 error. X/Y = end address.
func (m *Machine) handleLoad() {
	if m.d64Reader == nil {
		m.cpu.SR |= 0x01 // Set Carry = error
		return
	}

	data, err := m.d64Reader.ReadFile(m.hookFileName)
	if err != nil {
		m.cpu.SR |= 0x01 // Set Carry = error
		return
	}
	if len(data) < 2 {
		m.cpu.SR |= 0x01 // Set Carry = error
		return
	}

	// Determine load address
	var loadAddr uint16
	if m.cpu.A == 0 {
		// Use address from file header
		loadAddr = uint16(data[0]) | uint16(data[1])<<8
	} else {
		// Override with X/Y
		loadAddr = uint16(m.cpu.X) | uint16(m.cpu.Y)<<8
	}

	// Write file data to RAM (skip 2-byte header when using file address)
	start := 2
	if m.cpu.A != 0 {
		start = 0 // No header when overriding
	}
	for i := start; i < len(data); i++ {
		m.bus.Write(loadAddr+uint16(i-start), data[i])
	}

	// Set end address in X/Y
	endAddr := loadAddr + uint16(len(data)-start)
	m.cpu.X = uint8(endAddr & 0xFF)
	m.cpu.Y = uint8(endAddr >> 8)
	m.cpu.SR &^= 0x01 // Clear Carry = success
	fmt.Printf("[KERNAL LOAD] OK: loaded '%s' to $%04X-$%04X (%d bytes)\n", m.hookFileName, loadAddr, endAddr, len(data)-start)
}

func (m *Machine) CPU() *CPU6510 {
	return m.cpu
}

func (m *Machine) Bus() *Bus {
	return m.bus
}

func (m *Machine) VIC() *VIC2 {
	return m.vic
}

func (m *Machine) SID() *SID {
	return m.sid
}

func (m *Machine) CIA(idx int) *CIA {
	if idx >= 0 && idx < 2 {
		return m.cia[idx]
	}
	return nil
}

func (m *Machine) FrameBuffer() []uint32 {
	return m.vic.FrameBuffer()
}

func (m *Machine) ExportBMP(path string) error {
	w, h := VIC_SCREEN_WIDTH, VIC_SCREEN_HEIGHT
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	pixels := m.vic.FrameBuffer()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(pixels[y*w+x] >> 16),
				G: uint8(pixels[y*w+x] >> 8),
				B: uint8(pixels[y*w+x]),
				A: 255,
			})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

func (m *Machine) KeyPress(char rune) {
	if m.cia[0] == nil {
		return
	}
	col, row := getKeyMatrix(char)
	if col >= 0 {
		m.cia[0].SetKey(col, row, true)
	}
}

func (m *Machine) KeyRelease(char rune) {
	if m.cia[0] == nil {
		return
	}
	col, row := getKeyMatrix(char)
	if col >= 0 {
		m.cia[0].SetKey(col, row, false)
	}
}

func (m *Machine) ClearKeyboard() {
	if m.cia[0] != nil {
		m.cia[0].ClearKeyboard()
	}
}

// TypeKey writes a character directly into the KERNAL keyboard buffer ($0277)
// This bypasses the IRQ-based SCNKEY scanning and works even if CIA interrupts are broken.
func (m *Machine) TypeKey(char rune) {
	petscii := asciiToPetscii(char)
	if petscii < 0 {
		return
	}
	bufLen := m.bus.Read(0x00C6)
	if bufLen < 10 {
		m.bus.Write(0x0277+uint16(bufLen), uint8(petscii))
		m.bus.Write(0x00C6, bufLen+1)
	}
}

func asciiToPetscii(char rune) int {
	// C64 starts in "uppercase/graphics" mode after reset.
	// In this mode, PETSCII 65-90 are uppercase letters,
	// and PETSCII 97-122 are graphic symbols.
	// So we map lowercase ASCII to uppercase PETSCII.
	switch {
	case char >= 'a' && char <= 'z':
		return int(char - 'a' + 'A')
	case char >= 'A' && char <= 'Z':
		return int(char)
	case char >= '0' && char <= '9':
		return int(char)
	case char == ' ':
		return 0x20
	case char == '\r':
		return 0x0D
	case char == '\b':
		return 0x14
	case char == '+':
		return 0x2B
	case char == '-':
		return 0x2D
	case char == '*':
		return 0x2A
	case char == '/':
		return 0x2F
	case char == '=':
		return 0x3D
	case char == '@':
		return 0x40
	case char == ':':
		return 0x3A
	case char == ';':
		return 0x3B
	case char == ',':
		return 0x2C
	case char == '.':
		return 0x2E
	case char == '$':
		return 0x24
	case char == '%':
		return 0x25
	case char == '&':
		return 0x26
	case char == '(':
		return 0x28
	case char == ')':
		return 0x29
	}
	return -1
}

func (m *Machine) DebugInfo() string {
	return fmt.Sprintf("Charen len=%d", len(m.bus.Charen))
}

// TraceRun esegue la CPU per N step e chiama la callback ad ogni step
func (m *Machine) TraceRun(steps int, callback func(step int, pc uint16, opcode uint8, a, x, y, sp, sr uint8)) {
	for i := 0; i < steps; i++ {
		pc := m.cpu.PC
		opcode := m.bus.Read(pc)
		a, x, y, sp, sr := m.cpu.A, m.cpu.X, m.cpu.Y, m.cpu.SP, m.cpu.SR
		
		if callback != nil {
			callback(i, pc, opcode, a, x, y, sp, sr)
		}
		
		cycles := m.cpu.step()
		m.cpu.ClockAdd(uint64(cycles))
		for j := 0; j < int(cycles); j++ {
			m.vic.step()
			m.cia[0].Step()
			m.cia[1].Step()
			m.sid.step()
		}
		if m.cia[0].IRQ() || m.cia[1].IRQ() {
			m.cpu.IRQ()
			if m.cia[0].IRQ() {
				m.cia[0].AckIRQ()
			}
			if m.cia[1].IRQ() {
				m.cia[1].AckIRQ()
			}
		}
	}
}

// Special key runes for C64 keys not in ASCII
const (
	KeyF1      rune = 0xF001
	KeyF2      rune = 0xF002
	KeyF3      rune = 0xF003
	KeyF4      rune = 0xF004
	KeyF5      rune = 0xF005
	KeyF6      rune = 0xF006
	KeyF7      rune = 0xF007
	KeyF8      rune = 0xF008
	KeyRunStop rune = 0xF009
	KeyRestore rune = 0xF00A
)

func getKeyMatrix(char rune) (int, int) {
	switch char {
	case '1':
		return 0, 7
	case '2':
		return 3, 7
	case '3':
		return 0, 1
	case '4':
		return 3, 1
	case '5':
		return 0, 2
	case '6':
		return 3, 2
	case '7':
		return 0, 3
	case '8':
		return 3, 3
	case '9':
		return 0, 4
	case '0':
		return 3, 4
	case 'q', 'Q':
		return 6, 7
	case 'w', 'W':
		return 1, 1
	case 'e', 'E':
		return 6, 1
	case 'r', 'R':
		return 1, 2
	case 't', 'T':
		return 6, 2
	case 'y', 'Y':
		return 1, 3
	case 'u', 'U':
		return 6, 3
	case 'i', 'I':
		return 1, 4
	case 'o', 'O':
		return 6, 4
	case 'p', 'P':
		return 1, 5
	case 'a', 'A':
		return 2, 1
	case 's', 'S':
		return 5, 1
	case 'd', 'D':
		return 2, 2
	case 'f', 'F':
		return 5, 2
	case 'g', 'G':
		return 2, 3
	case 'h', 'H':
		return 5, 3
	case 'j', 'J':
		return 2, 4
	case 'k', 'K':
		return 5, 4
	case 'l', 'L':
		return 2, 5
	case 'z', 'Z':
		return 4, 1
	case 'x', 'X':
		return 7, 2
	case 'c', 'C':
		return 4, 2
	case 'v', 'V':
		return 7, 3
	case 'b', 'B':
		return 4, 3
	case 'n', 'N':
		return 7, 4
	case 'm', 'M':
		return 4, 4
	case ' ':
		return 4, 7
	case '\r':
		return 1, 0 // RETURN
	case '\b':
		return 0, 0 // DELETE (BACKSPACE)
	case '+':
		return 0, 5
	case '-':
		return 3, 5
	case '*':
		return 1, 6
	case '/':
		return 7, 6
	case '=':
		return 5, 6
	case '@':
		return 5, 5
	case ':':
		return 5, 5 // @ and : share a key; simplified
	case ';':
		return 2, 6
	case ',':
		return 7, 5
	case '.':
		return 4, 5
	case '$':
		return 3, 1 // shares with 4
	case '%':
		return 0, 2 // shares with 5
	case '&':
		return 3, 2 // shares with 6
	case '(':
		return 0, 3 // shares with 7
	case ')':
		return 3, 3 // shares with 8
	case KeyF1, KeyF2:
		return 4, 0
	case KeyF3, KeyF4:
		return 5, 0
	case KeyF5, KeyF6:
		return 6, 0
	case KeyF7, KeyF8:
		return 7, 0
	case KeyRunStop:
		return 7, 7
	}
	return -1, -1
}
