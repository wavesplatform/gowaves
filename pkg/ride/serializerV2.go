package ride

import (
	"bytes"
	"encoding/binary"
)

func serializeDAppV2(s *serializer, tree *Tree) error {
	if err := s.writeByte(byte(tree.LibVersion)); err != nil {
		return err
	}
	if err := s.writeByte(byte(tree.contentType)); err != nil {
		return err
	}
	if err := s.writeMeta(tree.Meta); err != nil {
		return err
	}
	if err := s.writeDeclarations(tree.Declarations); err != nil {
		return err
	}
	if err := s.writeFunctions(tree.Functions); err != nil {
		return err
	}
	if err := s.writeVerifier(tree.Verifier); err != nil {
		return err
	}
	return nil
}

func serializeScriptV2(s *serializer, tree *Tree) error {
	if err := s.writeByte(byte(tree.LibVersion)); err != nil {
		return err
	}
	if err := s.writeByte(byte(tree.contentType)); err != nil {
		return err
	}
	if err := s.walk(tree.Verifier); err != nil {
		return err
	}
	return nil
}

func writeUint16V2(buf *bytes.Buffer, v uint16) error {
	b := [binary.MaxVarintLen16]byte{}
	n := binary.PutUvarint(b[:], uint64(v))
	_, err := buf.Write(b[:n])
	return err
}

func writeUint32V2(buf *bytes.Buffer, v uint32) error {
	b := [binary.MaxVarintLen32]byte{}
	n := binary.PutUvarint(b[:], uint64(v))
	_, err := buf.Write(b[:n])
	return err
}

func writeInt64V2(buf *bytes.Buffer, v int64) error {
	b := [binary.MaxVarintLen64]byte{}
	n := binary.PutUvarint(b[:], uint64(v))
	_, err := buf.Write(b[:n])
	return err
}
