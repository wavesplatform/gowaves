package ast

import (
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/types"
)

func newMapAssetInfo(scheme proto.Scheme, state types.SmartState, transaction proto.IIssueTransaction) (object, error) {
	obj := newObject()
	id, err := transaction.GetID()
	if err != nil {
		return nil, err
	}
	obj["id"] = NewBytes(id)
	obj["quantity"] = NewLong(int64(transaction.GetQuantity()))
	obj["decimals"] = NewLong(int64(transaction.GetDecimals()))
	addr, err := proto.NewAddressFromPublicKey(scheme, transaction.GetSenderPK())
	if err != nil {
		return nil, err
	}
	obj["issuer"] = NewAddressFromProtoAddress(addr)
	pk := transaction.GetSenderPK()
	obj["issuerPublicKey"] = NewBytes(pk.Bytes())
	obj["reissuable"] = NewBoolean(transaction.GetReissuable())
	obj["scripted"] = NewBoolean(transaction.NonEmptyScript())
	dId, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return nil, err
	}
	sponsored, err := state.NewestAssetIsSponsored(dId)
	if err != nil {
		return nil, err
	}
	obj["sponsored"] = NewBoolean(sponsored)
	return obj, nil
}
