package connection

import (
	"context"
	"fmt"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"net"
	"time"
)

type connVersion struct {
	conn    net.Conn
	version proto.Version
}

type Connection struct {
	addr             string
	conn             net.Conn
	connected        bool
	readFromRemoteCh chan []byte
	writeToRemoteCh  chan []byte
	errCh            chan error
	version          proto.Version
	declAddr         proto.PeerInfo
	ctx              context.Context
	connectCh        chan connVersion
	disconnectCh     chan struct{}
	infoCh           chan interface{}
}

func NewConnection(ctx context.Context, addr string, v proto.Version, readFromRemoteCh chan []byte, writeToRemoteCh chan []byte, infoCh chan interface{}) *Connection {
	a := &Connection{
		addr:             addr,
		ctx:              ctx,
		version:          v,
		connectCh:        make(chan connVersion, 1),
		readFromRemoteCh: readFromRemoteCh,
		writeToRemoteCh:  writeToRemoteCh,
		connected:        false,
		infoCh:           infoCh,
	}
	go a.run()
	go a.connect()
	return a
}

func (a *Connection) run() {

	for {
		select {
		case <-a.ctx.Done():
			return
		case connVersion := <-a.connectCh:
			a.conn = connVersion.conn
			a.connected = true
			go func() {
				for {
					b := make([]byte, 65536)
					_, err := a.conn.Read(b)
					if err != nil {
						select {
						case a.errCh <- err:
						default:
						}
						return
					}

					a.readFromRemoteCh <- b
				}
			}()
		case bts := <-a.writeToRemoteCh:
			if a.connected {
				_, err := a.conn.Write(bts)
				if err != nil {
					select {
					case a.errCh <- err:
					default:
					}
				}
			}
		case <-a.disconnectCh:
			a.conn.Close()
			a.connected = false
		}
	}
}

func (a *Connection) connect() error {
	zap.S().Debug("called Connection connect")
	v := a.version
	minor := v.Minor

	for i := minor; i > 0; i-- {

		fmt.Println("i < v.Minor", i < v.Minor, i, v.Minor)

		if i < v.Minor {
			ticker := time.NewTimer(31 * time.Minute)
			select {
			case <-ticker.C:
			case <-a.ctx.Done():
				return a.ctx.Err()
			}
		}

		conn, err := net.Dial("tcp", a.addr)
		if err != nil {
			fmt.Println(err)
			zap.S().Error(err)
			continue
		}

		bytes, err := a.declAddr.MarshalBinary()
		if err != nil {
			fmt.Println(err)
			zap.S().Error(err)
			continue
		}

		handshake := proto.Handshake{
			Name:              "wavesW",
			Version:           proto.Version{Major: 0, Minor: minor, Patch: 0},
			NodeName:          "gowaves",
			NodeNonce:         0x0,
			DeclaredAddrBytes: bytes,
			Timestamp:         proto.NewTimestampFromTime(time.Now()),
		}

		_, err = handshake.WriteTo(conn)
		if err != nil {
			zap.S().Error("failed to send handshake: ", err)
			continue
		}
		_, err = handshake.ReadFrom(conn)
		if err != nil {
			fmt.Println(err)
			zap.S().Error("failed to read handshake: ", err)
			continue
		}

		a.connectCh <- connVersion{
			conn:    conn,
			version: proto.Version{Major: 0, Minor: minor, Patch: 0},
		}
		fmt.Println("connected")
		return nil
	}

	return nil
}
