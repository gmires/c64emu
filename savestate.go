package c64

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// SaveState holds a complete snapshot of the emulator state
type SaveState struct {
	Version uint32 // format version

	// CPU state
	CPU struct {
		A, X, Y, SP, SR uint8
		PC              uint16
		Clk             uint64
	}

	// RAM (64KB)
	RAM [65536]uint8

	// I/O area ($D000-$DFFF)
	IO [4096]uint8

	// Color RAM ($D800-$DBFF)
	Color [COLOR_MEM]uint8

	// CPU Port
	CPUPortDDR  uint8
	CPUPortData uint8

	// VIC-II
	VIC struct {
		MX            [8]uint16
		MY            [8]uint8
		MXExp         uint8
		MYExp         uint8
		SpritePrior   uint8
		SpriteMC      uint8
		SpriteColl    uint8
		SpriteBkgColl uint8
		SpriteEn      uint8
		Ctrl1         uint8
		Ctrl2         uint8
		IRQMask       uint8
		IRQFlags      uint8
		RasterIRQLine uint16
		ScrollX       uint8
		ScrollY       uint8
		BorderColor   uint8
		BGColor       [4]uint8
		Regs          [64]uint8
		Line          int32
		Cycle         int32
	}

	// CIA #0 and #1
	CIAs [2]struct {
		PortA       uint8
		PortB       uint8
		DDRA        uint8
		DDRB        uint8
		TimerA      uint16
		TimerB      uint16
		TimerALatch uint16
		TimerBLatch uint16
		CtrlA       uint8
		CtrlB       uint8
		IntMask     uint8
		IntFlags    uint8
		Ser         uint8
		IRQLine     bool
		Regs        [16]uint8
		Keyboard    [8][8]bool
		JoyA        uint8
		JoyB        uint8
	}

	// SID
	SID struct {
		Filter    uint8
		Resonance uint8
		Vol       uint8
		Osc3Val   uint8
		Osc       [3]struct {
			Freq     uint16
			PW       uint16
			Ctrl     uint8
			EnvState uint8
			EnvLevel uint8
			Acc      uint32
		}
	}
}

const saveStateVersion = 1

// SaveState serializes the machine state to a writer
func (m *Machine) SaveState(w io.Writer) error {
	var s SaveState
	s.Version = saveStateVersion

	// CPU
	s.CPU.A = m.cpu.A
	s.CPU.X = m.cpu.X
	s.CPU.Y = m.cpu.Y
	s.CPU.SP = m.cpu.SP
	s.CPU.SR = m.cpu.SR
	s.CPU.PC = m.cpu.PC
	s.CPU.Clk = m.cpu.clk

	// RAM and I/O
	copy(s.RAM[:], m.bus.RAM)
	copy(s.IO[:], m.bus.IO)
	s.Color = m.bus.Color

	// CPU Port
	s.CPUPortDDR = m.bus.cpuPortDDR
	s.CPUPortData = m.bus.cpuPortData

	// VIC-II
	vic := m.vic
	s.VIC.MX = vic.mx
	s.VIC.MY = vic.my
	s.VIC.MXExp = vic.mxExp
	s.VIC.MYExp = vic.myExp
	s.VIC.SpritePrior = vic.spritePrior
	s.VIC.SpriteMC = vic.spriteMC
	s.VIC.SpriteColl = vic.spriteColl
	s.VIC.SpriteBkgColl = vic.spriteBkgColl
	s.VIC.SpriteEn = vic.spriteEn
	s.VIC.Ctrl1 = vic.ctrl1
	s.VIC.Ctrl2 = vic.ctrl2
	s.VIC.IRQMask = vic.irqMask
	s.VIC.IRQFlags = vic.irqFlags
	s.VIC.RasterIRQLine = vic.rasterIRQLine
	s.VIC.ScrollX = vic.scrollX
	s.VIC.ScrollY = vic.scrollY
	s.VIC.BorderColor = vic.borderColor
	s.VIC.BGColor = vic.bgColor
	s.VIC.Regs = vic.regs
	s.VIC.Line = int32(vic.line)
	s.VIC.Cycle = int32(vic.cycle)

	// CIAs
	for i, cia := range m.cia {
		s.CIAs[i].PortA = cia.portA
		s.CIAs[i].PortB = cia.portB
		s.CIAs[i].DDRA = cia.ddrA
		s.CIAs[i].DDRB = cia.ddrB
		s.CIAs[i].TimerA = cia.timerA
		s.CIAs[i].TimerB = cia.timerB
		s.CIAs[i].TimerALatch = cia.timerALatch
		s.CIAs[i].TimerBLatch = cia.timerBLatch
		s.CIAs[i].CtrlA = cia.ctrlA
		s.CIAs[i].CtrlB = cia.ctrlB
		s.CIAs[i].IntMask = cia.intMask
		s.CIAs[i].IntFlags = cia.intFlags
		s.CIAs[i].Ser = cia.ser
		s.CIAs[i].IRQLine = cia.irqLine
		s.CIAs[i].Regs = cia.regs
		s.CIAs[i].Keyboard = cia.keyboard
		s.CIAs[i].JoyA = cia.joyA
		s.CIAs[i].JoyB = cia.joyB
	}

	// SID
	sid := m.sid
	s.SID.Filter = sid.filter
	s.SID.Resonance = sid.resonance
	s.SID.Vol = sid.vol
	s.SID.Osc3Val = sid.osc3val
	for i, osc := range sid.osc {
		s.SID.Osc[i].Freq = osc.freq
		s.SID.Osc[i].PW = osc.pw
		s.SID.Osc[i].Ctrl = osc.ctrl
		s.SID.Osc[i].EnvState = osc.envState
		s.SID.Osc[i].EnvLevel = osc.envLevel
		s.SID.Osc[i].Acc = osc.acc
	}

	return binary.Write(w, binary.LittleEndian, &s)
}

