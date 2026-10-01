//go:build nrf52 || nrf52833 || nrf52840

package board

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
)

func Output() (tinyplayer.Output, error) {
	err := machine.I2S0.Configure(machine.I2SConfig{
		SCK:            machine.P0_03,
		WS:             machine.P0_04,
		SDO:            machine.P0_28,
		SDI:            machine.NoPin,
		Mode:           machine.I2SModeSource,
		AudioFrequency: 44100,
		Stereo:         true,
	})
	return &machine.I2S0, err
}
