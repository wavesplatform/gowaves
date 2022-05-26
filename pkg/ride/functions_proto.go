package ride

import (
	"bytes"
	c1 "crypto"
	"crypto/rsa"
	sh256 "crypto/sha256"
	"crypto/x509"

	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	c2 "github.com/wavesplatform/gowaves/pkg/ride/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

func containsAddress(addr proto.WavesAddress, list []proto.WavesAddress) bool {
	for _, v := range list {
		if v == addr {
			return true
		}
	}
	return false
}

func extractOptionalAsset(v rideType) (proto.OptionalAsset, error) {
	switch tv := v.(type) {
	case rideBytes:
		asset, err := proto.NewOptionalAssetFromBytes(tv)
		if err != nil {
			return proto.OptionalAsset{}, err
		}
		return *asset, nil
	case rideUnit:
		return proto.NewOptionalAssetWaves(), nil
	default:
		return proto.OptionalAsset{}, errors.Errorf("unexpected type '%s'", v.instanceOf())
	}
}

func convertAttachedPayments(payments rideList) (proto.ScriptPayments, error) {
	res := make([]proto.ScriptPayment, len(payments))
	for i, value := range payments {
		payment, ok := value.(rideObject)
		if !ok {
			return nil, RuntimeError.Errorf("payments list has an unexpected element %d of type '%s'",
				i, payment.instanceOf())
		}
		amount, err := payment.get("amount")
		if err != nil {
			return nil, RuntimeError.Wrap(err, "attached payment")
		}
		intAmount, ok := amount.(rideInt)
		if !ok {
			return nil, RuntimeError.Errorf("property 'amount' of attached payment %d has an invalid type '%s'",
				i, amount.instanceOf())
		}
		assetID, err := payment.get("assetId")
		if err != nil {
			return nil, RuntimeError.Wrap(err, "attached payment")
		}
		asset, err := extractOptionalAsset(assetID)
		if err != nil {
			return nil, RuntimeError.Errorf("property 'assetId' of attached payment %d has an invalid type '%s': %v",
				i, assetID.instanceOf(), err)
		}
		res[i] = proto.ScriptPayment{Asset: asset, Amount: uint64(intAmount)}
	}
	return res, nil
}

func extractFunctionName(v rideType) (rideString, error) {
	switch tv := v.(type) {
	case rideUnit:
		return "default", nil
	case rideString:
		if tv == "" {
			return "default", nil
		}
		return tv, nil
	default:
		return "", RuntimeError.Errorf("unexpected type '%s'", v.instanceOf())
	}
}

type invocation interface {
	name() string
	blocklist() bool
}

type nonReentrantInvocation struct{}

func (i *nonReentrantInvocation) name() string {
	return "invoke"
}

func (i *nonReentrantInvocation) blocklist() bool {
	return true
}

type reentrantInvocation struct{}

func (i *reentrantInvocation) name() string {
	return "reentrantInvoke"
}

func (i *reentrantInvocation) blocklist() bool {
	return false
}

func performInvoke(invocation invocation, env environment, args ...rideType) (rideType, error) {
	ws, ok := env.state().(*WrappedState)
	if !ok {
		return nil, EvaluationFailure.Errorf("%s: wrong state", invocation.name())
	}
	ws.incrementInvCount()
	if ws.invCount() > 100 {
		return rideUnit{}, nil
	}

	callerAddress, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, RuntimeError.Errorf("%s: this has an unexpected type '%s'", invocation.name(), env.this().instanceOf())
	}

	if err := checkArgs(args, 4); err != nil {
		return nil, RuntimeError.Wrapf(err, "%s", invocation.name())
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to extract first argument", invocation.name())
	}
	recipient, err = ensureRecipientAddress(env, recipient)
	if err != nil {
		return nil, RuntimeError.Wrap(err, invocation.name())
	}
	fn, err := extractFunctionName(args[1])
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to extract second argument", invocation.name())
	}
	arguments, ok := args[2].(rideList)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of third argument", invocation.name(), args[2].instanceOf())
	}

	oldInvocationParam := env.invocation()
	invocationParam := oldInvocationParam.copy()
	invocationParam["caller"] = callerAddress
	callerPublicKey, err := env.state().NewestScriptPKByAddr(proto.WavesAddress(callerAddress))
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get caller public key by address", invocation.name())
	}
	invocationParam["callerPublicKey"] = rideBytes(common.Dup(callerPublicKey.Bytes()))
	payments, ok := args[3].(rideList)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of forth argument", invocation.name(), args[3].instanceOf())
	}
	invocationParam["payments"] = payments
	env.setInvocation(invocationParam)

	attachedPayments, err := convertAttachedPayments(payments)
	if err != nil {
		return nil, RuntimeError.Wrap(err, invocation.name())
	}
	// since RideV5 the limit of attached payments is 10
	if len(attachedPayments) > 10 {
		return nil, InternalInvocationError.Errorf("%s: no more than ten payments is allowed since RideV5 activation", invocation.name())
	}
	attachedPaymentActions := make([]proto.ScriptAction, len(attachedPayments))
	for i, payment := range attachedPayments {
		attachedPaymentActions[i] = &proto.AttachedPaymentScriptAction{
			Sender:    &callerPublicKey,
			Recipient: recipient,
			Amount:    int64(payment.Amount),
			Asset:     payment.Asset,
		}
	}
	address, err := env.state().NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, RuntimeError.Errorf("%s: failed to get address from dApp, invokeFunctionFromDApp", invocation.name())
	}
	env.setNewDAppAddress(*address)

	localActionsCountValidators := proto.NewScriptActionsCountValidator()

	err = ws.smartAppendActions(attachedPaymentActions, env, &localActionsCountValidators)
	if err != nil {
		if GetEvaluationErrorType(err) == Undefined {
			return nil, InternalInvocationError.Wrapf(err, "%s: failed to apply attached payments", invocation.name())
		}
		return nil, err
	}

	if invocation.blocklist() {
		// append a call to the stack to protect a user from the reentrancy attack
		ws.blocklist = append(ws.blocklist, proto.WavesAddress(callerAddress)) // push
		defer func() {
			ws.blocklist = ws.blocklist[:len(ws.blocklist)-1] // pop
		}()
	}

	if ws.invCount() > 1 {
		if containsAddress(*recipient.Address, ws.blocklist) && proto.WavesAddress(callerAddress) != *recipient.Address {
			return rideUnit{}, InternalInvocationError.Errorf(
				"%s: function call of %s with dApp address %s is forbidden because it had already been called once by 'invoke'",
				invocation.name(), fn, recipient.Address)
		}
	}

	res, err := invokeFunctionFromDApp(env, recipient, fn, arguments)
	if err != nil {
		return nil, EvaluationErrorPush(err, "%s at '%s' function '%s' with arguments %v", invocation.name(), recipient.Address.String(), fn, arguments)
	}

	err = ws.smartAppendActions(res.ScriptActions(), env, &localActionsCountValidators)
	if err != nil {
		if GetEvaluationErrorType(err) == Undefined {
			return nil, InternalInvocationError.Wrapf(err, "%s: failed to apply actions", invocation.name())
		}
		return nil, err
	}

	if env.validateInternalPayments() && !env.rideV6Activated() {
		err = ws.validateBalances(env.rideV6Activated())
	} else if env.rideV6Activated() {
		err = ws.validateBalances(env.rideV6Activated())
	}
	if err != nil {
		if ws.invCount() > 1 {
			return nil, RuntimeError.Wrapf(err, "%s: failed to validate balances", invocation.name())
		}
		return nil, InternalInvocationError.Wrapf(err, "%s: failed to validate balances", invocation.name())
	}

	env.setNewDAppAddress(proto.WavesAddress(callerAddress))
	env.setInvocation(oldInvocationParam)

	ws.totalComplexity += res.Complexity()

	if res.userResult() == nil {
		return rideUnit{}, nil
	}
	return res.userResult(), nil
}

