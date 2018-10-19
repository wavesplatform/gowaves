package server

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

type Server struct {
	BootPeerAddrs []string
	Listen        string
}

func handleRequest(conn net.Conn) {
}

func handleClient(conn net.Conn) {
	handshake := proto.Handshake{Name: "wavesT",
		VersionMajor:      0x0,
		VersionMinor:      0xe,
		VersionPatch:      0x4,
		NodeName:          "gowaves",
		NodeNonce:         0x0,
		DeclaredAddrBytes: []byte{},
		Timestamp:         uint64(time.Now().Unix())}

	_, err := handshake.WriteTo(conn)
	if err != nil {
		zap.S().Error("failed to send handshake: ", err)
		return
	}
	_, err = handshake.ReadFrom(conn)
	if err != nil {
		zap.S().Error("failed to read handshake: ", err)
		return
	}

	var b []byte
	b, e := json.Marshal(handshake)
	if e != nil {
		return
	}
	js := string(b)
	zap.S().Info("received handshake: ", js)

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
		conn, err := net.Dial("tcp", peer)
		if err != nil {
			zap.S().Error("failed to connect to peer: ", err)
			continue
		}

		go handleClient(conn)
	}
}
