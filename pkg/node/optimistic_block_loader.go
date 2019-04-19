package node

import (
	"github.com/go-errors/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	//"github.com/wavesplatform/gowaves/pkg/network/peer"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"sync"
)

type blockBytes []byte

type expectedBlocks struct {
	curPosition     int
	blockToPosition map[crypto.Signature]int
	lst             []blockBytes
	notify          chan blockBytes
	mu              sync.Mutex
}

func newExpectedBlocks(signatures []crypto.Signature, notify chan blockBytes) *expectedBlocks {
	blockToPosition := make(map[crypto.Signature]int, len(signatures))

	for idx, value := range signatures {
		blockToPosition[value] = idx
	}

	return &expectedBlocks{
		blockToPosition: blockToPosition,
		curPosition:     0,
		lst:             make([]blockBytes, len(signatures)),
		notify:          notify,
	}
}

func (a *expectedBlocks) add(block blockBytes) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, err := proto.BlockGetSignature(block)
	if err != nil {
		return err
	}

	n, ok := a.blockToPosition[s]
	if !ok {
		return errors.Errorf("unexpected block sig %s", s)
	}

	a.lst[n] = block

	for a.curPosition < len(a.lst) {
		if a.lst[a.curPosition] == nil {
			break
		}
		a.notify <- a.lst[a.curPosition]
		a.curPosition += 1
	}

	return nil
}

func (a *expectedBlocks) hasNext() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.curPosition < len(a.lst)
}
