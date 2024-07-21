package ride

import (
	"bytes"
	c1 "crypto"
	"crypto/rsa"
	sh256 "crypto/sha256"
	"crypto/x509"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"

	"github.com/wavesplatform/gowaves/pkg/consensus"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/keyvalue"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	c2 "github.com/wavesplatform/gowaves/pkg/ride/crypto"
	"github.com/wavesplatform/gowaves/pkg/util/common"
)

const invocationsLimit = 100

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
	case rideByteVector:
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
		if value.instanceOf() != attachedPaymentTypeName {
			return nil, RuntimeError.Errorf("payments list has an unexpected element %d of type '%s'",
				i, value.instanceOf())
		}
		amount, err := value.get(amountField)
		if err != nil {
			return nil, RuntimeError.Wrap(err, "attached payment")
		}
		intAmount, ok := amount.(rideInt)
		if !ok {
			return nil, RuntimeError.Errorf("property 'amount' of attached payment %d has an invalid type '%s'",
				i, amount.instanceOf())
		}
		assetID, err := value.get(assetIDField)
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
	ic := ws.invCount()
	if ic > invocationsLimit {
		return rideUnit{}, RuntimeError.Errorf("%s: too many internal invocations", invocation.name())
	}
	defer ws.setInvocationCount(ic)
	callerAddress, ok := env.this().(rideAddress)
	if !ok {
		return rideUnit{}, RuntimeError.Errorf("%s: this has an unexpected type '%s'", invocation.name(), env.this().instanceOf())
	}

	if err := checkArgs(args, 4); err != nil {
		return nil, RuntimeError.Wrapf(err, "%s", invocation.name())
	}
	recipient, err := extractRecipient(args[0])
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to extract recipient", invocation.name())
	}
	recipient, err = ensureRecipientAddress(env, recipient)
	if err != nil {
		return nil, RuntimeError.Wrap(err, invocation.name())
	}

	tree, err := env.state().NewestScriptByAccount(recipient)
	if err != nil {
		return nil, EvaluationFailure.Wrap(err, "failed to get script by recipient")
	}
	if tree.LibVersion < ast.LibV5 {
		return nil, RuntimeError.Errorf(
			"DApp %s invoked DApp %s that uses RIDE %d, but dApp-to-dApp invocation requires version 5 or higher",
			proto.WavesAddress(callerAddress),
			recipient.Address(),
			tree.LibVersion,
		)
	}

	oldLibVersion, err := env.libVersion()
	if err != nil {
		return rideUnit{}, RuntimeError.Wrap(err, invocation.name())
	}
	env.setLibVersion(tree.LibVersion)
	defer func() {
		env.setLibVersion(oldLibVersion)
	}()

	fn, err := extractFunctionName(args[1])
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to extract function name", invocation.name())
	}
	arguments, ok := args[2].(rideList)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of arguments", invocation.name(), args[2].instanceOf())
	}

	oldInvocationParam := env.invocation()
	originCallerRaw, err := oldInvocationParam.get(originCallerField)
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get field from oldInvocation", invocation.name())
	}
	originCaller, ok := originCallerRaw.(rideAddress)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of originCaller", invocation.name(), originCallerRaw.instanceOf())
	}
	feeAssetID, err := oldInvocationParam.get(feeAssetIDField)
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get field from oldInvocation", invocation.name())
	}
	transactionIDRaw, err := oldInvocationParam.get(transactionIDField)
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get field from oldInvocation", invocation.name())
	}
	transactionID, ok := transactionIDRaw.(rideByteVector)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of transactionID", invocation.name(), transactionIDRaw.instanceOf())
	}
	originCallerPublicKey, err := oldInvocationParam.get(originCallerPublicKeyField)
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get field from oldInvocation", invocation.name())
	}
	feeRaw, err := oldInvocationParam.get(feeField)
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get field from oldInvocation", invocation.name())
	}
	fee, ok := feeRaw.(rideInt)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of fee", invocation.name(), feeRaw.instanceOf())
	}
	callerPublicKey, err := env.state().NewestScriptPKByAddr(proto.WavesAddress(callerAddress))
	if err != nil {
		return nil, RuntimeError.Wrapf(err, "%s: failed to get caller public key by address", invocation.name())
	}
	payments, ok := args[3].(rideList)
	if !ok {
		return nil, RuntimeError.Errorf("%s: unexpected type '%s' of payments", invocation.name(), args[3].instanceOf())
	}
	env.setInvocation(newRideInvocationV5(
		originCaller,
		payments,
		common.Dup(callerPublicKey.Bytes()),
		feeAssetID,
		originCallerPublicKey,
		transactionID,
		callerAddress,
		fee,
	))

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

	oldChangedAccounts := ws.diff.replaceChangedAccounts(make(changedAccounts))
	defer func() {
		_ = ws.diff.replaceChangedAccounts(oldChangedAccounts)
	}()

	localActionsCountValidator := proto.NewScriptActionsCountValidator()

	// Check payments itself. We don't validate result balances in following function,
	// but apply payments to wrapped state as is.
	err = ws.smartAppendActions(attachedPaymentActions, env, &localActionsCountValidator)
	if err != nil {
		if GetEvaluationErrorType(err) == Undefined {
			return nil, InternalInvocationError.Wrapf(err, "%s: failed to apply attached payments", invocation.name())
		}
		return nil, err
	}
	checkPaymentsAfterApplication := func(errT EvaluationError) error {
		err = ws.validateBalancesAfterPaymentsApplication(env, proto.WavesAddress(callerAddress), attachedPayments)
		if err != nil && GetEvaluationErrorType(err) == Undefined {
			err = errT.Wrapf(err, "%s: failed to apply attached payments", invocation.name())
		}
		return err
	}
	lightNodeActivated := env.lightNodeActivated()
	if lightNodeActivated { // Check payments result balances here AFTER Light Node activation
		if pErr := checkPaymentsAfterApplication(NegativeBalanceAfterPayment); pErr != nil {
			return nil, pErr
		}
	}

	address, err := env.state().NewestRecipientToAddress(recipient)
	if err != nil {
		return nil, RuntimeError.Errorf("%s: failed to get address from dApp, invokeFunctionFromDApp", invocation.name())
	}
	recipientAddr := address
	env.setNewDAppAddress(recipientAddr)

	if invocation.blocklist() {
		// append a call to the stack to protect a user from the reentrancy attack
		ws.blocklist = append(ws.blocklist, proto.WavesAddress(callerAddress)) // push
		defer func() {
			ws.blocklist = ws.blocklist[:len(ws.blocklist)-1] // pop
		}()
	}

	if ws.invCount() > 1 {
		if containsAddress(recipientAddr, ws.blocklist) && proto.WavesAddress(callerAddress) != recipientAddr {
			return rideUnit{}, InternalInvocationError.Errorf(
				"%s: function call of %s with dApp address %s is forbidden because it had already been called once by 'invoke'",
				invocation.name(), fn, recipientAddr)
		}
	}

	res, err := invokeFunctionFromDApp(env, tree, fn, arguments)
	if err != nil {
		return nil, EvaluationErrorPushf(err, "%s at '%s' function %s with arguments %v",
			invocation.name(), recipientAddr, fn, arguments,
		)
	}

	if !lightNodeActivated { // Check payments result balances here BEFORE Light Node activation
		if pErr := checkPaymentsAfterApplication(InternalInvocationError); pErr != nil {
			return nil, pErr
		}
	}

	err = ws.smartAppendActions(res.ScriptActions(), env, &localActionsCountValidator)
	if err != nil {
		if GetEvaluationErrorType(err) == Undefined {
			return nil, InternalInvocationError.Wrapf(err, "%s: failed to apply actions", invocation.name())
		}
		return nil, err
	}

	if env.validateInternalPayments() || env.rideV6Activated() {
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
	if addr := recipient.Address(); addr != nil {
		return recipient, nil
	}
	alias := recipient.Alias()
	if alias == nil {
		return proto.Recipient{}, errors.New("empty recipient")
	}
	address, err := env.state().NewestAddrByAlias(*alias)
	if err != nil {
		return proto.Recipient{}, errors.Errorf("failed to get address by alias, %v", err)
	}
	return proto.NewRecipientFromAddress(address), nil
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
	script, err := env.state().NewestScriptBytesByAccount(recipient)
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
		return rideByteVector(hash.Bytes()), nil
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
	obj, err := transactionToObject(env, tx)
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
	case rideByteVector:
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
	assetBytes, ok := args[1].(rideByteVector)
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
	return rideByteVector(entry.Value), nil
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
	return rideByteVector(entry.Value), nil
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
	message, ok := args[0].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("sigVerify: unexpected argument type '%s'", args[0].instanceOf())
	}
	if env != nil {
		v, err := env.libVersion()
		if err != nil {
			return nil, errors.Wrap(err, "sigVerify")
		}
		if l := len(message); v == ast.LibV3 && !env.checkMessageLength(l) {
			return nil, errors.Errorf("sigVerify: invalid message size %d", l)
		}
	}
	signature, ok := args[1].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("sigVerify: unexpected argument type '%s'", args[1].instanceOf())
	}
	pkb, ok := args[2].(rideByteVector)
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
	return rideByteVector(d.Bytes()), nil
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
	return rideByteVector(d.Bytes()), nil
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
	return rideByteVector(d), nil
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
	if i <= 0 {
		return rideUnit{}, nil
	}
	height := proto.Height(i)
	blockInfo, err := env.state().NewestBlockInfoByHeight(height)
	if err != nil {
		if errors.Is(err, keyvalue.ErrNotFound) {
			return rideUnit{}, nil
		}
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	v, err := env.libVersion()
	if err != nil {
		return nil, errors.Wrap(err, "blockInfoByHeight")
	}
	return blockInfoToObject(blockInfo, v), nil
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
	switch t := tx.GetTypeInfo().Type; t {
	case proto.TransferTransaction:
		// ok, it's transfer tx
	case proto.EthereumMetamaskTransaction:
		ethTx, ok := tx.(*proto.EthereumTransaction)
		if !ok {
			return nil, errors.Errorf("transferByID: expected ethereum transaction, got (%T)", tx)
		}
		kindType, ktErr := proto.GuessEthereumTransactionKindType(ethTx.Data())
		if ktErr != nil {
			return nil, errors.Wrap(err, "transferByID: failed to guess ethereum transaction kind type")
		}
		switch kindType {
		case proto.EthereumTransferWavesKindType, proto.EthereumTransferAssetsKindType:
			// ok, it's an ethereum transfer tx (waves or asset)
		case proto.EthereumInvokeKindType:
			return rideUnit{}, nil // it's not an ethereum transfer tx
		default:
			return rideUnit{}, errors.Errorf("transferByID: unreachable point reached in eth kind type switch")
		}
	case proto.GenesisTransaction, proto.PaymentTransaction, proto.IssueTransaction,
		proto.ReissueTransaction, proto.BurnTransaction, proto.ExchangeTransaction,
		proto.LeaseTransaction, proto.LeaseCancelTransaction, proto.CreateAliasTransaction,
		proto.MassTransferTransaction, proto.DataTransaction, proto.SetScriptTransaction,
		proto.SponsorshipTransaction, proto.SetAssetScriptTransaction, proto.InvokeScriptTransaction,
		proto.UpdateAssetInfoTransaction, proto.InvokeExpressionTransaction:
		// it's not a transfer transaction
		return rideUnit{}, nil
	default:
		return rideUnit{}, errors.Errorf("transferByID: unreachable point reached in tx type switch")
	}
	obj, err := transactionToObject(env, tx)
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
	message, ok := args[1].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("rsaVerify: unexpected argument type '%s'", args[1].instanceOf())
	}
	if env != nil {
		v, err := env.libVersion()
		if err != nil {
			return nil, errors.Wrap(err, "rsaVerify")
		}
		if l := len(message); v == ast.LibV3 && !env.checkMessageLength(l) {
			return nil, errors.Errorf("rsaVerify: invalid message size %d", l)
		}
	}
	sig, ok := args[2].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("rsaVerify: unexpected argument type '%s'", args[2].instanceOf())
	}
	pk, ok := args[3].(rideByteVector)
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
	root, ok := args[0].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("checkMerkleProof: unexpected argument type '%s'", args[0].instanceOf())
	}
	proof, ok := args[1].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("checkMerkleProof: unexpected argument type '%s'", args[1].instanceOf())
	}
	leaf, ok := args[2].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("checkMerkleProof: unexpected argument type '%s'", args[2].instanceOf())
	}
	r, err := c2.MerkleRootHash(leaf, proof)
	if err != nil {
		return rideBoolean(false), nil
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

func calcAssetID(env environment, name, description rideString, decimals, quantity rideInt, reissuable rideBoolean, nonce rideInt) (rideByteVector, error) {
	pid, ok := env.txID().(rideByteVector)
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
	issue, ok := args[0].(rideIssue)
	if !ok {
		return nil, errors.Errorf("calculateAssetID: unexpected argument type '%s'", args[0])
	}
	return calcAssetID(env, issue.name, issue.description, issue.decimals, issue.quantity, issue.isReissuable, issue.nonce)
}

func simplifiedIssue(env environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 5); err != nil {
		return nil, errors.Wrap(err, "simplifiedIssue")
	}
	issue, err := issueConstructor(env, args[0], args[1], args[2], args[3], args[4], rideUnit{}, rideInt(0))
	if err != nil {
		return nil, errors.Wrap(err, "simplifiedIssue")
	}
	return issue, nil
}

