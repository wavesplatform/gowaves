package data

import (
	"bufio"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"os"
	"sort"
	"strings"
)

type Symbols struct {
	tickers map[string]crypto.Digest
	tokens  map[crypto.Digest]string
}

type Substitution struct {
	Symbol  string        `json:"symbol"`
	AssetID crypto.Digest `json:"assetID"`
}

type BySymbols []Substitution

func (a BySymbols) Len() int {
	return len(a)
}

func (a BySymbols) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a BySymbols) Less(i, j int) bool {
	return a[i].Symbol < a[j].Symbol
}

func (s *Symbols) put(ticker string, id crypto.Digest) {
	s.tickers[ticker] = id
	s.tokens[id] = ticker
}

func (s *Symbols) Count() int {
	return len(s.tickers)
}

func (s *Symbols) All() []Substitution {
	r := make([]Substitution, len(s.tickers))
	i := 0
	for k, v := range s.tickers {
		r[i] = Substitution{k, v}
		i++
	}
	sort.Sort(BySymbols(r))
	return r
}

func (s *Symbols) Tokens() map[crypto.Digest]string {
	return s.tokens
}

func (s *Symbols) ParseTicker(ticker string) (crypto.Digest, error) {
	id, err := crypto.NewDigestFromBase58(ticker)
	if err != nil {
		ticker = strings.ToUpper(ticker)
		if ticker == proto.WavesAssetName {
			return crypto.Digest{}, nil
		}
		id, ok := s.tickers[ticker]
		if !ok {
			return crypto.Digest{}, errors.Errorf("unknown ticker or invalid asset ID '%s'", ticker)
		}
		return id, nil
	}
	return id, nil
}

func ImportSymbols(name string) (*Symbols, error) {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to import symbols from file '%s'", name)
	}
	r := &Symbols{map[string]crypto.Digest{proto.WavesAssetName: WavesID}, map[crypto.Digest]string{WavesID: proto.WavesAssetName}}
	f, err := os.Open(name)
	if err != nil {
		return nil, wrapError(err)
	}
	s := bufio.NewScanner(f)
	i := 1
	for s.Scan() {
		fs := strings.Fields(s.Text())
		if l := len(fs); l < 2 {
			return nil, wrapError(errors.Errorf("incorrect fields count %d on line %d", l, i))
		}
		id, err := crypto.NewDigestFromBase58(fs[1])
		if err != nil {
			return nil, wrapError(err)
		}
		ticker := strings.ToUpper(fs[0])
		r.put(ticker, id)
		i++
	}
	return r, nil
}
