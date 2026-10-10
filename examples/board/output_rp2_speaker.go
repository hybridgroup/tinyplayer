//go:build (rp2040 || rp2350) && speaker

package board

import (
	"machine"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinyplayer/pwm"
)

// Output drives a speaker on GPIO2, with the inverted signal on GPIO3.
func Output() (tinyplayer.Output, error) {
	return pwm.New(machine.GPIO2, machine.GPIO3, 11)
}
