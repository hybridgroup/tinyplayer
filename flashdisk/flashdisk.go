// Package flashdisk turns a region of NOR flash into a disk with 512 byte
// sectors that can be rewritten in place, for use by FAT and USB MSC.
package flashdisk

import (
	"errors"
	"io"
	"sync/atomic"
)

// BlockDevice has the same methods as machine.BlockDevice.
type BlockDevice interface {
	io.ReaderAt
	io.WriterAt
	Size() int64
	WriteBlockSize() int64
	EraseBlockSize() int64
	EraseBlocks(start, len int64) error
}

const SectorSize = 512

var (
	ErrAlignment = errors.New("flashdisk: region not aligned to erase blocks")
	ErrBounds    = errors.New("flashdisk: access out of range")
	ErrTooSmall  = errors.New("flashdisk: device too small")
)

// Disk caches one erase block and erases flash only when a write needs it.
// Call Sync to write the cached block out.
type Disk struct {
	mu         lock
	dev        BlockDevice
	offset     int64
	size       int64
	eraseSize  int64
	cache      []byte
	cacheBlock int64
	dirty      bool
	changed    atomic.Uint32
	scratch    [SectorSize]byte
}

// New uses size bytes of dev starting at offset. Both must be multiples of
// the erase block size.
func New(dev BlockDevice, offset, size int64) (*Disk, error) {
	es := dev.EraseBlockSize()
	if es < SectorSize || es > 64*SectorSize || es%SectorSize != 0 || SectorSize%dev.WriteBlockSize() != 0 {
		return nil, ErrAlignment
	}
	if offset%es != 0 || size%es != 0 || size <= 0 {
		return nil, ErrAlignment
	}
	if offset < 0 || offset+size > dev.Size() {
		return nil, ErrTooSmall
	}
	return &Disk{
		dev:        dev,
		offset:     offset,
		size:       size,
		eraseSize:  es,
		cache:      make([]byte, es),
		cacheBlock: -1,
	}, nil
}

// Tail uses the last size bytes of dev. With machine.Flash that region stays
// in the same place when the program grows.
func Tail(dev BlockDevice, size int64) (*Disk, error) {
	es := dev.EraseBlockSize()
	end := dev.Size() / es * es
	if size > end {
		return nil, ErrTooSmall
	}
	return New(dev, end-size, size)
}

func (d *Disk) Size() int64 { return d.size }

func (d *Disk) WriteBlockSize() int64 { return SectorSize }

// EraseBlockSize returns the sector size, so EraseBlocks counts sectors.
func (d *Disk) EraseBlockSize() int64 { return SectorSize }

// EraseBlocks discards sectors. The data is left as is since a discard does
// not promise anything about later reads.
func (d *Disk) EraseBlocks(start, n int64) error {
	if start < 0 || n < 0 || (start+n)*SectorSize > d.size {
		return ErrBounds
	}
	return nil
}

// Changed returns a counter that goes up on every write.
func (d *Disk) Changed() uint32 { return d.changed.Load() }

// ReadBlocks reads whole sectors, as used by github.com/soypat/fat.
func (d *Disk) ReadBlocks(dst []byte, start int64) (int, error) {
	return d.ReadAt(dst, start*SectorSize)
}

// WriteBlocks writes whole sectors, as used by github.com/soypat/fat.
func (d *Disk) WriteBlocks(data []byte, start int64) (int, error) {
	return d.WriteAt(data, start*SectorSize)
}

func (d *Disk) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || off+int64(len(p)) > d.size {
		return 0, ErrBounds
	}
	// No defer since MSC calls this from an interrupt, see TinyGo
	// src/machine/usb/msc/scsi_readwrite.go readBlock.
	state := d.mu.lock()
	n, err := d.readAt(p, off)
	d.mu.unlock(state)
	return n, err
}

func (d *Disk) readAt(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		pos := off + int64(n)
		blk := pos / d.eraseSize
		in := pos % d.eraseSize
		m := min(int64(len(p)-n), d.eraseSize-in)
		if blk == d.cacheBlock {
			copy(p[n:], d.cache[in:in+m])
		} else if _, err := d.dev.ReadAt(p[n:n+int(m)], d.offset+pos); err != nil {
			return n, err
		}
		n += int(m)
	}
	return n, nil
}

func (d *Disk) WriteAt(p []byte, off int64) (int, error) {
	if off < 0 || off+int64(len(p)) > d.size {
		return 0, ErrBounds
	}
	state := d.mu.lock()
	n, err := d.writeAt(p, off)
	d.mu.unlock(state)
	return n, err
}

func (d *Disk) writeAt(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		pos := off + int64(n)
		blk := pos / d.eraseSize
		in := pos % d.eraseSize
		m := min(int64(len(p)-n), d.eraseSize-in)
		if err := d.load(blk); err != nil {
			return n, err
		}
		copy(d.cache[in:in+m], p[n:])
		d.dirty = true
		n += int(m)
	}
	d.changed.Add(1)
	return n, nil
}

// Sync writes the cached erase block to flash.
func (d *Disk) Sync() error {
	state := d.mu.lock()
	err := d.flush()
	d.mu.unlock(state)
	return err
}

func (d *Disk) load(blk int64) error {
	if blk == d.cacheBlock {
		return nil
	}
	if err := d.flush(); err != nil {
		return err
	}
	d.cacheBlock = -1
	if _, err := d.dev.ReadAt(d.cache, d.offset+blk*d.eraseSize); err != nil {
		return err
	}
	d.cacheBlock = blk
	return nil
}

// flush writes back only the sectors that changed. NOR flash writes can only
// clear bits, so the block is erased first when a changed sector sets any.
func (d *Disk) flush() error {
	if !d.dirty {
		return nil
	}
	base := d.offset + d.cacheBlock*d.eraseSize
	var changed uint64
	needErase := false
	for s := int64(0); s < d.eraseSize/SectorSize; s++ {
		if _, err := d.dev.ReadAt(d.scratch[:], base+s*SectorSize); err != nil {
			return err
		}
		c := d.cache[s*SectorSize:][:SectorSize]
		for i, old := range d.scratch {
			if old != c[i] {
				changed |= 1 << s
				if old&c[i] != c[i] {
					needErase = true
				}
			}
		}
	}
	if needErase {
		if err := d.dev.EraseBlocks(base/d.eraseSize, 1); err != nil {
			return err
		}
	}
	for s := int64(0); s < d.eraseSize/SectorSize; s++ {
		c := d.cache[s*SectorSize:][:SectorSize]
		if needErase && allFF(c) || !needErase && changed&(1<<s) == 0 {
			continue
		}
		if _, err := d.dev.WriteAt(c, base+s*SectorSize); err != nil {
			return err
		}
	}
	d.dirty = false
	return nil
}

func allFF(b []byte) bool {
	for _, v := range b {
		if v != 0xFF {
			return false
		}
	}
	return true
}
