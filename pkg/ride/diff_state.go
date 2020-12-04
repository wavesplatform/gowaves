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
	diffDelete  []proto.DeleteDataEntry
}

type diffState struct {
	diffDataEntr diffDataEntry
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
