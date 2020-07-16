package fride

import "github.com/pkg/errors"

type value interface {
	value()
}

type boolean bool

func (b boolean) value() {}

type long int64

func (l long) value() {}

type str string

func (s str) value() {}

type bytes []byte

func (b bytes) value() {}

type object map[str]value

func (o object) value() {}

type call struct {
	name  string
	count int
}

func (c call) value() {}

func fetch(from value, prop value) (value, error) {
	obj, ok := from.(object)
	if ok {
		name, ok := prop.(str)
		if !ok {
			return nil, errors.Errorf("unable to fetch by property of invalid type '%T'", prop)
		}
		prop, ok := obj[name]
		if ok {
			return prop, nil
		}
	}
	return nil, errors.Errorf("unable to fetch from non object type '%T'", from)
}
