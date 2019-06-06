package genesis_generator

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func Genesis(timestamp proto.Timestamp, transactions proto.Transactions) (*proto.Block, error) {

	buf := new(bytes.Buffer)
	_, err := transactions.WriteTo(buf)
	if err != nil {
		return nil, err
	}

	block := proto.Block{
		BlockHeader: proto.BlockHeader{
			Version:              1,
			Timestamp:            timestamp,
			Parent:               crypto.MustSignatureFromBase58("67rpwLCuS5DGA8KGZXKsVQ7dnPb9goRLoKfgGbLfQg9WoLUgNY77E2jT11fem3coV9nAkguBACzrU1iyZM4B8roQ"),
			FeaturesCount:        0,
			Features:             nil,
			ConsensusBlockLength: 0,
			NxtConsensus: proto.NxtConsensus{
				BaseTarget:   153722867,
				GenSignature: crypto.MustDigestFromBase58("11111111111111111111111111111111"),
			},
			TransactionBlockLength: uint32(len(buf.Bytes())),
			TransactionCount:       len(transactions),
		},
		Transactions: buf.Bytes(),
	}

	buf = new(bytes.Buffer)
	_, err = block.WriteTo(buf)
	if err != nil {
		return nil, err
	}
	kp := proto.NewKeyPair([]byte{})
	sig := crypto.Sign(kp.Private(), buf.Bytes())
	block.BlockHeader.BlockSignature = sig
	return &block, nil
}

func Generate(timestamp proto.Timestamp, schema byte, v ...interface{}) (*proto.Block, error) {
	if len(v)%2 != 0 {
		return nil, errors.Errorf("bad args, expected even argument count, found %d", len(v))
	}

	transactions := proto.Transactions{}
	for i := 0; i < len(v); i += 2 {
		t := proto.NewUnsignedGenesis(v[i].(proto.KeyPair).Addr(schema), uint64(v[i+1].(int)), timestamp)
		err := t.GenerateSigID()
		if err != nil {
			panic(err.Error())
		}
		transactions = append(transactions, t)
	}

	return Genesis(timestamp, transactions)
}
