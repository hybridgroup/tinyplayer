package wav

import (
	"encoding/binary"
	"io"
)

func (d *Decoder) readPCM(samples []int16) (int, error) {
	bps := d.format.BitsPerSample / 8
	align := d.format.BlockAlign
	n := 0
	for n < len(samples) && d.remaining >= uint32(align) {
		want := min((len(samples)-n)*bps, bufSize/align*align)
		if d.remaining < uint32(want) {
			want = int(d.remaining)
		}
		want = want / align * align
		got, err := io.ReadFull(d.r, d.buf[:want])
		got = got / align * align
		d.remaining -= uint32(got)
		b := d.buf[:got]
		if bps == 1 {
			for i, v := range b {
				samples[n+i] = int16(int(v)-128) << 8
			}
		} else {
			for i := 0; i < got; i += 2 {
				samples[n+i/2] = int16(binary.LittleEndian.Uint16(b[i:]))
			}
		}
		n += got / bps
		if err != nil {
			d.remaining = 0
		}
	}
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}
