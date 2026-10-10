//go:build rp2040 || rp2350

package pwm

import (
	"device/rp"
	"errors"
	"machine"
	"runtime"
	"runtime/volatile"
	"unsafe"
)

const bufWords = 1024

var (
	errPins = errors.New("pwm: second pin must be the other channel of the same PWM slice")
	errRate = errors.New("pwm: invalid sample rate")
)

type pwmGroup interface {
	Configure(machine.PWMConfig) error
	Channel(machine.Pin) (uint8, error)
	SetTop(uint32)
	SetInverting(uint8, bool)
	Enable(bool)
}

type dmaChannel struct {
	READ_ADDR   volatile.Register32
	WRITE_ADDR  volatile.Register32
	TRANS_COUNT volatile.Register32
	CTRL_TRIG   volatile.Register32
	_           [12]volatile.Register32
}

// PWM plays audio as a PWM duty on one pin, or on two pins in antiphase.
type PWM struct {
	group  pwmGroup
	cc     *volatile.Register32
	dma    *dmaChannel
	dmaIdx uint8
	dreq   uint32
	repeat uint32
	top    uint32
	bufs   [2][bufWords]uint32
	next   int
}

// New sets up PWM output on pin. If pinB is not machine.NoPin it must be the
// other channel of the same PWM slice and gets the inverted signal.
// dmaCh is the DMA channel to use, from 0 to 11.
func New(pin, pinB machine.Pin, dmaCh uint8) (*PWM, error) {
	slice, err := machine.PWMPeripheral(pin)
	if err != nil {
		return nil, err
	}
	groups := [...]pwmGroup{machine.PWM0, machine.PWM1, machine.PWM2, machine.PWM3,
		machine.PWM4, machine.PWM5, machine.PWM6, machine.PWM7}
	if int(slice) >= len(groups) || dmaCh > 11 {
		return nil, errors.ErrUnsupported
	}
	p := &PWM{
		group:  groups[slice],
		cc:     (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&rp.PWM.CH0_CC), 0x14*uintptr(slice))),
		dma:    &(*[12]dmaChannel)(unsafe.Pointer(rp.DMA))[dmaCh],
		dmaIdx: dmaCh,
		dreq:   rp.DREQ_PWM_WRAP0 + uint32(slice),
	}
	if err := p.group.Configure(machine.PWMConfig{Period: 1e9 / minCarrier}); err != nil {
		return nil, err
	}
	ch, err := p.group.Channel(pin)
	if err != nil {
		return nil, err
	}
	if pinB != machine.NoPin {
		sliceB, err := machine.PWMPeripheral(pinB)
		if err != nil || sliceB != slice {
			return nil, errPins
		}
		chB, err := p.group.Channel(pinB)
		if err != nil || chB == ch {
			return nil, errPins
		}
		p.group.SetInverting(chB, true)
	}
	if err := p.SetSampleFrequency(44100); err != nil {
		return nil, err
	}
	p.Enable(true)
	return p, nil
}

// SetSampleFrequency waits for queued audio to finish, then sets the rate.
func (p *PWM) SetSampleFrequency(freq uint32) error {
	if freq == 0 || freq > machine.CPUFrequency()/256 {
		return errRate
	}
	p.wait()
	p.repeat, p.top = timing(machine.CPUFrequency(), freq)
	p.group.SetTop(p.top)
	return nil
}

// WriteStereo mixes the frames to mono and returns once they are queued.
func (p *PWM) WriteStereo(b []uint32) (int, error) {
	for i := 0; i < len(b); {
		buf := p.bufs[p.next][:]
		n := fill(buf, b[i:], p.repeat, p.top)
		p.wait()
		p.start(buf[:n*int(p.repeat)])
		p.next ^= 1
		i += n
	}
	return len(b), nil
}

// Enable starts the output at mid scale, or stops it and drives the pins low.
func (p *PWM) Enable(enabled bool) {
	if !enabled {
		mask := uint32(1) << p.dmaIdx
		rp.DMA.CHAN_ABORT.Set(mask)
		for rp.DMA.CHAN_ABORT.Get()&mask != 0 {
		}
		p.cc.Set(0)
		p.group.Enable(false)
		return
	}
	mid := (p.top + 1) / 2
	p.cc.Set(mid | mid<<16)
	p.group.Enable(true)
}

func (p *PWM) wait() {
	for p.dma.CTRL_TRIG.Get()&rp.DMA_CH0_CTRL_TRIG_BUSY != 0 {
		runtime.Gosched()
	}
}

func (p *PWM) start(buf []uint32) {
	p.dma.READ_ADDR.Set(uint32(uintptr(unsafe.Pointer(&buf[0]))))
	p.dma.WRITE_ADDR.Set(uint32(uintptr(unsafe.Pointer(p.cc))))
	p.dma.TRANS_COUNT.Set(uint32(len(buf)))
	p.dma.CTRL_TRIG.Set(rp.DMA_CH0_CTRL_TRIG_EN |
		rp.DMA_CH0_CTRL_TRIG_INCR_READ |
		2<<rp.DMA_CH0_CTRL_TRIG_DATA_SIZE_Pos |
		uint32(p.dmaIdx)<<rp.DMA_CH0_CTRL_TRIG_CHAIN_TO_Pos |
		p.dreq<<rp.DMA_CH0_CTRL_TRIG_TREQ_SEL_Pos)
}
