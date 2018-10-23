package server

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/wavesplatform/gowaves/pkg/p2p"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type Server struct {
	BootPeerAddrs []string
	Listen        string
}

func handleRequest(conn net.Conn) {
}

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	var major, minor, patch uint32
	minor = 5

	dialer := net.Dialer{}

	for i := 0; i < 20; i++ {
		patch = uint32(i)
		time.Sleep(20 * time.Second)
		conn, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			continue
		}

		fmt.Printf("Trying to connect with version %v %v %v", major, minor, patch)
		handshake := proto.Handshake{Name: "wavesT",
			Version:           proto.Version{Major: major, Minor: minor, Patch: patch},
			NodeName:          "gowaves",
			NodeNonce:         0x0,
			DeclaredAddrBytes: []byte{},
			Timestamp:         uint64(time.Now().Unix())}

		_, err = handshake.WriteTo(conn)
		if err != nil {
			zap.S().Error("failed to send handshake: ", err)
			continue
		}
		_, err = handshake.ReadFrom(conn)
		if err != nil {
			zap.S().Error("failed to read handshake: ", err)
			continue
		}

		var b []byte
		b, e := json.Marshal(handshake)
		if e != nil {
			return nil, err
		}
		js := string(b)
		zap.S().Info("received handshake: ", js)

		return conn, nil
	}
	return nil, errors.New("TODO")
}

func handleClient(peer string) {
	customTransport := p2p.Transport{DialContext: dialContext}
	conn, _ := p2p.NewConn(
		p2p.WithVersion(proto.Version{Major: 0, Minor: 5, Patch: 14}),
		p2p.WithTransport(&customTransport),
		p2p.WithRemote("tcp", peer),
	)
	err := conn.DialContext(context.Background(), "tcp", peer)
	if err != nil {
		zap.S().Error("error while dialing: ", err)
		return
	}
	bufConnW := bufio.NewWriter(conn)
	bufConn := bufio.NewReader(conn)

	var gp proto.GetPeersMessage
	gp.WriteTo(bufConnW)
	bufConnW.Flush()

LOOP:
	for {
		buf, err := bufConn.Peek(9)
		if err != nil {
			zap.S().Error("error while reading from connection: ", err)
			break
		}

		switch msgType := buf[8]; msgType {
		case proto.ContentIDGetPeers:
			var gp proto.GetPeersMessage
			_, err := gp.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("error while receiving GetPeersMessage: ", err)
				break
			}

		case proto.ContentIDPeers:
			var p proto.PeersMessage
			_, err := p.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("failed to read Peers message: ", err)
				break
			}
			var b []byte
			b, e := json.Marshal(p)
			if e != nil {
				return
			}
			js := string(b)
			zap.S().Info("Got peers", js)
		case proto.ContentIDScore:
			var s proto.ScoreMessage
			_, err := s.ReadFrom(bufConn)
			if err != nil {
				zap.S().Error("failed to read Score message: ", err)
				break
			}
		default:
			l := binary.BigEndian.Uint32(buf[:4])
			arr := make([]byte, l)
			_, err := io.ReadFull(bufConn, arr)
			if err != nil {
				break LOOP
			}
			break LOOP
		}
	}
}

func (m *Server) Run() {
	if m.Listen == "" {
		return
	}

	l, err := net.Listen("tcp", m.Listen)

	if err != nil {
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			zap.S().Error("error while accepting connections: ", err)
			break
		}

		go handleRequest(conn)
	}
}

func (m *Server) RunClients() {

	for _, peer := range m.BootPeerAddrs {

		go handleClient(peer)
	}
}
