package ride

import (
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/pkg/errors"
)

const maxMessageLength = 32 * 1024

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

func stringAndIntArgs(args []rideType) (string, int, error) {
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
	return string(s), int(i), nil
}

func twoStringsAndIntArgs(args []rideType) (string, string, int, error) {
	if len(args) != 3 {
		return "", "", 0, errors.Errorf("invalid number of arguments %d, expected 3", len(args))
	}
	if args[0] == nil {
		return "", "", 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return "", "", 0, errors.Errorf("argument 2 is empty")
	}
	if args[2] == nil {
		return "", "", 0, errors.Errorf("argument 3 is empty")
	}
	s1, ok := args[0].(rideString)
	if !ok {
		return "", "", 0, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	s2, ok := args[1].(rideString)
	if !ok {
		return "", "", 0, errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	i, ok := args[2].(rideInt)
	if !ok {
		return "", "", 0, errors.Errorf("unexpected type of argument 3 '%s'", args[2].instanceOf())
	}
	return string(s1), string(s2), int(i), nil
}

func twoStringsArgs(args []rideType) (string, string, error) {
	if len(args) != 2 {
		return "", "", errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return "", "", errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return "", "", errors.Errorf("argument 2 is empty")
	}
	s1, ok := args[0].(rideString)
	if !ok {
		return "", "", errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	s2, ok := args[1].(rideString)
	if !ok {
		return "", "", errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	return string(s1), string(s2), nil
}

func concatStrings(_ environment, args ...rideType) (rideType, error) {
	s1, s2, err := twoStringsArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "concatStrings")
	}
	l := len(s1) + len(s2)
	if l > maxBytesLength {
		return nil, errors.Errorf("concatStrings: length of result (%d) is greater than allowed (%d)", l, maxBytesLength)
	}
	out := s1 + s2
	lengthInRunes := utf8.RuneCountInString(out)
	if lengthInRunes > maxMessageLength {
		return nil, errors.Errorf("concatStrings: length of result (%d) is greater than allowed (%d)", lengthInRunes, maxMessageLength)
	}
	return rideString(out), nil
}

func takeString(env environment, args ...rideType) (rideType, error) {
	s, n, err := stringAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeString")
	}
	return env.takeString(s, n), nil
}

func dropString(_ environment, args ...rideType) (rideType, error) {
	s, n, err := stringAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropString")
	}
	return dropRideString(s, n), nil
}

func sizeString(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeString")
	}
	return rideInt(utf8.RuneCountInString(string(s))), nil
}

func indexOfSubstring(_ environment, args ...rideType) (rideType, error) {
	s1, s2, err := twoStringsArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "indexOfSubstring")
	}
	i := runesIndex(s1, s2)
	if i == -1 {
		return rideUnit{}, nil
	}
	return rideInt(i), nil
}

func indexOfSubstringWithOffset(_ environment, args ...rideType) (rideType, error) {
	s1, s2, n, err := twoStringsAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "lastIndexOfSubstringWithOffset")
	}
	if n < 0 || n > utf8.RuneCountInString(s1) {
		return rideUnit{}, nil
	}
	i := runesIndex(runesDrop(s1, n), s2)
	if i == -1 {
		return rideUnit{}, nil
	}
	return rideInt(i + n), nil
}

func stringToBytes(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringToBytes")
	}
	return rideBytes(s), nil
}

func dropRightString(_ environment, args ...rideType) (rideType, error) {
	s, n, err := stringAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "dropRightString")
	}
	return takeRideString(s, utf8.RuneCountInString(s)-n), nil
}

func takeRightString(_ environment, args ...rideType) (rideType, error) {
	s, n, err := stringAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "takeRightString")
	}
	return dropRideString(s, utf8.RuneCountInString(s)-n), nil
}

