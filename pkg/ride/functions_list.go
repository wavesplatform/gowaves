package ride

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/math"
)

const maxListSize = 1000

func listAndStringArgs(args []RideType) (RideList, RideString, error) {
	if len(args) != 2 {
		return nil, "", errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, "", errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, "", errors.Errorf("argument 2 is empty")
	}
	l, ok := args[0].(RideList)
	if !ok {
		return nil, "", errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	s, ok := args[1].(RideString)
	if !ok {
		return nil, "", errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	return l, s, nil
}

func listAndIntArgs(args []RideType) (RideList, int, error) {
	if len(args) != 2 {
		return nil, 0, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, 0, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, 0, errors.Errorf("argument 2 is empty")
	}
	l, ok := args[0].(RideList)
	if !ok {
		return nil, 0, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	ri, ok := args[1].(RideInt)
	if !ok {
		return nil, 0, errors.Errorf("unexpected type of argument 2 '%s'", args[1].instanceOf())
	}
	i := int(ri)
	if i < 0 || i >= len(l) {
		return nil, 0, errors.Errorf("invalid index %d", i)
	}
	return l, i, nil
}

func listArg(args []RideType) (RideList, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("%d is invalid number of arguments, expected 1", len(args))
	}
	if args[0] == nil {
		return nil, errors.Errorf("argument is empty")
	}
	l, ok := args[0].(RideList)
	if !ok {
		return nil, errors.Errorf("unexpected type of argument '%s'", args[0].instanceOf())
	}
	return l, nil
}

func listAndElementArgs(args []RideType) (RideList, RideType, error) {
	if len(args) != 2 {
		return nil, nil, errors.Errorf("%d is invalid number of arguments, expected 2", len(args))
	}
	if args[0] == nil {
		return nil, nil, errors.Errorf("argument 1 is empty")
	}
	if args[1] == nil {
		return nil, nil, errors.Errorf("argument 2 is empty")
	}
	l, ok := args[0].(RideList)
	if !ok {
		return nil, nil, errors.Errorf("unexpected type of argument 1 '%s'", args[0].instanceOf())
	}
	return l, args[1], nil
}

func intFromArray(_ Environment, args ...RideType) (RideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "intFromArray")
	}
	item, err := findItem(list, key, "IntegerEntry", "Int")
	if err != nil {
		return nil, errors.Wrap(err, "intFromArray")
	}
	return item, nil
}

func booleanFromArray(_ Environment, args ...RideType) (RideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanFromArray")
	}
	item, err := findItem(list, key, "BooleanEntry", "Boolean")
	if err != nil {
		return nil, errors.Wrap(err, "booleanFromArray")
	}
	return item, nil
}

func bytesFromArray(_ Environment, args ...RideType) (RideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesFromArray")
	}
	item, err := findItem(list, key, "BinaryEntry", "ByteVector")
	if err != nil {
		return nil, errors.Wrap(err, "bytesFromArray")
	}
	return item, nil
}

func stringFromArray(_ Environment, args ...RideType) (RideType, error) {
	list, key, err := listAndStringArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringFromArray")
	}
	item, err := findItem(list, key, "StringEntry", "String")
	if err != nil {
		return nil, errors.Wrap(err, "stringFromArray")
	}
	return item, nil
}

