package ride

import (
	"bytes"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	_ "github.com/wavesplatform/gowaves/pkg/proto"
)

type diffDataEntry struct {
	diffInteger []proto.IntegerDataEntry
	diffBool    []proto.BooleanDataEntry
	diffString  []proto.StringDataEntry
	diffBinary  []proto.BinaryDataEntry
}

type diffBalance struct {
	account proto.Recipient
	assetID crypto.Digest
	amount  int64
}

type diffWavesBalance struct {
	account    proto.Recipient
	regular    int64
	generating int64
	available  int64
	effective  int64
}

type diffState struct {
	dataEntry    diffDataEntry
	balance      []diffBalance
	wavesBalance []diffWavesBalance
}

func (diffSt *diffState) findIntFromDataEntryByKey(key string) *proto.IntegerDataEntry {
	for _, intDataEntry := range diffSt.dataEntry.diffInteger {
		if key == intDataEntry.Key {
			return &intDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findBoolFromDataEntryByKey(key string) *proto.BooleanDataEntry {
	for _, boolDataEntry := range diffSt.dataEntry.diffBool {
		if key == boolDataEntry.Key {
			return &boolDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findStringFromDataEntryByKey(key string) *proto.StringDataEntry {
	for _, stringDataEntry := range diffSt.dataEntry.diffString {
		if key == stringDataEntry.Key {
			return &stringDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findBinaryFromDataEntryByKey(key string) *proto.BinaryDataEntry {
	for _, binaryDataEntry := range diffSt.dataEntry.diffBinary {
		if key == binaryDataEntry.Key {
			return &binaryDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) findWavesBalance(account proto.Recipient) *diffWavesBalance {
	for _, v := range diffSt.wavesBalance {
		if v.account == account {
			return &v
		}
	}
	return nil
}

func (diffSt *diffState) findBalance(account proto.Recipient, asset []byte) *diffBalance {
	for _, v := range diffSt.balance {
		if v.account == account && bytes.Equal(v.assetID.Bytes(), asset) {
			return &v
		}
	}
	return nil
}

//func newDiffState() *diffState {
//	diff := make(map[string]int64)
//
//	return &diffState{diff}
//}
//
//func (s *diffState) saveDiff(newDiff map[string]int64) error {
//	for key, balanceDiff := range newDiff {
//		if _, found := s.diff[key]; found {
//			// If diffState already has changes for this key, we summarize changes
//			s.diff[key] = s.diff[key] + balanceDiff
//			continue
//		}
//		// We don't have any changes for this key yet.
//		s.diff[key] = balanceDiff
//	}
//	return nil
//}
//
//
//func (s *diffState) findDiffByKey(key string) int64 {
//		if val, found := s.diff[key]; found {
//			return val
//		}
//		return 0
//}
