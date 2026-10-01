//go:build baremetal

package flashdisk

import "runtime/interrupt"

// USB MSC reads the disk from its interrupt handler, see TinyGo
// src/machine/usb/msc/scsi_readwrite.go readBlock, so a mutex could deadlock.
type lock struct{}

func (lock) lock() interrupt.State        { return interrupt.Disable() }
func (lock) unlock(state interrupt.State) { interrupt.Restore(state) }
