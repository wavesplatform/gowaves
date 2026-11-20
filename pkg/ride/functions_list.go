package ride

import (
	"slices"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

const (
	maxListSize = 1000
	// medianStackBufferSize is the size of the list that median sorts without a heap allocation.
	medianStackBufferSize = 64
)

func listAndStringArgs(args []rideType) (rideList, rideString, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, "", err
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, "", errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	s, ok := args[1].(rideString)
	if !ok {
		return nil, "", errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	return l, s, nil
}

func listAndIntArgs(args []rideType) (rideList, int, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, 0, err
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, 0, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	ri, ok := args[1].(rideInt)
	if !ok {
		return nil, 0, errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	i := int(ri)
	if i < 0 {
		return nil, 0, errors.Errorf("negative index value %d", i)
	}
	if i >= len(l) {
		return nil, 0, errors.New("index out of bounds")
	}
	return l, i, nil
}

func listArg(args []rideType) (rideList, error) {
	if err := checkArgs(args, 1); err != nil {
		return nil, err
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("unexpected type of argument '%s'", args[0].instanceOf())
	}
	return l, nil
}

func listAndElementArgs(args []rideType) (rideList, rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, nil, err
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, nil, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	return l, args[1], nil
}

func listAndIntAndElementArgs(args []rideType) (rideList, int, rideType, error) {
	const threeArgs = 3
	if err := checkArgs(args, threeArgs); err != nil {
		return nil, 0, nil, err
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, 0, nil, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	ri, ok := args[1].(rideInt)
	if !ok {
		return nil, 0, nil, errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	i := int(ri)
	if i < 0 {
		return nil, 0, nil, errors.Errorf("negative index value %d", i)
	}
	if i >= len(l) {
		return nil, 0, nil, errors.New("index out of bounds")
	}
	return l, i, args[2], nil
}

func intFromArray(_ environment, args ...rideType) (rideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "intFromArray")
	}
	return findFirstEntry(list, key, intTypeName), nil
}

func booleanFromArray(_ environment, args ...rideType) (rideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanFromArray")
	}
	return findFirstEntry(list, key, booleanTypeName), nil
}

func bytesFromArray(_ environment, args ...rideType) (rideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesFromArray")
	}
	return findFirstEntry(list, key, byteVectorTypeName), nil
}

func stringFromArray(_ environment, args ...rideType) (rideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringFromArray")
	}
	return findFirstEntry(list, key, stringTypeName), nil
}

func intFromArrayByIndex(_ environment, args ...rideType) (rideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "intFromArrayByIndex")
	}
	switch te := list[i].(type) {
	case rideDataEntry:
		if v, err := te.get(valueField); err == nil && v.instanceOf() == intTypeName {
			return v, nil
		}
		return nil, errors.Errorf("intFromArrayByIndex: unexpected value type %q of data entry", te.value.instanceOf())
	case rideIntegerEntry:
		return te.get(valueField)
	default:
		return nil, errors.Errorf("intFromArrayByIndex: unexpected type of list item %q", te.instanceOf())
	}
}

func booleanFromArrayByIndex(_ environment, args ...rideType) (rideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanFromArrayByIndex")
	}
	switch te := list[i].(type) {
	case rideDataEntry:
		if v, err := te.get(valueField); err == nil && v.instanceOf() == booleanTypeName {
			return v, nil
		}
		return nil, errors.Errorf("booleanFromArrayByIndex: unexpected value type %q of data entry", te.value.instanceOf())
	case rideBooleanEntry:
		return te.get(valueField)
	default:
		return nil, errors.Errorf("booleanFromArrayByIndex: unexpected type of list item %q", te.instanceOf())
	}
}

func bytesFromArrayByIndex(_ environment, args ...rideType) (rideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesFromArrayByIndex")
	}
	switch te := list[i].(type) {
	case rideDataEntry:
		if v, err := te.get(valueField); err == nil && v.instanceOf() == byteVectorTypeName {
			return v, nil
		}
		return nil, errors.Errorf("bytesFromArrayByIndex: unexpected value type %q of data entry", te.value.instanceOf())
	case rideBinaryEntry:
		return te.get(valueField)
	default:
		return nil, errors.Errorf("bytesFromArrayByIndex: unexpected type of list item %q", te.instanceOf())
	}
}

func stringFromArrayByIndex(_ environment, args ...rideType) (rideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringFromArrayByIndex")
	}
	switch te := list[i].(type) {
	case rideDataEntry:
		if v, err := te.get(valueField); err == nil && v.instanceOf() == stringTypeName {
			return v, nil
		}
		return nil, errors.Errorf("stringFromArrayByIndex: unexpected value type %q of data entry", te.value.instanceOf())
	case rideStringEntry:
		return te.get(valueField)
	default:
		return nil, errors.Errorf("stringFromArrayByIndex: unexpected type of list item %q", te.instanceOf())
	}
}

