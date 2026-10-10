// Package pwm plays mono audio on a speaker or piezo buzzer using a PWM pin.
package pwm

const minCarrier = 40000

// timing returns how many PWM periods each sample lasts and the PWM top value,
// so the carrier stays at minCarrier Hz or above.
func timing(clock, rate uint32) (repeat, top uint32) {
	repeat = max(1, (minCarrier+rate-1)/rate)
	top = clock/(rate*repeat) - 1
	return repeat, top
}

// fill mixes stereo frames to mono and writes each one repeat times as a duty
// from 0 to top+1, for both PWM channels. It returns the frames used.
func fill(dst, frames []uint32, repeat, top uint32) int {
	n := min(len(frames), len(dst)/int(repeat))
	i := 0
	for _, f := range frames[:n] {
		s := (int32(int16(f)) + int32(int16(f>>16))) >> 1
		d := uint32(s+32768) * (top + 1) >> 16
		d |= d << 16
		for range repeat {
			dst[i] = d
			i++
		}
	}
	return n
}
