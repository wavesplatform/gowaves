//nolint:cyclop,gocognit,gocyclo,godot,govet // Verbatim copy of the old parser, the test oracle.
package serialization

// This file keeps the io.Reader based parser that was used before the []byte cursor rewrite.
// It is the oracle for the equality tests: both parsers must return equal trees and fail on the same inputs.

import (
	"bytes"
	sh256 "crypto/sha256"
	"encoding/binary"
	"io"
	"math"
	"strconv"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
)

func legacyCheckHeader(script []byte) error {
	ok, err := verifyCheckSum(script)
	if err != nil {
		return errors.Wrap(err, "failed to parse script header")
	}
	if !ok {
		return errors.New("invalid script checksum")
	}
	r := bytes.NewReader(script)
	_, err = legacyParseHeader(r, [32]byte{})
	if err != nil {
		return errors.Wrap(err, "failed to parse script header")
	}
	return nil
}

func legacyParse(script []byte) (*ast.Tree, error) {
	ok, err := verifyCheckSum(script)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script")
	}
	if !ok {
		return nil, errors.New("invalid script checksum")
	}
	id := sh256.Sum256(script)
	r := bytes.NewReader(script)
	p, err := legacyParseHeader(r, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script header")
	}
	return p.parse()
}

type legacyParser struct {
	r           *bytes.Reader
	id          [32]byte
	header      scriptHeader
	seenBlockV2 bool
	readShort   func(*bytes.Reader) (int16, error)
	readInt     func(*bytes.Reader) (int32, error)
	readLong    func(*bytes.Reader) (int64, error)
	readMeta    func(p *legacyParser) (meta.DApp, error)
}

func (p *legacyParser) parse() (*ast.Tree, error) {
	switch p.header.content {
	case ast.ContentTypeExpression:
		return p.parseExpression()
	case ast.ContentTypeApplication:
		return p.parseDApp()
	default:
		return nil, errors.Errorf("unsupported content type '%d'", p.header.content)
	}
}

func (p *legacyParser) parseDApp() (*ast.Tree, error) {
	tree := ast.NewTree(p.header.content, p.header.library)
	m, err := p.readMeta(p)
	if err != nil {
		return nil, err
	}
	tree.Meta = m

	n, err := p.readCount("declarations")
	if err != nil {
		return nil, err
	}
	declarations := make([]ast.Node, n)
	for i := range n {
		d, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		declarations[i] = d
	}
	tree.Declarations = declarations

	n, err = p.readCount("functions")
	if err != nil {
		return nil, err
	}
	functions := make([]ast.Node, n)
	for i := range n {
		invocationParameter, err := p.readString()
		if err != nil {
			return nil, err
		}
		node, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		fn, ok := node.(*ast.FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("unexpected type of declaration %T", node)
		}
		fn.InvocationParameter = invocationParameter
		functions[i] = fn
		// Update callable name in tree's meta
		if len(tree.Meta.Functions) > i {
			tree.Meta.Functions[i].Name = fn.Name
		}
	}
	tree.Functions = functions

	vn, err := p.readInt(p.r)
	if err != nil {
		return nil, err
	}
	if vn != 0 {
		invocationParameter, err := p.readString()
		if err != nil {
			return nil, err
		}
		node, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		fn, ok := node.(*ast.FunctionDeclarationNode)
		if !ok {
			return nil, errors.Errorf("unexpected type of declaration %T", node)
		}
		fn.InvocationParameter = invocationParameter
		tree.Verifier = fn
	}
	tree.HasBlockV2 = p.seenBlockV2
	tree.Digest = p.id
	return tree, nil
}

func (p *legacyParser) parseExpression() (*ast.Tree, error) {
	tree := ast.NewTree(p.header.content, p.header.library)
	node, err := p.parseNext()
	if err != nil {
		return nil, err
	}
	tree.Verifier = node
	tree.HasBlockV2 = p.seenBlockV2
	tree.Digest = p.id
	return tree, nil
}

