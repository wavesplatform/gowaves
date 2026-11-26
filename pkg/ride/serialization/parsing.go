package serialization

import (
	"bytes"
	sh256 "crypto/sha256"
	"encoding/binary"
	"io"
	"math"
	"strconv"

	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
)

var errVarintOverflow = errors.New("varint overflows a 64-bit integer")

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

func CheckHeader(script []byte) error {
	ok, err := verifyCheckSum(script)
	if err != nil {
		return errors.Wrap(err, "failed to parse script header")
	}
	if !ok {
		return errors.New("invalid script checksum")
	}
	if _, err = parseHeader(script, [32]byte{}); err != nil {
		return errors.Wrap(err, "failed to parse script header")
	}
	return nil
}

func Parse(script []byte) (*ast.Tree, error) {
	ok, err := verifyCheckSum(script)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script")
	}
	if !ok {
		return nil, errors.New("invalid script checksum")
	}
	id := sh256.Sum256(script)
	p, err := parseHeader(script, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse script header")
	}
	return p.parse()
}

// parser reads the script from a byte slice. The cursor pos points to the first unread byte.
// The whole script, including the checksum, is kept in data, so the remaining bytes count is the same as
// it was with the io.Reader based parser. Integers are read according to the serialization mode: fixed
// size big-endian numbers in V1 and unsigned varints in V2.
type parser struct {
	data        []byte
	pos         int
	v2          bool
	id          [32]byte
	header      scriptHeader
	seenBlockV2 bool
}

// remaining returns the number of unread bytes.
func (p *parser) remaining() int {
	return len(p.data) - p.pos
}

const (
	shortSize = 2
	intSize   = 4
	longSize  = 8
)

func (p *parser) readByte() (byte, error) {
	if p.pos >= len(p.data) {
		return 0, io.EOF
	}
	b := p.data[p.pos]
	p.pos++
	return b, nil
}

// next returns the next n bytes and advances the cursor. The returned slice aliases the script.
func (p *parser) next(n int) ([]byte, error) {
	if rem := p.remaining(); n > rem {
		if rem == 0 {
			return nil, io.EOF
		}
		return nil, io.ErrUnexpectedEOF
	}
	b := p.data[p.pos : p.pos+n]
	p.pos += n
	return b, nil
}

func (p *parser) readUvarint() (uint64, error) {
	v, n := binary.Uvarint(p.data[p.pos:])
	switch {
	case n > 0:
		p.pos += n
		return v, nil
	case n == 0:
		if p.remaining() == 0 {
			return 0, io.EOF
		}
		return 0, io.ErrUnexpectedEOF
	default:
		return 0, errVarintOverflow
	}
}

// readVarint reads an unsigned varint of the V2 serialization mode and returns it as a two's complement number.
func (p *parser) readVarint() (int64, error) {
	v, err := p.readUvarint()
	if err != nil {
		return 0, err
	}
	return int64(v), nil //nolint:gosec // The varint holds the two's complement representation of the number.
}

func (p *parser) readShort() (int16, error) {
	if p.v2 {
		v, err := p.readVarint()
		if err != nil {
			return 0, err
		}
		if v < math.MinInt16 || v > math.MaxInt16 {
			return 0, errors.New("value out of int16 range")
		}
		return int16(v), nil
	}
	b, err := p.next(shortSize)
	if err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(b)), nil //nolint:gosec // Two's complement big-endian number.
}

func (p *parser) readInt() (int32, error) {
	if p.v2 {
		v, err := p.readVarint()
		if err != nil {
			return 0, err
		}
		if v < math.MinInt32 || v > math.MaxInt32 {
			return 0, errors.New("value out of int32 range")
		}
		return int32(v), nil
	}
	b, err := p.next(intSize)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(b)), nil //nolint:gosec // Two's complement big-endian number.
}

func (p *parser) readLong() (int64, error) {
	if p.v2 {
		return p.readVarint()
	}
	b, err := p.next(longSize)
	if err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(b)), nil //nolint:gosec // Two's complement big-endian number.
}

func (p *parser) readMeta() (meta.DApp, error) {
	if !p.v2 {
		v, err := p.readInt()
		if err != nil {
			return meta.DApp{}, err
		}
		b, err := p.readBytesView()
		if err != nil {
			return meta.DApp{}, err
		}
		if v != 0 {
			return meta.DApp{}, errors.Errorf("unsupported script meta version %d", v)
		}
		return unmarshalMeta(b)
	}
	b, err := p.readBytesView()
	if err != nil {
		return meta.DApp{}, err
	}
	return unmarshalMeta(b)
}

func unmarshalMeta(b []byte) (meta.DApp, error) {
	pbMeta := new(g.DAppMeta)
	if err := pbMeta.UnmarshalVT(b); err != nil {
		return meta.DApp{}, err
	}
	return meta.Convert(pbMeta)
}

