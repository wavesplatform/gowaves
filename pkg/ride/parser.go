package ride

import (
	"bytes"
	sh256 "crypto/sha256"
	"encoding/binary"
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	scriptApplicationVersion = 1
)

const (
	tokenLong byte = iota
	tokenBytes
	tokenString
	tokenIf
	tokenBlockV1
	tokenRef
	tokenTrue
	tokenFalse
	tokenGetter
	tokenFunctionCall
	tokenBlockV2
)

const (
	functionTypeNative byte = iota
	functionTypeUser
)

const (
	declarationTypeLet byte = iota
	declarationTypeFunction
)

func Parse(source []byte) (*Tree, error) {
	p, err := newParser(source)
	if err != nil {
		return nil, err
	}
	return p.parse()
}

type parser struct {
	r           *bytes.Reader
	seenBlockV2 bool
	id          [32]byte
}

func newParser(source []byte) (*parser, error) {
	id := sh256.Sum256(source)
	size := len(source) - 4
	if size <= 0 {
		return nil, errors.Errorf("invalid source length %d", size)
	}
	src, cs := source[:size], source[size:]
	digest, err := crypto.SecureHash(src)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(digest[:4], cs) {
		return nil, errors.New("invalid source checksum")
	}
	return &parser{r: bytes.NewReader(src), id: id}, nil
}

func (p *parser) parse() (*Tree, error) {
	vb, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch v := int(vb); v {
	case 0:
		return p.parseDApp()
	case 1, 2, 3, 4, 5:
		return p.parseScript(v)
	default:
		return nil, errors.Errorf("unsupported script version %d", v)
	}
}

func (p *parser) parseDApp() (*Tree, error) {
	av, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	lv, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	tree := &Tree{
		AppVersion: int(av),
		LibVersion: int(lv),
	}
	m, err := p.readMeta()
	if err != nil {
		return nil, err
	}
	tree.Meta = m

	n, err := p.readInt()
	if err != nil {
		return nil, err
	}
	declarations := make([]Node, n)
	for i := 0; i < int(n); i++ {
		d, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		declarations[i] = d
	}
	tree.Declarations = declarations

	n, err = p.readInt()
	if err != nil {
		return nil, err
	}
	functions := make([]Node, n)
	for i := 0; i < int(n); i++ {
		invocationParameter, err := p.readString()
		if err != nil {
			return nil, err
		}
		node, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		fn, ok := node.(*FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("unexpected type of declaration %T", node)
		}
		fn.invocationParameter = invocationParameter
		functions[i] = fn
		// Update callable name in tree's meta
		if len(tree.Meta.Functions) > i {
			tree.Meta.Functions[i].Name = fn.Name
		}
	}
	tree.Functions = functions

	n, err = p.readInt()
	if err != nil {
		return nil, err
	}
	if n != 0 {
		invocationParameter, err := p.readString()
		if err != nil {
			return nil, err
		}
		node, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		fn, ok := node.(*FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("unexpected type of declaration %T", node)
		}
		fn.invocationParameter = invocationParameter
		tree.Verifier = fn
	}
	tree.HasBlockV2 = p.seenBlockV2
	tree.Digest = p.id
	return tree, nil
}

func (p *parser) parseScript(v int) (*Tree, error) {
	tree := &Tree{
		AppVersion: scriptApplicationVersion,
		LibVersion: v,
	}
	node, err := p.parseNext()
	if err != nil {
		return nil, err
	}
	tree.Verifier = node
	tree.HasBlockV2 = p.seenBlockV2
	tree.Digest = p.id
	return tree, nil
}

