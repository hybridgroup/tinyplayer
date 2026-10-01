//go:build nrf52 || nrf52833 || nrf52840 || rp2040 || rp2350

package board

import (
	"machine"

	"github.com/hybridgroup/tinyplayer/flashdisk"
)

// Disk returns a FAT sized region at the end of the internal flash.
func Disk() (*flashdisk.Disk, error) {
	return flashdisk.Tail(machine.Flash, diskSize)
}
