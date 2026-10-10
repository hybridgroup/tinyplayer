// Package tinyplayer plays WAV audio on an I2S DAC or a PWM speaker output.
package tinyplayer

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/hybridgroup/tinyplayer/wav"
)

// Output is an I2S bus such as *machine.I2S or *piolib.I2S, or a *pwm.PWM.
type Output interface {
	SetSampleFrequency(freq uint32) error
	WriteStereo(b []uint32) (int, error)
	Enable(enabled bool)
}

const bufferFrames = 256

// Player decodes WAV audio and writes it to an Output.
type Player struct {
	out     Output
	dec     wav.Decoder
	rate    uint32
	volume  atomic.Int32
	stop    atomic.Bool
	paused  atomic.Bool
	played  atomic.Uint32
	total   atomic.Uint32
	samples [bufferFrames * 2]int16
	frames  [bufferFrames]uint32
}

// New returns a Player that writes to out at full volume.
func New(out Output) *Player {
	p := &Player{out: out}
	p.volume.Store(256)
	return p
}

// SetVolume sets the volume from 0 (silent) to 256 (full).
func (p *Player) SetVolume(v int) {
	p.volume.Store(int32(max(0, min(256, v))))
}

// Stop ends the current Play call. It is safe to call from another goroutine.
func (p *Player) Stop() {
	p.stop.Store(true)
}

// Pause pauses or resumes the current Play call. It is safe to call from
// another goroutine.
func (p *Player) Pause(paused bool) {
	p.paused.Store(paused)
}

// Paused reports whether playback is paused.
func (p *Player) Paused() bool {
	return p.paused.Load()
}

func (p *Player) waitPaused() {
	p.out.Enable(false)
	for p.paused.Load() && !p.stop.Load() {
		time.Sleep(10 * time.Millisecond)
	}
	p.out.Enable(true)
}

// Progress returns how much of the current or last stream has been queued and
// its length.
func (p *Player) Progress() (elapsed, total time.Duration) {
	rate := time.Duration(p.rate)
	if rate == 0 {
		return 0, 0
	}
	return time.Duration(p.played.Load()) * time.Second / rate,
		time.Duration(p.total.Load()) * time.Second / rate
}

// Play decodes the WAV stream in r and returns once all of it is queued.
func (p *Player) Play(r io.Reader) error {
	p.stop.Store(false)
	p.played.Store(0)
	p.total.Store(0)
	if err := p.dec.Reset(r); err != nil {
		return err
	}
	p.total.Store(uint32(p.dec.Frames()))
	f := p.dec.Format()
	if f.SampleRate != p.rate {
		if err := p.out.SetSampleFrequency(f.SampleRate); err != nil {
			return err
		}
		p.rate = f.SampleRate
	}
	ch := f.Channels
	for !p.stop.Load() {
		if p.paused.Load() {
			p.waitPaused()
			continue
		}
		n, err := p.dec.Read(p.samples[:bufferFrames*ch])
		if n > 0 {
			vol := p.volume.Load()
			frames := n / ch
			for i := 0; i < frames; i++ {
				l := int32(p.samples[i*ch])
				r := int32(p.samples[i*ch+ch-1])
				l = l * vol >> 8
				r = r * vol >> 8
				p.frames[i] = uint32(uint16(l)) | uint32(uint16(r))<<16
			}
			if _, werr := p.out.WriteStereo(p.frames[:frames]); werr != nil {
				return werr
			}
			p.played.Add(uint32(frames))
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}
