package ast

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func newMapFromBlockHeader(scheme proto.Scheme, h *proto.BlockHeader) (object, error) {
	obj := newObject()
	obj["timestamp"] = NewLong(int64(h.Timestamp))
	obj["version"] = NewLong(int64(h.Version))
	obj["reference"] = NewBytes(common.Dup(h.Parent.Bytes()))
	addr, err := proto.NewAddressFromPublicKey(scheme, h.GenPublicKey)
	if err != nil {
		return nil, err
	}
	obj["generator"] = NewAddressFromProtoAddress(addr)
	obj["generatorPublicKey"] = NewBytes(common.Dup(h.GenPublicKey.Bytes()))
	obj["signature"] = NewBytes(common.Dup(h.BlockSignature.Bytes()))
	obj["baseTarget"] = NewLong(int64(h.BaseTarget))
	obj["generationSignature"] = NewBytes(common.Dup(h.GenSignature.Bytes()))
	obj["transactionCount"] = NewLong(int64(h.TransactionCount))
	obj["featureVotes"] = makeFeatures(h.Features)
	return obj, nil
}
