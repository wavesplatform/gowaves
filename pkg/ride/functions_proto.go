package ride

import (
	"bytes"
	c1 "crypto"
	"crypto/rsa"
	sh256 "crypto/sha256"
	"crypto/x509"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	c2 "github.com/wavesplatform/gowaves/pkg/ride/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func invoke(env RideEnvironment, args ...rideType) (rideType, error) {
	env.incrementInvCount()
	if env.invCount() > 9 {
		return rideUnit{}, nil
	}
	callerAddress, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, errors.Errorf("invoke: this has an unexpected type '%s'", env.this().instanceOf())
	}

	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Errorf("invoke: unexpected argument type '%s'", args[0].instanceOf())
	}

	var fnName rideString
	switch fnN := args[1].(type) {
	case rideUnit:
		fnName = "default"
	case rideString:
		if fnN == "" {
			fnName = "default"
			break
		}
		fnName = fnN
	default:
		return nil, errors.Errorf("invoke: unexpected argument type '%s'", args[1].instanceOf())
	}

	listArg, ok := args[2].(rideList)
	if !ok {
		return nil, errors.Errorf("invoke: unexpected argument type '%s'", args[2].instanceOf())
	}

	var attachedPayments proto.ScriptPayments

	payments := args[3].(rideList)

	invocationParam := env.invocation()
	invocationParam["caller"] = callerAddress
	callerPublicKey, err := env.state().NewestScriptPKByAddr(proto.Address(callerAddress), false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get caller public key by address")
	}
	invocationParam["callerPublicKey"] = rideBytes(common.Dup(callerPublicKey.Bytes()))
	invocationParam["payments"] = payments
	env.SetInvocation(invocationParam)

	for _, value := range payments {
		payment, ok := value.(rideObject)
		if !ok {
			return nil, errors.Errorf("invoke: unexpected argument type '%s'", payment.instanceOf())
		}

		assetID := payment["assetId"]
		amount := payment["amount"]

		intAmount, ok := amount.(rideInt)
		if !ok {
			return nil, errors.Errorf("invoke: unexpected argument type '%s'", amount.instanceOf())
		}
		var asset crypto.Digest

		switch asID := assetID.(type) {
		case rideBytes:
			asset, _ = crypto.NewDigestFromBytes(asID)
		case rideUnit:
			asset = crypto.Digest{}
		default:
			return nil, errors.Errorf("attachedPayment: unexpected argument type '%s'", args[0].instanceOf())
		}
		optAsset := proto.OptionalAsset{ID: asset}

		attachedPayments = append(attachedPayments, proto.ScriptPayment{Asset: optAsset, Amount: uint64(intAmount)})
	}

	var paymentActions []proto.ScriptAction
	for _, payment := range attachedPayments {
		action := &proto.TransferScriptAction{Sender: callerPublicKey, Recipient: recipient, Amount: int64(payment.Amount), Asset: payment.Asset}
		paymentActions = append(paymentActions, action)
	}

	res, err := invokeFunctionFromDApp(env, recipient, fnName, listArg)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to get RideResult from invokeFunctionFromDApp")
	}

	err = env.smartAppendActions(paymentActions)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to apply attachedPayments")
	}

	if res.Result() {
		if res.UserError() != "" {
			return nil, errors.Errorf(res.UserError())
		}

		err = env.smartAppendActions(res.ScriptActions())
		env.setNewDAppAddress(proto.Address(callerAddress))
		if err != nil {
			return nil, err
		}

		if res.UserResult() == nil {
			return rideUnit{}, nil
		}
		return res.UserResult(), nil
	}

	return nil, errors.Errorf("Result of Invoke is false")
}

func addressFromString(env RideEnvironment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "addressFromString")
	}
	a, err := proto.NewAddressFromString(string(s))
	if err != nil {
		return rideUnit{}, nil
	}
	if a[1] != env.scheme() {
		return rideUnit{}, nil
	}
	return rideAddress(a), nil
}

func addressValueFromString(env RideEnvironment, args ...rideType) (rideType, error) {
	r, err := addressFromString(env, args...)
	if err != nil {
		return nil, errors.Wrap(err, "addressValueFromString")
	}
	if _, ok := r.(rideUnit); ok {
		return rideThrow("failed to extract from Unit value"), nil
	}
	return r, nil
}

