package state

import (
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type patchesStorage struct {
	hs     *historyStorage
	scheme proto.Scheme
}

func newPatchesStorage(hs *historyStorage, scheme proto.Scheme) *patchesStorage {
	return &patchesStorage{hs: hs, scheme: scheme}
}

type patchRecord struct {
	fixSnapshots []proto.AtomicSnapshot
}

func (r *patchRecord) marshalBinary() ([]byte, error) {
	var tss g.TransactionStateSnapshot
	for _, s := range r.fixSnapshots {
		if err := s.AppendToProtobuf(&tss); err != nil {
			return nil, err
		}
	}
	return tss.MarshalVTStrict()
}

func (r *patchRecord) unmarshalBinary(scheme proto.Scheme, data []byte) error {
	var tss g.TransactionStateSnapshot
	if err := tss.UnmarshalVT(data); err != nil {
		return err
	}
	fixSnapshots, err := proto.TxSnapshotsFromProtobufWithoutTxStatus(scheme, &tss)
	if err != nil {
		return err
	}
	r.fixSnapshots = fixSnapshots
	return nil
}

func (ps *patchesStorage) savePatch(blockID proto.BlockID, fixSnapshots []proto.AtomicSnapshot) error {
	if len(fixSnapshots) == 0 { // Nothing to save.
		return nil
	}
	pr := patchRecord{fixSnapshots: fixSnapshots}
	data, err := pr.marshalBinary()
	if err != nil {
		return err
	}
	key := patchKey{blockID}
	return ps.hs.addNewEntry(patches, key.bytes(), data, blockID)
}

func (ps *patchesStorage) newestPatch(blockID proto.BlockID) ([]proto.AtomicSnapshot, error) {
	key := patchKey{blockID}
	data, err := ps.hs.newestTopEntryData(key.bytes())
	if err != nil {
		if isNotFoundInHistoryOrDBErr(err) { // No patches for this block.
			return nil, nil
		}
		return nil, err
	}
	var pr patchRecord
	if ubErr := pr.unmarshalBinary(ps.scheme, data); ubErr != nil {
		return nil, ubErr
	}
	return pr.fixSnapshots, nil
}