func (p *legacyParser) parseNext() (ast.Node, error) {
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
		return ast.NewLongNode(v), nil

	case tokenBytes:
		v, err := p.readBytes()
		if err != nil {
			return nil, err
		}
		return ast.NewBytesNode(v), nil

	case tokenString:
		v, err := p.readString()
		if err != nil {
			return nil, err
		}
		return ast.NewStringNode(v), nil

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
		return ast.NewConditionalNode(condition, trueBranch, falseBranch), nil

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
		return ast.NewAssignmentNode(name, expr, block), nil

	case tokenRef:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		return ast.NewReferenceNode(name), nil

	case tokenTrue:
		return ast.NewBooleanNode(true), nil

	case tokenFalse:
		return ast.NewBooleanNode(false), nil

	case tokenGetter:
		object, err := p.parseNext()
		if err != nil {
			return nil, err
		}
		field, err := p.readString()
		if err != nil {
			return nil, err
		}
		return ast.NewPropertyNode(field, object), nil

	case tokenFunctionCall:
		ft, err := p.r.ReadByte()
		if err != nil {
			return nil, err
		}
		function, err := p.readFunctionName(ft)
		if err != nil {
			return nil, err
		}
		ac, err := p.readCount("function call arguments")
		if err != nil {
			return nil, err
		}
		arguments := make([]ast.Node, ac)
		for i := range ac {
			arg, err := p.parseNext()
			if err != nil {
				return nil, err
			}
			arguments[i] = arg
		}
		return ast.NewFunctionCallNode(function, arguments), nil

	case tokenBlockV2:
		p.seenBlockV2 = true
		declaration, err := p.readDeclaration()
		if err != nil {
			return nil, err
		}
		if ad, ok := declaration.(*ast.AssignmentNode); ok {
			ad.NewBlock = true
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

// readCount reads a length or an elements count and checks that it is not negative and does not exceed the number
// of unread bytes. Every element takes at least one byte, so a bigger count is always invalid. The check prevents
// panics and huge allocations on malformed input.
func (p *legacyParser) readCount(what string) (int, error) {
	v, err := p.readInt(p.r)
	if err != nil {
		return 0, err
	}
	n := int(v)
	if n < 0 {
		return 0, errors.Errorf("negative number of %s %d", what, n)
	}
	if rem := p.r.Len(); n > rem {
		return 0, errors.Errorf("number of %s %d exceeds the number of remaining bytes %d", what, n, rem)
	}
	return n, nil
}

func (p *legacyParser) readBytes() ([]byte, error) {
	n, err := p.readCount("bytes")
	if err != nil {
		return nil, err
	}
	buf := make([]byte, n)
	if n > 0 {
		if _, rErr := io.ReadFull(p.r, buf); rErr != nil {
			return nil, rErr
		}
	}
	return buf, nil
}

func (p *legacyParser) readString() (string, error) {
	b, err := p.readBytes()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *legacyParser) readFunctionName(ft byte) (ast.Function, error) {
	switch ft {
	case functionTypeNative:
		id, err := p.readShort(p.r)
		if err != nil {
			return nil, err
		}
		return ast.NativeFunction(strconv.Itoa(int(id))), nil
	case functionTypeUser:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		return ast.UserFunction(name), nil
	default:
		return nil, errors.Errorf("unsupported function type %d", ft)
	}
}

func (p *legacyParser) readDeclaration() (ast.Node, error) {
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
		return ast.NewAssignmentNode(name, exp, nil), nil
	case declarationTypeFunction:
		name, err := p.readString()
		if err != nil {
			return nil, err
		}
		ac, err := p.readCount("function declaration arguments")
		if err != nil {
			return nil, err
		}
		arguments := make([]string, ac)
		for i := range ac {
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
		return ast.NewFunctionDeclarationNode(name, arguments, body, nil), nil
	default:
		return nil, errors.Errorf("unsupported declaration type %d", dt)
	}
}

/*
Serialization mode V1 (LIBRARY_VERSION <= 5):
00 CONTENT_TYPE LIBRARY_VERSION <DAPP|EXPRESSION> - DApp, Expression
LIBRARY_VERSION <EXPRESSION> - Expression

Serialization mode V2 (since LIBRARY_VERSION >= 6):
LIBRARY_VERSION CONTENT_TYPE <DAPP|EXPRESSION> - DApp, Expression
*/
func legacyParseHeader(r *bytes.Reader, id [32]byte) (*legacyParser, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	if b < byte(ast.LibV6) {
		switch b {
		case 0:
			b, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			ct, err := ast.NewContentType(b)
			if err != nil {
				return nil, err
			}
			b, err = r.ReadByte()
			if err != nil {
				return nil, err
			}
			lv, err := ast.NewLibraryVersion(b)
			if err != nil {
				return nil, err
			}
			return legacyNewParserV1(r, id, scriptHeader{content: ct, library: lv}), nil
		default:
			lv := ast.LibraryVersion(b)
			return legacyNewParserV1(r, id, scriptHeader{content: ast.ContentTypeExpression, library: lv}), nil
		}
	}
	lv, err := ast.NewLibraryVersion(b)
	if err != nil {
		return nil, err
	}
	b, err = r.ReadByte()
	if err != nil {
		return nil, err
	}
	ct, err := ast.NewContentType(b)
	if err != nil {
		return nil, err
	}
	return legacyNewParserV2(r, id, scriptHeader{content: ct, library: lv}), nil
}

func legacyNewParserV1(r *bytes.Reader, id [32]byte, header scriptHeader) *legacyParser {
	p := &legacyParser{
		r:      r,
		id:     id,
		header: header,
	}
	p.readShort = legacyReadShortV1
	p.readInt = legacyReadIntV1
	p.readLong = legacyReadLongV1
	p.readMeta = legacyReadMetaV1
	return p
}

func legacyReadShortV1(r *bytes.Reader) (int16, error) {
	buf := [2]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf[:])), nil
}