func (p *parser) parseNext() (Node, error) {
	t, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch t {
	case tokenLong:
		v, err := p.readLong()
		if err != nil {
			return nil, err
		}
		return NewLongNode(v), nil

	case tokenBytes:
		v, err := p.readBytes()
		if err != nil {
			return nil, err
		}
		return NewBytesNode(v), nil

	case tokenString:
		v, err := p.readString()
		if err != nil {
			return nil, err
		}
		return NewStringNode(v), nil

	case tokenIf:
		condition, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		trueBranch, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		falseBranch, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		return NewConditionalNode(condition, trueBranch, falseBranch), nil

	case tokenBlockV1:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		expr, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		block, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		return NewAssignmentNode(name, expr, block), nil

	case tokenRef:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		return NewReferenceNode(name), nil

	case tokenTrue:
		return NewBooleanNode(true), nil

	case tokenFalse:
		return NewBooleanNode(false), nil

	case tokenGetter:
		object, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		field, err := p.readString()
		if err != nil {
			return nil, err
		}
		return NewPropertyNode(field, object), nil

	case tokenFunctionCall:
		ft, err := p.r.ReadByte()
		if err != nil {
			return nil, err
		}
		function, err := p.readFunctionName(ft)
		if err != nil {
			return nil, err
		}
		argumentsCount, err := p.readInt()
		if err != nil {
			return nil, err
		}
		ac := int(argumentsCount)
		arguments := make([]Node, ac)
		for i := 0; i < ac; i++ {
			arg, err := p.parseNext()
			if err != nil {
				return nil, err
			}
			arguments[i] = arg
		}
		return NewFunctionCallNode(function, arguments), nil

	case tokenBlockV2:
		p.seenBlockV2 = true
		declaration, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		block, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		declaration.SetBlock(block)
		return declaration, nil

	default:
		return nil, errors.Errorf("unsupported token %x", t)
	}

}

func (p *parser) readShort() (int16, error) {
	buf := make([]byte, 2)
	_, err := p.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf)), nil
}

func (p *parser) readInt() (int32, error) {
	buf := make([]byte, 4)
	_, err := p.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf)), nil
}

func (p *parser) readLong() (int64, error) {
	buf := make([]byte, 8)
	_, err := p.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf)), nil
}

func (p *parser) readBytes() ([]byte, error) {
	n, err := p.readInt()
	if err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	if n > 0 {
		_, err = p.r.Read(buf)
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}

func (p *parser) readString() (string, error) {
	b, err := p.readBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *parser) readFunctionName(ft byte) (function, error) {
	switch ft {
	case functionTypeNative:
		id, err := p.readShort()
		if err != nil {
			return nil, err
		}
		return nativeFunction(strconv.Itoa(int(id))), nil
	case functionTypeUser:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		return userFunction(name), nil
	default:
		return nil, errors.Errorf("unsupported function type %d", ft)
	}
}

func (p *parser) readDeclaration() (Node, error) {
	dt, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch dt {
	case declarationTypeLet:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		exp, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		return NewAssignmentNode(name, exp, nil), nil
	case declarationTypeFunction:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		argumentsCount, err := p.readInt()
		if err != nil {
			return nil, err
		}
		ac := int(argumentsCount)
		arguments := make([]string, ac)
		for i := 0; i < ac; i++ {
			arg, err := p.readString()
			if err != nil {
				return nil, err
			}
			arguments[i] = arg
		}
		body, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		return NewFunctionDeclarationNode(name, arguments, body, nil), nil
	default:
		return nil, errors.Errorf("unsupported declaration type %d", dt)
	}
}

func (p *parser) readMeta() (meta.DApp, error) {
	v, err := p.readInt()
	if err != nil {
		return meta.DApp{}, err
	}
	b, err := p.readBytes()
	if err != nil {
		return meta.DApp{}, err
	}
	switch v {
	case 0:
		pbMeta := new(g.DAppMeta)
		if err := protobuf.Unmarshal(b, pbMeta); err != nil {
			return meta.DApp{}, err
		}
		m, err := meta.Convert(pbMeta)
		if err != nil {
			return meta.DApp{}, err
		}
		return m, nil
	default:
		return meta.DApp{}, errors.Errorf("unsupported script meta version %d", v)
	}
}

func IsDApp(data []byte) bool {
	return len(data) > 0 && data[0] == 0
}
