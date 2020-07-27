package fride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func addressFromString(args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "addressFromString")
	}
	v, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("addressFromString: first argument is not a string value but '%v' of type '%T'", args[0], args[0])
	}
	a, err := proto.NewAddressFromString(string(v))
	if err != nil {
		return nil, errors.Wrap(err, "addressFromString")
	}
	return rideAddress(a), nil
}

func transactionByID(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func transactionHeightByID(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func assetBalanceV3(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func intFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func bytesFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func stringFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func booleanFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func addressFromRecipient(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unlimitedSigVerify(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unlimitedKeccak256(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unlimitedBlake2b256(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unlimitedSha256(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func addressFromPublicKey(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func address(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func alias(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func assetPair(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func dataEntry(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func dataTransaction(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func wavesBalanceV3(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func assetInfoV3(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func blockInfoByHeight(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func transferByID(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func addressToString(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unlimitedRSAVerify(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func checkMerkleProof(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func intValueFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func booleanValueFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func bytesValueFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func stringValueFromState(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func ceiling(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func down(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func floor(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func halfDown(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func halfEven(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func halfUp(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func md5(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func noAlg(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func scriptResult(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func scriptTransfer(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha1(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha224(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func rideSha256(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha3224(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha3256(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha3384(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha3512(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha384(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sha512(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func transferSet(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unit(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func up(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func writeSet(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func assetBalanceV4(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func assetInfoV4(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func transferFromProtobuf(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func calculateAssetID(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func simplifiedIssue(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func fullIssue(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func rebuildMerkleRoot(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func unlimitedGroth16Verify(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func ecRecover(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func checkedBytesDataEntry(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func checkedBooleanDataEntry(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func burn(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func checkedDeleteEntry(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func checkedIntDataEntry(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func checkedStringDataEntry(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func issue(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func reissue(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func sponsorship(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func wavesBalanceV4(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}
