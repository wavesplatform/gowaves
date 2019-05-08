package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var Cli struct {
	WavesNetwork string `kong:"wavesnetwork,short='n',help='Waves network.',required"`
	Address      string `kong:"address,short='a',help='Address connect to.',required"`
	Version      string `kong:"version,short='v',help='Version,(0.15.1).',required"`
}

func init() {
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
}

func main() {
	kong.Parse(&Cli)
	zap.S().Infof("%+v", Cli)

	version, err := parseVersion(Cli.Version)
	if err != nil {
		zap.S().Error(err)
		return
	}

	handshake := proto.Handshake{
		AppName:      Cli.WavesNetwork,
		Version:      version,
		NodeName:     "nodename",
		NodeNonce:    0x0,
		DeclaredAddr: proto.HandshakeTCPAddr{},
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	conn, err := net.Dial("tcp", Cli.Address)
	if err != nil {
		zap.S().Error(err)
		return
	}

	defer conn.Close()

	_, err = handshake.WriteTo(conn)
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Info("reading handshake")
	readH := proto.Handshake{}
	_, err = readH.ReadFrom(bufio.NewReader(conn))
	if err != nil {
		zap.S().Error(err)
		return
	}

	zap.S().Infof("readed handshake %+v", readH)

	go func() {
		expectedContentID := byte(0x15)

		for {
			bts, err := readPacket(conn)
			if err != nil {
				zap.S().Error(err)
				return
			}

			zap.S().Info("readed==", bts)

			if bts[proto.HeaderContentIDPosition] == expectedContentID {
				zap.S().Info(bts)
			}
			continue
		}
	}()

	sig, _ := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")

	sigs := proto.GetSignaturesMessage{
		Blocks: []crypto.Signature{sig},
	}

	zap.S().Info("writing GetSignaturesMessage bytes")
	_, err = sigs.WriteTo(conn)
	if err != nil {
		zap.S().Error(err)
		return
	}

	time.Sleep(5 * time.Minute)

}

func readPacket(r io.Reader) ([]byte, error) {
	packetLen := [4]byte{}
	_, err := io.ReadFull(r, packetLen[:])
	if err != nil {
		return nil, err
	}
	l := binary.BigEndian.Uint32(packetLen[:])
	buf := make([]byte, l+4)
	copy(buf, packetLen[:])
	_, err = io.ReadFull(r, buf[4:])
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func parseVersion(v string) (proto.Version, error) {
	rs := strings.Split(v, ".")
	if len(rs) != 3 {
		return proto.Version{}, errors.Errorf("incorrect version %s", v)
	}

	major, _ := strconv.ParseUint(rs[0], 10, 64)
	minot, _ := strconv.ParseUint(rs[1], 10, 64)
	patch, _ := strconv.ParseUint(rs[2], 10, 64)

	return proto.Version{
		Major: uint32(major),
		Minor: uint32(minot),
		Patch: uint32(patch),
	}, nil

}
