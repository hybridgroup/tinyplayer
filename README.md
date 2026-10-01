# tinyplayer

Play WAV audio on TinyGo devices using an I2S DAC such as the PCM5102 or MAX98357.

- Sound effects, speech and music
- 8 and 16 bit PCM, and IMA ADPCM, mono or stereo, at any sample rate the I2S hardware supports
- Audio embedded in the program, or in WAV files on a FAT volume in flash
- On boards with USB, the flash volume shows up as a USB drive so you can copy WAV files to it

## Supported hardware

| Chip | I2S | Flash files | USB drive |
|---|---|---|---|
| ESP32-C3 | machine.I2S0 | yes | no |
| ESP32-C6, ESP32-S3 | machine.I2S0 | no | no |
| nRF52840 | machine.I2S0 | yes | yes |
| nRF52832, nRF52833 | machine.I2S0 | yes | no |
| RP2040, RP2350 | [pio](https://github.com/tinygo-org/pio) piolib.I2S | yes | yes |

The ESP32 and nRF52 I2S drivers are in the TinyGo dev branch.

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

`Play` takes any `io.Reader` with a WAV file in it. It sets the sample rate from the file and returns when all the audio is queued. Call `Stop` from another goroutine to end playback early. `SetVolume` takes 0 to 256.

To embed a sound use a string so it stays in flash:

```go
//go:embed beep.wav
var beep string
```

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

Wiring for the examples:

| Board | SCK/BCK | WS/LCK | SDO/DIN |
|---|---|---|---|
| ESP32 | GPIO3 | GPIO4 | GPIO5 |
| nRF52 | P0.03 | P0.04 | P0.28 |
| RP2040, RP2350 | GPIO3 | GPIO4 | GPIO2 |

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

The `fat_noexfat` tag leaves out exFAT support, which these small volumes do not need.

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
- RP2040 and RP2350 need the DMA and sample rate fixes to `piolib.I2S` from [tinygo-org/pio#64](https://github.com/tinygo-org/pio/pull/64). `go.mod` uses the pio main branch until the next release.
