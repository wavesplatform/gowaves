package server

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func TestGetTransactions(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/lease_genesis.json")
	require.NoError(t, err)
	st, stateCloser := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, sets)
	err = server.initServer(st, utxpool.New(utxSize, utxpool.NewValidator(st, ntptime.Stub{}), sets), sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		err := conn.Close()
		require.NoError(t, err)
		stateCloser()
	}()

	id, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	require.NoError(t, err)
	tx, err := st.TransactionByID(id.Bytes())
	require.NoError(t, err)
	leaseTx, ok := tx.(*proto.LeaseWithSig)
	assert.Equal(t, true, ok)
	recipient := *leaseTx.Recipient.Address
	sender, err := proto.NewAddressFromPublicKey(server.scheme, leaseTx.SenderPK)
	require.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)

	// By sender.
	senderBody := sender.Body()
	req := &g.TransactionsRequest{
		Sender: senderBody,
	}
	stream, err := cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	correctRes, err := server.transactionToTransactionResponse(tx, true)
	require.NoError(t, err)
	res, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient.
	recipientBody := recipient.Body()
	req = &g.TransactionsRequest{
		Recipient: &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: recipientBody}},
	}
	stream, err = cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient and ID.
	req = &g.TransactionsRequest{
		Recipient:      &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: recipientBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	stream, err = cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By sender, recipient and ID.
	req = &g.TransactionsRequest{
		Sender:         senderBody,
		Recipient:      &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: recipientBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	stream, err = cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestGetStatuses(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	require.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize, utxpool.NoOpValidator{}, settings.MainNetSettings)
	err = server.initServer(st, utx, sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		err := conn.Close()
		require.NoError(t, err)
		err = st.Close()
		require.NoError(t, err)
		err = os.RemoveAll(dataDir)
		require.NoError(t, err)
	}()

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	require.NoError(t, err)
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), &proto.LegacyAttachment{Value: []byte("attachment")})
	err = tx.Sign(server.scheme, sk)
	require.NoError(t, err)
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)
	// Add tx to UTX.
	err = utx.AddWithBytes(tx, txBytes)
	require.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)
	// id0 is from Mainnet genesis block.
	id0 := crypto.MustSignatureFromBase58("2DVtfgXjpMeFf2PQCqvwxAiaGbiDsxDjSdNQkc5JQ74eWxjWFYgwvqzC4dn7iB1AhuM32WxEiVi1SGijsBtYQwn8")
	// id1 should be in UTX.
	id1, err := tx.GetID(settings.MainNetSettings.AddressSchemeCharacter)
	require.NoError(t, err)
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
	require.NoError(t, err)
	for _, correctRes := range correstResults {
		res, err := stream.Recv()
		require.NoError(t, err)
		assert.Equal(t, correctRes, res)
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestGetUnconfirmed(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	require.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize, utxpool.NoOpValidator{}, settings.MainNetSettings)
	err = server.initServer(st, utx, sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		err = conn.Close()
		require.NoError(t, err)
		err = st.Close()
		require.NoError(t, err)
		err = os.RemoveAll(dataDir)
		require.NoError(t, err)
	}()

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	addrBody := addr.Body()
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	require.NoError(t, err)
	senderAddr, err := proto.NewAddressFromPublicKey(server.scheme, pk)
	require.NoError(t, err)
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), &proto.LegacyAttachment{Value: []byte("attachment")})
	err = tx.Sign(server.scheme, sk)
	require.NoError(t, err)
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err)
	// Add tx to UTX.
	err = utx.AddWithBytes(tx, txBytes)
	require.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)

	// By sender.
	senderAddrBody := senderAddr.Body()
	req := &g.TransactionsRequest{
		Sender: senderAddrBody,
	}
	stream, err := cl.GetUnconfirmed(ctx, req)
	require.NoError(t, err)
	correctRes, err := server.transactionToTransactionResponse(tx, false)
	require.NoError(t, err)
	res, err := stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient.
	req = &g.TransactionsRequest{
		Recipient: &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: addrBody}},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By ID.
	id, err := tx.GetID(settings.MainNetSettings.AddressSchemeCharacter)
	require.NoError(t, err)
	req = &g.TransactionsRequest{
		TransactionIds: [][]byte{id},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By sender, recipient and ID.
	senderAddrBody = senderAddr.Body()
	req = &g.TransactionsRequest{
		Sender:         senderAddrBody,
		Recipient:      &g.Recipient{Recipient: &g.Recipient_PublicKeyHash{PublicKeyHash: addrBody}},
		TransactionIds: [][]byte{id},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assert.Equal(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestSign(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	require.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, settings.MainNetSettings)

	err = server.initServer(st, nil, sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		err = conn.Close()
		require.NoError(t, err)
		err = st.Close()
		require.NoError(t, err)
		err = os.RemoveAll(dataDir)
		require.NoError(t, err)
	}()

	pk := keyPairs[0].Public

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), &proto.LegacyAttachment{Value: []byte("attachment")})
	err = tx.GenerateID(server.scheme)
	require.NoError(t, err)
	require.NoError(t, err)
	txProto, err := tx.ToProtobuf(server.scheme)
	require.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)
	req := &g.SignRequest{Transaction: txProto, SignerPublicKey: pk.Bytes()}
	res, err := cl.Sign(ctx, req)
	require.NoError(t, err)
	var c proto.ProtobufConverter
	resTx, err := c.SignedTransaction(res)
	require.NoError(t, err)
	transfer, ok := resTx.(*proto.TransferWithSig)
	assert.Equal(t, true, ok)
	ok, err = transfer.Verify(server.scheme, pk)
	require.NoError(t, err)
	assert.Equal(t, true, ok)
}

func TestBroadcast(t *testing.T) {
	dataDir, err := ioutil.TempDir(os.TempDir(), "dataDir")
	require.NoError(t, err)
	params := defaultStateParams()
	st, err := state.NewState(dataDir, params, settings.MainNetSettings)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	sch := createWallet(ctx, st, settings.MainNetSettings)
	utx := utxpool.New(utxSize, utxpool.NoOpValidator{}, settings.MainNetSettings)
	err = server.initServer(st, utx, sch)
	require.NoError(t, err)

	conn := connect(t, grpcTestAddr)
	defer func() {
		cancel()
		err = conn.Close()
		require.NoError(t, err)
		err = st.Close()
		require.NoError(t, err)
		err = os.RemoveAll(dataDir)
		require.NoError(t, err)
	}()

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	require.NoError(t, err)
	waves := proto.OptionalAsset{Present: false}
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), &proto.LegacyAttachment{Value: []byte("attachment")})
	err = tx.Sign(server.scheme, sk)
	require.NoError(t, err)
	txProto, err := tx.ToProtobufSigned(server.scheme)
	require.NoError(t, err)

	// tx is originally not in UTX.
	assert.Equal(t, false, utx.Exists(tx))
	cl := g.NewTransactionsApiClient(conn)
	_, err = cl.Broadcast(ctx, txProto)
	require.NoError(t, err)

	// tx should now be in UTX.
	assert.Equal(t, true, utx.Exists(tx))
}
