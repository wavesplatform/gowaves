package server

import (
	"context"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	pb "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/libs/ntptime"
	"github.com/wavesplatform/gowaves/pkg/miner/utxpool"
	"github.com/wavesplatform/gowaves/pkg/mock"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestGetTransactions(t *testing.T) {
	genesisPath, err := globalPathFromLocal("testdata/genesis/lease_genesis.json")
	require.NoError(t, err)
	st := stateWithCustomGenesis(t, genesisPath)
	sets, err := st.BlockchainSettings()
	require.NoError(t, err)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)
	validator, err := utxpool.NewValidator(st, ntptime.Stub{}, 24*time.Hour)
	require.NoError(t, err)
	err = server.initServer(st, utxpool.New(utxSize, validator, sets), sch)
	require.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

	id, err := crypto.NewDigestFromBase58("ADXuoPsKMJ59HyLMGzLBbNQD8p2eJ93dciuBPJp3Qhx")
	require.NoError(t, err)
	tx, err := st.TransactionByID(id.Bytes())
	require.NoError(t, err)
	leaseTx, ok := tx.(*proto.LeaseWithSig)
	assert.Equal(t, true, ok)
	recipient := *leaseTx.Recipient.Address()
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
	correctRes, err := server.transactionToTransactionResponse(tx, true, false)
	require.NoError(t, err)
	res, err := stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient.
	recipientBody := recipient.Body()
	req = &g.TransactionsRequest{
		Recipient: &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: recipientBody}},
	}
	stream, err = cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient and ID.
	req = &g.TransactionsRequest{
		Recipient:      &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: recipientBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	stream, err = cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By sender, recipient and ID.
	req = &g.TransactionsRequest{
		Sender:         senderBody,
		Recipient:      &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: recipientBody}},
		TransactionIds: [][]byte{id.Bytes()},
	}
	stream, err = cl.GetTransactions(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestGetStatuses(t *testing.T) {
	params := defaultStateParams()
	st := newTestState(t, true, params, settings.MainNetSettings)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)
	utx := utxpool.New(utxSize, utxpool.NoOpValidator{}, settings.MainNetSettings)
	err := server.initServer(st, utx, sch)
	require.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	require.NoError(t, err)
	waves := proto.NewOptionalAssetWaves()
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), proto.Attachment("attachment"))
	err = tx.Sign(server.scheme, sk)
	require.NoError(t, err)
	txBytes, err := tx.MarshalBinary(scheme)
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
		{Id: id0.Bytes(), Height: 1, Status: g.TransactionStatus_CONFIRMED, ApplicationStatus: g.ApplicationStatus_SUCCEEDED},
		{Id: id1, Status: g.TransactionStatus_UNCONFIRMED, ApplicationStatus: g.ApplicationStatus_UNKNOWN},
		{Id: id2, Status: g.TransactionStatus_NOT_EXISTS, ApplicationStatus: g.ApplicationStatus_UNKNOWN},
	}

	req := &g.TransactionsByIdRequest{TransactionIds: ids}
	stream, err := cl.GetStatuses(ctx, req)
	require.NoError(t, err)
	for _, correctRes := range correstResults {
		res, err := stream.Recv()
		require.NoError(t, err)
		assertTransactionStatusesEqual(t, correctRes, res)
	}
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestGetUnconfirmed(t *testing.T) {
	params := defaultStateParams()
	st := newTestState(t, true, params, settings.MainNetSettings)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)
	utx := utxpool.New(utxSize, utxpool.NoOpValidator{}, settings.MainNetSettings)
	err := server.initServer(st, utx, sch)
	require.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	addrBody := addr.Body()
	sk, pk, err := crypto.GenerateKeyPair([]byte("whatever"))
	require.NoError(t, err)
	senderAddr, err := proto.NewAddressFromPublicKey(server.scheme, pk)
	require.NoError(t, err)
	waves := proto.NewOptionalAssetWaves()
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), []byte("attachment"))
	err = tx.Sign(server.scheme, sk)
	require.NoError(t, err)
	txBytes, err := tx.MarshalBinary(scheme)
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
	correctRes, err := server.transactionToTransactionResponse(tx, false, false)
	require.NoError(t, err)
	res, err := stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By recipient.
	req = &g.TransactionsRequest{
		Recipient: &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: addrBody}},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
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
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)

	// By sender, recipient and ID.
	senderAddrBody = senderAddr.Body()
	req = &g.TransactionsRequest{
		Sender:         senderAddrBody,
		Recipient:      &pb.Recipient{Recipient: &pb.Recipient_PublicKeyHash{PublicKeyHash: addrBody}},
		TransactionIds: [][]byte{id},
	}
	stream, err = cl.GetUnconfirmed(ctx, req)
	require.NoError(t, err)
	res, err = stream.Recv()
	require.NoError(t, err)
	assertTransactionResponsesEqual(t, correctRes, res)
	_, err = stream.Recv()
	assert.Equal(t, io.EOF, err)
}

func TestSign(t *testing.T) {
	params := defaultStateParams()
	st := newTestState(t, true, params, settings.MainNetSettings)
	ctx := withAutoCancel(t, context.Background())
	sch := createTestNetWallet(t)

	err := server.initServer(st, nil, sch)
	require.NoError(t, err)

	conn := connectAutoClose(t, grpcTestAddr)

	pk := keyPairs[0].Public

	addr, err := proto.NewAddressFromString("3PAWwWa6GbwcJaFzwqXQN5KQm7H96Y7SHTQ")
	require.NoError(t, err)
	waves := proto.NewOptionalAssetWaves()
	tx := proto.NewUnsignedTransferWithSig(pk, waves, waves, 100, 1, 100, proto.NewRecipientFromAddress(addr), []byte("attachment"))
	err = tx.GenerateID(server.scheme)
	require.NoError(t, err)
	require.NoError(t, err)
	txProto, err := tx.ToProtobuf(server.scheme)
	require.NoError(t, err)

	cl := g.NewTransactionsApiClient(conn)
	req := &g.SignRequest{Transaction: txProto, SignerPublicKey: pk.Bytes()}
	_, err = cl.Sign(ctx, req)
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unimplemented, s.Code())
	require.Equal(t, "method Sign not implemented", s.Message())
}

func TestBroadcast(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	h := mock.NewMockGrpcHandlers(ctrl)
	h.EXPECT().Broadcast(gomock.Any(), gomock.Any()).Return(&pb.SignedTransaction{}, nil)

	gRPCServer := createGRPCServerWithHandlers(h)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lis, err := net.Listen("tcp", "127.0.0.1:")
	require.NoError(t, err)
	defer lis.Close()
	go func() {
		if err := gRPCServer.Serve(lis); err != nil {
			log.Fatalf("server.Run(): %v\n", err)
		}
	}()
	defer gRPCServer.Stop()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	cl := g.NewTransactionsApiClient(conn)
	_, err = cl.Broadcast(ctx, &pb.SignedTransaction{})
	require.NoError(t, err)
}