func transactionByID(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "transactionByID")
	}
	tx, err := env.state().NewestTransactionByID(b)
	if err != nil {
		if env.state().IsNotFound(err) {
			return rideUnit{}, nil
		}
		return nil, errors.Wrap(err, "transactionByID")
	}
	obj, err := transactionToObject(env.scheme(), tx)
	if err != nil {
		return nil, errors.Wrap(err, "transactionByID")
	}
	return obj, nil
}

func transactionHeightByID(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "transactionHeightByID")
	}
	h, err := env.state().NewestTransactionHeightByID(b)
	if err != nil {
		if env.state().IsNotFound(err) {
			return rideUnit{}, nil
		}
		return nil, errors.Wrap(err, "transactionHeightByID")
	}
	return rideInt(h), nil
}

func assetBalanceV3(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	asset, err := extractAsset(args[1])
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	balance, err := env.state().NewestAccountBalance(recipient, asset)
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	return rideInt(balance), nil
}

func assetBalanceV4(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	asset, err := extractAsset(args[1])
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	if len(asset) == 0 { // Additional check, empty asset's ID is not allowed any more
		return nil, errors.New("assetBalanceV4: empty asset ID")
	}
	balance, err := env.state().NewestAccountBalance(recipient, asset)
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	return rideInt(balance), nil
}

func intFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	r, k, err := extractRecipientAndKey(args)
	if err != nil {
		return rideUnit{}, nil
	}
	entry, err := env.state().RetrieveNewestIntegerEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideInt(entry.Value), nil
}

func bytesFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	r, k, err := extractRecipientAndKey(args)
	if err != nil {
		return rideUnit{}, nil
	}
	entry, err := env.state().RetrieveNewestBinaryEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideBytes(entry.Value), nil
}

func stringFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	r, k, err := extractRecipientAndKey(args)
	if err != nil {
		return rideUnit{}, nil
	}
	entry, err := env.state().RetrieveNewestStringEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideString(entry.Value), nil
}

func booleanFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	r, k, err := extractRecipientAndKey(args)
	if err != nil {
		return rideUnit{}, nil
	}
	entry, err := env.state().RetrieveNewestBooleanEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideBoolean(entry.Value), nil
}

func addressFromRecipient(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "addressFromRecipient")
	}
	switch r := args[0].(type) {
	case rideRecipient:
		if r.Address != nil {
			return rideAddress(*r.Address), nil
		}
		if r.Alias != nil {
			addr, err := env.state().NewestAddrByAlias(*r.Alias)
			if err != nil {
				return nil, errors.Wrap(err, "addressFromRecipient")
			}
			return rideAddress(addr), nil
		}
		return nil, errors.Errorf("addressFromRecipient: unable to get address from recipient '%s'", r)
	case rideAddress:
		return r, nil
	default:
		return nil, errors.Errorf("addressFromRecipient: unexpected argument type '%s'", args[0].instanceOf())
	}
}

func sigVerify(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "sigVerify")
	}
	message, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("sigVerify: unexpected argument type '%s'", args[0].instanceOf())
	}
	if l := len(message); env != nil && !env.checkMessageLength(l) {
		return nil, errors.Errorf("sigVerify: invalid message size %d", l)
	}
	signature, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("sigVerify: unexpected argument type '%s'", args[1].instanceOf())
	}
	pkb, ok := args[2].(rideBytes)
	if !ok {
		return nil, errors.Errorf("sigVerify: unexpected argument type '%s'", args[2].instanceOf())
	}
	pk, err := crypto.NewPublicKeyFromBytes(pkb)
	if err != nil {
		return rideBoolean(false), nil
	}
	sig, err := crypto.NewSignatureFromBytes(signature)
	if err != nil {
		return rideBoolean(false), nil
	}
	ok = crypto.Verify(pk, sig, message)
	return rideBoolean(ok), nil
}

func keccak256(env RideEnvironment, args ...rideType) (rideType, error) {
	data, err := bytesOrStringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "keccak256")
	}
	if l := len(data); env != nil && !env.checkMessageLength(l) {
		return nil, errors.Errorf("keccak256: invalid data size %d", l)
	}
	d, err := crypto.Keccak256(data)
	if err != nil {
		return nil, errors.Wrap(err, "keccak256")
	}
	return rideBytes(d.Bytes()), nil
}

