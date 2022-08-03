package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"io"
	"net"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
)

var (
	wavesNetwork = flag.String("waves-network", "", "Waves network.")
	address      = flag.String("address", "", "Address connect to.")
	version      = flag.String("version", "", "Version, for example: (0.15.1).")
)

func printCLIArgsToLog() {
	type cliArgs struct {
		wavesNetwork string
		address      string
		version      string
	}
	cli := cliArgs{
		wavesNetwork: *wavesNetwork,
		address:      *address,
		version:      *version,
	}

	zap.S().Infof("CLI args: %+v", cli)
}

func init() {
	flag.Parse()
	logger, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(logger)
	printCLIArgsToLog()
}

func main() {
	if *wavesNetwork == "" {
		zap.S().Fatal("please, provide 'waves-network' CLI argument")
	}
	if *address == "" {
		zap.S().Fatal("please, provide 'address' CLI argument")
	}
	if *version == "" {
		zap.S().Fatal("please, provide 'version' CLI argument")
	}

	parsedVersion, err := proto.NewVersionFromString(*version)
	if err != nil {
		zap.S().Error(err)
		return
	}

	handshake := proto.Handshake{
		AppName:      *wavesNetwork,
		Version:      parsedVersion,
		NodeName:     "nodename",
		NodeNonce:    0x0,
		DeclaredAddr: proto.HandshakeTCPAddr{},
		Timestamp:    proto.NewTimestampFromTime(time.Now()),
	}

	conn, err := net.Dial("tcp", *address)
	if err != nil {
		zap.S().Error(err)
		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			zap.S().Errorf("failed to close connetion: %v", err)
		}
	}()

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
		const expectedContentID = byte(proto.ContentIDSignatures)

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
		Signatures: []crypto.Signature{sig},
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