func sizeList(_ environment, args ...rideType) (rideType, error) {
	l, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeList")
	}
	return rideInt(len(l)), nil
}

func getList(_ environment, args ...rideType) (rideType, error) {
	l, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "getList")
	}
	return l[i], nil
}

func createList(_ environment, args ...rideType) (rideType, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("createList: %d is invalid number of arguments, expected %d", len(args), 2)
	}
	if args[0] == nil {
		return nil, errors.Errorf("createList: empty head")
	}
	if args[1] == nil {
		return rideList{args[0]}, nil
	}
	tail, ok := args[1].(rideList)
	if !ok {
		return nil, errors.Errorf("createList: unexpected argument type '%s'", args[1].instanceOf())
	}
	if len(tail) == 0 {
		return rideList{args[0]}, nil
	}
	return append(rideList{args[0]}, tail...), nil
}

func intValueFromArray(env environment, args ...rideType) (rideType, error) {
	v, err := intFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromArray(env environment, args ...rideType) (rideType, error) {
	v, err := booleanFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromArray(env environment, args ...rideType) (rideType, error) {
	v, err := bytesFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromArray(env environment, args ...rideType) (rideType, error) {
	v, err := stringFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func intValueFromArrayByIndex(env environment, args ...rideType) (rideType, error) {
	v, err := intFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromArrayByIndex(env environment, args ...rideType) (rideType, error) {
	v, err := booleanFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromArrayByIndex(env environment, args ...rideType) (rideType, error) {
	v, err := bytesFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromArrayByIndex(env environment, args ...rideType) (rideType, error) {
	v, err := stringFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func limitedCreateList(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "limitedCreateList")
	}
	tail, ok := args[1].(rideList)
	if !ok {
		return nil, errors.Errorf("limitedCreateList: unexpected argument type '%s'", args[1].instanceOf())
	}
	if len(tail) == maxListSize {
		return nil, RuntimeError.Errorf("limitedCreateList: resulting list size exceeds %d elements", maxListSize)
	}
	if len(tail) == 0 {
		return rideList{args[0]}, nil
	}
	return append(rideList{args[0]}, tail...), nil
}

func appendToList(_ environment, args ...rideType) (rideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "appendToList")
	}
	if len(list) == maxListSize {
		return nil, errors.Errorf("appendToList: resulting list size exceeds %d elements", maxListSize)
	}
	if len(list) == 0 {
		return rideList{e}, nil
	}
	return append(list, e), nil
}

func concatList(_ environment, args ...rideType) (rideType, error) {
	list1, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "concatList")
	}
	list2, ok := e.(rideList)
	if !ok {
		return nil, errors.Errorf("concatList: unexpected argument type '%s'", args[1])
	}
	l1 := len(list1)
	l2 := len(list2)
	if l1+l2 > maxListSize {
		return nil, errors.Errorf("concatList: resulting list size exceeds %d elements", maxListSize)
	}
	r := make(rideList, l1+l2)
	copy(r, list1)
	copy(r[l1:], list2)
	return r, nil
}

func indexOfList(_ environment, args ...rideType) (rideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "indexOfList")
	}
	if len(list) > maxListSize {
		return nil, errors.Errorf("indexOfList: list size exceeds %d elements", maxListSize)
	}
	if i := indexOfElement(list, e); i >= 0 {
		return rideInt(i), nil
	}
	return rideUnit{}, nil // not found returns Unit
}

func lastIndexOfList(_ environment, args ...rideType) (rideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "lastIndexOfList")
	}
	if len(list) > maxListSize {
		return nil, errors.Errorf("lastIndexOfList: list size exceeds %d elements", maxListSize)
	}
	if i := lastIndexOfElement(list, e); i >= 0 {
		return rideInt(i), nil
	}
	return rideUnit{}, nil // not found returns Unit
}

func median(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "median")
	}
	size := len(list)
	if size > maxListSize || size < 2 {
		return nil, errors.Errorf("median: invalid list size %d", size)
	}
	var buf [medianStackBufferSize]int
	items := buf[:0]
	if size > len(buf) {
		items = make([]int, 0, size)
	}
	items, err = appendInts(items, list)
	if err != nil {
		return nil, errors.Wrap(err, "median")
	}
	slices.Sort(items)
	half := size / 2
	if size%2 == 1 {
		return rideInt(items[half]), nil
	} else {
		return rideInt(math.FloorDiv(int64(items[half-1])+int64(items[half]), 2)), nil
	}
}

