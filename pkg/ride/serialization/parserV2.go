package serialization

import (
	"bytes"
	"encoding/binary"
	"math"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
	protobuf "google.golang.org/protobuf/proto"
)

func newParserV2(r *bytes.Reader, id [32]byte, header scriptHeader) *parser {
	p := &parser{
		r:      r,
		id:     id,
		header: header,
	}
	p.readShort = readShortV2
	p.readInt = readIntV2
	p.readLong = readLongV2
	p.readMeta = readMetaV2
	return p
}

func readShortV2(r *bytes.Reader) (int16, error) {
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

func readIntV2(r *bytes.Reader) (int32, error) {
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

func readLongV2(r *bytes.Reader) (int64, error) {
	v, err := binary.ReadUvarint(r)
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

func readMetaV2(p *parser) (meta.DApp, error) {
	b, err := p.readBytes()
	if err != nil {
		return meta.DApp{}, err
	}
	pbMeta := new(g.DAppMeta)
	if err := protobuf.Unmarshal(b, pbMeta); err != nil {
		return meta.DApp{}, err
	}
	m, err := meta.Convert(pbMeta)
	if err != nil {
		return meta.DApp{}, err
	}
	return m, nil
}