func blake2b256(env RideEnvironment, args ...rideType) (rideType, error) {
	data, err := bytesOrStringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "blake2b256")
	}
	if l := len(data); env != nil && !env.checkMessageLength(l) {
		return nil, errors.Errorf("blake2b256: invalid data size %d", l)
	}
	d, err := crypto.FastHash(data)
	if err != nil {
		return nil, errors.Wrap(err, "blake2b256")
	}
	return rideBytes(d.Bytes()), nil
}

func sha256(env RideEnvironment, args ...rideType) (rideType, error) {
	data, err := bytesOrStringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sha256")
	}
	if l := len(data); env != nil && !env.checkMessageLength(l) {
		return nil, errors.Errorf("sha256: invalid data size %d", l)
	}
	h := sh256.New()
	if _, err = h.Write(data); err != nil {
		return nil, errors.Wrap(err, "sha256")
	}
	d := h.Sum(nil)
	return rideBytes(d), nil
}

func addressFromPublicKey(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "addressFromPublicKey")
	}
	addr, err := proto.NewAddressLikeFromAnyBytes(env.scheme(), b)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideAddress(addr), nil
}

func wavesBalanceV3(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV3")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV3")
	}
	balance, err := env.state().NewestAccountBalance(recipient, nil)
	if err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV3")
	}
	return rideInt(balance), nil
}

func wavesBalanceV4(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV4")
	}
	r, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV4")
	}
	balance, err := env.state().NewestFullWavesBalance(r)
	if err != nil {
		return nil, errors.Wrapf(err, "wavesBalanceV4(%s)", r.String())
	}
	return balanceDetailsToObject(balance), nil
}

func assetInfoV3(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "assetInfoV3")
	}
	asset, err := crypto.NewDigestFromBytes(b)
	if err != nil {
		return rideUnit{}, nil
	}
	info, err := env.state().NewestAssetInfo(asset)
	if err != nil {
		return rideUnit{}, nil
	}
	return assetInfoToObject(info), nil
}

func assetInfoV4(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "assetInfoV4")
	}
	asset, err := crypto.NewDigestFromBytes(b)
	if err != nil {
		return rideUnit{}, nil
	}
	info, err := env.state().NewestFullAssetInfo(asset)
	if err != nil {
		return rideUnit{}, nil
	}
	return fullAssetInfoToObject(info), nil
}

func blockInfoByHeight(env RideEnvironment, args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	height := proto.Height(i)
	header, err := env.state().NewestHeaderByHeight(height)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	vrf, err := env.state().BlockVRF(header, height)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	obj, err := blockHeaderToObject(env.scheme(), header, vrf)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	return obj, nil
}

func transferByID(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "transferByID")
	}
	tx, err := env.state().NewestTransactionByID(b)
	if err != nil {
		if env.state().IsNotFound(err) {
			return rideUnit{}, nil
		}
		return nil, errors.Wrap(err, "transferByID")
	}
	obj, err := transactionToObject(env.scheme(), tx)
	if err != nil {
		return nil, errors.Wrap(err, "transferByID")
	}
	return obj, nil
}

func addressToString(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "addressToString")
	}
	switch a := args[0].(type) {
	case rideAddress:
		return rideString(proto.Address(a).String()), nil
	case rideRecipient:
		if a.Address == nil {
			return nil, errors.Errorf("addressToString: recipient is not an Address '%s'", args[0].instanceOf())
		}
		return rideString(a.Address.String()), nil
	default:
		return nil, errors.Errorf("addressToString: invalid argument type '%s'", args[0].instanceOf())
	}
}

func rsaVerify(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 4); err != nil {
		return nil, errors.Wrap(err, "rsaVerify")
	}
	digest, err := digest(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "rsaVerify")
	}
	message, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("rsaVerify: unexpected argument type '%s'", args[1].instanceOf())
	}
	sig, ok := args[2].(rideBytes)
	if !ok {
		return nil, errors.Errorf("rsaVerify: unexpected argument type '%s'", args[2].instanceOf())
	}
	pk, ok := args[3].(rideBytes)
	if !ok {
		return nil, errors.Errorf("rsaVerify unexpected argument type '%s'", args[3].instanceOf())
	}
	key, err := x509.ParsePKIXPublicKey(pk)
	if err != nil {
		return nil, errors.Wrap(err, "rsaVerify: invalid public key")
	}
	k, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("rsaVerify: not an RSA key")
	}
	d := message
	if digest != 0 {
		h := digest.New()
		_, _ = h.Write(message)
		d = h.Sum(nil)
	}
	ok, err = c2.VerifyPKCS1v15(k, digest, d, sig)
	if err != nil {
		return nil, errors.Wrap(err, "rsaVerify")
	}
	return rideBoolean(ok), nil
}

