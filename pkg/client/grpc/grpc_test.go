package grpc

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"io"
	"testing"
	"time"
)

func TestTransactionsAPIClient(t *testing.T) {
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
		fmt.Println(msg)
	}
	assert.Equal(t, io.EOF, err)
}

func TestBlocksAPIClient(t *testing.T) {
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
	h := 1
	for b, err = getBlock(h); err == nil; b, err = getBlock(h) {
		fmt.Println("HEIGHT:", b.Height, "BLOCK:", b.Block)
		h++
	}
	assert.Equal(t, io.EOF, err)
}

func connect(t *testing.T) *grpc.ClientConn {
	conn, err := grpc.Dial("52.30.47.67:6870", grpc.WithInsecure())
	require.NoError(t, err)
	return conn
}
