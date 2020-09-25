package fride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func transactionToObject(scheme byte, tx proto.Transaction) (rideObject, error) {
	return nil, errors.New("not implemented")
}

func assetInfoToObject(info proto.AssetInfo) rideObject {
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("Asset")
	obj["id"] = rideBytes(info.ID.Bytes())
	obj["quantity"] = rideInt(info.Quantity)
	obj["decimals"] = rideInt(info.Decimals)
	obj["issuer"] = rideAddress(info.Issuer)
	obj["issuerPublicKey"] = rideBytes(common.Dup(info.IssuerPublicKey.Bytes()))
	obj["reissuable"] = rideBoolean(info.Reissuable)
	obj["scripted"] = rideBoolean(info.Scripted)
	obj["sponsored"] = rideBoolean(info.Sponsored)
	return obj
}

func fullAssetInfoToObject(info proto.FullAssetInfo) rideObject {
	obj := assetInfoToObject(info.AssetInfo)
	obj["name"] = rideString(info.Name)
	obj["description"] = rideString(info.Description)
	return obj
}

func blockHeaderToObject(scheme byte, header *proto.BlockHeader, height proto.Height) (rideObject, error) {
	return nil, errors.New("not implemented")
}

//func transferToObject(scheme byte, tx *proto.Transaction) (rideObject, error) {
//switch t := tx.(type) {
//case *proto.TransferWithProofs:
//	rs, err := transferTonewVariablesFromTransferWithProofs(s.Scheme(), t)
//	if err != nil {
//		return nil, errors.Wrap(err, funcName)
//	}
//	return NewObject(rs), nil
//case *proto.TransferWithSig:
//	rs, err := newVariablesFromTransferWithSig(s.Scheme(), t)
//	if err != nil {
//		return nil, errors.Wrap(err, funcName)
//	}
//	return NewObject(rs), nil
//default:
//	return NewUnit(), nil
//}

//return nil, errors.New("not implemented")
//}

func transferWithProofsToObject(scheme byte, tx proto.TransferWithProofs) (rideObject, error) {
	return nil, errors.New("not implemented")
}

func balanceDetailsToObject(balance *proto.FullWavesBalance) (rideObject, error) {
	return nil, errors.New("not implemented")
}
