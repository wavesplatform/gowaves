package main

type ByteVector struct {
	base  string
	value []byte
}

func NewByteVector(base string, value []byte) (ByteVector, error) {
	return ByteVector{
		base:  base,
		value: value,
	}, nil
}

type Declaration struct {
	name  string
	value interface{}
}

func NewDeclaration(name string, value interface{}) Declaration {
	return Declaration{
		name:  name,
		value: value,
	}
}