func fullIssue(env environment, args ...rideType) (rideType, error) {
	issue, err := issueConstructor(env, args...)
	if err != nil {
		return nil, errors.Wrap(err, "fullIssue")
	}
	return issue, nil
}

// rebuildMerkleRoot == createMerkleRoot(proofs: List[ByteVector], leaf: ByteVector, index: Int) returns ByteVector
// Recalculates the Merkle root hash from the Merkle proof list, leaf hash, and leaf index.
// proofs - list of proofs in the order from leaf to root.
func rebuildMerkleRoot(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "rebuildMerkleRoot")
	}
	proofsLeafsToRootOrder, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("rebuildMerkleRoot: unexpected argument type '%s'", args[0].instanceOf())
	}
	const maxProofsCount = 16
	if l := len(proofsLeafsToRootOrder); l > maxProofsCount {
		return nil, errors.New("rebuildMerkleRoot: no more than 16 proofs is allowed")
	}
	proofsRootToLeafsOrder := make([]crypto.Digest, 0, len(proofsLeafsToRootOrder))
	for i := len(proofsLeafsToRootOrder) - 1; i >= 0; i-- { // reverse order
		x := proofsLeafsToRootOrder[i]
		b, ok := x.(rideByteVector)
		if !ok {
			return nil, errors.Errorf("rebuildMerkleRoot: unexpected proof type '%s' at position %d", x.instanceOf(), i)
		}
		d, err := crypto.NewDigestFromBytes(b)
		if err != nil {
			return nil, errors.Wrap(err, "rebuildMerkleRoot")
		}
		proofsRootToLeafsOrder = append(proofsRootToLeafsOrder, d)
	}
	leaf, ok := args[1].(rideByteVector)
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
	root := tree.RebuildRoot(lf, proofsRootToLeafsOrder, idx)
	return rideByteVector(root[:]), nil
}