func checkMerkleProof(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "checkMerkleProof")
	}
	root, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("checkMerkleProof: unexpected argument type '%s'", args[0].instanceOf())
	}
	proof, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("checkMerkleProof: unexpected argument type '%s'", args[1].instanceOf())
	}
	leaf, ok := args[2].(rideBytes)
	if !ok {
		return nil, errors.Errorf("checkMerkleProof: unexpected argument type '%s'", args[2].instanceOf())
	}
	r, err := c2.MerkleRootHash(leaf, proof)
	if err != nil {
		return nil, errors.Wrap(err, "checkMerkleProof")
	}
	return rideBoolean(bytes.Equal(root, r)), nil
}

func intValueFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	v, err := intFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	v, err := booleanFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	v, err := bytesFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromState(env RideEnvironment, args ...rideType) (rideType, error) {
	v, err := stringFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func transferFromProtobuf(env RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "transferFromProtobuf")
	}
	tx := new(proto.TransferWithProofs)
	err = tx.UnmarshalSignedFromProtobuf(b)
	if err != nil {
		return nil, errors.Wrap(err, "transferFromProtobuf")
	}
	//TODO: using scope's scheme is not quite correct here, because it should be possible to validate transfers from other networks
	err = tx.GenerateID(env.scheme())
	if err != nil {
		return nil, errors.Wrap(err, "transferFromProtobuf")
	}
	obj, err := transferWithProofsToObject(env.scheme(), tx)
	if err != nil {
		return nil, errors.Wrap(err, "transferFromProtobuf")
	}
	return obj, nil
}

func calcAssetID(env RideEnvironment, name, description rideString, decimals, quantity rideInt, reissuable rideBoolean, nonce rideInt) (rideBytes, error) {
	pid, ok := env.txID().(rideBytes)
	if !ok {
		return nil, errors.New("calculateAssetID: no parent transaction ID found")
	}
	d, err := crypto.NewDigestFromBytes(pid)
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	id := proto.GenerateIssueScriptActionID(string(name), string(description), int64(decimals), int64(quantity), bool(reissuable), int64(nonce), d)
	return id.Bytes(), nil
}

func calculateAssetID(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	if t := args[0].instanceOf(); t != "Issue" {
		return nil, errors.Errorf("calculateAssetID: unexpected argument type '%s'", t)
	}
	issue, ok := args[0].(rideObject)
	if !ok {
		return nil, errors.New("calculateAssetID: not an object")
	}
	name, err := stringProperty(issue, "name")
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	description, err := stringProperty(issue, "description")
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	decimals, err := intProperty(issue, "decimals")
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	quantity, err := intProperty(issue, "quantity")
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	reissuable, err := booleanProperty(issue, "isReissuable")
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	nonce, err := intProperty(issue, "nonce")
	if err != nil {
		return nil, errors.Wrap(err, "calculateAssetID")
	}
	return calcAssetID(env, name, description, decimals, quantity, reissuable, nonce)
}

func simplifiedIssue(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 5); err != nil {
		return nil, errors.Wrap(err, "simplifiedIssue")
	}
	name, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("simplifiedIssue: unexpected argument type '%s'", args[0].instanceOf())
	}
	description, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("simplifiedIssue: unexpected argument type '%s'", args[1].instanceOf())
	}
	quantity, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("simplifiedIssue: unexpected argument type '%s'", args[2].instanceOf())
	}
	decimals, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("simplifiedIssue: unexpected argument type '%s'", args[3].instanceOf())
	}
	reissuable, ok := args[4].(rideBoolean)
	if !ok {
		return nil, errors.Errorf("simplifiedIssue: unexpected argument type '%s'", args[4].instanceOf())
	}
	return newIssue(name, description, quantity, decimals, reissuable, rideUnit{}, 0), nil
}

