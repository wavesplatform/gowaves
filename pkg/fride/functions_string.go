package fride

import (
	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"
)

const maxRunesLength = 32767

func stringArgs(args []rideType, count int) ([]rideString, error) {
	if len(args) != count {
		return nil, errors.Errorf("%d is invalid number of arguments, expected %d", len(args), count)
	}
	r := make([]rideString, len(args))
	for n, arg := range args {
		if arg == nil {
			return nil, errors.Errorf("argument %d is empty", n+1)
		}
		l, ok := arg.(rideString)
		if !ok {
			return nil, errors.Errorf("argument %d is not of type 'String' but '%s'", n+1, arg.instanceOf())
		}
		r[n] = l
	}
	return r, nil
}

func stringArg(args []rideType) (rideString, error) {
	if len(args) != 1 {
		return "", errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return "", errors.Errorf("argument 1 is empty")
	}
	s, ok := args[0].(rideString)
	if !ok {
		return "", errors.Errorf("argument 1 is not of type 'String' but '%s'", args[0].instanceOf())
	}
	return s, nil
}

func stringAndIntArgs(args []rideType) (rideString, rideInt, error) {
	if len(args) != 2 {
		return "", 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return "", 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return "", 0, errors.Errorf("argument 2 is empty")
	}
	s, ok := args[0].(rideString)
	if !ok {
		return "", 0, errors.Errorf("argument 1 is not of type 'String' but '%s'", args[0].instanceOf())
	}
	i, ok := args[1].(rideInt)
	if !ok {
		return "", 0, errors.Errorf("argument 2 is not of type 'Int' but '%s'", args[1].instanceOf())
	}
	return s, i, nil
}

func stringArgs2(args []rideType) (rideBytes, rideBytes, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	b1, ok := args[0].(rideBytes)
	if !ok {
		return nil, nil, errors.Errorf("argument 1 is not of type 'ByteVector' but '%s'", args[0].instanceOf())
	}
	b2, ok := args[1].(rideBytes)
	if !ok {
		return nil, nil, errors.Errorf("argument 2 is not of type 'ByteVector' but '%s'", args[1].instanceOf())
	}
	return b1, b2, nil
}

func throw(args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "throw")
	}
	return nil, Throw{Message: string(s)}
}

func concatStrings(args ...rideType) (rideType, error) {
	s1, s2, err := stringArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "concatStrings")
	}
	l := len(s1) + len(s2)
	if l > maxBytesLength {
		return nil, errors.Errorf("concatStrings: length of result (%d) is greater than allowed (%d)", l, maxBytesLength)
	}
	out := string(s1) + string(s2)
	lengthInRunes := utf8.RuneCountInString(out)
	if lengthInRunes > maxRunesLength {
		return nil, errors.Errorf("concatStrings: length of result (%d) is greater than allowed (%d)", lengthInRunes, maxRunesLength)
	}
	return rideString(out), nil
}

func takeStrings(args ...rideType) (rideType, error) {
	s, i, err := stringAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeStrings")
	}
	l := utf8.RuneCountInString(string(s))
	t := int(i)
	if t > l {
		t = l
	}
	if t < 0 {
		t = 0
	}
	return rideString(runesTake(string(s), t)), nil
}

func dropStrings(args ...rideType) (rideType, error) {
	s, i, err := stringAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropStrings")
	}
	l := utf8.RuneCountInString(string(s))
	d := int(i)
	if d > l {
		d = l
	}
	if d < 0 {
		d = 0
	}
	return rideString(runesDrop(string(s), d)), nil
}

func sizeStrings(args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeStrings")
	}
	return rideInt(utf8.RuneCountInString(string(s))), nil
}

func indexOfSubstring(args ...rideType) (rideType, error) {
	s1, s2, err := stringArgs2(args)
	if err != nil {
		return nil, errors.Wrap(err, "indexOfSubstring")
	}
	i := runesIndex(string(s1), string(s2))
	if i == -1 {
		return &rideUnit{}, nil
	}
	return rideInt(i), nil
}

func indexOfSubstringWithOffset(args ...rideType) (rideType, error) {
	if err := checkArgs(args, 3); err != nil {
		return nil, errors.Wrap(err, "indexOfSubstringWithOffset")
	}
	s1, ok := args[0].(rideString)
	if !ok {
		return nil, errors.Errorf("indexOfSubstringWithOffset: argument 1 is not of type 'String' but '%s'", args[0].instanceOf())
	}
	s2, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("indexOfSubstringWithOffset: argument 2 is not of type 'String' but '%s'", args[1].instanceOf())
	}
	i, ok := args[2].(rideInt)
	if !ok {
		return nil, errors.Errorf("indexOfSubstringWithOffset: argument 3 is not of type 'Int' but '%s'", args[2].instanceOf())
	}
	offset := int(i)
	if offset < 0 || offset > utf8.RuneCountInString(string(s1)) {
		return rideUnit{}, nil
	}
	idx := runesIndex(runesDrop(string(s1), offset), string(s2))
	if idx == -1 {
		return rideUnit{}, nil
	}
	return rideInt(int64(i) + int64(offset)), nil
}

func stringToBytes(args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringToBytes")
	}
	return rideBytes(s), nil
}

func runesIndex(s, sub string) int {
	if i := strings.Index(s, sub); i >= 0 {
		return utf8.RuneCountInString(s[:i])
	}
	return -1
}

func runesDrop(s string, n int) string {
	runes := []rune(s)
	out := make([]rune, len(runes)-n)
	copy(out, runes[n:])
	res := string(out)
	return res
}

func runesTake(s string, n int) string {
	out := make([]rune, n)
	copy(out, []rune(s)[:n])
	return string(out)
}

func dropRightString(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func takeRightString(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func splitString(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func parseInt(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func parseIntValue(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func lastIndexOfSubstring(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func lastIndexOfSubstringWithOffset(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}

func makeString(...rideType) (rideType, error) {
	return nil, errors.New("not implemented")
}
