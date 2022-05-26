package serialization

import (
	"bytes"
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	protobuf "google.golang.org/protobuf/proto"
)

func serializeDAppV1(s *serializer, tree *ast.Tree) error {
	if err := s.writeByte(0x00); err != nil {
		return err
	}
	if err := s.writeByte(byte(tree.ContentType)); err != nil {
		return err
	}
	if err := s.writeByte(byte(tree.LibVersion)); err != nil {
		return err
	}
	if err := s.writeMeta(s, tree.Meta); err != nil {
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

func serializeScriptV1(s *serializer, tree *ast.Tree) error {
	if err := s.writeByte(byte(tree.LibVersion)); err != nil {
		return err
	}
	if err := s.walk(tree.Verifier); err != nil {
		return err
	}
	return nil
}

func writeUint16V1(buf *bytes.Buffer, v uint16) error {
	b := [2]byte{}
	binary.BigEndian.PutUint16(b[:], v)
	_, err := buf.Write(b[:])
	return err
}

func writeUint32V1(buf *bytes.Buffer, v uint32) error {
	b := [4]byte{}
	binary.BigEndian.PutUint32(b[:], v)
	_, err := buf.Write(b[:])
	return err
}

func writeInt64V1(buf *bytes.Buffer, v int64) error {
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], uint64(v))
	_, err := buf.Write(b[:])
	return err
}

func writeMetaV1(s *serializer, m meta.DApp) error {
	if err := s.writeUint32(s.buf, 0); err != nil { // Meta version is always 0
		return err
	}
	pbMeta, err := meta.Build(m)
	if err != nil {
		return err
	}
	mb, err := protobuf.Marshal(pbMeta)
	if err != nil {
		return err
	}
	if err := s.writeBytes(mb); err != nil {
		return err
	}
	return nil
}