func reentrantInvoke(env environment, args ...rideType) (rideType, error) {
	return performInvoke(&reentrantInvocation{}, env, args...)
}

func invoke(env environment, args ...rideType) (rideType, error) {
	return performInvoke(&nonReentrantInvocation{}, env, args...)
}

func ensureRecipientAddress(env environment, recipient proto.Recipient) (proto.Recipient, error) {
	if recipient.Address == nil {
		if recipient.Alias == nil {
			return proto.Recipient{}, errors.New("empty recipient")
		}
		address, err := env.state().NewestAddrByAlias(*recipient.Alias)
		if err != nil {
			return proto.Recipient{}, errors.Errorf("failed to get address by alias, %v", err)
		}
		recipient.Address = &address
		return recipient, nil
	}
	return recipient, nil
}

func recipientArg(args []rideType) (proto.Recipient, error) {
	if len(args) != 1 {
		return proto.Recipient{}, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return proto.Recipient{}, errors.Errorf("argument 1 is empty")
	}
	return extractRecipient(args[0])
}

func hashScriptAtAddress(env environment, args ...rideType) (rideType, error) {
	recipient, err := recipientArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "hashScriptAtAddress")
	}
	script, err := env.state().GetByteTree(recipient)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return rideUnit{}, nil
		}
		return nil, errors.Errorf("hashScriptAtAddress: failed to get script by recipient, %v", err)
	}

	if len(script) != 0 {
		hash, err := crypto.FastHash(script)
		if err != nil {
			return nil, errors.Errorf("hashScriptAtAddress: failed to get hash of script, %v", err)
		}
		return rideBytes(hash.Bytes()), nil
	}

	return rideUnit{}, nil
}