func fullIssue(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 7); err != nil {
		return nil, errors.Wrap(err, "fullIssue")
	}
	name, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[0].instanceOf())
	}
	description, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[1].instanceOf())
	}
	quantity, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[2].instanceOf())
	}
	decimals, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[3].instanceOf())
	}
	reissuable, ok := args[4].(rideBoolean)
	if !ok {
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[4].instanceOf())
	}
	var script rideType
	switch s := args[5].(type) {
	case rideBytes:
		script = s
	case rideUnit:
		script = s
	default:
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[5].instanceOf())
	}
	nonce, ok := args[6].(rideInt)
	if !ok {
		return nil, errors.Errorf("fullIssue: unexpected argument type '%s'", args[6].instanceOf())
	}
	return newIssue(name, description, quantity, decimals, reissuable, script, nonce), nil
}

func rebuildMerkleRoot(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "rebuildMerkleRoot")
	}
	proofs, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("rebuildMerkleRoot: unexpected argument type '%s'", args[0].instanceOf())
	}
	if l := len(proofs); l > 16 {
		return nil, errors.New("rebuildMerkleRoot: no more than 16 proofs is allowed")
	}
	pfs := make([]crypto.Digest, len(proofs))
	for i, x := range proofs {
		b, ok := x.(rideBytes)
		if !ok {
			return nil, errors.Errorf("rebuildMerkleRoot: unexpected proof type '%s' at position %d", x.instanceOf(), i)
		}
		d, err := crypto.NewDigestFromBytes(b)
		if err != nil {
			return nil, errors.Wrap(err, "rebuildMerkleRoot")
		}
		pfs[i] = d
	}
	leaf, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("rebuildMerkleRoot: unexpected argument type '%s'", args[1].instanceOf())
	}
	lf, err := crypto.NewDigestFromBytes(leaf)
	if err != nil {
		return nil, errors.Wrap(err, "rebuildMerkleRoot")
	}
	index, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("rebuildMerkleRoot: unexpected argument type '%s'", args[2].instanceOf())
	}
	idx := uint64(index)
	tree, err := crypto.NewMerkleTree()
	if err != nil {
		return nil, errors.Wrap(err, "rebuildMerkleRoot")
	}
	root := tree.RebuildRoot(lf, pfs, idx)
	return rideBytes(root[:]), nil
}

func bls12Groth16Verify(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "bls12Groth16Verify")
	}
	key, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bls12Groth16Verify: unexpected argument type '%s'", args[0].instanceOf())
	}
	proof, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bls12Groth16Verify: unexpected argument type '%s'", args[1].instanceOf())
	}
	inputs, ok := args[2].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bls12Groth16Verify: unexpected argument type '%s'", args[2].instanceOf())
	}
	ok, err := crypto.Bls12381{}.Groth16Verify(key, proof, inputs)
	if err != nil {
		return nil, errors.Wrap(err, "bls12Groth16Verify")
	}
	return rideBoolean(ok), nil
}

func bn256Groth16Verify(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "bn256Groth16Verify")
	}
	key, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bn256Groth16Verify: unexpected argument type '%s'", args[0].instanceOf())
	}
	proof, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bn256Groth16Verify: unexpected argument type '%s'", args[1].instanceOf())
	}
	inputs, ok := args[2].(rideBytes)
	if !ok {
		return nil, errors.Errorf("bn256Groth16Verify: unexpected argument type '%s'", args[2].instanceOf())
	}
	ok, err := crypto.Bn256{}.Groth16Verify(key, proof, inputs)
	if err != nil {
		return nil, errors.Wrap(err, "bn256Groth16Verify")
	}
	return rideBoolean(ok), nil
}

func ecRecover(_ RideEnvironment, args ...rideType) (rideType, error) {
	digest, signature, err := bytesArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "ecRecover")
	}
	if l := len(digest); l != 32 {
		return nil, errors.Errorf("ecRecover: invalid message digest size %d, expected 32 bytes", l)
	}
	if l := len(signature); l != 65 {
		return nil, errors.Errorf("ecRecover: invalid signature size %d, expected 65 bytes", l)
	}
	pk, err := crypto.ECDSARecoverPublicKey(digest, signature)
	if err != nil {
		return nil, errors.Wrapf(err, "ecRecover")
	}
	pkb := pk.SerializeUncompressed()
	//We have to drop first byte because in bitcoin library where is a length.
	return rideBytes(pkb[1:]), nil
}

func checkedBytesDataEntry(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "checkedBytesDataEntry")
	}
	key, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("checkedBytesDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	value, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("checkedBytesDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	return newDataEntry("BinaryEntry", key, value), nil
}

