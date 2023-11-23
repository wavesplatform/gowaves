package proto

import (
	"encoding/binary"
	"github.com/pkg/errors"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
)

type BlockSnapshot struct {
	TxSnapshots [][]AtomicSnapshot
}

func (bs BlockSnapshot) MarshallBinary() ([]byte, error) {
	result := []byte{}
	var txSnCnt [4]byte
	binary.BigEndian.PutUint32(txSnCnt[0:4], uint32(len(bs.TxSnapshots)))
	result = append(result, txSnCnt[:]...)
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
		var bytesLen [4]byte
		binary.BigEndian.PutUint32(bytesLen[0:4], uint32(len(tsBytes)))
		result = append(result, bytesLen[:]...)
		result = append(result, tsBytes...)
	}
	return result, nil
}

func (bs *BlockSnapshot) UnmarshalBinary(data []byte, scheme Scheme) error {
	txSnCnt := binary.BigEndian.Uint32(data[0:4])
	data = data[4:]
	for i := uint32(0); i <= txSnCnt; i++ {
		tsBytesLen := binary.BigEndian.Uint32(data[0:4])
		var tsProto *g.TransactionStateSnapshot
		err := tsProto.UnmarshalVT(data[0:tsBytesLen])
		if err != nil {
			return err
		}
		atomicTs, err := TxSnapshotsFromProtobuf(scheme, tsProto)
		if err != nil {
			return err
		}
		bs.TxSnapshots = append(bs.TxSnapshots, atomicTs)
		data = data[tsBytesLen:]
	}
	return nil
}
