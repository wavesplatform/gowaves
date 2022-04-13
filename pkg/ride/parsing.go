package ride

import (
	"bytes"
	sh256 "crypto/sha256"
	"strconv"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
)

type contentType byte

const (
	contentTypeExpression contentType = iota + 1
	contentTypeApplication
)

func newContentType(b byte) (contentType, error) {
	ct := contentType(b)
	switch ct {
	case contentTypeExpression, contentTypeApplication:
		return ct, nil
	default:
		return 0, errors.Errorf("unsupported content type '%d'", b)
	}
}

type libraryVersion byte

const (
	libV1 libraryVersion = iota + 1
	libV2
	libV3
	libV4
	libV5
	libV6
)

func newLibraryVersion(b byte) (libraryVersion, error) {
	lv := libraryVersion(b)
	switch lv {
	case libV1, libV2, libV3, libV4, libV5, libV6:
		return lv, nil
	default:
		return 0, errors.Errorf("unsupported library version '%d'", b)
	}
}

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

func Parse(script []byte) (*Tree, error) {
	ok, err := verifyCheckSum(script)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script")
	}
	if !ok {
		return nil, errors.New("invalid script checksum")
	}
	id := sh256.Sum256(script)
	r := bytes.NewReader(script)
	p, err := parseHeader(r, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script header")
	}
	return p.parse()
}

type parser struct {
	r           *bytes.Reader
	id          [32]byte
	header      scriptHeader
	seenBlockV2 bool
	readShort   func(*bytes.Reader) (int16, error)
	readInt     func(*bytes.Reader) (int32, error)
	readLong    func(*bytes.Reader) (int64, error)
	readMeta    func(p *parser) (meta.DApp, error)
}

func (p *parser) parse() (*Tree, error) {
	switch p.header.content {
	case contentTypeExpression:
		return p.parseExpression()
	case contentTypeApplication:
		return p.parseDApp()
	default:
		return nil, errors.Errorf("unsupported content type '%d'", p.header.content)
	}
}

func (p *parser) parseDApp() (*Tree, error) {
	tree := &Tree{
		contentType: p.header.content,
		LibVersion:  int(p.header.library),
	}
	m, err := p.readMeta(p)
	if err != nil {
		return nil, err
	}
	tree.Meta = m

	n, err := p.readInt(p.r)
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

	n, err = p.readInt(p.r)
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

	n, err = p.readInt(p.r)
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

func (p *parser) parseExpression() (*Tree, error) {
	tree := &Tree{
		contentType: p.header.content,
		LibVersion:  int(p.header.library),
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
		v, err := p.readLong(p.r)
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
		argumentsCount, err := p.readInt(p.r)
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
		if ad, ok := declaration.(*AssignmentNode); ok {
			ad.newBlock = true
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

func (p *parser) readBytes() ([]byte, error) {
	n, err := p.readInt(p.r)
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
		id, err := p.readShort(p.r)
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
		argumentsCount, err := p.readInt(p.r)
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

type scriptHeader struct {
	content contentType
	library libraryVersion
}

/*
	Serialization mode V1 (LIBRARY_VERSION <= 5):
	00 CONTENT_TYPE LIBRARY_VERSION <DAPP|EXPRESSION> - DApp, Expression
	LIBRARY_VERSION <EXPRESSION> - Expression

	Serialization mode V2 (since LIBRARY_VERSION >= 6):
	LIBRARY_VERSION CONTENT_TYPE <DAPP|EXPRESSION> - DApp, Expression
*/
func parseHeader(r *bytes.Reader, id [32]byte) (*parser, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if b < byte(libV6) {
		switch b {
		case 0:
			b, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			ct, err := newContentType(b)
			if err != nil {
				return nil, err
			}
			b, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			lv, err := newLibraryVersion(b)
			if err != nil {
				return nil, err
			}
			return newParserV1(r, id, scriptHeader{content: ct, library: lv}), nil
		default:
			lv := libraryVersion(b)
			return newParserV1(r, id, scriptHeader{content: contentTypeExpression, library: lv}), nil
		}
	}
	lv, err := newLibraryVersion(b)
	if err != nil {
		return nil, err
	}
	b, err = r.ReadByte()
	if err != nil {
		return nil, err
	}
	ct, err := newContentType(b)
	if err != nil {
		return nil, err
	}
	return newParserV2(r, id, scriptHeader{content: ct, library: lv}), nil
}

func verifyCheckSum(scr []byte) (bool, error) {
	size := len(scr) - 4
	if size <= 0 {
		return false, errors.Errorf("invalid source length %d", size)
	}
	body, cs := scr[:size], scr[size:]
	digest, err := crypto.SecureHash(body)
	if err != nil {
		return false, errors.Wrap(err, "failed to verify check sum")
	}
	return bytes.Equal(digest[:4], cs), nil
}
