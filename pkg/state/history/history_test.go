package history

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	rollbackMax  = 100
	rollbackEdge = 1000
	totalBlocks  = 2000
)

type mockHeightInfo struct {
	blockIDToHeight map[crypto.Signature]uint64
}

func newMockHeightInfo() (*mockHeightInfo, error) {
	return &mockHeightInfo{blockIDToHeight: make(map[crypto.Signature]uint64)}, nil
}

func (m *mockHeightInfo) IsValidBlock(blockID crypto.Signature) (bool, error) {
	height, ok := m.blockIDToHeight[blockID]
	if !ok {
		return false, errors.New("ID not found")
	}
	return height <= rollbackEdge, nil
}

func (m *mockHeightInfo) Height() (uint64, error) {
	return uint64(len(m.blockIDToHeight)), nil
}

func (m *mockHeightInfo) BlockIDToHeight(blockID crypto.Signature) (uint64, error) {
	height, ok := m.blockIDToHeight[blockID]
	if !ok {
		return 0, errors.New("ID not found")
	}
	return height, nil
}

func (m *mockHeightInfo) RollbackMax() uint64 {
	return rollbackMax
}

func (m *mockHeightInfo) addBlock(blockID crypto.Signature) error {
	m.blockIDToHeight[blockID] = uint64(len(m.blockIDToHeight) + 1)
	return nil
}

func TestAddRecord(t *testing.T) {
	hinfo, err := newMockHeightInfo()
	if err != nil {
		t.Fatalf("newMockHeightInfo(): %v\n", err)
	}
	blockID := make([]byte, crypto.SignatureSize)
	_, err = rand.Read(blockID)
	if err != nil {
		t.Fatalf("rand.Read(): %v\n", err)
	}
	hfmt, err := NewHistoryFormatter(crypto.SignatureSize+1, crypto.SignatureSize, hinfo, hinfo)
	if err != nil {
		t.Fatalf("NewHistoryFormatter(): %v\n", err)
	}
	var history []byte
	// Test record rewrite.
	firstRecord := append([]byte{0}, blockID...)
	history, err = hfmt.AddRecord(history, firstRecord)
	if err != nil {
		t.Fatalf("AddRecord(): %v\n", err)
	}
	secondRecord := append([]byte{1}, blockID...)
	history, err = hfmt.AddRecord(history, secondRecord)
	if err != nil {
		t.Fatalf("AddRecord(): %v\n", err)
	}
	if !bytes.Equal(history, secondRecord) {
		t.Errorf("History formatter did not rewrite record with same ID.")
	}
	// Test record append.
	_, err = rand.Read(blockID)
	if err != nil {
		t.Fatalf("rand.Read(): %v\n", err)
	}
	thirdRecord := append([]byte{2}, blockID...)
	history, err = hfmt.AddRecord(history, thirdRecord)
	if err != nil {
		t.Fatalf("AddRecord(): %v\n", err)
	}
	if !bytes.Equal(history, append(secondRecord, thirdRecord...)) {
		t.Errorf("History formatter did not append record with new ID.")
	}
}

func TestNormalize(t *testing.T) {
	hinfo, err := newMockHeightInfo()
	if err != nil {
		t.Fatalf("newMockHeightInfo(): %v\n", err)
	}
	hfmt, err := NewHistoryFormatter(crypto.SignatureSize, crypto.SignatureSize, hinfo, hinfo)
	if err != nil {
		t.Fatalf("NewHistoryFormatter(): %v\n", err)
	}
	var history []byte
	for i := 0; i < totalBlocks; i++ {
		blockIDBytes := make([]byte, crypto.SignatureSize)
		_, err = rand.Read(blockIDBytes)
		if err != nil {
			t.Fatalf("rand.Read(): %v\n", err)
		}
		blockID, err := crypto.NewSignatureFromBytes(blockIDBytes)
		if err != nil {
			t.Fatalf("NewSignatureFromBytes(): %v\n", err)
		}
		if err := hinfo.addBlock(blockID); err != nil {
			t.Fatalf("addBlock(): %v\n", err)
		}
		history, err = hfmt.AddRecord(history, blockID[:])
		if err != nil {
			t.Fatalf("AddRecord(): %v\n", err)
		}
	}
	history, err = hfmt.Normalize(history)
	if err != nil {
		t.Fatalf("Normalize(): %v\n", err)
	}
	height, err := hinfo.Height()
	if err != nil {
		t.Fatalf("Height(): %v\n", err)
	}
	oldRecordNumber := 0
	for i := 0; i <= len(history)-crypto.SignatureSize; i += crypto.SignatureSize {
		record := history[i : i+crypto.SignatureSize]
		blockID, err := crypto.NewSignatureFromBytes(record)
		if err != nil {
			t.Fatalf("NewSignatureFromBytes(): %v\n", err)
		}
		recordHeight, err := hinfo.BlockIDToHeight(blockID)
		if err != nil {
			t.Fatalf("BlockIDToHeight(): %v\n", err)
		}
		if recordHeight < height-rollbackMax {
			oldRecordNumber++
		}
		if recordHeight > rollbackEdge {
			t.Errorf("History formatter did not erase invalid blocks.")
		}
	}
	if oldRecordNumber > 1 {
		t.Errorf("History formatter did not cut old blocks.")
	}
}
