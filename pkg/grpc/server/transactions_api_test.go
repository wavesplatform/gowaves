package server

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/grpc/client"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetTransactions(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/lease_genesis.json")
	assert.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, sets)
	err = server.initServer(st, utxpool.New(utxSize), sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		stateCloser()
	}()

	id, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	assert.NoError(t, err)
	tx, err := st.TransactionByID(id.Bytes())
	assert.NoError(t, err)
	leaseTx, ok := tx.(*proto.LeaseV1)
	assert.Equal(t, true, ok)
	recipient := *leaseTx.Recipient.Address
	recipientBody, err := recipient.Body()
	assert.NoError(t, err)
	sender, err := proto.NewAddressFromPublicKey(server.scheme, leaseTx.SenderPK)
	assert.NoError(t, err)
	senderBody, err := sender.Body()
	assert.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)

	// By sender.
	req := &g.TransactionsRequest{
		Sender: senderBody,
	}
	stream, err := cl.GetTransactions(ctx, req)
	assert.NoError(t, err)
	correctRes, err := server.transactionToTransactionResponse(tx, true)
	assert.NoError(t, err)
	res, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient.
	req = &g.TransactionsRequest{
		Recipient: &g.Recipient{Recipient: &g.Recipient_Address{Address: recipientBody}},
	}
	stream, err = cl.GetTransactions(ctx, req)
	assert.NoError(t, err)
	res, err = stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient and ID.
	req = &g.TransactionsRequest{
		Recipient:      &g.Recipient{Recipient: &g.Recipient_Address{Address: recipientBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	stream, err = cl.GetTransactions(ctx, req)
	assert.NoError(t, err)
	res, err = stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By sender, recipient and ID.
	req = &g.TransactionsRequest{
		Sender:         senderBody,
		Recipient:      &g.Recipient{Recipient: &g.Recipient_Address{Address: recipientBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	stream, err = cl.GetTransactions(ctx, req)
	assert.NoError(t, err)
	res, err = stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestGetStatuses(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize)
	err = server.initServer(st, utx, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
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

func TestGetUnconfirmed(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize)
	err = server.initServer(st, utx, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
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
	addrBody, err := addr.Body()
	assert.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	assert.NoError(t, err)
	senderAddr, err := proto.NewAddressFromPublicKey(server.scheme, pk)
	assert.NoError(t, err)
	senderAddrBody, err := senderAddr.Body()
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

	// By sender.
	req := &g.TransactionsRequest{
		Sender: senderAddrBody,
	}
	stream, err := cl.GetUnconfirmed(ctx, req)
	assert.NoError(t, err)
	correctRes, err := server.transactionToTransactionResponse(tx, false)
	assert.NoError(t, err)
	res, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient.
	req = &g.TransactionsRequest{
		Recipient: &g.Recipient{Recipient: &g.Recipient_Address{Address: addrBody}},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	assert.NoError(t, err)
	res, err = stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By ID.
	id, err := tx.GetID()
	assert.NoError(t, err)
	req = &g.TransactionsRequest{
		TransactionIds: [][]byte{id},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	assert.NoError(t, err)
	res, err = stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By sender, recipient and ID.
	req = &g.TransactionsRequest{
		Sender:         senderAddrBody,
		Recipient:      &g.Recipient{Recipient: &g.Recipient_Address{Address: addrBody}},
		TransactionIds: [][]byte{id},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	assert.NoError(t, err)
	res, err = stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestSign(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize)
	err = server.initServer(st, utx, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		conn.Close()
		err = st.Close()
		assert.NoError(t, err)
		err = os.RemoveAll(dataDir)
		assert.NoError(t, err)
	}()

	pk := keyPairs[0].Public

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	assert.NoError(t, err)
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferV1(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), "attachment")
	tx.GenerateID()
	txProto, err := tx.ToProtobuf(server.scheme)
	assert.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)
	req := &g.SignRequest{Transaction: txProto, SignerPublicKey: pk.Bytes()}
	res, err := cl.Sign(ctx, req)
	assert.NoError(t, err)
	var c client.SafeConverter
	resTx, err := c.SignedTransaction(res)
	assert.NoError(t, err)
	transfer, ok := resTx.(*proto.TransferV1)
	assert.Equal(t, true, ok)
	ok, err = transfer.Verify(pk)
	assert.NoError(t, err)
	assert.Equal(t, true, ok)
}

func TestBroadcast(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	assert.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	assert.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createScheduler(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize)
	err = server.initServer(st, utx, sch)
	assert.NoError(t, err)

	conn := connect(t, grpcTestAddr)
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
	txProto, err := tx.ToProtobufSigned(server.scheme)
	assert.NoError(t, err)

	// tx is originally not in UTX.
	assert.Equal(t, false, utx.Exists(tx))
	cl := g.NewTransactionsApiClient(conn)
	_, err = cl.Broadcast(ctx, txProto)
	assert.NoError(t, err)

	// tx should now be in UTX.
	assert.Equal(t, true, utx.Exists(tx))
}
