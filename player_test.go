package tinyplayer

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/hybridgroup/tinyplayer/wav"
)

type fakeOutput struct {
	rate    uint32
	frames  []uint32
	onWrite func()
	enables []bool
}

func (f *fakeOutput) SetSampleFrequency(freq uint32) error { f.rate = freq; return nil }
func (f *fakeOutput) Enable(e bool)                        { f.enables = append(f.enables, e) }
func (f *fakeOutput) WriteStereo(b []uint32) (int, error) {
	f.frames = append(f.frames, b...)
	if f.onWrite != nil {
		f.onWrite()
	}
	return len(b), nil
}

func TestPlayMono(t *testing.T) {
	data, _ := os.ReadFile("wav/testdata/mono_s16_list.wav")
	want, _ := wav.NewDecoder(bytes.NewReader(data))
	samples := make([]int16, 4000)
	n, _ := want.Read(samples)

	out := &fakeOutput{}
	p := New(out)
	p.SetVolume(128)
	if err := p.Play(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if el, tot := p.Progress(); el != tot || tot != time.Duration(n)*time.Second/16000 {
		t.Fatalf("progress %v of %v", el, tot)
	}
	if out.rate != 16000 || len(out.frames) != n {
		t.Fatalf("rate %d frames %d, want 16000 and %d", out.rate, len(out.frames), n)
	}
	for i, fr := range out.frames {
		v := uint16(int16(int32(samples[i]) * 128 >> 8))
		if fr != uint32(v)|uint32(v)<<16 {
			t.Fatalf("frame %d = %08x", i, fr)
		}
	}
}

func TestPlayStereoStop(t *testing.T) {
	data, _ := os.ReadFile("wav/testdata/stereo_adpcm.wav")
	out := &fakeOutput{}
	p := New(out)
	out.onWrite = p.Stop
	if err := p.Play(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if len(out.frames) != bufferFrames {
		t.Fatalf("got %d frames after Stop", len(out.frames))
	}
	d, _ := wav.NewDecoder(bytes.NewReader(data))
	s := make([]int16, 2)
	d.Read(s)
	if out.frames[0] != uint32(uint16(s[0]))|uint32(uint16(s[1]))<<16 {
		t.Fatalf("frame 0 = %08x, want L %d R %d", out.frames[0], s[0], s[1])
	}
}

func TestPause(t *testing.T) {
	data, _ := os.ReadFile("wav/testdata/mono_s16_list.wav")
	ref := &fakeOutput{}
	New(ref).Play(bytes.NewReader(data))

	out := &fakeOutput{}
	p := New(out)
	out.onWrite = func() {
		out.onWrite = nil
		p.Pause(true)
		go func() {
			time.Sleep(30 * time.Millisecond)
			p.Pause(false)
		}()
	}
	if err := p.Play(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if len(out.frames) != len(ref.frames) {
		t.Fatalf("got %d frames, want %d", len(out.frames), len(ref.frames))
	}
	if len(out.enables) != 2 || out.enables[0] || !out.enables[1] {
		t.Fatalf("enables %v", out.enables)
	}
}
