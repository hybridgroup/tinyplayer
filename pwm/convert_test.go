package pwm

import "testing"

func TestTiming(t *testing.T) {
	tests := []struct {
		rate, repeat, top uint32
	}{
		{44100, 1, 3400},
		{22050, 2, 3400},
		{16000, 3, 3124},
		{8000, 5, 3749},
	}
	for _, tt := range tests {
		repeat, top := timing(150_000_000, tt.rate)
		if repeat != tt.repeat || top != tt.top {
			t.Errorf("timing(%d) = %d, %d, want %d, %d", tt.rate, repeat, top, tt.repeat, tt.top)
		}
		if carrier := 150_000_000 / (top + 1); carrier < minCarrier {
			t.Errorf("rate %d: carrier %d Hz below %d", tt.rate, carrier, minCarrier)
		}
	}
}

func frame(l, r int16) uint32 {
	return uint32(uint16(l)) | uint32(uint16(r))<<16
}

func TestFill(t *testing.T) {
	const top = 999
	frames := []uint32{
		frame(-32768, -32768),
		frame(0, 0),
		frame(32767, 32767),
		frame(32767, -32768),
		frame(16384, 0),
	}
	want := []uint32{0, 500, 999, 499, 625}
	dst := make([]uint32, 2*len(frames))
	if n := fill(dst, frames, 2, top); n != len(frames) {
		t.Fatalf("fill used %d frames, want %d", n, len(frames))
	}
	for i, w := range want {
		for j := range 2 {
			got := dst[2*i+j]
			if got != w|w<<16 {
				t.Errorf("frame %d copy %d = %#x, want %#x", i, j, got, w|w<<16)
			}
		}
	}
}

func TestFillShortDst(t *testing.T) {
	frames := make([]uint32, 10)
	dst := make([]uint32, 7)
	if n := fill(dst, frames, 3, 100); n != 2 {
		t.Errorf("fill used %d frames, want 2", n)
	}
}
