package serialization

import (
	"bytes"
	"encoding/binary"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	protobuf "google.golang.org/protobuf/proto"
)

func serializeDAppV2(s *serializer, tree *ast.Tree) error {
	if err := s.writeByte(byte(tree.LibVersion)); err != nil {
		return err
	}
	if err := s.writeByte(byte(tree.ContentType)); err != nil {
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

func serializeScriptV2(s *serializer, tree *ast.Tree) error {
	if err := s.writeByte(byte(tree.LibVersion)); err != nil {
		return err
	}
	if err := s.writeByte(byte(tree.ContentType)); err != nil {
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

func writeMetaV2(s *serializer, m meta.DApp) error {
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