// LoadState deserializes the machine state from a reader
func (m *Machine) LoadState(r io.Reader) error {
	var s SaveState
	if err := binary.Read(r, binary.LittleEndian, &s); err != nil {
		return fmt.Errorf("reading save state: %w", err)
	}
	if s.Version != saveStateVersion {
		return fmt.Errorf("unsupported save state version: %d (expected %d)", s.Version, saveStateVersion)
	}

	// CPU
	m.cpu.A = s.CPU.A
	m.cpu.X = s.CPU.X
	m.cpu.Y = s.CPU.Y
	m.cpu.SP = s.CPU.SP
	m.cpu.SR = s.CPU.SR
	m.cpu.PC = s.CPU.PC
	m.cpu.clk = s.CPU.Clk

	// RAM and I/O
	copy(m.bus.RAM, s.RAM[:])
	copy(m.bus.IO, s.IO[:])
	m.bus.Color = s.Color

	// CPU Port
	m.bus.cpuPortDDR = s.CPUPortDDR
	m.bus.cpuPortData = s.CPUPortData

	// VIC-II
	vic := m.vic
	vic.mx = s.VIC.MX
	vic.my = s.VIC.MY
	vic.mxExp = s.VIC.MXExp
	vic.myExp = s.VIC.MYExp
	vic.spritePrior = s.VIC.SpritePrior
	vic.spriteMC = s.VIC.SpriteMC
	vic.spriteColl = s.VIC.SpriteColl
	vic.spriteBkgColl = s.VIC.SpriteBkgColl
	vic.spriteEn = s.VIC.SpriteEn
	vic.ctrl1 = s.VIC.Ctrl1
	vic.ctrl2 = s.VIC.Ctrl2
	vic.irqMask = s.VIC.IRQMask
	vic.irqFlags = s.VIC.IRQFlags
	vic.rasterIRQLine = s.VIC.RasterIRQLine
	vic.scrollX = s.VIC.ScrollX
	vic.scrollY = s.VIC.ScrollY
	vic.borderColor = s.VIC.BorderColor
	vic.bgColor = s.VIC.BGColor
	vic.regs = s.VIC.Regs
	vic.line = int(s.VIC.Line)
	vic.cycle = int(s.VIC.Cycle)

	// CIAs
	for i, cia := range m.cia {
		cia.portA = s.CIAs[i].PortA
		cia.portB = s.CIAs[i].PortB
		cia.ddrA = s.CIAs[i].DDRA
		cia.ddrB = s.CIAs[i].DDRB
		cia.timerA = s.CIAs[i].TimerA
		cia.timerB = s.CIAs[i].TimerB
		cia.timerALatch = s.CIAs[i].TimerALatch
		cia.timerBLatch = s.CIAs[i].TimerBLatch
		cia.ctrlA = s.CIAs[i].CtrlA
		cia.ctrlB = s.CIAs[i].CtrlB
		cia.intMask = s.CIAs[i].IntMask
		cia.intFlags = s.CIAs[i].IntFlags
		cia.ser = s.CIAs[i].Ser
		cia.irqLine = s.CIAs[i].IRQLine
		cia.regs = s.CIAs[i].Regs
		cia.keyboard = s.CIAs[i].Keyboard
		cia.joyA = s.CIAs[i].JoyA
		cia.joyB = s.CIAs[i].JoyB
	}

	// SID
	sid := m.sid
	sid.filter = s.SID.Filter
	sid.resonance = s.SID.Resonance
	sid.vol = s.SID.Vol
	sid.osc3val = s.SID.Osc3Val
	for i, osc := range sid.osc {
		osc.freq = s.SID.Osc[i].Freq
		osc.pw = s.SID.Osc[i].PW
		osc.ctrl = s.SID.Osc[i].Ctrl
		osc.envState = s.SID.Osc[i].EnvState
		osc.envLevel = s.SID.Osc[i].EnvLevel
		osc.acc = s.SID.Osc[i].Acc
	}

	return nil
}

// SaveStateToFile saves the machine state to a file
func (m *Machine) SaveStateToFile(path string) error {
	var buf bytes.Buffer
	if err := m.SaveState(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// LoadStateFromFile loads the machine state from a file
func (m *Machine) LoadStateFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return m.LoadState(bytes.NewReader(data))
}
