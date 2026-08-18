package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"unicode"

	"github.com/wavesplatform/gowaves/pkg/crypto/bls"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

func main() {
	log.SetOutput(os.Stderr)
	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		log.Println(capitalize(err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg := config{}
	if err := cfg.parse(os.Args); err != nil {
		return err
	}

	bSK, err := bls.GenerateSecretKey(cfg.sk.Bytes(), bls.WithNoPreHash())
	if err != nil {
		return fmt.Errorf("failed to derive BLS secret key: %w", err)
	}
	bPK, err := bSK.PublicKey()
	if err != nil {
		return fmt.Errorf("failed to derive BLS public key: %w", err)
	}
	_, cs, err := bls.ProvePoP(bSK, bPK, cfg.height)
	if err != nil {
		return fmt.Errorf("failed to create proof of possession: %w", err)
	}

	tx := proto.NewUnsignedCommitToGenerationWithProofs(
		proto.MaxCommitToGenerationTransactionVersion,
		cfg.pk,
		cfg.height,
		bPK,
		cs,
		cfg.fee,
		cfg.timestamp,
	)
	if jsErr := json.NewEncoder(os.Stdout).Encode(tx); jsErr != nil {
		return fmt.Errorf("failed to encode transaction to JSON: %w", jsErr)
	}
	return nil
}

func capitalize(str string) string {
	if len(str) == 0 {
		return str
	}
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
