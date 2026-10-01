//go:build esp32c3 || nrf52 || nrf52833 || nrf52840 || rp2040 || rp2350

// Plays the WAV files on a FAT volume in flash. If there are none it copies
// the example sounds there first.
package main

import (
	"time"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinyplayer/examples/board"
	"github.com/hybridgroup/tinyplayer/examples/sounds"
	"github.com/hybridgroup/tinyplayer/storage"
	"github.com/soypat/fat"
)

func main() {
	time.Sleep(2 * time.Second)

	out, err := board.Output()
	if err != nil {
		failed("could not configure I2S", err)
	}
	player := tinyplayer.New(out)

	disk, err := board.Disk()
	if err != nil {
		failed("could not open flash", err)
	}
	vol, formatted, err := storage.Mount(disk)
	if err != nil {
		failed("could not mount FAT", err)
	}
	if formatted {
		println("formatted new FAT volume")
	}

	names, err := vol.ListWAV(nil)
	if err != nil {
		failed("could not list files", err)
	}
	if len(names) == 0 {
		for _, s := range sounds.All {
			println("copying", s.Name)
			if err := vol.WriteFile("/"+s.Name, s.Data); err != nil {
				failed("could not write file", err)
			}
		}
		names, _ = vol.ListWAV(nil)
	}

	var f fat.File
	for {
		for _, name := range names {
			println("playing", name)
			if err := vol.Open(&f, name); err != nil {
				println("error:", err.Error())
				continue
			}
			if err := player.Play(&f); err != nil {
				println("error:", err.Error())
			}
			f.Close()
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func failed(msg string, err error) {
	for {
		println(msg+":", err.Error())
		time.Sleep(time.Second)
	}
}