func isDataStorageUntouched(env environment, args ...rideType) (rideType, error) {
	recipient, err := recipientArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "isDataStorageUntouched")
	}
	isUntouched, err := env.state().IsStateUntouched(recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "isDataStorageUntouched")
	}
	return rideBoolean(isUntouched), nil
}

func addressFromString(env environment, args ...rideType) (rideType, error) {
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

func addressValueFromString(env environment, args ...rideType) (rideType, error) {
	r, err := addressFromString(env, args...)
	if err != nil {
		return nil, errors.Wrap(err, "addressValueFromString")
	}
	if _, ok := r.(rideUnit); ok {
		return nil, UserError.New("failed to extract from Unit value")
	}
	return r, nil
}

func transactionByID(env environment, args ...rideType) (rideType, error) {
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

func transactionHeightByID(env environment, args ...rideType) (rideType, error) {
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

func assetBalanceV3(env environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	var balance uint64
	switch assetBytes := args[1].(type) {
	case rideUnit:
		balance, err = env.state().NewestWavesBalance(recipient)
	case rideBytes:
		asset, digestErr := crypto.NewDigestFromBytes(assetBytes)
		if digestErr != nil {
			return rideInt(0), nil // according to the scala node implementation
		}
		balance, err = env.state().NewestAssetBalance(recipient, asset)
	default:
		return nil, errors.Errorf("assetBalanceV3: unable to extract asset ID from '%s'", assetBytes.instanceOf())
	}
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV3")
	}
	return rideInt(balance), nil
}

func assetBalanceV4(env environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	assetBytes, ok := args[1].(rideBytes)
	if !ok {
		return nil, errors.Errorf("assetBalanceV4: unable to extract asset ID from '%s'", args[1].instanceOf())
	}
	asset, digestErr := crypto.NewDigestFromBytes(assetBytes)
	if digestErr != nil {
		return rideInt(0), nil // according to the scala node implementation
	}
	balance, err := env.state().NewestAssetBalance(recipient, asset)
	if err != nil {
		return nil, errors.Wrap(err, "assetBalanceV4")
	}
	return rideInt(balance), nil
}

func intFromState(env environment, args ...rideType) (rideType, error) {
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

func intFromSelfState(env environment, args ...rideType) (rideType, error) {
	k, err := keyArg(args)
	if err != nil {
		return rideUnit{}, nil
	}
	a, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, nil
	}
	r := proto.NewRecipientFromAddress(proto.WavesAddress(a))
	entry, err := env.state().RetrieveNewestIntegerEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideInt(entry.Value), nil
}

func bytesFromState(env environment, args ...rideType) (rideType, error) {
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

func bytesFromSelfState(env environment, args ...rideType) (rideType, error) {
	k, err := keyArg(args)
	if err != nil {
		return rideUnit{}, nil
	}
	a, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, nil
	}
	r := proto.NewRecipientFromAddress(proto.WavesAddress(a))
	entry, err := env.state().RetrieveNewestBinaryEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideBytes(entry.Value), nil
}

func stringFromState(env environment, args ...rideType) (rideType, error) {
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

func stringFromSelfState(env environment, args ...rideType) (rideType, error) {
	k, err := keyArg(args)
	if err != nil {
		return rideUnit{}, nil
	}
	a, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, nil
	}
	r := proto.NewRecipientFromAddress(proto.WavesAddress(a))
	entry, err := env.state().RetrieveNewestStringEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideString(entry.Value), nil
}

func booleanFromState(env environment, args ...rideType) (rideType, error) {
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

func booleanFromSelfState(env environment, args ...rideType) (rideType, error) {
	k, err := keyArg(args)
	if err != nil {
		return rideUnit{}, nil
	}
	a, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, nil
	}
	r := proto.NewRecipientFromAddress(proto.WavesAddress(a))
	entry, err := env.state().RetrieveNewestBooleanEntry(r, k)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideBoolean(entry.Value), nil
}

func addressFromRecipient(env environment, args ...rideType) (rideType, error) {
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
	case rideAlias:
		addr, err := env.state().NewestAddrByAlias(proto.Alias(r))
		if err != nil {
			return nil, errors.Wrap(err, "addressFromRecipient")
		}
		return rideAddress(addr), nil
	default:
		return nil, errors.Errorf("addressFromRecipient: unexpected argument type '%s'", args[0].instanceOf())
	}
}

func sigVerify(env environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "sigVerify")
	}
	message, ok := args[0].(rideBytes)
	if !ok {
		return nil, errors.Errorf("sigVerify: unexpected argument type '%s'", args[0].instanceOf())
	}
	if l := len(message); env != nil && env.libVersion() == ast.LibV3 && !env.checkMessageLength(l) {
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

func keccak256(env environment, args ...rideType) (rideType, error) {
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

func blake2b256(env environment, args ...rideType) (rideType, error) {
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

func sha256(env environment, args ...rideType) (rideType, error) {
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

func addressFromPublicKey(env environment, args ...rideType) (rideType, error) {
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

func addressFromPublicKeyStrict(env environment, args ...rideType) (rideType, error) {
	b, err := bytesArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "addressFromPublicKeyStrict")
	}
	switch len(b) {
	case crypto.PublicKeySize:
		pk, err := crypto.NewPublicKeyFromBytes(b)
		if err != nil {
			return nil, errors.Wrap(err, "addressFromPublicKeyStrict")
		}
		a, err := proto.NewAddressFromPublicKey(env.scheme(), pk)
		if err != nil {
			return nil, errors.Wrap(err, "addressFromPublicKeyStrict")
		}
		return rideAddress(a), nil

	case proto.EthereumPublicKeyLength:
		pk, err := proto.NewEthereumPublicKeyFromBytes(b)
		if err != nil {
			return nil, errors.Wrap(err, "addressFromPublicKeyStrict")
		}
		ea := pk.EthereumAddress()
		a, err := ea.ToWavesAddress(env.scheme())
		if err != nil {
			return nil, errors.Wrap(err, "addressFromPublicKeyStrict")
		}
		return rideAddress(a), nil

	default:
		return nil, errors.Errorf("addressFromPublicKeyStrict: unexpected public key length '%d'", len(b))
	}
}

func wavesBalanceV3(env environment, args ...rideType) (rideType, error) {
	recipient, err := recipientArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV3")
	}
	balance, err := env.state().NewestWavesBalance(recipient)
	if err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV3")
	}
	return rideInt(balance), nil
}

func wavesBalanceV4(env environment, args ...rideType) (rideType, error) {
	r, err := recipientArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "wavesBalanceV4")
	}
	balance, err := env.state().NewestFullWavesBalance(r)
	if err != nil {
		return nil, errors.Wrapf(err, "wavesBalanceV4(%s)", r.String())
	}
	return balanceDetailsToObject(balance), nil
}