func legacyReadIntV1(r *bytes.Reader) (int32, error) {
	buf := [4]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf[:])), nil
}

func legacyReadLongV1(r *bytes.Reader) (int64, error) {
	buf := [8]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf[:])), nil
}

func legacyReadMetaV1(p *legacyParser) (meta.DApp, error) {
	v, err := p.readInt(p.r)
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
		if unmrshlErr := pbMeta.UnmarshalVT(b); unmrshlErr != nil {
			return meta.DApp{}, unmrshlErr
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

func legacyNewParserV2(r *bytes.Reader, id [32]byte, header scriptHeader) *legacyParser {
	p := &legacyParser{
		r:      r,
		id:     id,
		header: header,
	}
	p.readShort = legacyReadShortV2
	p.readInt = legacyReadIntV2
	p.readLong = legacyReadLongV2
	p.readMeta = legacyReadMetaV2
	return p
}

func legacyReadShortV2(r *bytes.Reader) (int16, error) {
	v, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	vv := int64(v)
	if vv < math.MinInt16 || vv > math.MaxInt16 {
		return 0, errors.New("value out of int16 range")
	}
	return int16(v), nil
}

func legacyReadIntV2(r *bytes.Reader) (int32, error) {
	v, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	vv := int64(v)
	if vv < math.MinInt32 || vv > math.MaxInt32 {
		return 0, errors.New("value out of int32 range")
	}
	return int32(v), nil
}

func legacyReadLongV2(r *bytes.Reader) (int64, error) {
	v, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

func legacyReadMetaV2(p *legacyParser) (meta.DApp, error) {
	b, err := p.readBytes()
	if err != nil {
		return meta.DApp{}, err
	}
	pbMeta := new(g.DAppMeta)
	err = pbMeta.UnmarshalVT(b)
	if err != nil {
		return meta.DApp{}, err
	}
	m, err := meta.Convert(pbMeta)
	if err != nil {
		return meta.DApp{}, err
	}
	return m, nil
}

func legacyNativeFunctionName(id int16) ast.NativeFunction {
	return ast.NativeFunction(strconv.Itoa(int(id)))
}
