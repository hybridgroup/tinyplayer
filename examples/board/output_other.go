//go:build !(esp32c3 || esp32c6 || esp32s3 || nrf52 || nrf52833 || nrf52840 || rp2040 || rp2350)

package board

import (
	"errors"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinyplayer/flashdisk"
)

func Output() (tinyplayer.Output, error) {
	return nil, errors.New("board: no I2S output for this target")
}

func Disk() (*flashdisk.Disk, error) {
	return nil, errors.New("board: no flash disk for this target")
}
