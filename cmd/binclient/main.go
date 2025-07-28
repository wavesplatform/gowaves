package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/logging"
	"github.com/wavesplatform/gowaves/pkg/proto"
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

	slog.Info("Application arguments", "args", cli)
}

func init() {
	flag.Parse()
	slog.SetDefault(slog.New(logging.NewHandler(logging.LoggerPrettyNoColor, slog.LevelDebug)))
	printCLIArgsToLog()
}

func main() {
	if *wavesNetwork == "" {
		slog.Error("Please, provide 'waves-network' CLI argument")
		os.Exit(1)
	}
	if *address == "" {
		slog.Error("Please, provide 'address' CLI argument")
		os.Exit(1)
	}
	if *version == "" {
		slog.Error("Please, provide 'version' CLI argument")
		os.Exit(1)
	}

	parsedVersion, err := proto.NewVersionFromString(*version)
	if err != nil {
		slog.Error("Failed to parse version", logging.Error(err))
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
		slog.Error("Failed to dial", logging.Error(err))
		return
	}

	defer func() {
		if clErr := conn.Close(); clErr != nil {
			slog.Error("Failed to close connection", logging.Error(clErr))
		}
	}()

	_, err = handshake.WriteTo(conn)
	if err != nil {
		slog.Error("Failed to write handshake", logging.Error(err))
		return
	}

	slog.Info("Reading handshake")
	readH := proto.Handshake{}
	_, err = readH.ReadFrom(bufio.NewReader(conn))
	if err != nil {
		slog.Error("Failed to read handshake", logging.Error(err))
		return
	}

	slog.Info("Handshake complete", "handshake", readH)

	go func() {
		const expectedContentID = byte(proto.ContentIDSignatures)

		for {
			bts, rErr := readPacket(conn)
			if rErr != nil {
				slog.Error("Failed to read packet", logging.Error(rErr))
				return
			}

			slog.Info("Got packet", "data", hex.EncodeToString(bts))

			if bts[proto.HeaderContentIDPosition] == expectedContentID {
				slog.Info("Received Signatures message", "data", hex.EncodeToString(bts))
			}
			continue
		}
	}()

	sig, _ := crypto.NewSignatureFromBase58("FSH8eAAzZNqnG8xgTZtz5xuLqXySsXgAjmFEC25hXMbEufiGjqWPnGCZFt6gLiVLJny16ipxRNAkkzjjhqTjBE2")

	sigs := proto.GetSignaturesMessage{
		Signatures: []crypto.Signature{sig},
	}

	slog.Info("Sending GetSignatures message")
	_, err = sigs.WriteTo(conn)
	if err != nil {
		slog.Error("Failed to write GetSignatures message", logging.Error(err))
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
