// Plays WAV files that are embedded in the program.
package main

import (
	"strings"
	"time"

	"github.com/hybridgroup/tinyplayer"
	"github.com/hybridgroup/tinyplayer/examples/board"
	"github.com/hybridgroup/tinyplayer/examples/sounds"
)

func main() {
	time.Sleep(2 * time.Second)

	out, err := board.Output()
	if err != nil {
		println("could not configure I2S:", err.Error())
		return
	}
	player := tinyplayer.New(out)

	var r strings.Reader
	for {
		for _, s := range sounds.All {
			println("playing", s.Name)
			r.Reset(s.Data)
			if err := player.Play(&r); err != nil {
				println("error:", err.Error())
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
