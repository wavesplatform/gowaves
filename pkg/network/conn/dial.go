package conn

// TODO seems that file not used anymore

import (
	"bufio"
	"context"
	"io"
)

type Connector struct {
	d    Dialer
	pool Pool
}

func NewConnector(d Dialer, pool Pool) *Connector {
	return &Connector{
		d:    d,
		pool: pool,
	}
}

type dialParams struct {
	addr         string
	network      string
	toRemoteCh   chan []byte
	fromRemoteCh chan []byte
	errCh        chan error
	sendFunc     func(conn io.Writer, ctx context.Context, toRemoteCh chan []byte, errCh chan error)
	recvFunc     func(pool Pool, reader io.Reader, fromRemoteCh chan []byte, errCh chan error)
}

func (a *Connector) dial(params dialParams) (Connection, error) {
	conn, err := a.d(params.network, params.addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	impl := &ConnectionImpl{
		cancel: cancel,
		conn:   conn,
	}

	bufReader := bufio.NewReaderSize(conn, 2*1024*1024)

	go params.recvFunc(a.pool, bufReader, params.fromRemoteCh, params.errCh)
	go params.sendFunc(conn, ctx, params.toRemoteCh, params.errCh)

	return impl, nil
}

func (a *Connector) Dial(network string, addr string, toRemoteCh chan []byte, fromRemoteCh chan []byte, errCh chan error) (Connection, error) {
	params := dialParams{
		addr:         addr,
		network:      network,
		toRemoteCh:   toRemoteCh,
		fromRemoteCh: fromRemoteCh,
		errCh:        errCh,
		sendFunc:     sendToRemote,
		recvFunc:     recvFromRemote,
	}

	return a.dial(params)
}
