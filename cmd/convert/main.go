package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"unicode"

	"github.com/wavesplatform/gowaves/pkg/proto"
)

const inputFileSizeLimit = 10 * 1024 * 1024

func main() {
	log.SetOutput(os.Stderr)
	if err := run(); err != nil {
		log.Println(capitalize(err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg := config{}
	if cfgErr := cfg.parse(); cfgErr != nil {
		return cfgErr
	}
	defer cfg.close()

	data, err := io.ReadAll(io.LimitReader(cfg.in, inputFileSizeLimit))
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	if json.Valid(data) {
		return handleJSON(data, cfg)
	}
	return handleBinary(data, cfg)
}

func handleJSON(data []byte, cfg config) error {
	tx, rErr := fromJSON(data, cfg)
	if rErr != nil {
		return rErr
	}
	tx, sErr := sign(tx, cfg)
	if sErr != nil {
		return sErr
	}
	if cfg.toJSON {
		return toJSON(tx, cfg)
	}
	return toBinary(tx, cfg)
}

func handleBinary(data []byte, cfg config) error {
	tx, rErr := fromBinary(data, cfg)
	if rErr != nil {
		return rErr
	}
	tx, sErr := sign(tx, cfg)
	if sErr != nil {
		return sErr
	}
	if cfg.toBinary {
		return toBinary(tx, cfg)
	}
	return toJSON(tx, cfg)
}

func sign(tx proto.Transaction, cfg config) (proto.Transaction, error) {
	if cfg.sk != nil {
		if err := tx.Sign(cfg.scheme, *cfg.sk); err != nil {
			return nil, fmt.Errorf("failed to sign transaction: %w", err)
		}
	}
	return tx, nil
}

func fromJSON(data []byte, cfg config) (proto.Transaction, error) {
	tt := proto.TransactionTypeVersion{}
	if err := json.Unmarshal(data, &tt); err != nil {
		return nil, fmt.Errorf("failed read transaction from JSON: %w", err)
	}
	tx, err := proto.GuessTransactionType(&tt)
	if err != nil {
		return nil, fmt.Errorf("failed read transaction from JSON: %w", err)
	}
	if umErr := proto.UnmarshalTransactionFromJSON(data, cfg.scheme, tx); umErr != nil {
		return nil, fmt.Errorf("failed read transaction from JSON: %w", err)
	}
	return tx, nil
}

func toJSON(tx proto.Transaction, cfg config) error {
	js, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	if _, wErr := cfg.out.Write(js); wErr != nil {
		return fmt.Errorf("failed to write transaction: %w", wErr)
	}
	return nil
}

func toBinary(tx proto.Transaction, cfg config) error {
	bts, err := proto.MarshalTx(cfg.scheme, tx)
	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %w", err)
	}
	var w = cfg.out
	if cfg.base64 {
		w = base64.NewEncoder(base64.StdEncoding, cfg.out)
		defer func(w io.Closer) {
			if clErr := w.Close(); clErr != nil {
				log.Printf("failed to close Base64 encoder: %v", clErr)
			}
		}(w)
	}
	if _, wErr := w.Write(bts); wErr != nil {
		return fmt.Errorf("failed to write transaction: %w", wErr)
	}
	return nil
}

func fromBinary(data []byte, cfg config) (proto.Transaction, error) {
	var bts []byte
	if cfg.base64 {
		bts = make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		cnt, err := base64.StdEncoding.Decode(bts, data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode from Base64: %w", err)
		}
		bts = bts[:cnt]
	} else {
		bts = data
	}
	tx, err := proto.SignedTxFromProtobuf(bts)
	if err != nil {
		if tx, err = proto.BytesToTransaction(bts, cfg.scheme); err != nil {
			return nil, fmt.Errorf("failed to read transaction from binary: %w", err)
		}
	}
	return tx, nil
}

func capitalize(str string) string {
	if len(str) == 0 {
		return str
	}
	runes := []rune(str)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
