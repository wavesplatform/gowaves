package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/cmd/blockcmp/internal"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves"
	"github.com/wavesplatform/gowaves/pkg/grpc/generated/waves/node/grpc"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
	"github.com/wavesplatform/gowaves/pkg/util/common"
	"go.uber.org/zap"
	g "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type report struct {
	scheme       byte
	height       int
	endpoints    []string
	blockIDs     []proto.BlockID
	transactions [][]proto.Transaction
	results      [][]*waves.InvokeScriptResult
}

func newReport(height, capacity int, scheme byte, endpoints []string) *report {
	return &report{
		scheme:       scheme,
		height:       height,
		endpoints:    endpoints,
		blockIDs:     make([]proto.BlockID, capacity),
		transactions: make([][]proto.Transaction, capacity),
		results:      make([][]*waves.InvokeScriptResult, capacity),
	}
}

func (r *report) setBlockID(i int, id proto.BlockID) {
	r.blockIDs[i] = id
}

func (r *report) addTransactions(i int, txs []proto.Transaction) {
	r.transactions[i] = txs
}

func (r *report) addResults(i int, results []*waves.InvokeScriptResult) {
	r.results[i] = results
}

func (r *report) String() string {
	sb := new(strings.Builder)
	for i := 1; i < len(r.endpoints); i++ {
		if r.blockIDs[0] != r.blockIDs[i] {
			sb.WriteString(fmt.Sprintf("Endpoint %s has different block ID at height %d: %s != %s",
				r.endpoints[i], r.height, r.blockIDs[0].String(), r.blockIDs[i].String()))
		}
		if len(r.transactions[0]) != len(r.transactions[i]) {
			sb.WriteString(fmt.Sprintf("Endpoint %s has different transactions count at height %d: %d != %d",
				r.endpoints[i], r.height, len(r.transactions[0]), len(r.transactions[i])))
		}
		for j := 0; j < len(r.transactions[0]); j++ {
			if r.results[0][j] != r.results[i][j] {
				id, err := r.transactions[i][j].GetID(r.scheme)
				if err != nil {
					panic(err)
				}
				diff := resultDiff(r.results[0][j], r.results[i][j], r.scheme)
				if diff != "" {
					sb.WriteString(fmt.Sprintf("Endpoint %s has different result for transaction '%s':\n%s",
						r.endpoints[i], base58.Encode(id), diff))
					sb.WriteString("\n")
				}
			}
		}
	}
	return sb.String()
}

func main() {
	err := run()
	if err != nil {
		os.Exit(1)
	}
}

func run() error {
	var (
		nodes          string
		height         int
		blockchainType string
	)

	common.SetupLogger("INFO")

	flag.StringVar(&nodes, "nodes", "", "Nodes gRPC API URLs separated by comma")
	flag.IntVar(&height, "height", 0, "Height to compare blocks at")
	flag.StringVar(&blockchainType, "blockchain-type", "mainnet",
		"Blockchain type mainnet/testnet/stagenet, default value is mainnet")
	flag.Parse()

	if nodes == "" {
		err := errors.New("empty nodes list")
		zap.S().Errorf("Failed to parse nodes' gRPC API addresses: %v", err)
		return err
	}
	if height == 0 {
		err := errors.Errorf("zero height")
		zap.S().Errorf("Failed to intialize: %v", err)
		return err
	}
	bs, err := settings.BlockchainSettingsByTypeName(blockchainType)
	if err != nil {
		zap.S().Errorf("Failed to load blockchain settings: %v", err)
		return err
	}

	endpoints, err := parseNodesList(nodes)
	if err != nil {
		zap.S().Errorf("Failed to parse nodes' gRPC API addresses: %v", err)
		return err
	}
	if len(endpoints) < 2 {
		err := errors.New("not enough nodes to compare")
		zap.S().Errorf("Failed to initialize: %v", err)
		return err
	}
	clients, err := dialEndpoints(endpoints)
	if err != nil {
		zap.S().Errorf("Failed to connect to gRPC endpoints: %v", err)
		return err
	}
	rep, err := compareBlocks(clients, bs.AddressSchemeCharacter, height)
	if err != nil {
		zap.S().Errorf("Failed to compare blocks: %v", err)
		return err
	}
	fmt.Println(rep.String())
	return nil
}

func parseNodesList(nodes string) ([]string, error) {
	parts := strings.Split(nodes, ",")
	r := make([]string, 0, len(parts))
	for _, p := range parts {
		r = append(r, strings.TrimSpace(p))
	}
	return r, nil
}

func dialEndpoints(endpoints []string) ([]*g.ClientConn, error) {
	r := make([]*g.ClientConn, len(endpoints))
	for i, e := range endpoints {
		c, err := g.Dial(e, g.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
		r[i] = c
	}
	return r, nil
}

func compareBlocks(clients []*g.ClientConn, scheme byte, height int) (*report, error) {
	endpoints := make([]string, len(clients))
	for i, cl := range clients {
		endpoints[i] = cl.Target()
	}
	rep := newReport(height, len(clients), scheme, endpoints)
	for i, c := range clients {
		h, err := nodeHeight(c)
		if err != nil {
			return nil, err
		}
		if height > h {
			return nil, errors.Errorf("height %d is above of blockchain tip (%d) at node %s",
				height, h, c.Target())
		}
		blockID, txs, err := blockTransactions(c, scheme, height)
		if err != nil {
			return nil, err
		}
		results, err := transactionResults(c, scheme, txs)
		if err != nil {
			return nil, err
		}
		rep.setBlockID(i, blockID)
		rep.addTransactions(i, txs)
		rep.addResults(i, results)
	}
	return rep, nil
}

func nodeHeight(c *g.ClientConn) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	api := grpc.NewBlocksApiClient(c)
	h, err := api.GetCurrentHeight(ctx, &empty.Empty{}, g.EmptyCallOption{})
	if err != nil {
		return 0, err
	}
	return int(h.Value), nil
}

