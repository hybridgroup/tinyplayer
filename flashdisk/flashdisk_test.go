package flashdisk

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
)

// nor acts like NOR flash. Writes can only clear bits and erases set a whole
// block to 0xFF.
type nor struct {
	data    []byte
	erases  int
	wbs     int64
	badBits bool
}

func newNOR(size int) *nor {
	return &nor{data: bytes.Repeat([]byte{0xFF}, size), wbs: 256}
}

func (n *nor) ReadAt(p []byte, off int64) (int, error) {
	return copy(p, n.data[off:]), nil
}

func (n *nor) WriteAt(p []byte, off int64) (int, error) {
	if off%n.wbs != 0 || int64(len(p))%n.wbs != 0 {
		return 0, errors.New("unaligned write")
	}
	for i, v := range p {
		if n.data[off+int64(i)]&v != v {
			n.badBits = true
		}
		n.data[off+int64(i)] &= v
	}
	return len(p), nil
}

func (n *nor) Size() int64           { return int64(len(n.data)) }
func (n *nor) WriteBlockSize() int64 { return n.wbs }
func (n *nor) EraseBlockSize() int64 { return 4096 }
func (n *nor) EraseBlocks(start, count int64) error {
	n.erases += int(count)
	for i := start * 4096; i < (start+count)*4096; i++ {
		n.data[i] = 0xFF
	}
	return nil
}

func TestRandomWrites(t *testing.T) {
	dev := newNOR(64 * 1024)
	d, err := Tail(dev, 32*1024)
	if err != nil {
		t.Fatal(err)
	}
	if d.offset != 32*1024 {
		t.Fatalf("offset = %d", d.offset)
	}
	ref := bytes.Repeat([]byte{0xFF}, int(d.Size()))
	rng := rand.New(rand.NewSource(1))
	buf := make([]byte, 3*SectorSize)
	for i := 0; i < 500; i++ {
		n := (1 + rng.Intn(3)) * SectorSize
		off := int64(rng.Intn(int(d.Size())/SectorSize-3)) * SectorSize
		rng.Read(buf[:n])
		if _, err := d.WriteAt(buf[:n], off); err != nil {
			t.Fatal(err)
		}
		copy(ref[off:], buf[:n])
		if i%7 == 0 {
			if err := d.Sync(); err != nil {
				t.Fatal(err)
			}
		}
		got := make([]byte, n)
		if _, err := d.ReadAt(got, off); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, ref[off:off+int64(n)]) {
			t.Fatalf("write %d: read back differs", i)
		}
	}
	d.Sync()
	if dev.badBits {
		t.Fatal("wrote 1 bits over 0 bits without erasing")
	}
	if !bytes.Equal(dev.data[32*1024:], ref) {
		t.Fatal("flash contents differ")
	}
	if !bytes.Equal(dev.data[:32*1024], bytes.Repeat([]byte{0xFF}, 32*1024)) {
		t.Fatal("wrote outside the region")
	}
}

func TestSkipErase(t *testing.T) {
	dev := newNOR(16 * 1024)
	d, _ := New(dev, 0, 16*1024)
	sector := bytes.Repeat([]byte{0x5A}, SectorSize)
	d.WriteAt(sector, 0)
	d.Sync()
	d.WriteAt(sector, SectorSize)
	d.Sync()
	if dev.erases != 0 {
		t.Fatalf("erased %d times on blank flash", dev.erases)
	}
	d.WriteAt(bytes.Repeat([]byte{0xA5}, SectorSize), 0)
	d.Sync()
	if dev.erases != 1 || dev.badBits {
		t.Fatalf("erases = %d badBits = %v", dev.erases, dev.badBits)
	}
	if !bytes.Equal(dev.data[SectorSize:2*SectorSize], sector) {
		t.Fatal("neighbor sector lost after erase")
	}
}

func TestBadRegion(t *testing.T) {
	dev := newNOR(16 * 1024)
	if _, err := New(dev, 512, 4096); err != ErrAlignment {
		t.Errorf("unaligned offset: %v", err)
	}
	if _, err := New(dev, 0, 32*1024); err != ErrTooSmall {
		t.Errorf("too big: %v", err)
	}
	d, _ := New(dev, 0, 4096)
	if _, err := d.WriteAt(make([]byte, 512), 4096); err != ErrBounds {
		t.Errorf("write past end: %v", err)
	}
}
