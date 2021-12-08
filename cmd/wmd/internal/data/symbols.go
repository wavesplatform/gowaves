package data

import (
	"bufio"
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	g "github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type Substitution struct {
	Symbol  string  `json:"symbol"`
	AssetID AssetID `json:"assetID"`
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

type Symbols struct {
	tickers map[string]crypto.Digest
	tokens  map[crypto.Digest]string
	mu      sync.RWMutex
	oracle  proto.WavesAddress
	scheme  byte
}

func NewSymbolsFromFile(name string, oracle proto.WavesAddress, scheme byte) (*Symbols, error) {
	wrapError := func(err error) error {
		return errors.Wrapf(err, "failed to import symbols from file '%s'", name)
	}
	r := &Symbols{
		tickers: map[string]crypto.Digest{proto.WavesAssetName: WavesID},
		tokens:  map[crypto.Digest]string{WavesID: proto.WavesAssetName},
		oracle:  oracle,
		scheme:  scheme,
	}
	f, err := os.Open(name) // #nosec: in this case check for prevent G304 (CWE-22) is not necessary
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

func (s *Symbols) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.tickers)
}

func (s *Symbols) All() []Substitution {
	r := make([]Substitution, len(s.tickers))
	i := 0
	s.mu.RLock()
	for k, v := range s.tickers {
		r[i] = Substitution{k, AssetID(v)}
		i++
	}
	s.mu.RUnlock()
	sort.Sort(BySymbols(r))
	return r
}

func (s *Symbols) Token(id crypto.Digest) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.tokens[id]
	return v, ok
}

func (s *Symbols) ParseTicker(ticker string) (crypto.Digest, error) {
	id, err := crypto.NewDigestFromBase58(ticker)
	if err != nil {
		ticker = strings.ToUpper(ticker)
		if ticker == proto.WavesAssetName {
			return crypto.Digest{}, nil
		}
		s.mu.RLock()
		defer s.mu.RUnlock()
		id, ok := s.tickers[ticker]
		if !ok {
			return crypto.Digest{}, errors.Errorf("unknown ticker or invalid asset ID '%s'", ticker)
		}
		return id, nil
	}
	return id, nil
}

func (s *Symbols) UpdateFromOracle(conn *grpc.ClientConn) error {
	if state := conn.GetState(); state != connectivity.Ready {
		return errors.Errorf("invalid gRPC connection state: %s", state.String())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := g.NewAccountsApiClient(conn)
	req := &g.DataRequest{Address: s.oracle.Bytes()}
	dc, err := c.GetDataEntries(ctx, req)
	if err != nil {
		return err
	}
	var msg g.DataEntryResponse
	converter := proto.ProtobufConverter{FallbackChainID: s.scheme}
	count := 0
	s.mu.Lock()
	defer s.mu.Unlock()
	for err = dc.RecvMsg(&msg); err == nil; err = dc.RecvMsg(&msg) {
		entry, err := converter.Entry(msg.Entry)
		if err != nil {
			return err
		}
		switch te := entry.(type) {
		case *proto.StringDataEntry:
			ticker := te.Key
			if strings.HasPrefix(ticker, "wpo_") {
				continue
			}
			id, err := crypto.NewDigestFromBase58(te.Value)
			if err != nil {
				continue
			}
			s.put(ticker, id)
			count++
		default:
			continue
		}
	}
	if err != io.EOF {
		return err
	}
	zap.S().Infof("Oracle: %d tickers updated", count)
	return nil
}

func (s *Symbols) put(ticker string, id crypto.Digest) {
	s.tickers[ticker] = id
	s.tokens[id] = ticker
}
