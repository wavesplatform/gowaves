package ride

import (
	"bytes"
	"encoding/binary"
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	protobuf "google.golang.org/protobuf/proto"
)

type serializerV2 struct {
	buf bytes.Buffer
}

func (s *serializerV2) serialize(tree *Tree) ([]byte, error) {
	if tree.IsDApp() {
		if err := s.serializeDApp(tree); err != nil {
			return nil, err
		}
	} else {
		if err := s.serializeScript(tree); err != nil {
			return nil, err
		}
	}
	body := s.buf.Bytes()
	digest, err := crypto.SecureHash(body)
	if err != nil {
		return nil, err
	}
	_, err = s.buf.Write(digest[:4])
	if err != nil {
		return nil, err
	}
	return s.buf.Bytes(), nil
}

func (s *serializerV2) serializeDApp(tree *Tree) error {
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

func (s *serializerV2) serializeScript(tree *Tree) error {
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

func (s *serializerV2) writeMeta(m meta.DApp) error {
	if err := s.writeUint32(0); err != nil { // Meta version is always 0
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

func (s *serializerV2) writeDeclarations(declarations []Node) error {
	if err := s.writeUint32(uint32(len(declarations))); err != nil {
		return err
	}
	for _, d := range declarations {
		if err := s.writeDeclaration(d); err != nil {
			return err
		}
	}
	return nil
}

func (s *serializerV2) writeDeclaration(declaration Node) error {
	switch d := declaration.(type) {
	case *FunctionDeclarationNode:
		if err := s.writeFunctionDeclaration(d); err != nil {
			return err
		}
		return nil
	case *AssignmentNode:
		if err := s.writeByte(declarationTypeLet); err != nil {
			return err
		}
		if err := s.writeAssignmentDeclaration(d); err != nil {
			return err
		}
		return nil
	default:
		return errors.Errorf("unexpected declaration type '%T'", d)
	}
}

func (s *serializerV2) writeFunctions(functions []Node) error {
	if err := s.writeUint32(uint32(len(functions))); err != nil {
		return err
	}
	for _, f := range functions {
		fn, ok := f.(*FunctionDeclarationNode)
		if !ok {
			return errors.Errorf("unexpected function declaration type '%T'", f)
		}
		if err := s.writeFunction(fn); err != nil {
			return err
		}
	}
	return nil
}

func (s *serializerV2) writeVerifier(verifier Node) error {
	if verifier != nil {
		if err := s.writeUint32(1); err != nil {
			return err
		}
		fn, ok := verifier.(*FunctionDeclarationNode)
		if !ok {
			return errors.Errorf("invalid type of verifier node '%T'", verifier)
		}
		if err := s.writeFunction(fn); err != nil {
			return err
		}
		return nil
	}
	if err := s.writeUint32(0); err != nil {
		return err
	}
	return nil
}

func (s *serializerV2) writeFunction(function *FunctionDeclarationNode) error {
	if err := s.writeString(function.invocationParameter); err != nil {
		return err
	}
	if err := s.writeFunctionDeclaration(function); err != nil {
		return err
	}
	return nil
}

func (s *serializerV2) writeAssignmentDeclaration(assignment *AssignmentNode) error {
	if err := s.writeString(assignment.Name); err != nil {
		return err
	}
	return s.walk(assignment.Expression)
}

func (s *serializerV2) writeFunctionDeclaration(function *FunctionDeclarationNode) error {
	if err := s.writeByte(declarationTypeFunction); err != nil {
		return err
	}
	if err := s.writeString(function.Name); err != nil {
		return err
	}
	if err := s.writeUint32(uint32(len(function.Arguments))); err != nil {
		return err
	}
	for _, arg := range function.Arguments {
		if err := s.writeString(arg); err != nil {
			return err
		}
	}
	return s.walk(function.Body)
}

func (s *serializerV2) walk(node Node) error {
	switch n := node.(type) {
	case *LongNode:
		if err := s.writeByte(tokenLong); err != nil {
			return err
		}
		if err := s.writeLong(n.Value); err != nil {
			return err
		}
		return nil
	case *BytesNode:
		if err := s.writeByte(tokenBytes); err != nil {
			return err
		}
		if err := s.writeBytes(n.Value); err != nil {
			return err
		}
		return nil
	case *BooleanNode:
		if n.Value {
			if err := s.writeByte(tokenTrue); err != nil {
				return err
			}
		} else {
			if err := s.writeByte(tokenFalse); err != nil {
				return err
			}
		}
		return nil
	case *StringNode:
		if err := s.writeByte(tokenString); err != nil {
			return err
		}
		if err := s.writeString(n.Value); err != nil {
			return err
		}
		return nil

	case *ConditionalNode:
		if err := s.writeByte(tokenIf); err != nil {
			return err
		}
		if err := s.walk(n.Condition); err != nil {
			return err
		}
		if err := s.walk(n.TrueExpression); err != nil {
			return err
		}
		if err := s.walk(n.FalseExpression); err != nil {
			return err
		}
		return nil

	case *AssignmentNode:
		if n.newBlock {
			if err := s.writeByte(tokenBlockV2); err != nil {
				return err
			}
			if err := s.writeByte(declarationTypeLet); err != nil {
				return err
			}
		} else {
			if err := s.writeByte(tokenBlockV1); err != nil {
				return err
			}
		}
		if err := s.writeAssignmentDeclaration(n); err != nil {
			return err
		}
		return s.walk(n.Block)

	case *ReferenceNode:
		if err := s.writeByte(tokenRef); err != nil {
			return err
		}
		if err := s.writeString(n.Name); err != nil {
			return err
		}
		return nil

	case *FunctionDeclarationNode:
		if err := s.writeByte(tokenBlockV2); err != nil {
			return err
		}
		if err := s.writeDeclaration(n); err != nil {
			return err
		}
		return s.walk(n.Block)

	case *FunctionCallNode:
		if err := s.writeByte(tokenFunctionCall); err != nil {
			return err
		}
		switch tf := n.Function.(type) {
		case nativeFunction:
			if err := s.writeByte(functionTypeNative); err != nil {
				return err
			}
			id, err := strconv.ParseUint(tf.Name(), 10, 16)
			if err != nil {
				return err
			}
			if err := s.writeUint16(uint16(id)); err != nil {
				return err
			}
		case userFunction:
			if err := s.writeByte(functionTypeUser); err != nil {
				return err
			}
			if err := s.writeString(tf.Name()); err != nil {
				return err
			}
		default:
			return errors.Errorf("unsupported function type '%T'", n.Function)
		}
		if err := s.writeUint32(uint32(len(n.Arguments))); err != nil {
			return err
		}
		for _, arg := range n.Arguments {
			if err := s.walk(arg); err != nil {
				return err
			}
		}
		return nil

	case *PropertyNode:
		if err := s.writeByte(tokenGetter); err != nil {
			return err
		}
		if err := s.walk(n.Object); err != nil {
			return err
		}
		if err := s.writeString(n.Name); err != nil {
			return err
		}
		return nil

	default:
		return errors.Errorf("unsupported type of node '%T'", node)
	}
}

func (s *serializerV2) writeByte(b byte) error {
	return s.buf.WriteByte(b)
}

func (s *serializerV2) writeUint16(v uint16) error {
	b := [binary.MaxVarintLen16]byte{}
	n := binary.PutUvarint(b[:], uint64(v))
	_, err := s.buf.Write(b[:n])
	return err
}

func (s *serializerV2) writeUint32(v uint32) error {
	b := [binary.MaxVarintLen32]byte{}
	n := binary.PutUvarint(b[:], uint64(v))
	_, err := s.buf.Write(b[:n])
	return err
}

func (s *serializerV2) writeLong(v int64) error {
	b := [binary.MaxVarintLen64]byte{}
	n := binary.PutUvarint(b[:], uint64(v))
	_, err := s.buf.Write(b[:n])
	return err
}

func (s *serializerV2) writeBytes(data []byte) error {
	if err := s.writeUint32(uint32(len(data))); err != nil {
		return err
	}
	_, err := s.buf.Write(data)
	return err
}

func (s *serializerV2) writeString(str string) error {
	return s.writeBytes([]byte(str))
}
