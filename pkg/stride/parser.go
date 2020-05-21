package stride

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
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

type Parser struct {
	r           *bytes.Reader
	seenBlockV2 bool
	id          [32]byte
}

func NewParser(source []byte) (*Parser, error) {
	id := sha256.Sum256(source)
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
	return &Parser{r: bytes.NewReader(src), id: id}, nil
}

func (p *Parser) Parse() (*SourceTree, error) {
	vb, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch v := int(vb); v {
	case 0:
		return p.parseDApp()
	case 1, 2, 3:
		return p.parseScript(v)
	default:
		return nil, errors.Errorf("unsupported script version %d", v)
	}
}

func (p *Parser) parseDApp() (*SourceTree, error) {
	av, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	lv, err := p.r.ReadByte()
	if err != nil {
		return nil, err
	}
	tree := &SourceTree{
		AppVersion: int(av),
		LibVersion: int(lv),
	}
	meta, err := p.readMeta()
	if err != nil {
		return nil, err
	}
	tree.Meta = meta

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
		if node.Kind != FunctionDeclarationExpression {
			return nil, errors.Errorf("unexpected type of declaration %d", node.Kind)
		}
		node.invocationParameter = invocationParameter
		functions[i] = node
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
		if node.Kind != FunctionDeclarationExpression {
			return nil, errors.Errorf("unexpected type of declaration %d", node.Kind)
		}
		node.invocationParameter = invocationParameter
		tree.Verifier = node
	}
	tree.HasBlockV2 = p.seenBlockV2
	tree.Digest = p.id
	return tree, nil
}

func (p *Parser) parseScript(v int) (*SourceTree, error) {
	tree := &SourceTree{
		AppVersion: scriptApplicationVersion,
		LibVersion: v,
	}
	node, err := p.parse()
	if err != nil {
		return nil, err
	}
	tree.Verifier = node
	tree.HasBlockV2 = p.seenBlockV2
	tree.Digest = p.id
	return tree, nil
}

func (p *Parser) parse() (Node, error) {
	t, err := p.r.ReadByte()
	if err != nil {
		return Node{}, err
	}
	switch t {
	case tokenLong:
		v, err := p.readLong()
		if err != nil {
			return Node{}, err
		}
		return NewLongLiteral(v), nil

	case tokenBytes:
		v, err := p.readBytes()
		if err != nil {
			return Node{}, err
		}
		return NewBytesLiteral(v), nil

	case tokenString:
		v, err := p.readString()
		if err != nil {
			return Node{}, err
		}
		return NewStringLiteral(v), nil

	case tokenIf:
		condition, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		trueBranch, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		falseBranch, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		return NewIfExpression(condition, trueBranch, falseBranch), nil

	case tokenBlockV1:
		name, err := p.readString()
		if err != nil {
			return Node{}, err
		}
		expr, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		body, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		n := NewLetExpression(name, expr)
		n.Siblings = append(n.Siblings, body)
		return n, nil

	case tokenRef:
		name, err := p.readString()
		if err != nil {
			return Node{}, err
		}
		return NewReferenceExpression(name), nil

	case tokenTrue:
		return NewBooleanLiteral(true), nil

	case tokenFalse:
		return NewBooleanLiteral(false), nil

	case tokenGetter:
		object, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		field, err := p.readString()
		if err != nil {
			return Node{}, err
		}
		return NewGetterExpression(object, field), nil

	case tokenFunctionCall:
		ft, err := p.r.ReadByte()
		if err != nil {
			return Node{}, err
		}
		name, err := p.readFunctionName(ft)
		if err != nil {
			return Node{}, err
		}
		argumentsCount, err := p.readInt()
		if err != nil {
			return Node{}, err
		}
		ac := int(argumentsCount)
		arguments := make([]Node, ac)
		for i := 0; i < int(ac); i++ {
			arg, err := p.parse()
			if err != nil {
				return Node{}, err
			}
			arguments[i] = arg
		}
		return NewFunctionCallExpression(name, arguments), nil

	case tokenBlockV2:
		p.seenBlockV2 = true
		declaration, err := p.readDeclaration()
		if err != nil {
			return Node{}, err
		}
		body, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		declaration.Siblings = append(declaration.Siblings, body)
		return declaration, nil

	default:
		return Node{}, errors.Errorf("unsupported token %x", t)
	}

}

func (p *Parser) readShort() (int16, error) {
	buf := make([]byte, 2)
	_, err := p.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf)), nil
}

func (p *Parser) readInt() (int32, error) {
	buf := make([]byte, 4)
	_, err := p.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf)), nil
}

func (p *Parser) readLong() (int64, error) {
	buf := make([]byte, 8)
	_, err := p.r.Read(buf)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf)), nil
}

func (p *Parser) readBytes() ([]byte, error) {
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

func (p *Parser) readString() (string, error) {
	b, err := p.readBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *Parser) readFunctionName(ft byte) (string, error) {
	switch ft {
	case functionTypeNative:
		id, err := p.readShort()
		if err != nil {
			return "", err
		}
		return strconv.Itoa(int(id)), nil
	case functionTypeUser:
		return p.readString()
	default:
		return "", errors.Errorf("unsupported function type %d", ft)
	}
}

func (p *Parser) readDeclaration() (Node, error) {
	dt, err := p.r.ReadByte()
	if err != nil {
		return Node{}, err
	}
	switch dt {
	case declarationTypeLet:
		name, err := p.readString()
		if err != nil {
			return Node{}, err
		}
		exp, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		return NewLetExpression(name, exp), nil
	case declarationTypeFunction:
		name, err := p.readString()
		if err != nil {
			return Node{}, err
		}
		argumentsCount, err := p.readInt()
		if err != nil {
			return Node{}, err
		}
		ac := int(argumentsCount)
		arguments := make([]string, ac)
		for i := 0; i < ac; i++ {
			arg, err := p.readString()
			if err != nil {
				return Node{}, err
			}
			arguments[i] = arg
		}
		body, err := p.parse()
		if err != nil {
			return Node{}, err
		}
		return NewFunctionDeclarationExpression(name, arguments, body), nil
	default:
		return Node{}, errors.Errorf("unsupported declaration type %d", dt)
	}
}

func (p *Parser) readMeta() (ScriptMeta, error) {
	v, err := p.readInt()
	if err != nil {
		return ScriptMeta{}, err
	}
	b, err := p.readBytes()
	if err != nil {
		return ScriptMeta{}, err
	}
	return ScriptMeta{
		Version: int(v),
		Bytes:   b,
	}, nil
}
