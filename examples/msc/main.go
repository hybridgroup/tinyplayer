//go:build nrf52840 || rp2040 || rp2350

// Shows the flash FAT volume as a USB drive and plays the WAV files on it.
// Copy new WAV files to the drive and they play after the copy finishes.
package main

import (
	"machine"
	"machine/usb/msc"
	"sync/atomic"
	"time"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinyplayer/examples/board"
	"github.com/hybridgroup/tinyplayer/flashdisk"
	"github.com/hybridgroup/tinyplayer/storage"
	"github.com/soypat/fat"
)

var refresh atomic.Bool

func main() {
	// MSC replaces the USB descriptor, so it must be set up before the host
	// enumerates, see TinyGo src/machine/usb.go ConfigureUSBEndpoint.
	disk, err := board.Disk()
	if err != nil {
		failed("could not open flash", err)
	}
	m := msc.Port(disk)
	m.SetVendorID("TinyGo")
	m.SetProductID("tinyplayer")
	machine.USBDev.Configure(machine.UARTConfig{})

	vol, formatted, err := storage.Mount(disk)
	if err != nil {
		failed("could not mount FAT", err)
	}
	if formatted {
		println("formatted new FAT volume")
	}

	out, err := board.Output()
	if err != nil {
		failed("could not configure I2S", err)
	}
	player := tinyplayer.New(out)
	go watch(disk, player)

	var names []string
	var f fat.File
	for {
		if refresh.Swap(false) {
			println("drive changed, reading files again")
			if err := vol.Remount(); err != nil {
				println("error:", err.Error())
			}
		}
		names, err = vol.ListWAV(names[:0])
		if err != nil {
			println("error:", err.Error())
		}
		if len(names) == 0 {
			println("no WAV files, copy some to the USB drive")
			time.Sleep(2 * time.Second)
			continue
		}
		for _, name := range names {
			if refresh.Load() {
				break
			}
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

// watch waits until the host has stopped writing for a second, then writes
// the cached flash block out and asks main to read the volume again.
func watch(disk *flashdisk.Disk, player *tinyplayer.Player) {
	last := disk.Changed()
	idle := 0
	for {
		time.Sleep(100 * time.Millisecond)
		now := disk.Changed()
		if now != last {
			last = now
			idle = 1
			continue
		}
		if idle == 0 {
			continue
		}
		idle++
		if idle > 10 {
			idle = 0
			if err := disk.Sync(); err != nil {
				println("error:", err.Error())
			}
			refresh.Store(true)
			player.Stop()
		}
	}
}

func failed(msg string, err error) {
	for {
		println(msg+":", err.Error())
		time.Sleep(time.Second)
	}
}
