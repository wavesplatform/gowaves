package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"google.golang.org/grpc"
)

func TestTransactionsAPIClient(t *testing.T) {
	t.SkipNow()
	conn := connect(t)
	defer conn.Close()

	c := NewTransactionsApiClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req := TransactionsRequest{}
	var err error
	uc, err := c.GetUnconfirmed(ctx, &req, grpc.EmptyCallOption{})
	require.NoError(t, err)
	var msg TransactionResponse
	for err = uc.RecvMsg(&msg); err == nil; err = uc.RecvMsg(&msg) {
		c := SafeConverter{}
		tx, err := c.SignedTransaction(msg.Transaction)
		require.NoError(t, err)
		js, err := json.Marshal(tx)
		require.NoError(t, err)
		fmt.Println(string(js))
	}
	assert.Equal(t, io.EOF, err)
}

func TestBlocksAPIClient(t *testing.T) {
	t.SkipNow()
	conn := connect(t)
	defer conn.Close()

	c := NewBlocksApiClient(conn)

	getBlock := func(h int) (*BlockWithHeight, error) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(15*time.Second))
		defer cancel()
		req := &BlockRequest{IncludeTransactions: true, Request: &BlockRequest_Height{int32(h)}}
		return c.GetBlock(ctx, req, grpc.EmptyCallOption{})
	}

	var err error
	var b *BlockWithHeight
	cnv := SafeConverter{}
	h := 1
	for b, err = getBlock(h); err == nil; b, err = getBlock(h) {
		cnv.Reset()
		txs, err := cnv.BlockTransactions(b)
		require.NoError(t, err)
		sb := strings.Builder{}
		sb.WriteRune('[')
		sb.WriteString(strconv.Itoa(len(txs)))
		sb.WriteRune(']')
		sb.WriteRune(' ')
		for _, tx := range txs {
			js, err := json.Marshal(tx)
			require.NoError(t, err)
			sb.WriteString(string(js))
			sb.WriteRune(',')
		}
		header, err := cnv.BlockHeader(b)
		require.NoError(t, err)
		bjs, err := json.Marshal(header)
		require.NoError(t, err)
		fmt.Println("HEIGHT:", b.Height, "BLOCK:", string(bjs), "TXS:", sb.String())
		h++
	}
	assert.Equal(t, io.EOF, err)
}

func TestAccountData(t *testing.T) {
	t.SkipNow()
	conn := connect(t)
	defer conn.Close()

	c := NewAccountsApiClient(conn)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(20*time.Second))
	defer cancel()
	addr, err := proto.NewAddressFromString("3MxHxW5VWq4KrWcbhFfxKrafXm4mL6rZHfj")
	require.NoError(t, err)
	b := make([]byte, len(addr.Bytes()))
	copy(b, addr.Bytes())
	req := &DataRequest{Address: b /*, Key: "13QuhSAkAueic5ncc8YRwyNxGQ6tRwVSS44a7uFgWsnk"*/}
	dc, err := c.GetDataEntries(ctx, req)
	require.NoError(t, err)
	var msg DataEntryResponse
	for err = dc.RecvMsg(&msg); err == nil; err = dc.RecvMsg(&msg) {
		con := SafeConverter{}
		e := con.entry(msg.Entry)
		require.NoError(t, con.err)
		fmt.Println(e.GetKey(), ":", e)
	}
	assert.Equal(t, io.EOF, err)
}

func connect(t *testing.T) *grpc.ClientConn {
	conn, err := grpc.Dial("testnet-aws-fr-1.wavesnodes.com:6870", grpc.WithInsecure())
	require.NoError(t, err)
	return conn
}
