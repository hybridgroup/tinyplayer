//go:build esp32c3

package board

import (
	"machine"

	"github.com/hybridgroup/tinyplayer/flashdisk"
)

const diskSize = 1 << 20

// Disk returns a region that ends at 2 MiB. Writes past that fail since TinyGo
// src/machine/machine_esp32c3_flash.go never sets the ROM flash chip size.
func Disk() (*flashdisk.Disk, error) {
	const romLimit = 2 << 20
	start := int64(machine.FlashDataStart() - 0x3C000000)
	return flashdisk.New(machine.Flash, romLimit-start-diskSize, diskSize)
}
