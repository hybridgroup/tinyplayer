package storage

import (
	"bytes"
	"os"
	"slices"
	"testing"

	"github.com/hybridgroup/tinyplayer/flashdisk"
	"github.com/hybridgroup/tinyplayer/wav"
	"github.com/soypat/fat"
)

type nor struct{ data []byte }

func (n *nor) ReadAt(p []byte, off int64) (int, error) { return copy(p, n.data[off:]), nil }
func (n *nor) WriteAt(p []byte, off int64) (int, error) {
	for i, v := range p {
		n.data[off+int64(i)] &= v
	}
	return len(p), nil
}
func (n *nor) Size() int64           { return int64(len(n.data)) }
func (n *nor) WriteBlockSize() int64 { return 4 }
func (n *nor) EraseBlockSize() int64 { return 4096 }
func (n *nor) EraseBlocks(start, count int64) error {
	clear := n.data[start*4096 : (start+count)*4096]
	for i := range clear {
		clear[i] = 0xFF
	}
	return nil
}

func TestVolume(t *testing.T) {
	speech, err := os.ReadFile("../examples/sounds/speech.wav")
	if err != nil {
		t.Fatal(err)
	}
	beep, _ := os.ReadFile("../examples/sounds/beep.wav")
	dev := &nor{data: bytes.Repeat([]byte{0xFF}, 1<<20+100*1024)}

	for _, size := range []int64{256 * 1024, 1 << 20} {
		disk, err := flashdisk.Tail(dev, size)
		if err != nil {
			t.Fatal(err)
		}
		v, formatted, err := Mount(disk)
		if err != nil || !formatted {
			t.Fatalf("size %d: formatted=%v err=%v", size, formatted, err)
		}
		if err := v.WriteFile("/speech.wav", string(speech)); err != nil {
			t.Fatal(err)
		}
		if err := v.WriteFile("/Beep.WAV", string(beep)); err != nil {
			t.Fatal(err)
		}
		if err := v.WriteFile("/notes.txt", "hello"); err != nil {
			t.Fatal(err)
		}

		disk, _ = flashdisk.Tail(dev, size)
		v, formatted, err = Mount(disk)
		if err != nil || formatted {
			t.Fatalf("remount: formatted=%v err=%v", formatted, err)
		}
		names, err := v.ListWAV(nil)
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(names)
		if !slices.Equal(names, []string{"/Beep.WAV", "/speech.wav"}) {
			t.Fatalf("names = %q", names)
		}

		var f fat.File
		if err := v.Open(&f, "/speech.wav"); err != nil {
			t.Fatal(err)
		}
		d, err := wav.NewDecoder(&f)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := wav.NewDecoder(bytes.NewReader(speech))
		a := make([]int16, 300)
		b := make([]int16, 300)
		total := 0
		for {
			n, err := d.Read(a)
			m, _ := want.Read(b)
			if n != m || !slices.Equal(a[:n], b[:m]) {
				t.Fatalf("decoded audio differs after %d samples", total)
			}
			total += n
			if err != nil {
				break
			}
		}
		f.Close()
		if total < 40000 {
			t.Fatalf("only %d samples", total)
		}
		for i := range dev.data {
			dev.data[i] = 0xFF
		}
	}
}
