package common

import (
	"sync"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

// Keep last hash of network message.
type DuplicateChecker struct {
	last crypto.Digest
	lock sync.Mutex
}

// Creates new DuplicateChecker.
func NewDuplicateChecker() *DuplicateChecker {
	return &DuplicateChecker{}
}

// Compares new bytes with previous, if equal message is now new.
func (a *DuplicateChecker) Add(b []byte) (isNew bool) {
	digest := crypto.MustFastHash(b)
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.last == digest {
		return false
	}
	a.last = digest
	return true
}
