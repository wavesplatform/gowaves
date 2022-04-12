package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/util/genesis_generator"
)

func main() {
	if err := run(); err != nil {
		log.Printf("[ERROR] %s", errorToLog(err))
		os.Exit(1)
	}
}

func run() error {
	var (
		scheme     string
		seed       string
		amounts    string
		pairs      string
		baseTarget uint64
		timestamp  int64
	)
	flag.StringVar(&scheme, "scheme", "C", "Network scheme byte, defaults to 'C'")
	flag.StringVar(&seed, "seed", "", "Master seed as Base58 string")
	flag.StringVar(&amounts, "amounts", "", "Comma separated transaction amounts")
	flag.StringVar(&pairs, "pairs", "", "Comma separated pairs of address or account seed and amount to produce genesis transactions, eg '3MvRmBpZf6Cm14dY5Nrrq2pj4587EzGTnj4:100_000_000,8GVECo9addsbFumLsmnAU3Cfz7UiF5TGm64zkZnfntdA:100_000'")
	flag.Uint64Var(&baseTarget, "base-target", 0, "Base Target value")
	flag.Int64Var(&timestamp, "timestamp", time.Now().UnixMilli(), "Block and transactions timestamp in ms, defaults to current time")
	flag.Parse()

	var (
		sc           byte
		transactions []genesis_generator.GenesisTransactionInfo
		bt           uint64
		ts           uint64
	)
	if len(scheme) != 1 {
		return errors.Errorf("invalid scheme '%s'", scheme)
	}
	sc = scheme[0]
	if timestamp <= 0 {
		return errors.Errorf("ivalid timestamp '%d'", timestamp)
	}
	ts = uint64(timestamp)
	switch {
	case len(pairs) != 0 && len(seed) == 0 && len(amounts) == 0:
		txs, err := parsePairs(pairs, sc, ts)
		if err != nil {
			return err
		}
		transactions = txs
	case len(pairs) == 0 && len(seed) != 0 && len(amounts) != 0:
		as, err := parseAmounts(amounts)
		if err != nil {
			return err
		}
		txs, err := makeTransactions([]byte(seed), as, sc, ts)
		if err != nil {
			return err
		}
		transactions = txs
	default:
		return errors.New("invalid combination of 'pairs' or 'seed' and 'amounts' parameters")
	}
	if baseTarget == 0 {
		return errors.New("no Base Target value")
	}
	bt = baseTarget

	block, err := genesis_generator.GenerateGenesisBlock(sc, transactions, bt, ts)
	if err != nil {
		return err
	}
	js, err := json.Marshal(block)
	if err != nil {
		return err
	}
	fmt.Println(string(js))
	return nil
}

func errorToLog(err error) string {
	if err == nil {
		return ""
	}
	msg := []rune(err.Error())
	msg[0] = unicode.ToUpper(msg[0])
	return string(msg)
}

func parsePairs(s string, scheme byte, ts uint64) ([]genesis_generator.GenesisTransactionInfo, error) {
	pairs := strings.Split(s, ",")
	r := make([]genesis_generator.GenesisTransactionInfo, 0, len(pairs))
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		amount, err := strconv.ParseUint(strings.Replace(parts[1], "_", "", -1), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount '%s'", parts[1])
		}
		addr, err := proto.NewAddressFromString(parts[0])
		if err != nil {
			seed, err := crypto.NewDigestFromBase58(parts[0])
			if err != nil {
				return nil, errors.Errorf("failed to convert '%s' to address or account seed", parts[0])
			}
			_, pk, err := crypto.GenerateKeyPair(seed[:])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate address from seed '%s'", seed.String())
			}
			addr, err = proto.NewAddressFromPublicKey(scheme, pk)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate address from seed '%s'", seed.String())
			}
		}
		r = append(r, genesis_generator.GenesisTransactionInfo{Address: addr, Amount: amount, Timestamp: ts})
	}
	return r, nil
}

func parseAmounts(s string) ([]uint64, error) {
	parts := strings.Split(s, ",")
	r := make([]uint64, 0, len(parts))
	for _, p := range parts {
		a, err := strconv.ParseUint(strings.Replace(p, "_", "", -1), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount '%s'", p)
		}
		r = append(r, a)
	}
	return r, nil
}

func makeTransactions(seed []byte, amounts []uint64, scheme byte, ts uint64) ([]genesis_generator.GenesisTransactionInfo, error) {
	r := make([]genesis_generator.GenesisTransactionInfo, 0, len(amounts))
	for i, amount := range amounts {
		iv := make([]byte, 4)
		binary.BigEndian.PutUint32(iv, uint32(i))
		s := append(iv, seed...)
		h, err := crypto.SecureHash(s)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate address from seed '%s'", string(seed))
		}
		_, pk, err := crypto.GenerateKeyPair(h[:])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate address from seed '%s'", string(seed))
		}
		addr, err := proto.NewAddressFromPublicKey(scheme, pk)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate address from seed '%s'", string(seed))
		}
		r = append(r, genesis_generator.GenesisTransactionInfo{Address: addr, Amount: amount, Timestamp: ts})
	}
	return r, nil
}
