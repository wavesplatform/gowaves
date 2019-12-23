package server

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/scheduler"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetStatuses(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := state.DefaultTestingStateParams()
	// State should store addl data for gRPC API.
	params.StoreExtendedApiData = true
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	sch := scheduler.NewScheduler(st, keyPairs, settings.MainNetSettings)
	utx := utxpool.New(utxSize)
	err = server.initServer(st, utx, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	assert.NoError(t, err)
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferV1(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), "attachment")
	err = tx.Sign(sk)
	assert.NoError(t, err)
	txBytes, err := tx.MarshalBinary()
	assert.NoError(t, err)
	// Add tx to UTX.
	added := utx.AddWithBytes(tx, txBytes)
	assert.Equal(t, true, added)

	cl := g.NewTransactionsApiClient(conn)
	// id0 is from Mainnet genesis block.
	id0 := crypto.MustSignatureFromBase58("2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8")
	// id1 should be in UTX.
	id1, err := tx.GetID()
	assert.NoError(t, err)
	// id2 is unknown.
	id2 := []byte{2}
	ids := [][]byte{id0.Bytes(), id1, id2}
	correstResults := []*g.TransactionStatus{
		{Id: id0.Bytes(), Height: 1, Status: g.TransactionStatus_CONFIRMED},
		{Id: id1, Status: g.TransactionStatus_UNCONFIRMED},
		{Id: id2, Status: g.TransactionStatus_NOT_EXISTS},
	}

	req := &g.TransactionsByIdRequest{TransactionIds: ids}
	stream, err := cl.GetStatuses(ctx, req)
	assert.NoError(t, err)
	for _, correctRes := range correstResults {
		res, err := stream.Recv()
		assert.NoError(t, err)
		assert.Equal(t, correctRes, res)
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}
