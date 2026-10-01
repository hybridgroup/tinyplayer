package wav

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"testing"
)

type onlyReader struct{ r io.Reader }

func (o onlyReader) Read(p []byte) (int, error) { return o.r.Read(p) }

func TestDecodeReference(t *testing.T) {
	tests := []struct {
		name     string
		codec    Codec
		channels int
		rate     uint32
	}{
		{"mono_adpcm", CodecIMAADPCM, 1, 16000},
		{"stereo_adpcm", CodecIMAADPCM, 2, 22050},
		{"stereo_u8", CodecPCM, 2, 8000},
		{"mono_s16_list", CodecPCM, 1, 16000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, err := os.ReadFile("testdata/" + tt.name + ".wav")
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile("testdata/" + tt.name + ".raw")
			if err != nil {
				t.Fatal(err)
			}
			want := make([]int16, len(raw)/2)
			binary.Read(bytes.NewReader(raw), binary.LittleEndian, want)

			for _, seek := range []bool{true, false} {
				var r io.Reader = bytes.NewReader(in)
				if !seek {
					r = onlyReader{r}
				}
				d, err := NewDecoder(r)
				if err != nil {
					t.Fatal(err)
				}
				f := d.Format()
				if f.Codec != tt.codec || f.Channels != tt.channels || f.SampleRate != tt.rate {
					t.Fatalf("format = %+v", f)
				}
				got := readAll(t, d, 37*tt.channels)
				if len(got) != len(want) {
					t.Fatalf("got %d samples, want %d", len(got), len(want))
				}
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("sample %d = %d, want %d", i, got[i], want[i])
					}
				}
			}
		})
	}
}

func readAll(t *testing.T, d *Decoder, chunk int) []int16 {
	var out []int16
	buf := make([]int16, chunk)
	for {
		n, err := d.Read(buf)
		if n%d.Format().Channels != 0 {
			t.Fatalf("partial frame read: %d", n)
		}
		out = append(out, buf[:n]...)
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestErrors(t *testing.T) {
	if _, err := NewDecoder(bytes.NewReader([]byte("not a wav file at all"))); !errors.Is(err, ErrNotWAV) {
		t.Errorf("garbage: %v", err)
	}
	in, _ := os.ReadFile("testdata/mono_s16_list.wav")
	float := bytes.Clone(in)
	binary.LittleEndian.PutUint16(float[20:], 3)
	if _, err := NewDecoder(bytes.NewReader(float)); !errors.Is(err, ErrUnsupported) {
		t.Errorf("float: %v", err)
	}
	if _, err := NewDecoder(bytes.NewReader(in[:40])); !errors.Is(err, ErrNoData) {
		t.Errorf("truncated header: %v", err)
	}
}

func TestReuse(t *testing.T) {
	in, _ := os.ReadFile("testdata/stereo_adpcm.wav")
	var d Decoder
	for i := 0; i < 2; i++ {
		if err := d.Reset(bytes.NewReader(in)); err != nil {
			t.Fatal(err)
		}
		if n := len(readAll(t, &d, 256)); n != 24408/2 {
			t.Fatalf("pass %d: %d samples", i, n)
		}
	}
}