func intFromArrayByIndex(_ Environment, args ...RideType) (RideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "intFromArrayByIndex")
	}
	e := list[i]
	o, ok := e.(rideObject)
	if !ok {
		return nil, errors.Errorf("intFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
	switch {
	case o.instanceOf() == "DataEntry" && o["value"].instanceOf() == "Int":
		return o["value"], nil
	case o.instanceOf() == "IntegerEntry":
		return o["value"], nil
	default:
		return nil, errors.Errorf("intFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
}

func booleanFromArrayByIndex(_ Environment, args ...RideType) (RideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "booleanFromArrayByIndex")
	}
	e := list[i]
	o, ok := e.(rideObject)
	if !ok {
		return nil, errors.Errorf("booleanFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
	switch {
	case o.instanceOf() == "DataEntry" && o["value"].instanceOf() == "Boolean":
		return o["value"], nil
	case o.instanceOf() == "BooleanEntry":
		return o["value"], nil
	default:
		return nil, errors.Errorf("booleanFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
}

func bytesFromArrayByIndex(_ Environment, args ...RideType) (RideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "bytesFromArrayByIndex")
	}
	e := list[i]
	o, ok := e.(rideObject)
	if !ok {
		return nil, errors.Errorf("bytesFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
	switch {
	case o.instanceOf() == "DataEntry" && o["value"].instanceOf() == "ByteVector":
		return o["value"], nil
	case o.instanceOf() == "BinaryEntry":
		return o["value"], nil
	default:
		return nil, errors.Errorf("bytesFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
}

func stringFromArrayByIndex(_ Environment, args ...RideType) (RideType, error) {
	list, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "stringFromArrayByIndex")
	}
	e := list[i]
	o, ok := e.(rideObject)
	if !ok {
		return nil, errors.Errorf("stringFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
	switch {
	case o.instanceOf() == "DataEntry" && o["value"].instanceOf() == "String":
		return o["value"], nil
	case o.instanceOf() == "StringEntry":
		return o["value"], nil
	default:
		return nil, errors.Errorf("stringFromArrayByIndex: unexpected type of list item '%s'", e.instanceOf())
	}
}

func sizeList(_ Environment, args ...RideType) (RideType, error) {
	l, err := listArg(args)
	if err != nil {
		return nil, errors.Wrap(err, "sizeList")
	}
	return RideInt(len(l)), nil
}

func getList(_ Environment, args ...RideType) (RideType, error) {
	l, i, err := listAndIntArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "getList")
	}
	return l[i], nil
}

func createList(_ Environment, args ...RideType) (RideType, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("createList: %d is invalid number of arguments, expected %d", len(args), 2)
	}
	if args[0] == nil {
		return nil, errors.Errorf("createList: empty head")
	}
	if args[1] == nil {
		return RideList{args[0]}, nil
	}
	tail, ok := args[1].(RideList)
	if !ok {
		return nil, errors.Errorf("createList: unexpected argument type '%s'", args[1].instanceOf())
	}
	if len(tail) == 0 {
		return RideList{args[0]}, nil
	}
	return append(RideList{args[0]}, tail...), nil
}

func intValueFromArray(env Environment, args ...RideType) (RideType, error) {
	v, err := intFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromArray(env Environment, args ...RideType) (RideType, error) {
	v, err := booleanFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromArray(env Environment, args ...RideType) (RideType, error) {
	v, err := bytesFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromArray(env Environment, args ...RideType) (RideType, error) {
	v, err := stringFromArray(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func intValueFromArrayByIndex(env Environment, args ...RideType) (RideType, error) {
	v, err := intFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func booleanValueFromArrayByIndex(env Environment, args ...RideType) (RideType, error) {
	v, err := booleanFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func bytesValueFromArrayByIndex(env Environment, args ...RideType) (RideType, error) {
	v, err := bytesFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func stringValueFromArrayByIndex(env Environment, args ...RideType) (RideType, error) {
	v, err := stringFromArrayByIndex(env, args...)
	if err != nil {
		return nil, err
	}
	return extractValue(v)
}

func limitedCreateList(_ Environment, args ...RideType) (RideType, error) {
	if err := checkArgs(args, 2); err != nil {
		return nil, errors.Wrap(err, "limitedCreateList")
	}
	tail, ok := args[1].(RideList)
	if !ok {
		return nil, errors.Errorf("limitedCreateList: unexpected argument type '%s'", args[1].instanceOf())
	}
	if len(tail) == maxListSize {
		return nil, errors.Errorf("limitedCreateList: resulting list size exceeds %d elements", maxListSize)
	}
	if len(tail) == 0 {
		return RideList{args[0]}, nil
	}
	return append(RideList{args[0]}, tail...), nil
}

func appendToList(_ Environment, args ...RideType) (RideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "appendToList")
	}
	if len(list) == maxListSize {
		return nil, errors.Errorf("appendToList: resulting list size exceeds %d elements", maxListSize)
	}
	if len(list) == 0 {
		return RideList{e}, nil
	}
	return append(list, e), nil
}

func concatList(_ Environment, args ...RideType) (RideType, error) {
	list1, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "concatList")
	}
	list2, ok := e.(RideList)
	if !ok {
		return nil, errors.Errorf("concatList: unexpected argument type '%s'", args[1])
	}
	l1 := len(list1)
	l2 := len(list2)
	if l1+l2 > maxListSize {
		return nil, errors.Errorf("concatList: resulting list size exceeds %d elements", maxListSize)
	}
	r := make(RideList, l1+l2)
	copy(r, list1)
	copy(r[l1:], list2)
	return r, nil
}

func indexOfList(_ Environment, args ...RideType) (RideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "indexOfList")
	}
	if len(list) > maxListSize {
		return nil, errors.Errorf("indexOfList: list size exceeds %d elements", maxListSize)
	}
	for i := 0; i < len(list); i++ {
		if e.eq(list[i]) {
			return RideInt(i), nil
		}
	}
	return rideUnit{}, nil // not found returns Unit
}

func lastIndexOfList(_ Environment, args ...RideType) (RideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "lastIndexOfList")
	}
	if len(list) > maxListSize {
		return nil, errors.Errorf("lastIndexOfList: list size exceeds %d elements", maxListSize)
	}
	for i := len(list) - 1; i >= 0; i-- {
		if e.eq(list[i]) {
			return RideInt(i), nil
		}
	}
	return rideUnit{}, nil // not found returns Unit
}

func median(_ Environment, args ...RideType) (RideType, error) {
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
		return RideInt(items[half]), nil
	} else {
		return RideInt(math.FloorDiv(int64(items[half-1])+int64(items[half]), 2)), nil
	}
}

func max(_ Environment, args ...RideType) (RideType, error) {
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
	return RideInt(max), nil
}

func min(_ Environment, args ...RideType) (RideType, error) {
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
	return RideInt(min), nil
}

func containsElement(_ Environment, args ...RideType) (RideType, error) {
	list, e, err := listAndElementArgs(args)
	if err != nil {
		return nil, errors.Wrap(err, "containsElement")
	}
	for i := 0; i < len(list); i++ {
		if e.eq(list[i]) {
			return RideBoolean(true), nil
		}
	}
	return RideBoolean(false), nil
}

func listRemoveByIndex(_ Environment, args ...RideType) (RideType, error) {
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
	r := make(RideList, l-1)
	copy(r, list[:i])
	copy(r[i:], list[i+1:])
	return r, nil
}

func findItem(list RideList, key RideString, entryType, valueType string) (RideType, error) {
	for _, item := range list {
		o, ok := item.(rideObject)
		if !ok {
			return nil, errors.Errorf("unexpected type of list item '%s'", item.instanceOf())
		}
		switch o.instanceOf() {
		case "DataEntry":
			if o["key"].eq(key) {
				v := o["value"]
				if v.instanceOf() == valueType {
					return v, nil
				}
			}
		case entryType:
			if o["key"].eq(key) {
				return o["value"], nil
			}
		}
	}
	return rideUnit{}, nil
}

func intSlice(list RideList) ([]int, error) {
	items := make([]int, len(list))
	for i, el := range list {
		item, ok := el.(RideInt)
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
