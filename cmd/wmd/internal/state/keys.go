package state

import (
	"encoding/binary"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

const (
	heightKeyPrefix byte = iota
	blockKeyPrefix

	candleKeyPrefix
	candleHistoryKeyPrefix

	tradeKeyPrefix
	tradeHistoryKeyPrefix
	marketTradesKeyPrefix
	addressTradesKeyPrefix

	earliestTimeFrameKeyPrefix
	earliestHeightKeyPrefix

	assetKeyPrefix
	assetInfoHistoryKeyPrefix

	marketKeyPrefix
	marketHistoryKeyPrefix

	aliasToAddressKeyPrefix
	aliasHistoryKeyPrefix

	assetIssuerKeyPrefix
	assetBalanceKeyPrefix
	assetBalanceHistoryKeyPrefix
)

var (
	minDigest = crypto.Digest{}
	maxDigest = crypto.Digest{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
)

type uint32Key struct {
	prefix byte
	key    uint32
}

func (k uint32Key) bytes() []byte {
	buf := make([]byte, 1+4)
	buf[0] = k.prefix
	binary.BigEndian.PutUint32(buf[1:], k.key)
	return buf
}
