package ride

import (
	"bytes"
)

func SerializeTreeV1(tree *Tree) ([]byte, error) {
	s := serializerV1{
		buf: bytes.Buffer{},
	}
	return s.serialize(tree)
}

func SerializeTreeV2(tree *Tree) ([]byte, error) {
	s := serializerV2{
		buf: bytes.Buffer{},
	}
	return s.serialize(tree)
}