func checkedBooleanDataEntry(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "checkedBooleanDataEntry")
	}
	key, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("checkedBooleanDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	value, ok := args[1].(rideBoolean)
	if !ok {
		return nil, errors.Errorf("checkedBooleanDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	return newDataEntry("BooleanEntry", key, value), nil
}

func checkedDeleteEntry(_ RideEnvironment, args ...rideType) (rideType, error) {
	key, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "checkedDeleteEntry")
	}
	return newDataEntry("DeleteEntry", key, rideUnit{}), nil
}

func checkedIntDataEntry(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "checkedIntDataEntry")
	}
	key, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("checkedIntDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	value, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("checkedIntDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	return newDataEntry("IntegerEntry", key, value), nil
}

func checkedStringDataEntry(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "checkedStringDataEntry")
	}
	key, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("checkedStringDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	value, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("checkedStringDataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	return newDataEntry("StringEntry", key, value), nil
}

// Constructors

func address(_ RideEnvironment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "address")
	}
	addr, err := proto.NewAddressFromBytes(b)
	if err != nil {
		return rideAddressLike(b), nil
	}
	return rideAddress(addr), nil
}

func alias(env RideEnvironment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "alias")
	}
	alias := proto.NewAlias(env.scheme(), string(s))
	return rideAlias(*alias), nil
}

func assetPair(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "assetPair")
	}
	aa, ok := checkAsset(args[0])
	if !ok {
		return nil, errors.Errorf("assetPair: unexpected argument type '%s'", args[0].instanceOf())
	}
	pa, ok := checkAsset(args[1])
	if !ok {
		return nil, errors.Errorf("assetPair: unexpected argument type '%s'", args[1].instanceOf())
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("AssetPair")
	obj["amountAsset"] = aa
	obj["priceAsset"] = pa
	return obj, nil
}

func burn(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "burn")
	}
	assetID, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("burn: unexpected argument type '%s'", args[0].instanceOf())
	}
	quantity, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("burn: unexpected argument type '%s'", args[1].instanceOf())
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("Burn")
	obj["assetId"] = assetID
	obj["quantity"] = quantity
	return obj, nil
}

func dataEntry(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "dataEntry")
	}
	key, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("dataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	var value rideType
	switch v := args[1].(type) {
	case rideInt, rideBytes, rideBoolean, rideString:
		value = v
	default:
		return nil, errors.Errorf("dataEntry: unexpected argument type '%s'", args[0].instanceOf())
	}
	return newDataEntry("DataEntry", key, value), nil
}

func dataTransaction(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 9); err != nil {
		return nil, errors.Wrap(err, "dataTransaction")
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("DataTransaction")
	entries, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[0].instanceOf())
	}
	obj["data"] = entries
	id, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[1].instanceOf())
	}
	obj["id"] = id
	fee, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[2].instanceOf())
	}
	obj["fee"] = fee
	timestamp, ok := args[3].(rideInt)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[3].instanceOf())
	}
	obj["timestamp"] = timestamp
	version, ok := args[4].(rideInt)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[4].instanceOf())
	}
	obj["version"] = version
	addr, ok := args[5].(rideAddress)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[5].instanceOf())
	}
	obj["sender"] = addr
	pk, ok := args[6].(rideBytes)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[6].instanceOf())
	}
	obj["senderPublicKey"] = pk
	body, ok := args[7].(rideBytes)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[7].instanceOf())
	}
	obj["bodyBytes"] = body
	proofs, ok := args[8].(rideList)
	if !ok {
		return nil, errors.Errorf("dataTransaction: unexpected argument type '%s'", args[8].instanceOf())
	}
	obj["proofs"] = proofs
	return obj, nil
}

func scriptResult(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "scriptResult")
	}
	if args[0].instanceOf() != "WriteSet" {
		return nil, errors.Errorf("scriptResult: unexpected argument type '%s'", args[0].instanceOf())
	}
	if args[1].instanceOf() != "TransferSet" {
		return nil, errors.Errorf("scriptResult: unexpected argument type '%s'", args[1].instanceOf())
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("ScriptResult")
	obj["writeSet"] = args[0]
	obj["transferSet"] = args[1]
	return obj, nil
}

func writeSet(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "writeSet")
	}
	list, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("writeSet: unexpected argument type '%s'", args[0].instanceOf())
	}
	var entries rideList
	for _, item := range list {
		e, ok := item.(rideObject)
		if !ok || e.instanceOf() != "DataEntry" {
			return nil, errors.Errorf("writeSet: unexpected list item type '%s'", item.instanceOf())
		}
		entries = append(entries, e)
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("WriteSet")
	obj["data"] = entries
	return obj, nil
}