func splitString(_ environment, args ...rideType) (rideType, error) {
	s1, s2, err := twoStringsArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "splitString")
	}
	r := make(rideList, 0)
	for _, p := range strings.Split(s1, s2) {
		r = append(r, rideString(p))
	}
	return r, nil
}

func parseInt(_ environment, args ...rideType) (rideType, error) {
	s, err := stringArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "parseInt")
	}
	i, err := strconv.ParseInt(string(s), 10, 64)
	if err != nil {
		return rideUnit{}, nil
	}
	return rideInt(i), nil
}

func parseIntValue(env environment, args ...rideType) (rideType, error) {
	maybeInt, err := parseInt(env, args...)
	if err != nil {
		return nil, errors.Wrap(err, "parseIntValue")
	}
	return extractValue(maybeInt)
}

func lastIndexOfSubstring(_ environment, args ...rideType) (rideType, error) {
	s1, s2, err := twoStringsArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "lastIndexOfSubstring")
	}
	i := strings.LastIndex(s1, s2)
	if i == -1 {
		return rideUnit{}, nil
	}
	return rideInt(i), nil
}

func lastIndexOfSubstringWithOffset(_ environment, args ...rideType) (rideType, error) {
	s1, s2, n, err := twoStringsAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "lastIndexOfSubstringWithOffset")
	}
	if n < 0 {
		return rideUnit{}, nil
	}
	i := strings.LastIndex(s1, s2)
	for i > n {
		i = strings.LastIndex(s1[:i], s2)
	}
	if i == -1 {
		return rideUnit{}, nil
	}
	return rideInt(i), nil
}

func makeString(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "makeString")
	}
	list, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("makeString: unexpected argument type '%s'", args[0].instanceOf())
	}
	parts := make([]string, len(list))
	pl := 0
	pc := 0
	for i, item := range list {
		var str string
		switch ti := item.(type) {
		case rideString:
			str = string(ti)
		case rideInt:
			str = strconv.Itoa(int(ti))
		default:
			return nil, errors.Errorf("makeString: unexpected list item type '%s'", item.instanceOf())
		}
		parts[i] = str
		pl += len(str)
		pc++
	}
	s, ok := args[1].(rideString)
	if !ok {
		return nil, errors.Errorf("makeString: unexpected argument type '%s'", args[1].instanceOf())
	}
	sep := string(s)
	var expectedLength = 0
	if pc > 1 {
		expectedLength = pl + (pc-1)*len(sep)
	}
	if expectedLength > maxMessageLength {
		return nil, errors.Errorf("makeString: resulting length %d exceeds maximum allowed %d", expectedLength, maxMessageLength)
	}
	return rideString(strings.Join(parts, sep)), nil
}

func contains(_ environment, args ...rideType) (rideType, error) {
	s1, s2, err := twoStringsArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "contains")
	}
	return rideBoolean(strings.Contains(s1, s2)), nil
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

// This is the CORRECT implementation of takeString function that handles runes in UTF-8 string correct
func runesTake(s string, n int) string {
	out := make([]rune, n)
	copy(out, []rune(s)[:n])
	return string(out)
}

func takeRideString(s string, n int) rideString {
	l := utf8.RuneCountInString(s)
	t := n
	if t > l {
		t = l
	}
	if t < 0 {
		t = 0
	}
	return rideString(runesTake(s, t))
}

// This is the WRONG implementation of takeString function that handles runes in UTF-8 string INCORRECT
func takeRideStringWrong(s string, n int) rideString {
	b := utf16.Encode([]rune(s))
	l := len(b)
	t := n
	if t > l {
		t = l
	}
	if t < 0 {
		t = 0
	}
	return rideString(strings.ReplaceAll(string(utf16.Decode(b[:t])), "�", "?"))
}

func dropRideString(s string, n int) rideString {
	l := utf8.RuneCountInString(s)
	d := n
	if d > l {
		d = l
	}
	if d < 0 {
		d = 0
	}
	return rideString(runesDrop(s, d))
}
