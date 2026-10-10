// Package sounds has the example WAV files. Run generate.sh to make them again.
package sounds

import _ "embed"

var (
	//go:embed beep.wav
	Beep string
	//go:embed click.wav
	Click string
	//go:embed speech.wav
	Speech string
	//go:embed speech_ja.wav
	SpeechJA string
	//go:embed music.wav
	Music string
)

// All lists the sounds with file names.
var All = []struct {
	Name string
	Data string
}{
	{"beep.wav", Beep},
	{"click.wav", Click},
	{"speech.wav", Speech},
	{"speech_ja.wav", SpeechJA},
	{"music.wav", Music},
}
