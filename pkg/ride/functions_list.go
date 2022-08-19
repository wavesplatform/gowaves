package ride

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

const maxListSize = 1000

func listAndStringArgs(args []rideType) (rideList, rideString, error) {
	if len(args) != 2 {
		return nil, "", errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, "", errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, "", errors.Errorf("argument 2 is empty")
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
	if len(args) != 2 {
		return nil, 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, 0, errors.Errorf("argument 2 is empty")
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
	if i < 0 || i >= len(l) {
		return nil, 0, errors.Errorf("invalid index %d", i)
	}
	return l, i, nil
}

func listArg(args []rideType) (rideList, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument is empty")
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, errors.Errorf("unexpected type of argument '%s'", args[0].instanceOf())
	}
	return l, nil
}

func listAndElementArgs(args []rideType) (rideList, rideType, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	l, ok := args[0].(rideList)
	if !ok {
		return nil, nil, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	return l, args[1], nil
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
	return findFirstEntry(list, key, bytesTypeName), nil
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
		if v, err := te.get(valueField); err == nil && v.instanceOf() == bytesTypeName {
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
		return nil, errors.Errorf("limitedCreateList: resulting list size exceeds %d elements", maxListSize)
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
	for i := 0; i < len(list); i++ {
		if e.eq(list[i]) {
			return rideInt(i), nil
		}
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
	for i := len(list) - 1; i >= 0; i-- {
		if e.eq(list[i]) {
			return rideInt(i), nil
		}
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
	items, err := intSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "median")
	}
	sort.Ints(items)
	half := size / 2
	if size%2 == 1 {
		return rideInt(items[half]), nil
	} else {
		return rideInt(math.FloorDiv(int64(items[half-1])+int64(items[half]), 2)), nil
	}
}

func max(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "max")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("max: invalid list size %d", size)
	}
	items, err := intSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "max")
	}
	_, max := minMax(items)
	return rideInt(max), nil
}

func min(_ environment, args ...rideType) (rideType, error) {
	list, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "min")
	}
	size := len(list)
	if size > maxListSize || size == 0 {
		return nil, errors.Errorf("min: invalid list size %d", size)
	}
	items, err := intSlice(list)
	if err != nil {
		return nil, errors.Wrap(err, "min")
	}
	min, _ := minMax(items)
	return rideInt(min), nil
}

func containsElement(_ environment, args ...rideType) (rideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "containsElement")
	}
	for i := 0; i < len(list); i++ {
		if e.eq(list[i]) {
			return rideBoolean(true), nil
		}
	}
	return rideBoolean(false), nil
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
	if i < 0 {
		return nil, errors.Errorf("listRemoveByIndex: negative index value %d", i)
	}
	if i >= l {
		return nil, errors.Errorf("listRemoveByIndex: index out of bounds")
	}
	r := make(rideList, l-1)
	copy(r, list[:i])
	copy(r[i:], list[i+1:])
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
			if ti.key == key && expectedValueType == bytesTypeName {
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

func intSlice(list rideList) ([]int, error) {
	items := make([]int, len(list))
	for i, el := range list {
		item, ok := el.(rideInt)
		if !ok {
			return nil, errors.Errorf("unexpected type of list element '%s'", el.instanceOf())
		}
		items[i] = int(item)
	}
	return items, nil
}

func minMax(items []int) (int, int) {
	if len(items) == 0 {
		panic("empty slice")
	}
	max := items[0]
	min := items[0]
	for _, i := range items {
		if max < i {
			max = i
		}
		if min > i {
			min = i
		}
	}
	return min, max
}
