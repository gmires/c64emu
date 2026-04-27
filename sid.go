package c64

type SID struct {
	clk       uint64
	bus       *Bus
	osc       [3]oscillator
	filter    uint8
	resonance uint8
	vol       uint8
	osc3val   uint8
	// Filter state
	filterCutoff uint16 // 11-bit from $D415/$D416
	filterMode   uint8  // $D418 bits 6-4: HP, BP, LP
	filterVoice  uint8  // $D417 bits 2-0: which voices pass through filter
	// Simple digital filter state
	filterZ1     float32 // previous input
	filterZ2     float32 // previous output
}

type oscillator struct {
	freq     uint16
	pw       uint16
	ctrl     uint8
	envState uint8 // 0=idle, 1=attack, 2=decay, 3=sustain, 4=release
	envLevel uint8
	acc      uint32
}

func NewSID(bus *Bus) *SID {
	return &SID{
		clk: 0,
		bus: bus,
		vol: 0x0F,
	}
}

func (s *SID) step() uint8 {
	s.clk++
	for i := range s.osc {
		s.updateOsc(i)
	}
	return 1
}

func (s *SID) updateOsc(idx int) {
	osc := &s.osc[idx]
	if osc.freq == 0 {
		return
	}
	osc.acc += uint32(osc.freq)
	
	switch osc.envState {
	case 1: // attack
		if osc.envLevel < 0xFF {
			osc.envLevel += 2
		}
	case 2: // decay
		if osc.envLevel > 0 {
			osc.envLevel -= 1
		}
	case 3: // sustain
		// hold
	case 4: // release
		if osc.envLevel > 0 {
			osc.envLevel -= 1
		}
	}
}

func (s *SID) Sample() float32 {
	var out float32 = 0
	var filtered float32 = 0
	for i := range s.osc {
		osc := &s.osc[i]
		if osc.ctrl == 0 {
			continue
		}
		var sample float32 = 0
		if osc.ctrl&0x40 != 0 { // noise
			sample = float32(osc.acc>>24)/128.0 - 1.0
		} else if osc.ctrl&0x20 != 0 { // pulse
			if osc.acc>>16 < uint32(osc.pw) {
				sample = 1.0
			} else {
				sample = -1.0
			}
		} else if osc.ctrl&0x10 != 0 { // sawtooth
			sample = float32(osc.acc>>20)/2048.0 - 1.0
		} else if osc.ctrl&0x08 != 0 { // triangle
			sample = float32((osc.acc>>19)&0x7FF)/1024.0 - 1.0
			if osc.acc&(1<<31) != 0 {
				sample = -sample
			}
		}
		voiceOut := sample * float32(osc.envLevel) / 255.0
		out += voiceOut
		if s.filterVoice&(1<<i) != 0 {
			filtered += voiceOut
		}
	}

	// Apply filter if any voice is routed through it
	if s.filterVoice != 0 {
		filtered = s.applyFilter(filtered)
	}

	// Mix filtered and unfiltered voices
	unfiltered := out - filtered
	result := unfiltered + filtered
	return result * float32(s.vol&0x0F) / 15.0
}

func (s *SID) applyFilter(input float32) float32 {
	if s.filterCutoff == 0 {
		return 0
	}
	// Simple digital approximation of SID filter
	// Cutoff range: 0-2047 (11-bit)
	// Normalize to 0.0 - 0.99
	cutoff := float32(s.filterCutoff) / 2048.0
	if cutoff > 0.99 {
		cutoff = 0.99
	}
	resonance := float32(s.resonance) / 15.0 // 0-1

	var output float32

	// Apply selected filter mode
	switch s.filterMode {
	case 0x01: // LP
		// Simple 1-pole low pass
		s.filterZ2 = s.filterZ2*(1.0-cutoff) + input*cutoff
		output = s.filterZ2
	case 0x02: // BP
		// Simple band pass (difference of two low passes)
		lp1 := s.filterZ1*(1.0-cutoff) + input*cutoff
		lp2 := s.filterZ2*(1.0-cutoff) + lp1*cutoff
		s.filterZ1 = lp1
		s.filterZ2 = lp2
		output = lp1 - lp2
	case 0x04: // HP
		// Simple high pass
		s.filterZ2 = s.filterZ2*(1.0-cutoff) + input*cutoff
		output = input - s.filterZ2
	default:
		// Multiple modes or off - just pass through
		output = input
	}

	// Apply resonance boost around cutoff (simplified)
	if resonance > 0 && (s.filterMode&0x03 != 0) {
		output *= 1.0 + resonance*0.5
	}

	return output
}

func (s *SID) handleReg(addr uint8, val uint8) {
	addr &= 0x1F

	switch addr {
	case 0x15:
		s.filterCutoff = (s.filterCutoff & 0x0700) | uint16(val)
	case 0x16:
		s.filterCutoff = (s.filterCutoff & 0x00FF) | (uint16(val&0x07) << 8)
		s.resonance = (val >> 4) & 0x0F
	case 0x17:
		s.filterVoice = val & 0x07
		s.filter = val
	case 0x18:
		s.vol = val & 0x0F
		s.filterMode = (val >> 4) & 0x07
	default:
		if addr >= 0x18 {
			return // reserved / unused registers
		}
		voice := addr >> 3
		osc := &s.osc[voice]
		switch addr & 0x07 {
		case 0x00:
			osc.freq = (osc.freq & 0xFF00) | uint16(val)
		case 0x01:
			osc.freq = (osc.freq & 0x00FF) | (uint16(val) << 8)
		case 0x02:
			osc.pw = (osc.pw & 0xFF00) | uint16(val)
		case 0x03:
			osc.pw = (osc.pw & 0x00FF) | (uint16(val&0x0F) << 8)
		case 0x04:
			gate := val&0x01 != 0
			if gate && osc.envState == 0 {
				osc.envState = 1 // attack
			} else if !gate && osc.envState != 0 {
				osc.envState = 4 // release
			}
			osc.ctrl = val
		case 0x05:
			// attack/decay
		case 0x06:
			// sustain/release
		}
	}
}

func (s *SID) readReg(addr uint8) uint8 {
	switch addr & 0x1F {
	case 0x1B:
		return uint8(s.osc[2].acc >> 24)
	case 0x1C:
		return s.osc[2].envLevel
	default:
		return 0
	}
}

func (s *SID) Clock() uint64 {
	return s.clk
}
