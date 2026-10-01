package wav

import (
	"encoding/binary"
	"io"
)

// IMA ADPCM tables from the IMA Digital Audio Focus and Technical Working
// Groups, "Recommended Practices for Enhancing Digital Audio Compatibility", 1992.
var imaIndexTable = [16]int8{-1, -1, -1, -1, 2, 4, 6, 8, -1, -1, -1, -1, 2, 4, 6, 8}

var imaStepTable = [89]int16{
	7, 8, 9, 10, 11, 12, 13, 14, 16, 17, 19, 21, 23, 25, 28, 31,
	34, 37, 41, 45, 50, 55, 60, 66, 73, 80, 88, 97, 107, 118, 130, 143,
	157, 173, 190, 209, 230, 253, 279, 307, 337, 371, 408, 449, 494, 544, 598, 658,
	724, 796, 876, 963, 1060, 1166, 1282, 1411, 1552, 1707, 1878, 2066, 2272, 2499, 2749, 3024,
	3327, 3660, 4026, 4428, 4871, 5358, 5894, 6484, 7132, 7845, 8630, 9493, 10442, 11487, 12635, 13899,
	15289, 16818, 18500, 20350, 22385, 24623, 27086, 29794, 32767,
}

func (d *Decoder) readADPCM(samples []int16) (int, error) {
	n := 0
	for n < len(samples) {
		if d.pos == d.end && !d.decodeBlock() {
			break
		}
		c := copy(samples[n:], d.decoded[d.pos:d.end])
		n += c
		d.pos += c
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

// decodeBlock decodes one block laid out as in Microsoft Multimedia Standards
// Update, "IMA ADPCM Wave Type", 1994. The last block may be short.
func (d *Decoder) decodeBlock() bool {
	ch := d.format.Channels
	size := d.format.BlockAlign
	if d.remaining < uint32(size) {
		size = int(d.remaining)
	}
	if size <= 4*ch {
		d.remaining = 0
		return false
	}
	got, _ := io.ReadFull(d.r, d.block[:size])
	d.remaining -= uint32(size)
	if got < size {
		d.remaining = 0
		size = got
		if size <= 4*ch {
			return false
		}
	}
	b := d.block[:size]

	var pred, idx [maxChannels]int32
	for c := 0; c < ch; c++ {
		h := b[c*4:]
		pred[c] = int32(int16(binary.LittleEndian.Uint16(h)))
		idx[c] = min(int32(h[2]), 88)
		d.decoded[c] = int16(pred[c])
	}

	groups := (size - 4*ch) / (4 * ch)
	frames := min(1+groups*8, d.format.SamplesPerBlock)
	data := b[4*ch:]
	for g := 0; g < groups; g++ {
		for c := 0; c < ch; c++ {
			chunk := data[(g*ch+c)*4:]
			p, i := pred[c], idx[c]
			for k := 0; k < 8; k++ {
				fi := 1 + g*8 + k
				if fi >= frames {
					break
				}
				nib := chunk[k/2] >> (4 * (k & 1)) & 0xF
				// Rounded form of the IMA shift and add steps, matching
				// FFmpeg libavcodec/adpcm.c adpcm_ima_wav_expand_nibble.
				diff := (2*int32(nib&7) + 1) * int32(imaStepTable[i]) >> 3
				if nib&8 != 0 {
					p -= diff
				} else {
					p += diff
				}
				p = max(-32768, min(32767, p))
				i = max(0, min(88, i+int32(imaIndexTable[nib])))
				d.decoded[fi*ch+c] = int16(p)
			}
			pred[c], idx[c] = p, i
		}
	}
	d.pos = 0
	d.end = frames * ch
	return true
}
