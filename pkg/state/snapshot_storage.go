package state

//
//import (
//	"errors"
//
//	"github.com/elliotchance/orderedmap/v2"
//	"github.com/wavesplatform/gowaves/pkg/crypto"
//	"github.com/wavesplatform/gowaves/pkg/proto"
//)
//
//type rollbackMap[K comparable, V any] struct {
//	mapData      map[K]V
//	rollbackActs []func()
//}
//
//func newRollbackMap[K comparable, V any]() rollbackMap[K, V] {
//	return rollbackMap[K, V]{
//		mapData:      make(map[K]V),
//		rollbackActs: nil,
//	}
//}
//
//func (m rollbackMap[K, V]) Set(key K, newV V) {
//	if oldV, ok := m.mapData[key]; ok {
//		m.rollbackActs = append(m.rollbackActs, func() {
//			m.mapData[key] = oldV
//		})
//	} else {
//		m.rollbackActs = append(m.rollbackActs, func() {
//			delete(m.mapData, key)
//		})
//	}
//	m.mapData[key] = newV
//}
//
//func (m rollbackMap[K, V]) Get(key K) (V, bool) {
//	value, ok := m.mapData[key]
//	return value, ok
//}
//
//func (m rollbackMap[K, V]) Rollback() {
//	l := len(m.rollbackActs)
//	if l == 0 {
//		return
//	}
//	fn := m.rollbackActs[l-1]
//	m.rollbackActs = m.rollbackActs[:l-1]
//	fn()
//}
//
////type historyMap[K comparable, V any] struct {
////	historyData map[K][]V
////}
////
////func newHistoryMap[K comparable, V any]() historyMap[K, V] {
////	return historyMap[K, V]{
////		historyData: make(map[K][]V),
////	}
////}
////
////func (m historyMap[K, V]) Set(key K, value V) {
////	m.historyData[key] = append(m.historyData[key], value)
////}
////
////func (m historyMap[K, V]) Get(key K) (value V, ok bool) {
////	data := m.historyData[key]
////	if len(data) == 0 {
////		return value, false
////	}
////	return data[len(data)-1], true
////}
////
////func (m historyMap[K, V]) Rollback(key K) bool {
////	data := m.historyData[key]
////	if len(data) == 0 {
////		return false
////	}
////	m.historyData[key] = data[:len(data)-1]
////	return true
////}
//
//type orderMap[K comparable, V any] struct {
//	*orderedmap.OrderedMap[K, V]
//}
//
//func newOrderMap[K comparable, V any]() orderMap[K, V] {
//	return orderMap[K, V]{orderedmap.NewOrderedMap[K, V]()}
//}
//
//func (o orderMap[K, V]) PopBack() (key K, value V, ok bool) {
//	elem := o.Back()
//	if elem == nil {
//		return key, value, false
//	}
//	popped := o.Delete(elem.Key)
//	return elem.Key, elem.Value, popped
//}
//
//type rangeIdx struct{ since, till int }
//
//type leasesSnapshotStorage struct {
//	leaseStatesByID      rollbackMap[crypto.Digest, *leasing]
//	blockIDToLeaseStates orderMap[proto.BlockID, []*leasing]
//}
//
//func (l *leasesSnapshotStorage) RollbackInMem(depth int) (rem int, err error) {
//	stateRollbackDepth := l.blockIDToLeaseStates.Len() - depth
//	if stateRollbackDepth <= 0 {
//		l.leaseStatesByID = newRollbackMap[crypto.Digest, *leasing]()
//		l.blockIDToLeaseStates = newOrderMap[proto.BlockID, []*leasing]()
//		return -1 * stateRollbackDepth, nil
//	}
//	for i := 0; i < stateRollbackDepth; i++ {
//		_, snapshots, ok := l.blockIDToLeaseStates.PopBack()
//		if !ok {
//			return 0, errors.New("failed to pop back leasesSnapshotStorage")
//		}
//		for range snapshots { // call rollback len(snapshots) times
//			l.leaseStatesByID.Rollback()
//		}
//	}
//	return stateRollbackDepth, nil
//}