func scriptTransfer(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "scriptTransfer")
	}
	var recipient rideType
	switch tr := args[0].(type) {
	case rideRecipient, rideAlias, rideAddress:
		recipient = tr
	default:
		return nil, errors.Errorf("scriptTransfer: unexpected argument type '%s'", args[0].instanceOf())
	}
	amount, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("scriptTransfer: unexpected argument type '%s'", args[1].instanceOf())
	}
	asset, ok := checkAsset(args[2])
	if !ok {
		return nil, errors.Errorf("scriptTransfer: unexpected argument type '%s'", args[2].instanceOf())
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("ScriptTransfer")
	obj["recipient"] = recipient
	obj["amount"] = amount
	obj["asset"] = asset
	return obj, nil
}

func transferSet(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "transferSet")
	}
	list, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("transferSet: unexpected argument type '%s'", args[0].instanceOf())
	}
	var transfers rideList
	for _, item := range list {
		t, ok := item.(rideObject)
		if !ok || t.instanceOf() != "ScriptTransfer" {
			return nil, errors.Errorf("transferSet: unexpected list item type '%s'", item.instanceOf())
		}
		transfers = append(transfers, t)
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("TransferSet")
	obj["transfers"] = transfers
	return obj, nil
}

func unit(_ RideEnvironment, _ ...rideType) (rideType, error) {
	return rideUnit{}, nil
}

func reissue(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "reissue")
	}
	assetID, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("reissue: unexpected argument type '%s'", args[0].instanceOf())
	}
	quantity, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("reissue: unexpected argument type '%s'", args[1].instanceOf())
	}
	reissuable, ok := args[2].(rideBoolean)
	if !ok {
		return nil, errors.Errorf("reissue: unexpected argument type '%s'", args[2].instanceOf())
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("Reissue")
	obj["assetId"] = assetID
	obj["quantity"] = quantity
	obj["isReissuable"] = reissuable
	return obj, nil
}

func sponsorship(_ RideEnvironment, args ...rideType) (rideType, error) {
	asset, fee, err := bytesAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "sponsorship")
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("SponsorFee")
	obj["assetId"] = rideBytes(asset)
	obj["minSponsoredAssetFee"] = rideInt(fee)
	return obj, nil
}

func attachedPayment(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "attachedPayment")
	}

	r := make(rideObject)
	r[instanceFieldName] = rideString("AttachedPayment")

	var assetID rideType
	switch assID := args[0].(type) {
	case rideBytes, rideUnit:
		assetID = assID
	default:
		return nil, errors.Errorf("attachedPayment: unexpected argument type '%s'", args[0].instanceOf())
	}
	r["assetId"] = assetID

	amount, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("attachedPayment: unexpected argument type '%s'", args[1].instanceOf())
	}
	r["amount"] = amount
	return r, nil
}

func extractRecipient(v rideType) (proto.Recipient, error) {
	var r proto.Recipient
	switch a := v.(type) {
	case rideAddress:
		r = proto.NewRecipientFromAddress(proto.Address(a))
	case rideAlias:
		r = proto.NewRecipientFromAlias(proto.Alias(a))
	case rideRecipient:
		r = proto.Recipient(a)
	default:
		return proto.Recipient{}, errors.Errorf("unable to extract recipient from '%s'", v.instanceOf())
	}
	return r, nil
}

func extractAsset(v rideType) ([]byte, error) {
	switch a := v.(type) {
	case rideBytes:
		return a, nil
	case rideUnit:
		return nil, nil
	default:
		return nil, errors.Errorf("unable to extract asset ID from '%s'", v.instanceOf())
	}
}

func extractRecipientAndKey(args []rideType) (proto.Recipient, string, error) {
	if err := checkArgs(args, 2); err != nil {
		return proto.Recipient{}, "", err
	}
	r, err := extractRecipient(args[0])
	if err != nil {
		return proto.Recipient{}, "", err
	}
	key, ok := args[1].(rideString)
	if !ok {
		return proto.Recipient{}, "", errors.Errorf("unexpected argument '%s'", args[1].instanceOf())
	}
	return r, string(key), nil
}

