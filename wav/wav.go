// Package wav decodes WAV files with 8 or 16 bit PCM or IMA ADPCM audio.
package wav

import (
	"encoding/binary"
	"errors"
	"io"
)

type Codec uint16

const (
	CodecPCM        Codec = 0x0001
	CodecIMAADPCM   Codec = 0x0011
	codecExtensible Codec = 0xFFFE
)

var (
	ErrNotWAV      = errors.New("wav: not a WAV file")
	ErrUnsupported = errors.New("wav: unsupported format")
	ErrNoData      = errors.New("wav: missing data chunk")
)

// Format describes the audio in a WAV file.
type Format struct {
	Codec           Codec
	Channels        int
	SampleRate      uint32
	BitsPerSample   int
	BlockAlign      int
	SamplesPerBlock int
}

const (
	bufSize       = 512
	maxADPCMBlock = 4096
	maxChannels   = 2
)

// Decoder reads samples from a WAV stream. The zero value is ready for Reset.
type Decoder struct {
	r         io.Reader
	format    Format
	remaining uint32
	buf       [bufSize]byte

	block    []byte
	decoded  []int16
	pos, end int
}

// NewDecoder reads the WAV header from r.
func NewDecoder(r io.Reader) (*Decoder, error) {
	d := &Decoder{}
	if err := d.Reset(r); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Decoder) Format() Format {
	return d.format
}

// Reset reads a new WAV header from r so the decoder can be reused.
func (d *Decoder) Reset(r io.Reader) error {
	d.r = r
	d.format = Format{}
	d.remaining = 0
	d.pos, d.end = 0, 0

	hdr := d.buf[:12]
	if _, err := io.ReadFull(r, hdr); err != nil {
		return ErrNotWAV
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return ErrNotWAV
	}

	haveFmt := false
	for {
		ch := d.buf[:8]
		if _, err := io.ReadFull(r, ch); err != nil {
			return ErrNoData
		}
		id := string(ch[0:4])
		size := binary.LittleEndian.Uint32(ch[4:8])
		switch id {
		case "fmt ":
			if err := d.readFmt(size); err != nil {
				return err
			}
			haveFmt = true
		case "data":
			if !haveFmt {
				return ErrNotWAV
			}
			d.remaining = size
			return d.setup()
		default:
			if err := d.skip(int64(size) + int64(size&1)); err != nil {
				return ErrNoData
			}
		}
	}
}

func (d *Decoder) readFmt(size uint32) error {
	if size < 16 || size > bufSize {
		return ErrNotWAV
	}
	n := int(size + size&1)
	b := d.buf[:n]
	if _, err := io.ReadFull(d.r, b); err != nil {
		return ErrNotWAV
	}
	f := Format{
		Codec:         Codec(binary.LittleEndian.Uint16(b[0:2])),
		Channels:      int(binary.LittleEndian.Uint16(b[2:4])),
		SampleRate:    binary.LittleEndian.Uint32(b[4:8]),
		BlockAlign:    int(binary.LittleEndian.Uint16(b[12:14])),
		BitsPerSample: int(binary.LittleEndian.Uint16(b[14:16])),
	}
	// WAVE_FORMAT_EXTENSIBLE keeps the codec in the SubFormat GUID at offset 24.
	// https://learn.microsoft.com/en-us/windows/win32/api/mmreg/ns-mmreg-waveformatextensible
	if f.Codec == codecExtensible {
		if size < 26 {
			return ErrNotWAV
		}
		f.Codec = Codec(binary.LittleEndian.Uint16(b[24:26]))
	}
	if f.Codec == CodecIMAADPCM && size >= 20 {
		f.SamplesPerBlock = int(binary.LittleEndian.Uint16(b[18:20]))
	}
	d.format = f
	return nil
}

func (d *Decoder) setup() error {
	f := &d.format
	if f.Channels < 1 || f.Channels > maxChannels || f.SampleRate == 0 {
		return ErrUnsupported
	}
	switch f.Codec {
	case CodecPCM:
		if f.BitsPerSample != 8 && f.BitsPerSample != 16 {
			return ErrUnsupported
		}
		if f.BlockAlign != f.Channels*f.BitsPerSample/8 {
			return ErrUnsupported
		}
		f.SamplesPerBlock = 1
	case CodecIMAADPCM:
		if f.BitsPerSample != 4 || f.BlockAlign <= 4*f.Channels || f.BlockAlign > maxADPCMBlock {
			return ErrUnsupported
		}
		spb := (f.BlockAlign-4*f.Channels)*2/f.Channels + 1
		if f.SamplesPerBlock == 0 {
			f.SamplesPerBlock = spb
		}
		if f.SamplesPerBlock > spb {
			return ErrUnsupported
		}
		if cap(d.block) < f.BlockAlign {
			d.block = make([]byte, f.BlockAlign)
		}
		if n := f.SamplesPerBlock * f.Channels; cap(d.decoded) < n {
			d.decoded = make([]int16, n)
		}
	default:
		return ErrUnsupported
	}
	return nil
}

func (d *Decoder) skip(n int64) error {
	if s, ok := d.r.(io.Seeker); ok {
		_, err := s.Seek(n, io.SeekCurrent)
		return err
	}
	for n > 0 {
		m := min(n, bufSize)
		if _, err := io.ReadFull(d.r, d.buf[:m]); err != nil {
			return err
		}
		n -= m
	}
	return nil
}

// Read fills samples with interleaved 16 bit samples and returns how many it
// wrote. The count is always a multiple of the channel count.
func (d *Decoder) Read(samples []int16) (int, error) {
	ch := d.format.Channels
	if ch == 0 {
		return 0, ErrNoData
	}
	samples = samples[:len(samples)/ch*ch]
	if d.format.Codec == CodecIMAADPCM {
		return d.readADPCM(samples)
	}
	return d.readPCM(samples)
}
