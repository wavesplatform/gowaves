package ride

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

const defaultThrowMessage = "Explicit script termination"

func checkArgs(args []RideType, count int) error {
	if len(args) != count {
		return errors.Errorf("%d is invalid number of arguments, expected %d", len(args), count)
	}
	for n, arg := range args {
		if arg == nil {
			return errors.Errorf("argument %d is empty", n+1)
		}
	}
	return nil
}

func eq(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "eq")
	}
	return RideBoolean(args[0].eq(args[1])), nil
}

func neq(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "neq")
	}
	return RideBoolean(!args[0].eq(args[1])), nil
}

func instanceOf(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "instanceOf")
	}
	t, ok := args[1].(RideString)
	if !ok {
		return nil, errors.Errorf("instanceOf: second argument is not a String value but '%s'", args[1].instanceOf())
	}
	return RideBoolean(args[0].instanceOf() == string(t)), nil
}

func extract(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "extract")
	}
	if args[0].instanceOf() == "Unit" {
		return rideThrow("extract() called on unit value"), nil
	}
	return args[0], nil
}

func isDefined(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "isDefined")
	}
	if args[0].instanceOf() == "Unit" {
		return RideBoolean(false), nil
	}
	return RideBoolean(true), nil
}

func throw(_ Environment, args ...RideType) (RideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "throw")
	}
	return rideThrow(s), nil
}

func throw0(_ Environment, _ ...RideType) (RideType, error) {
	return rideThrow(defaultThrowMessage), nil
}

func value(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "value")
	}
	if args[0].instanceOf() == "Unit" {
		return rideThrow(defaultThrowMessage), nil
	}
	return args[0], nil
}

func valueOrErrorMessage(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "valueOrErrorMessage")
	}
	msg, ok := args[1].(RideString)
	if !ok {
		return nil, errors.Errorf("valueOrErrorMessage: unexpected argument type '%s'", args[1])
	}
	if args[0].instanceOf() == "Unit" {
		return rideThrow(msg), nil
	}
	return args[0], nil
}

func valueOrElse(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "valueOrErrorMessage")
	}
	if args[0].instanceOf() == "Unit" {
		return args[1], nil
	}
	return args[0], nil
}

func bytesProperty(obj RideType, key string) (RideBytes, error) {
	p, err := obj.get(key)
	if err != nil {
		return nil, err
	}
	r, ok := p.(RideBytes)
	if !ok {
		return nil, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func digestProperty(obj RideType, key string) (crypto.Digest, error) {
	p, err := obj.get(key)
	if err != nil {
		return crypto.Digest{}, err
	}
	b, ok := p.(RideBytes)
	if !ok {
		return crypto.Digest{}, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	r, err := crypto.NewDigestFromBytes(b)
	if err != nil {
		return crypto.Digest{}, err
	}
	return r, nil
}

func stringProperty(obj RideType, key string) (RideString, error) {
	p, err := obj.get(key)
	if err != nil {
		return "", err
	}
	r, ok := p.(RideString)
	if !ok {
		return "", errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func intProperty(obj RideType, key string) (RideInt, error) {
	p, err := obj.get(key)
	if err != nil {
		return 0, err
	}
	r, ok := p.(RideInt)
	if !ok {
		return 0, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func booleanProperty(obj RideType, key string) (RideBoolean, error) {
	p, err := obj.get(key)
	if err != nil {
		return false, err
	}
	r, ok := p.(RideBoolean)
	if !ok {
		return false, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return r, nil
}

func optionalAssetProperty(obj RideType, key string) (proto.OptionalAsset, error) {
	p, err := obj.get(key)
	if err != nil {
		return proto.OptionalAsset{}, err
	}
	switch v := p.(type) {
	case rideUnit:
		return proto.NewOptionalAssetWaves(), nil
	case RideBytes:
		a, err := proto.NewOptionalAssetFromBytes(v)
		if err != nil {
			return proto.OptionalAsset{}, err
		}
		return *a, nil
	default:
		return proto.OptionalAsset{}, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
}

func recipientProperty(obj RideType, key string) (proto.Recipient, error) {
	p, err := obj.get(key)
	if err != nil {
		return proto.Recipient{}, err
	}
	var recipient proto.Recipient
	switch tp := p.(type) {
	case rideRecipient:
		recipient = proto.Recipient(tp)
	case rideAddress:
		recipient = proto.NewRecipientFromAddress(proto.WavesAddress(tp))
	case rideAlias:
		recipient = proto.NewRecipientFromAlias(proto.Alias(tp))
	default:
		return proto.Recipient{}, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
	return recipient, nil
}

func extractValue(v RideType) (RideType, error) {
	if _, ok := v.(rideUnit); ok {
		return rideThrow("failed to extract from Unit value"), nil
	}
	return v, nil
}