func assetInfoV3(env environment, args ...rideType) (rideType, error) {
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

func assetInfoV4(env environment, args ...rideType) (rideType, error) {
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

func blockInfoByHeight(env environment, args ...rideType) (rideType, error) {
	i, err := intArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	height := proto.Height(i)
	header, err := env.state().NewestHeaderByHeight(height)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	vrf, err := env.state().BlockVRF(header, height-1)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	obj, err := blockHeaderToObject(env.scheme(), height, header, vrf)
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	return obj, nil
}

func transferByID(env environment, args ...rideType) (rideType, error) {
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

func addressToString(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "addressToString")
	}
	switch a := args[0].(type) {
	case rideAddress:
		return rideString(proto.WavesAddress(a).String()), nil
	case rideRecipient:
		if a.Address == nil {
			return nil, errors.Errorf("addressToString: recipient is not an WavesAddress '%s'", args[0].instanceOf())
		}
		return rideString(a.Address.String()), nil
	case rideAddressLike:
		return rideString(base58.Encode(a)), nil
	default:
		return nil, errors.Errorf("addressToString: invalid argument type '%s'", args[0].instanceOf())
	}
}

func rsaVerify(env environment, args ...rideType) (rideType, error) {
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
	if l := len(message); env != nil && env.libVersion() == ast.LibV3 && !env.checkMessageLength(l) {
		return nil, errors.Errorf("sigVerify: invalid message size %d", l)
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

func checkMerkleProof(_ environment, args ...rideType) (rideType, error) {
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

func intValueFromState(env environment, args ...rideType) (rideType, error) {
	v, err := intFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func intValueFromSelfState(env environment, args ...rideType) (rideType, error) {
	v, err := intFromSelfState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromState(env environment, args ...rideType) (rideType, error) {
	v, err := booleanFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromSelfState(env environment, args ...rideType) (rideType, error) {
	v, err := booleanFromSelfState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromState(env environment, args ...rideType) (rideType, error) {
	v, err := bytesFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromSelfState(env environment, args ...rideType) (rideType, error) {
	v, err := bytesFromSelfState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromState(env environment, args ...rideType) (rideType, error) {
	v, err := stringFromState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromSelfState(env environment, args ...rideType) (rideType, error) {
	v, err := stringFromSelfState(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func transferFromProtobuf(env environment, args ...rideType) (rideType, error) {
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

func calcAssetID(env environment, name, description rideString, decimals, quantity rideInt, reissuable rideBoolean, nonce rideInt) (rideBytes, error) {
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

func calculateAssetID(env environment, args ...rideType) (rideType, error) {
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

func simplifiedIssue(_ environment, args ...rideType) (rideType, error) {
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

func fullIssue(_ environment, args ...rideType) (rideType, error) {
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

func rebuildMerkleRoot(_ environment, args ...rideType) (rideType, error) {
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

func bls12Groth16Verify(_ environment, args ...rideType) (rideType, error) {
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

func bn256Groth16Verify(_ environment, args ...rideType) (rideType, error) {
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

func ecRecover(_ environment, args ...rideType) (rideType, error) {
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

func checkedBytesDataEntry(_ environment, args ...rideType) (rideType, error) {
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

func checkedBooleanDataEntry(_ environment, args ...rideType) (rideType, error) {
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

func checkedDeleteEntry(_ environment, args ...rideType) (rideType, error) {
	key, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "checkedDeleteEntry")
	}
	return newDataEntry("DeleteEntry", key, rideUnit{}), nil
}

func checkedIntDataEntry(_ environment, args ...rideType) (rideType, error) {
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

func checkedStringDataEntry(_ environment, args ...rideType) (rideType, error) {
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

func address(_ environment, args ...rideType) (rideType, error) {
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

func alias(env environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "alias")
	}
	alias := proto.NewAlias(env.scheme(), string(s))
	return rideAlias(*alias), nil
}

func assetPair(_ environment, args ...rideType) (rideType, error) {
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

func burn(_ environment, args ...rideType) (rideType, error) {
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

func dataEntry(_ environment, args ...rideType) (rideType, error) {
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

func dataTransaction(_ environment, args ...rideType) (rideType, error) {
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

func transferObject(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "transferObject")
	}
	obj := make(rideObject)
	obj[instanceFieldName] = rideString("Transfer")
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, errors.Errorf("transferObject: unexpected argument type '%s'", args[0].instanceOf())
	}
	obj["recipient"] = rideRecipient(recipient)
	amount, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("transferObject: unexpected argument type '%s'", args[1].instanceOf())
	}
	obj["amount"] = amount
	return obj, nil
}

func scriptResult(_ environment, args ...rideType) (rideType, error) {
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

func writeSet(_ environment, args ...rideType) (rideType, error) {
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

func scriptTransfer(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "scriptTransfer")
	}
	var recipient rideType
	switch tr := args[0].(type) {
	case rideRecipient, rideAlias, rideAddress, rideAddressLike:
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

func transferSet(_ environment, args ...rideType) (rideType, error) {
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

func unit(_ environment, _ ...rideType) (rideType, error) {
	return rideUnit{}, nil
}

func reissue(_ environment, args ...rideType) (rideType, error) {
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

func sponsorship(_ environment, args ...rideType) (rideType, error) {
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

func attachedPayment(_ environment, args ...rideType) (rideType, error) {
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
		r = proto.NewRecipientFromAddress(proto.WavesAddress(a))
	case rideAlias:
		r = proto.NewRecipientFromAlias(proto.Alias(a))
	case rideRecipient:
		r = proto.Recipient(a)
	default:
		return proto.Recipient{}, errors.Errorf("unable to extract recipient from '%s'", v.instanceOf())
	}
	return r, nil
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

func keyArg(args []rideType) (string, error) {
	if err := checkArgs(args, 1); err != nil {
		return "", err
	}
	key, ok := args[0].(rideString)
	if !ok {
		return "", errors.Errorf("unexpected key type '%s'", args[0].instanceOf())
	}
	return string(key), nil
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

func calcLeaseID(env environment, recipient proto.Recipient, amount, nonce rideInt) (rideBytes, error) {
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

func calculateLeaseID(env environment, args ...rideType) (rideType, error) {
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

func simplifiedLease(_ environment, args ...rideType) (rideType, error) {
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

func fullLease(_ environment, args ...rideType) (rideType, error) {
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

func leaseCancel(_ environment, args ...rideType) (rideType, error) {
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
