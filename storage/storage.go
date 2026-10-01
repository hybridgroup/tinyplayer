// Package storage keeps WAV files on a FAT volume.
package storage

import (
	"io"
	"strings"

	"github.com/soypat/fat"
)

// Disk is a block device with 512 byte sectors.
type Disk interface {
	fat.BlockDevice
	Size() int64
}

const sectorSize = 512

// Volume is a mounted FAT filesystem on a Disk.
type Volume struct {
	FS   fat.FS
	disk Disk
}

// Mount mounts the FAT volume on disk. If there is none it formats the disk
// first and reports that it did.
func Mount(disk Disk) (v *Volume, formatted bool, err error) {
	v = &Volume{disk: disk}
	if v.FS.Mount(disk, sectorSize, fat.ModeRW) == nil {
		return v, false, nil
	}
	var f fat.Formatter
	n := int(disk.Size() / sectorSize)
	err = f.Format(disk, sectorSize, n, fat.FormatParams{Format: fat.FormatFAT16})
	if err != nil {
		err = f.Format(disk, sectorSize, n, fat.FormatParams{Format: fat.FormatFAT12})
	}
	if err == nil {
		err = v.sync()
	}
	if err == nil {
		err = v.FS.Mount(disk, sectorSize, fat.ModeRW)
	}
	if err != nil {
		return nil, false, err
	}
	return v, true, nil
}

// Remount reads the volume again, for example after a USB host changed it.
func (v *Volume) Remount() error {
	v.FS.Unmount()
	return v.FS.Mount(v.disk, sectorSize, fat.ModeRW)
}

func (v *Volume) Open(f *fat.File, path string) error {
	return v.FS.OpenFile(f, path, fat.ModeRead)
}

// ListWAV appends the paths of the .wav files in the root directory to dst.
func (v *Volume) ListWAV(dst []string) ([]string, error) {
	var dir fat.Dir
	if err := v.FS.OpenDir(&dir, "/"); err != nil {
		return dst, err
	}
	defer dir.Close()
	err := dir.ForEachFile(func(fi *fat.FileInfo) error {
		name := fi.Name()
		if !fi.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(strings.ToLower(name), ".wav") {
			dst = append(dst, "/"+name)
		}
		return nil
	})
	return dst, err
}

// WriteFile creates or replaces the file at path and writes it to flash.
func (v *Volume) WriteFile(path string, data string) error {
	var f fat.File
	if err := v.FS.OpenFile(&f, path, fat.ModeCreateAlways|fat.ModeWrite); err != nil {
		return err
	}
	if _, err := io.WriteString(&f, data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return v.sync()
}

func (v *Volume) sync() error {
	if s, ok := v.disk.(interface{ Sync() error }); ok {
		return s.Sync()
	}
	return nil
}