func listMax(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "listMax")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("listMax: invalid list size %d", size)
	}
	_, mx, err := intListMinMax(list)
	if err != nil {
		return nil, errors.Wrap(err, "listMax")
	}
	return rideInt(mx), nil
}

func listMin(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "listMin")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("listMin: invalid list size %d", size)
	}
	mn, _, err := intListMinMax(list)
	if err != nil {
		return nil, errors.Wrap(err, "listMin")
	}
	return rideInt(mn), nil
}

func containsElement(_ environment, args ...rideType) (rideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "containsElement")
	}
	return rideBoolean(indexOfElement(list, e) >= 0), nil
}

func listRemoveByIndex(_ environment, args ...rideType) (rideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "listRemoveByIndex")
	}
	l := len(list)
	if l == 0 {
		return nil, errors.New("listRemoveByIndex: can't remove an element from empty list")
	}
	r := make(rideList, l-1)
	copy(r, list[:i])
	copy(r[i:], list[i+1:])
	return r, nil
}

func listReplaceByIndex(_ environment, args ...rideType) (rideType, error) {
	list, i, el, err := listAndIntAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "listReplaceByIndex")
	}
	l := len(list)
	if l == 0 {
		return nil, errors.New("listReplaceByIndex: can't replace an element in the empty list")
	}
	r := make(rideList, l)
	copy(r, list)
	r[i] = el
	return r, nil
}

// fillList creates a list of the specified length, filled with the specified element.
func fillList(_ environment, args ...rideType) (rideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "fill")
	}
	l, ok := args[0].(rideInt)
	if !ok {
		return nil, errors.Errorf("fill: unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	if l <= 0 || l > maxListSize {
		return nil, errors.New("fill: invalid length of list")
	}

	e := args[1]

	r := make(rideList, l)
	for i := range l {
		r[i] = e
	}
	return r, nil
}

func findFirstEntry(list rideList, key rideString, expectedValueType string) rideType {
	for _, item := range list {
		switch ti := item.(type) {
		case rideDataEntry:
			if ti.key == key && ti.value.instanceOf() == expectedValueType {
				return ti.value
			}
		case rideIntegerEntry:
			if ti.key == key && expectedValueType == intTypeName {
				return ti.value
			}
		case rideBooleanEntry:
			if ti.key == key && expectedValueType == booleanTypeName {
				return ti.value
			}
		case rideBinaryEntry:
			if ti.key == key && expectedValueType == byteVectorTypeName {
				return ti.value
			}
		case rideStringEntry:
			if ti.key == key && expectedValueType == stringTypeName {
				return ti.value
			}
		}
	}
	return rideUnit{}
}

// indexOfElement returns the index of the first list item equal to e, or -1 if there is no such item.
// A plain loop with a direct method call is notably faster here than slices.IndexFunc(list, e.eq).
func indexOfElement(list rideList, e rideType) int {
	for i := range list {
		if e.eq(list[i]) {
			return i
		}
	}
	return -1
}

// lastIndexOfElement returns the index of the last list item equal to e, or -1 if there is no such item.
// A plain loop is about two times faster here than iterating over slices.Backward(list).
func lastIndexOfElement(list rideList, e rideType) int {
	for i := len(list); i > 0; i-- {
		if e.eq(list[i-1]) {
			return i - 1
		}
	}
	return -1
}

// appendInts appends the values of the list of Int elements to items.
// It fails on the first element that is not an Int.
func appendInts(items []int, list rideList) ([]int, error) {
	for _, el := range list {
		item, ok := el.(rideInt)
		if !ok {
			return nil, unexpectedListElementTypeError(el)
		}
		items = append(items, int(item))
	}
	return items, nil
}

// intListMinMax returns the minimal and maximal values of the non-empty list of Int elements.
// It fails on the first element that is not an Int.
func intListMinMax(list rideList) (int, int, error) {
	if len(list) == 0 {
		panic("empty list")
	}
	var mn, mx int
	for i, el := range list {
		item, ok := el.(rideInt)
		if !ok {
			return 0, 0, unexpectedListElementTypeError(el)
		}
		v := int(item)
		if i == 0 {
			mn, mx = v, v
			continue
		}
		mn = min(mn, v)
		mx = max(mx, v)
	}
	return mn, mx, nil
}

func unexpectedListElementTypeError(el rideType) error {
	return errors.Errorf("unexpected type of list element '%s'", el.instanceOf())
}