func blockTransactions(c *g.ClientConn, scheme byte, height int) (proto.BlockID, []proto.Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	api := grpc.NewBlocksApiClient(c)
	request := &grpc.BlockRequest{
		Request:             &grpc.BlockRequest_Height{Height: int32(height)},
		IncludeTransactions: true,
	}
	b, err := api.GetBlock(ctx, request, g.EmptyCallOption{})
	if err != nil {
		return proto.BlockID{}, nil, err
	}

	cnv := proto.ProtobufConverter{FallbackChainID: scheme}
	header, err := cnv.BlockHeader(b.GetBlock())
	if err != nil {
		return proto.BlockID{}, nil, err
	}

	txs, err := cnv.SignedTransactions(b.GetBlock().GetTransactions())
	if err != nil {
		return proto.BlockID{}, nil, err
	}
	return header.ID, txs, nil
}

func transactionResults(c *g.ClientConn, scheme byte, txs []proto.Transaction) ([]*waves.InvokeScriptResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	api := grpc.NewTransactionsApiClient(c)
	r := make([]*waves.InvokeScriptResult, len(txs))
	for i := 0; i < len(txs); i++ {
		id, err := txs[i].GetID(scheme)
		if err != nil {
			return nil, err
		}
		request := &grpc.TransactionsRequest{
			TransactionIds: [][]byte{id},
		}
		//goland:noinspection GoDeprecation
		stream, err := api.GetStateChanges(ctx, request, g.EmptyCallOption{}) // nolint
		if err != nil {
			return nil, err
		}
		rsp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			return nil, err
		}
		r[i] = rsp.GetResult()
	}
	return r, nil
}

func resultDiff(a, b *waves.InvokeScriptResult, scheme byte) string {
	sb := new(strings.Builder)
	if len(a.GetData()) != 0 || len(b.GetData()) != 0 {
		addDataDiff(sb, internal.ExtractDataEntries(a), internal.ExtractDataEntries(b))
	}
	if len(a.GetTransfers()) != 0 || len(b.GetTransfers()) != 0 {
		addTransfersDiff(sb, internal.ExtractTransfers(a, scheme), internal.ExtractTransfers(b, scheme))
	}
	if len(a.GetIssues()) != 0 || len(b.GetIssues()) != 0 {
		addIssuesDiff(sb, internal.ExtractIssues(a), internal.ExtractIssues(b))
	}
	if len(a.GetReissues()) != 0 || len(b.GetReissues()) != 0 {
		addReissuesDiff(sb, internal.ExtractReissues(a), internal.ExtractReissues(b))
	}
	if len(a.GetBurns()) != 0 || len(b.GetBurns()) != 0 {
		addBurnsDiff(sb, internal.ExtractBurns(a), internal.ExtractBurns(b))
	}
	if len(a.GetSponsorFees()) != 0 || len(b.GetSponsorFees()) != 0 {
		addSponsorFeesDiff(sb, internal.ExtractSponsorships(a), internal.ExtractSponsorships(b))
	}
	if len(a.GetLeases()) != 0 || len(b.GetLeases()) != 0 {
		addLeasesDiff(sb, internal.ExtractLeases(a, scheme), internal.ExtractLeases(b, scheme))
	}
	if len(a.GetLeaseCancels()) != 0 || len(b.GetLeaseCancels()) != 0 {
		addLeaseCancelsDiff(sb, internal.ExtractLeaseCancels(a), internal.ExtractLeaseCancels(b))
	}
	if a.GetErrorMessage().GetText() != b.GetErrorMessage().GetText() {
		sb.WriteString("\tError:\n")
		sb.WriteString(fmt.Sprintf("\t-%s\n\t+%s\n",
			a.GetErrorMessage().GetText(), b.GetErrorMessage().GetText()))
	}
	return sb.String()
}

func addDataDiff(sb *strings.Builder, a, b []internal.DataEntry) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tData Entries:\n")
		sb.WriteString(lsb.String())
	}
}

func addTransfersDiff(sb *strings.Builder, a, b []internal.Transfer) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tTransfers:\n")
		sb.WriteString(lsb.String())
	}
}

func addIssuesDiff(sb *strings.Builder, a, b []internal.Issue) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tIssues:\n")
		sb.WriteString(lsb.String())
	}
}

func addReissuesDiff(sb *strings.Builder, a, b []internal.Reissue) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tReissues:\n")
		sb.WriteString(lsb.String())
	}
}

func addBurnsDiff(sb *strings.Builder, a, b []internal.Burn) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tBurns:\n")
		sb.WriteString(lsb.String())
	}
}

func addSponsorFeesDiff(sb *strings.Builder, a, b []internal.Sponsorship) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tSponsorships:\n")
		sb.WriteString(lsb.String())
	}
}

func addLeasesDiff(sb *strings.Builder, a, b []internal.Lease) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tLeases:\n")
		sb.WriteString(lsb.String())
	}
}

func addLeaseCancelsDiff(sb *strings.Builder, a, b []internal.LeaseCancel) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !a[i].Equal(b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", a[i].String()))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", a[i].String()))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", b[i].String()))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tLease Cancels:\n")
		sb.WriteString(lsb.String())
	}
}

func minmax(a, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}