func (p *parser) parse() (*ast.Tree, error) {
	switch p.header.content {
	case ast.ContentTypeExpression:
		return p.parseExpression()
	case ast.ContentTypeApplication:
		return p.parseDApp()
	default:
		return nil, errors.Errorf("unsupported content type '%d'", p.header.content)
	}
}

func (p *parser) parseDApp() (*ast.Tree, error) {
	tree := ast.NewTree(p.header.content, p.header.library)
	m, err := p.readMeta()
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

	vn, err := p.readInt()
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

func (p *parser) parseExpression() (*ast.Tree, error) {
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

func (p *parser) parseNext() (ast.Node, error) {
	t, tErr := p.readByte()
	if tErr != nil {
		return nil, tErr
	}
	switch t {
	case tokenLong:
		v, err := p.readLong()
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
		ft, err := p.readByte()
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
func (p *parser) readCount(what string) (int, error) {
	v, err := p.readInt()
	if err != nil {
		return 0, err
	}
	n := int(v)
	if n < 0 {
		return 0, errors.Errorf("negative number of %s %d", what, n)
	}
	if rem := p.remaining(); n > rem {
		return 0, errors.Errorf("number of %s %d exceeds the number of remaining bytes %d", what, n, rem)
	}
	return n, nil
}

// readBytesView reads a length prefixed byte vector and returns a slice that aliases the script.
// Use it only for the data that is decoded right away and not kept in the tree.
func (p *parser) readBytesView() ([]byte, error) {
	n, err := p.readCount("bytes")
	if err != nil {
		return nil, err
	}
	return p.next(n) // readCount guarantees that n bytes are available.
}

// readBytes reads a length prefixed byte vector into a new slice, the tree must not alias the script bytes.
func (p *parser) readBytes() ([]byte, error) {
	b, err := p.readBytesView()
	if err != nil {
		return nil, err
	}
	r := make([]byte, len(b))
	copy(r, b)
	return r, nil
}

// readString reads a length prefixed string with a single allocation.
func (p *parser) readString() (string, error) {
	b, err := p.readBytesView()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// nativeFunctionNamesCount is the size of the table of native function names. All existing native function
// identifiers are less than this value, others are converted to strings on the fly.
const nativeFunctionNamesCount = 1 << 11

// nativeFunctionNames is a static table of native function names, so the parser doesn't call strconv.Itoa
// and doesn't allocate a string for every native function call.
//
//nolint:gochecknoglobals // Immutable lookup table, initialized once.
var nativeFunctionNames = func() [nativeFunctionNamesCount]ast.NativeFunction {
	var r [nativeFunctionNamesCount]ast.NativeFunction
	for i := range r {
		r[i] = ast.NativeFunction(strconv.Itoa(i))
	}
	return r
}()

func nativeFunctionName(id int16) ast.NativeFunction {
	if id >= 0 && int(id) < nativeFunctionNamesCount {
		return nativeFunctionNames[id]
	}
	return ast.NativeFunction(strconv.Itoa(int(id)))
}

func (p *parser) readFunctionName(ft byte) (ast.Function, error) {
	switch ft {
	case functionTypeNative:
		id, err := p.readShort()
		if err != nil {
			return nil, err
		}
		return nativeFunctionName(id), nil
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

func (p *parser) readDeclaration() (ast.Node, error) {
	dt, err := p.readByte()
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

type scriptHeader struct {
	content ast.ContentType
	library ast.LibraryVersion
}

/*
Serialization mode V1 (LIBRARY_VERSION <= 5):
00 CONTENT_TYPE LIBRARY_VERSION <DAPP|EXPRESSION> - DApp, Expression
LIBRARY_VERSION <EXPRESSION> - Expression

Serialization mode V2 (since LIBRARY_VERSION >= 6):
LIBRARY_VERSION CONTENT_TYPE <DAPP|EXPRESSION> - DApp, Expression
*/
func parseHeader(script []byte, id [32]byte) (*parser, error) {
	p := &parser{data: script, id: id}
	b, err := p.readByte()
	if err != nil {
		return nil, err
	}
	if b < byte(ast.LibV6) {
		if b != 0 {
			p.header = scriptHeader{content: ast.ContentTypeExpression, library: ast.LibraryVersion(b)}
			return p, nil
		}
		if b, err = p.readByte(); err != nil {
			return nil, err
		}
		ct, ctErr := ast.NewContentType(b)
		if ctErr != nil {
			return nil, ctErr
		}
		if b, err = p.readByte(); err != nil {
			return nil, err
		}
		lv, lvErr := ast.NewLibraryVersion(b)
		if lvErr != nil {
			return nil, lvErr
		}
		p.header = scriptHeader{content: ct, library: lv}
		return p, nil
	}
	lv, err := ast.NewLibraryVersion(b)
	if err != nil {
		return nil, err
	}
	if b, err = p.readByte(); err != nil {
		return nil, err
	}
	ct, err := ast.NewContentType(b)
	if err != nil {
		return nil, err
	}
	p.v2 = true
	p.header = scriptHeader{content: ct, library: lv}
	return p, nil
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