func bls12Groth16Verify(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "bls12Groth16Verify")
	}
	key, ok := args[0].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bls12Groth16Verify: unexpected argument type '%s'", args[0].instanceOf())
	}
	proof, ok := args[1].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bls12Groth16Verify: unexpected argument type '%s'", args[1].instanceOf())
	}
	inputs, ok := args[2].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bls12Groth16Verify: unexpected argument type '%s'", args[2].instanceOf())
	}
	ok, err := crypto.Groth16Verify(key, proof, inputs, ecc.BLS12_381)
	if err != nil {
		return nil, errors.Wrap(err, "bls12Groth16Verify")
	}
	return rideBoolean(ok), nil
}

func bn256Groth16Verify(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "bn256Groth16Verify")
	}
	key, ok := args[0].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bn256Groth16Verify: unexpected argument type '%s'", args[0].instanceOf())
	}
	proof, ok := args[1].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bn256Groth16Verify: unexpected argument type '%s'", args[1].instanceOf())
	}
	inputs, ok := args[2].(rideByteVector)
	if !ok {
		return nil, errors.Errorf("bn256Groth16Verify: unexpected argument type '%s'", args[2].instanceOf())
	}
	ok, err := crypto.Groth16Verify(key, proof, inputs, ecc.BN254)
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
	return rideByteVector(pkb[1:]), nil
}

