package c64

import (
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const (
	AudioSampleRate = 44100
	cyclesPerSample = 985248 / AudioSampleRate // ~22 cycles per sample for PAL
)

// SIDAudioStream implements io.Reader to provide continuous audio playback
type SIDAudioStream struct {
	sid     *SID
	machine *Machine
	buffer  []byte
	pos     int
}

func NewSIDAudioStream(machine *Machine) *SIDAudioStream {
	return &SIDAudioStream{
		sid:     machine.SID(),
		machine: machine,
		buffer:  make([]byte, 0),
	}
}

func (s *SIDAudioStream) Read(p []byte) (int, error) {
	// Generate samples on demand
	n := len(p) / 2 // 16-bit samples
	if n == 0 {
		return 0, nil
	}
	for i := 0; i < n; i++ {
		// Run machine for ~22 cycles to generate one sample
		for j := 0; j < cyclesPerSample; j++ {
			s.machine.Step()
		}
		sample := s.sid.Sample()
		// Clamp sample to [-1, 1]
		if sample > 1.0 {
			sample = 1.0
		} else if sample < -1.0 {
			sample = -1.0
		}
		val := int16(sample * 32767)
		p[i*2] = byte(val)
		p[i*2+1] = byte(val >> 8)
	}
	return len(p), nil
}

type AudioSystem struct {
	ctx    *audio.Context
	player *audio.Player
	stream *SIDAudioStream
}

func NewAudioSystem(machine *Machine) (*AudioSystem, error) {
	ctx := audio.NewContext(AudioSampleRate)
	stream := NewSIDAudioStream(machine)
	player, err := ctx.NewPlayer(stream)
	if err != nil {
		return nil, err
	}
	return &AudioSystem{
		ctx:    ctx,
		player: player,
		stream: stream,
	}, nil
}

func (as *AudioSystem) Start() {
	as.player.Play()
}

func (as *AudioSystem) Stop() {
	as.player.Pause()
}

func (as *AudioSystem) SetVolume(vol float64) {
	as.player.SetVolume(vol)
}
