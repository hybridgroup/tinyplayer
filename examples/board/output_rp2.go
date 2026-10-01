//go:build rp2040 || rp2350

package board

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
)

// Output uses DIN on GPIO2, BCK on GPIO3 and LCK on GPIO4.
func Output() (tinyplayer.Output, error) {
	sm, err := pio.PIO0.ClaimStateMachine()
	if err != nil {
		return nil, err
	}
	i2s, err := piolib.NewI2S(sm, machine.GPIO2, machine.GPIO3)
	if err != nil {
		return nil, err
	}
	return i2s, i2s.SetSampleFrequency(44100)
}
