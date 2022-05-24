package serialization

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/ride/meta"
	g "github.com/wavesplatform/gowaves/pkg/ride/meta/generated"
	protobuf "google.golang.org/protobuf/proto"
)

func newParserV1(r *bytes.Reader, id [32]byte, header scriptHeader) *parser {
	p := &parser{
		r:      r,
		id:     id,
		header: header,
	}
	p.readShort = readShortV1
	p.readInt = readIntV1
	p.readLong = readLongV1
	p.readMeta = readMetaV1
	return p
}

func readShortV1(r *bytes.Reader) (int16, error) {
	buf := [2]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int16(binary.BigEndian.Uint16(buf[:])), nil
}

func readIntV1(r *bytes.Reader) (int32, error) {
	buf := [4]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(buf[:])), nil
}

func readLongV1(r *bytes.Reader) (int64, error) {
	buf := [8]byte{}
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(buf[:])), nil
}

func readMetaV1(p *parser) (meta.DApp, error) {
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