func calculateDelay(env environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	var addr []byte
	switch arg0 := args[0].(type) {
	case rideAddress:
		addr = proto.WavesAddress(arg0).Bytes()
	case rideAddressLike:
		addr = arg0
	default:
		return nil, errors.Errorf("calculateDelay: unexpected argument type '%s'", args[0].instanceOf())
	}
	balance, ok := args[1].(rideInt)
	if !ok {
		return nil, errors.Errorf("calculateDelay: unexpected argument type '%s'", args[1].instanceOf())
	}
	if balance <= 0 {
		return nil, errors.Errorf("calculateDelay: balance '%d' should by positive", balance)
	}
	vrf, err := env.block().get(vrfField)
	if err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	hitSource, ok := vrf.(rideByteVector)
	if !ok {
		return nil, errors.New("calculateDelay: empty hit source")
	}
	bt, err := env.block().get(baseTargetField)
	if err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	baseTarget, ok := bt.(rideInt)
	if !ok {
		return nil, errors.Errorf("calculateDelay: invalid type '%T' of baseTarget", bt)
	}
	h, err := blake2b.New256(nil)
	if err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	_, err = h.Write(hitSource)
	if err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	_, err = h.Write(addr)
	if err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	hs := h.Sum(nil)[:consensus.HitSize]
	hit := big.NewInt(0).SetBytes(hs)

	pos := consensus.NewFairPosCalculator(0, 0)
	delay, err := pos.CalculateDelay(hit, uint64(baseTarget), uint64(balance))
	if err != nil {
		return nil, errors.Wrap(err, "calculateDelay")
	}
	return rideInt(delay), nil
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

func unit(_ environment, _ ...rideType) (rideType, error) {
	return rideUnit{}, nil
}

func extractRecipient(v rideType) (proto.Recipient, error) {
	var r proto.Recipient
	switch a := v.(type) {
	case rideAddress:
		r = proto.NewRecipientFromAddress(proto.WavesAddress(a))
	case rideAlias:
		r = proto.NewRecipientFromAlias(proto.Alias(a))
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

func bytesOrStringArg(args []rideType) (rideByteVector, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument is empty")
	}
	switch a := args[0].(type) {
	case rideByteVector:
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

func calcLeaseID(env environment, recipient proto.Recipient, amount, nonce rideInt) (rideByteVector, error) {
	pid, ok := env.txID().(rideByteVector)
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
	lease, ok := args[0].(rideLease)
	if !ok {
		return nil, errors.Errorf("calculateLeaseID: unexpected argument type '%s'", args[0])
	}
	recipient, err := recipientProperty(lease, recipientField)
	if err != nil {
		return nil, errors.Wrap(err, "calculateLeaseID")
	}
	return calcLeaseID(env, recipient, lease.amount, lease.nonce)
}

func simplifiedLease(env environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "simplifiedLease")
	}
	rideLease, err := leaseConstructor(env, args[0], args[1], rideInt(0))
	if err != nil {
		return nil, errors.Wrap(err, "simplifiedLease")
	}
	return rideLease, nil
}

func fullLease(env environment, args ...rideType) (rideType, error) {
	rideLease, err := leaseConstructor(env, args...)
	if err != nil {
		return nil, errors.Wrap(err, "fullLease")
	}
	return rideLease, nil
}
