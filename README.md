# tinyplayer

<img src="images/tinyplayer-gopher.png" alt="tinyplayer gopher" width="300">

Play WAV audio on TinyGo devices using an I2S DAC such as the PCM5102 or MAX98357, or a speaker or piezo buzzer on a PWM pin.

- Sound effects, speech and music
- 8 and 16 bit PCM, and IMA ADPCM, mono or stereo, at any sample rate the I2S hardware supports
- Audio embedded in the program, or in WAV files on a FAT volume in flash
- On boards with USB, the flash volume shows up as a USB drive so you can copy WAV files to it

## Supported hardware

| Chip | I2S | PWM speaker | Flash files | USB drive |
|---|---|---|---|---|
| ESP32-C3 | machine.I2S0 | no | yes | no |
| ESP32-C6, ESP32-S3 | machine.I2S0 | no | no | no |
| nRF52840 | machine.I2S0 | no | yes | yes |
| nRF52832, nRF52833 | machine.I2S0 | no | yes | no |
| RP2040, RP2350 | [pio](https://github.com/tinygo-org/pio) piolib.I2S | pwm.PWM | yes | yes |

See [AUDIO.md](AUDIO.md) for how to connect an I2S DAC, a buzzer or a speaker.

## Usage

```go
out := &machine.I2S0
out.Configure(machine.I2SConfig{
	SCK:    machine.GPIO3,
	WS:     machine.GPIO4,
	SDO:    machine.GPIO5,
	SDI:    machine.NoPin,
	Mode:   machine.I2SModeSource,
	Stereo: true,
})

player := tinyplayer.New(out)
player.Play(strings.NewReader(sound))
```

`Play` takes any `io.Reader` with a WAV file in it. It sets the sample rate from the file and returns when all the audio is queued. Call `Stop` from another goroutine to end playback early, or `Pause` to pause and resume it. `Progress` returns the elapsed and total time. `SetVolume` takes 0 to 256.

To embed a sound use a string so it stays in flash:

```go
//go:embed beep.wav
var beep string
```

### Speaker or buzzer

On RP2040 and RP2350 the `pwm` package plays audio as a PWM duty cycle. It is mono and about 11 bits, with the PWM carrier at 40 kHz or above. DMA feeds the PWM so the CPU does no work per sample.

```go
out, _ := pwm.New(machine.GPIO2, machine.GPIO3, 11)
player := tinyplayer.New(out)
```

The second pin gets the inverted signal. It must be the other channel of the same PWM slice, so an even pin and the pin after it. Pass `machine.NoPin` to use one pin. The last argument is the DMA channel.

Wiring, with more detail and photos in [AUDIO.md](AUDIO.md):

- Passive piezo buzzer: connect it between the two pins. Driving both ends gives twice the swing of one pin.
- Small speaker: drive a logic level N-channel MOSFET or NPN transistor from one pin, with the speaker between the drain and 3.3V or 5V and a diode across the speaker. Never connect a speaker straight to a pin.
- Amplifier such as the PAM8302 or LM386: put a 1k resistor and 10nF capacitor low pass filter between one pin and the amp input.

Active buzzers, including the Grove Buzzer, have their own oscillator and only switch on and off. They can play simple tones but not WAV audio. Use a passive piezo, a speaker, or an amplifier such as the Grove Speaker.

### Files on flash

`flashdisk` makes a region of NOR flash usable as a disk with 512 byte sectors. It erases flash only when it has to. `storage` mounts a FAT volume on it using [github.com/soypat/fat](https://github.com/soypat/fat), and formats it the first time.

```go
disk, _ := flashdisk.Tail(machine.Flash, 1<<20)
vol, _, _ := storage.Mount(disk)
names, _ := vol.ListWAV(nil)

var f fat.File
vol.Open(&f, names[0])
player.Play(&f)
f.Close()
```

The same `disk` can be passed to `msc.Port` to show it as a USB drive.

## Examples

Wiring for the examples. See [AUDIO.md](AUDIO.md) for details on each kind of audio hardware.

| Board | SCK/BCK | WS/LCK | SDO/DIN |
|---|---|---|---|
| ESP32 | GPIO3 | GPIO4 | GPIO5 |
| nRF52 | P0.03 | P0.04 | P0.28 |
| RP2040, RP2350 | GPIO3 | GPIO4 | GPIO2 |

For a speaker or buzzer on RP2040 or RP2350 add `-tags speaker` and connect it to GPIO2 and GPIO3.

```
tinygo flash -tags speaker -target=xiao-rp2350 ./examples/embed
```

- `examples/embed` plays sounds built into the program.

  ```
  tinygo flash -target=xiao-esp32c3 ./examples/embed
  ```

- `examples/flash` plays the WAV files on the flash volume. If there are none it copies the example sounds there first.

  ```
  tinygo flash -tags fat_noexfat -target=xiao-esp32c3 ./examples/flash
  ```

- `examples/msc` shows the flash volume as a USB drive. Copy WAV files to it and they play once the copy is done.

  ```
  tinygo flash -tags fat_noexfat -target=pico ./examples/msc
  ```

- `examples/tinypod` is an iPod classic style music player for a XIAO board on the Seeed XIAO expansion board. It plays WAV files from the microSD card and shows menus on the OLED. See [tinypod](#tinypod) below.

  ```
  tinygo flash -tags fat_noexfat -target=xiao-rp2350 ./examples/tinypod
  ```

The `fat_noexfat` tag leaves out exFAT support, which these small volumes do not need.

## tinypod

`examples/tinypod` runs on the xiao-esp32c3 and xiao-rp2350 plugged into the [Seeed XIAO expansion board](https://wiki.seeedstudio.com/Seeeduino-XIAO-Expansion-Board/). The board's microSD slot holds the music and its OLED shows the menus. Connect an I2S DAC to D0, D6 and D7 as shown in [AUDIO.md](AUDIO.md#tinypod).

Put WAV files on a FAT32 SDHC card. Each folder in `/Music` shows up as an album, and its subfolders are added to it. If there is no `/Music` folder the root of the card is used.

```
/Music/Album One/01 Song.wav
/Music/Album Two/Disc 1/01 Song.wav
```

Controls with an [Adafruit Mini I2C Gamepad QT](https://www.adafruit.com/product/5743) plugged into the Grove I2C port:

| Control | Menus | Now Playing |
|---|---|---|
| Joystick up/down | move | volume |
| Joystick left/right | back/open | previous/next song |
| A | open | play/pause |
| B or Select | back | back to menus |
| Start | play/pause | play/pause |
| X/Y | volume | volume |
| Expansion board button | back | back to menus |

With no gamepad the expansion board button does everything. Short press moves down, hold for half a second to open or play/pause, and hold for 1.5 seconds to go back.

16 bit stereo at 22.05 kHz and 44.1 kHz plays without gaps on both boards. tinypod reads each song ahead into a 32 KB buffer, so SD reads and screen updates do not interrupt the audio.

## Making WAV files

Mono IMA ADPCM at 16 kHz is a good choice for speech and effects. It uses about 8 KB per second.

```
ffmpeg -i in.mp3 -ac 1 -ar 16000 -c:a adpcm_ima_wav out.wav
```

16 bit PCM for the best quality:

```
ffmpeg -i in.mp3 -ac 2 -ar 44100 -c:a pcm_s16le out.wav
```

`examples/sounds/generate.sh` makes the example sounds.

## Known issues

- On ESP32-C3 the ROM flash routines refuse writes past 2 MiB, so the example volume ends there.
- ESP32-S3 has no `machine.Flash` yet.
- tinypod needs the `sd` read fix from [tinygo-org/drivers#905](https://github.com/tinygo-org/drivers/pull/905). The `sd` package reads several blocks at once correctly only on SDHC cards, so use a card of 4 GB or more.
- On RP2040 and RP2350, piolib I2S and SPI0 use the same DMA channel unless [tinygo-org/pio#65](https://github.com/tinygo-org/pio/pull/65) is applied. `go.mod` uses the branches of these PRs until they are merged.
- RP2040 and RP2350 need the DMA and sample rate fixes to `piolib.I2S` from [tinygo-org/pio#64](https://github.com/tinygo-org/pio/pull/64). `go.mod` uses the pio main branch until the next release.
