package ride

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	_ "github.com/wavesplatform/gowaves/pkg/proto"
)

type diffDataEntry struct {
	diffInteger []proto.IntegerDataEntry
	diffBool    []proto.BooleanDataEntry
	diffString  []proto.StringDataEntry
	diffBinary  []proto.BinaryDataEntry
}

type diffState struct {
	diffDataEntr diffDataEntry
}

func (diffSt *diffState) getIntFromDataEntryByKey(key string) *proto.IntegerDataEntry {
	for _, intDataEntry := range diffSt.diffDataEntr.diffInteger {
		if key == intDataEntry.Key {
			return &intDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) getBoolFromDataEntryByKey(key string) *proto.BooleanDataEntry {
	for _, boolDataEntry := range diffSt.diffDataEntr.diffBool {
		if key == boolDataEntry.Key {
			return &boolDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) getStringFromDataEntryByKey(key string) *proto.StringDataEntry {
	for _, stringDataEntry := range diffSt.diffDataEntr.diffString {
		if key == stringDataEntry.Key {
			return &stringDataEntry
		}
	}
	return nil
}

func (diffSt *diffState) getBinaryFromDataEntryByKey(key string) *proto.BinaryDataEntry {
	for _, binaryDataEntry := range diffSt.diffDataEntr.diffBinary {
		if key == binaryDataEntry.Key {
			return &binaryDataEntry
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
