//go:build esp32c3 || esp32c6 || esp32s3

package board

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
)

func Output() (tinyplayer.Output, error) {
	err := machine.I2S0.Configure(machine.I2SConfig{
		SCK:            machine.GPIO3,
		WS:             machine.GPIO4,
		SDO:            machine.GPIO5,
		SDI:            machine.NoPin,
		Mode:           machine.I2SModeSource,
		AudioFrequency: 44100,
		Stereo:         true,
	})
	return &machine.I2S0, err
}
