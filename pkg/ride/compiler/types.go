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
