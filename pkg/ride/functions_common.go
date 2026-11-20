package ride

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const defaultThrowMessage = "Explicit script termination"

func checkArgs(args []rideType, count int) error {
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

func eq(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "eq")
	}
	return rideBoolean(args[0].eq(args[1])), nil
}

func neq(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "neq")
	}
	return rideBoolean(!args[0].eq(args[1])), nil
}

func instanceOf(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "instanceOf")
	}
	t, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("instanceOf: second argument is not a String value but '%s'", args[1].instanceOf())
	}
	return rideBoolean(args[0].instanceOf() == string(t)), nil
}

func getType(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "getType")
	}
	return rideString(args[0].instanceOf()), nil
}

func extract(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "extract")
	}
	if args[0].instanceOf() == unitTypeName {
		return nil, UserError.New("extract() called on unit value")
	}
	return args[0], nil
}

func isDefined(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "isDefined")
	}
	if args[0].instanceOf() == unitTypeName {
		return rideBoolean(false), nil
	}
	return rideBoolean(true), nil
}

func throw(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "throw")
	}
	return nil, UserError.New(string(s))
}

func throw0(_ environment, _ ...rideType) (rideType, error) {
	return nil, UserError.New(defaultThrowMessage)
}

func value(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "value")
	}
	if args[0].instanceOf() == unitTypeName {
		return nil, UserError.New(defaultThrowMessage)
	}
	return args[0], nil
}

func valueOrErrorMessage(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "valueOrErrorMessage")
	}
	msg, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("valueOrErrorMessage: unexpected argument type '%s'", args[1])
	}
	if args[0].instanceOf() == unitTypeName {
		return nil, UserError.New(string(msg))
	}
	return args[0], nil
}

func valueOrElse(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "valueOrErrorMessage")
	}
	if args[0].instanceOf() == unitTypeName {
		return args[1], nil
	}
	return args[0], nil
}

func sizeTuple(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, errors.Wrap(err, "sizeTuple")
	}
	if t, ok := args[0].(rideTuple); ok {
		return rideInt(t.size()), nil
	}
	return nil, errors.Errorf("sizeTuple: unexpected argument type '%s'", args[0].instanceOf())
}

// optionalAssetValue converts the value p of the property key to an optional asset.
func optionalAssetValue(p rideType, key string) (proto.OptionalAsset, error) {
	switch v := p.(type) {
	case rideUnit:
		return proto.NewOptionalAssetWaves(), nil
	case rideByteVector:
		a, err := proto.NewOptionalAssetFromBytes(v)
		if err != nil {
			return proto.OptionalAsset{}, err
		}
		return *a, nil
	default:
		return proto.OptionalAsset{}, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
}

func recipientProperty(obj rideType, key string) (proto.Recipient, error) {
	p, err := obj.get(key)
	if err != nil {
		return proto.Recipient{}, err
	}
	return recipientValue(p, key)
}

// recipientValue converts the value p of the property key to a recipient.
func recipientValue(p rideType, key string) (proto.Recipient, error) {
	switch tp := p.(type) {
	case rideAddress:
		return proto.NewRecipientFromAddress(proto.WavesAddress(tp)), nil
	case rideAlias:
		return proto.NewRecipientFromAlias(proto.Alias(tp)), nil
	default:
		return proto.Recipient{}, errors.Errorf("unexpected type '%s' of property '%s'", p.instanceOf(), key)
	}
}

func extractValue(v rideType) (rideType, error) {
	if _, ok := v.(rideUnit); ok {
		return nil, UserError.New("failed to extract from Unit value")
	}
	return v, nil
}

func checkTakeDropNumberLimit(inputArgType string, limit int, checkLimits bool, n int, fName, rideFName string) error {
	if !checkLimits {
		return nil
	}
	if n < 0 {
		return RuntimeError.Errorf("%s: unexpected negative number = %d passed to %s()",
			fName, n, rideFName,
		)
	}
	if n > limit {
		return RuntimeError.Errorf("%s: number = %d passed to %s() exceeds %s limit = %d",
			fName, n, rideFName, inputArgType, limit,
		)
	}
	return nil
}