func bytesOrStringArg(args []rideType) (rideBytes, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument is empty")
	}
	switch a := args[0].(type) {
	case rideBytes:
		return a, nil
	case rideString:
		return []byte(a), nil
	default:
		return nil, errors.Errorf("unexpected argument type '%s'", args[0].instanceOf())
	}
}

func digest(v rideType) (c1.Hash, error) {
	switch v.instanceOf() {
	case "NoAlg":
		return 0, nil
	case "Md5":
		return c1.MD5, nil
	case "Sha1":
		return c1.SHA1, nil
	case "Sha224":
		return c1.SHA224, nil
	case "Sha256":
		return c1.SHA256, nil
	case "Sha384":
		return c1.SHA384, nil
	case "Sha512":
		return c1.SHA512, nil
	case "Sha3224":
		return c1.SHA3_224, nil
	case "Sha3256":
		return c1.SHA3_256, nil
	case "Sha3384":
		return c1.SHA3_384, nil
	case "Sha3512":
		return c1.SHA3_512, nil
	default:
		return 0, errors.Errorf("unexpected argument type '%s'", v.instanceOf())
	}
}

func newIssue(name, description rideString, quantity, decimals rideInt, reissuable rideBoolean, script rideType, nonce rideInt) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = rideString("Issue")
	r["name"] = name
	r["description"] = description
	r["quantity"] = quantity
	r["decimals"] = decimals
	r["isReissuable"] = reissuable
	r["compiledScript"] = script
	r["nonce"] = nonce
	return r
}

func newDataEntry(name, key rideString, value rideType) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = name
	r["key"] = key
	r["value"] = value
	return r
}

func checkAsset(v rideType) (rideType, bool) {
	switch v.(type) {
	case rideUnit, rideBytes:
		return v, true
	default:
		return nil, false
	}
}

func calcLeaseID(env RideEnvironment, recipient proto.Recipient, amount, nonce rideInt) (rideBytes, error) {
	pid, ok := env.txID().(rideBytes)
	if !ok {
		return nil, errors.New("calcLeaseID: no parent transaction ID found")
	}
	d, err := crypto.NewDigestFromBytes(pid)
	if err != nil {
		return nil, errors.Wrap(err, "calcLeaseID")
	}
	id := proto.GenerateLeaseScriptActionID(recipient, int64(amount), int64(nonce), d)
	return id.Bytes(), nil
}

func calculateLeaseID(env RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "calculateLeaseID")
	}
	if t := args[0].instanceOf(); t != "Lease" {
		return nil, errors.Errorf("calculateLeaseID: unexpected argument type '%s'", t)
	}
	lease, ok := args[0].(rideObject)
	if !ok {
		return nil, errors.New("calculateLeaseID: not an object")
	}
	if lease.instanceOf() != "Lease" {
		return nil, errors.Errorf("calculateLeaseID: unexpected object type '%s'", lease.instanceOf())
	}
	recipient, err := recipientProperty(lease, "recipient")
	if err != nil {
		return nil, errors.Wrap(err, "calculateLeaseID")
	}
	amount, err := intProperty(lease, "amount")
	if err != nil {
		return nil, errors.Wrap(err, "calculateLeaseID")
	}
	nonce, err := intProperty(lease, "nonce")
	if err != nil {
		return nil, errors.Wrap(err, "calculateLeaseID")
	}
	return calcLeaseID(env, recipient, amount, nonce)
}

func newLease(recipient rideRecipient, amount, nonce rideInt) rideObject {
	r := make(rideObject)
	r[instanceFieldName] = rideString("Lease")
	r["recipient"] = recipient
	r["amount"] = amount
	r["nonce"] = nonce
	return r
}

func simplifiedLease(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "simplifiedLease")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "simplifiedLease")
	}
	amount, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("simplifiedLease: unexpected argument type '%s'", args[1].instanceOf())
	}
	return newLease(rideRecipient(recipient), amount, 0), nil
}

func fullLease(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "fullLease")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "simplifiedLease")
	}
	amount, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("fullLease: unexpected argument type '%s'", args[1].instanceOf())
	}
	nonce, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("fullLease: unexpected argument type '%s'", args[6].instanceOf())
	}
	return newLease(rideRecipient(recipient), amount, nonce), nil
}

func leaseCancel(_ RideEnvironment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "leaseCancel")
	}
	id, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("leaseCancel: unexpected argument type '%s'", args[0].instanceOf())
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("LeaseCancel")
	obj["leaseId"] = id
	return obj, nil
}
