//go:build !baremetal

package flashdisk

import "sync"

type lock struct{ mu sync.Mutex }

func (l *lock) lock() struct{} {
	l.mu.Lock()
	return struct{}{}
}

func (l *lock) unlock(struct{}) { l.mu.Unlock() }
