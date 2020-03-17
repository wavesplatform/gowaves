package genesis_generator

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func Genesis(timestamp proto.Timestamp, transactions proto.Transactions, scheme proto.Scheme) (*proto.Block, error) {
	block, err := proto.CreateBlock(
		transactions,
		timestamp,
		crypto.MustSignatureFromBase58("67rpwLCuS5DGA8KGZXKsVQ7dnPb9goRLoKfgGbLfQg9WoLUgNY77E2jT11fem3coV9nAkguBACzrU1iyZM4B8roQ"),
		crypto.PublicKey{},
		proto.NxtConsensus{
			BaseTarget:   153722867,
			GenSignature: crypto.MustBytesFromBase58("11111111111111111111111111111111"),
		},
		proto.GenesisBlockVersion,
		nil,
		-1,
		scheme,
	)

	if err != nil {
		return nil, err
	}

	kp := proto.MustKeyPair([]byte{})
	err = block.Sign(scheme, kp.Secret)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func Generate(timestamp proto.Timestamp, schema byte, v ...interface{}) (*proto.Block, error) {
	if len(v)%2 != 0 {
		return nil, errors.Errorf("bad args, expected even argument count, found %d", len(v))
	}

	transactions := proto.Transactions{}
	for i := 0; i < len(v); i += 2 {
		addr, err := v[i].(proto.KeyPair).Addr(schema)
		if err != nil {
			return nil, err
		}
		t := proto.NewUnsignedGenesis(addr, uint64(v[i+1].(int)), timestamp)
		err = t.GenerateSigID(schema)
		if err != nil {
			panic(err.Error())
		}
		transactions = append(transactions, t)
	}

	return Genesis(timestamp, transactions, schema)
}
