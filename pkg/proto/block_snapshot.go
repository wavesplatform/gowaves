package proto

import (
	"encoding/binary"

	"github.com/pkg/errors"

	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type BlockSnapshot struct {
	TxSnapshots [][]AtomicSnapshot
}

func (bs *BlockSnapshot) AppendTxSnapshot(txSnapshot []AtomicSnapshot) {
	bs.TxSnapshots = append(bs.TxSnapshots, txSnapshot)
}

func (bs BlockSnapshot) MarshallBinary() ([]byte, error) {
	result := binary.BigEndian.AppendUint32([]byte{}, uint32(len(bs.TxSnapshots)))
	for _, ts := range bs.TxSnapshots {
		var res g.TransactionStateSnapshot
		for _, atomicSnapshot := range ts {
			if err := atomicSnapshot.AppendToProtobuf(&res); err != nil {
				return nil, errors.Wrap(err, "failed to marshall TransactionSnapshot to proto")
			}
		}
		tsBytes, err := res.MarshalVTStrict()
		if err != nil {
			return nil, err
		}
		result = binary.BigEndian.AppendUint32(result, uint32(len(tsBytes)))
		result = append(result, tsBytes...)
	}
	return result, nil
}

func (bs *BlockSnapshot) UnmarshalBinary(data []byte, scheme Scheme) error {
	if len(data) < uint32Size {
		return errors.Errorf("BlockSnapshot UnmarshallBinary: invalid data size")
	}
	txSnCnt := binary.BigEndian.Uint32(data[0:uint32Size])
	data = data[uint32Size:]
	var txSnapshots [][]AtomicSnapshot
	for i := uint32(0); i < txSnCnt; i++ {
		if len(data) < uint32Size {
			return errors.Errorf("BlockSnapshot UnmarshallBinary: invalid data size")
		}
		tsBytesLen := binary.BigEndian.Uint32(data[0:uint32Size])
		var tsProto g.TransactionStateSnapshot
		data = data[uint32Size:]
		if uint32(len(data)) < tsBytesLen {
			return errors.Errorf("BlockSnapshot UnmarshallBinary: invalid snapshot size")
		}
		err := tsProto.UnmarshalVT(data[0:tsBytesLen])
		if err != nil {
			return err
		}
		atomicTS, err := TxSnapshotsFromProtobuf(scheme, &tsProto)
		if err != nil {
			return err
		}
		txSnapshots = append(txSnapshots, atomicTS)
		data = data[tsBytesLen:]
	}
	bs.TxSnapshots = txSnapshots
	return nil
}
