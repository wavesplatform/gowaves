package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
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

	txs, err := cnv.BlockTransactions(b.Block)
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
		addDataDiff(sb, a.GetData(), b.GetData())
	}
	if len(a.GetTransfers()) != 0 || len(b.GetTransfers()) != 0 {
		addTransfersDiff(sb, a.GetTransfers(), b.GetTransfers(), scheme)
	}
	if len(a.GetIssues()) != 0 || len(b.GetIssues()) != 0 {
		addIssuesDiff(sb, a.GetIssues(), b.GetIssues())
	}
	if len(a.GetReissues()) != 0 || len(b.GetReissues()) != 0 {
		addReissuesDiff(sb, a.GetReissues(), b.GetReissues())
	}
	if len(a.GetBurns()) != 0 || len(b.GetBurns()) != 0 {
		addBurnsDiff(sb, a.GetBurns(), b.GetBurns())
	}
	if len(a.GetSponsorFees()) != 0 || len(b.GetSponsorFees()) != 0 {
		addSponsorFeesDiff(sb, a.GetSponsorFees(), b.GetSponsorFees())
	}
	if len(a.GetLeases()) != 0 || len(b.GetLeases()) != 0 {
		addLeasesDiff(sb, a.GetLeases(), b.GetLeases(), scheme)
	}
	if len(a.GetLeaseCancels()) != 0 || len(b.GetLeaseCancels()) != 0 {
		addLeaseCancelsDiff(sb, a.GetLeaseCancels(), b.GetLeaseCancels())
	}
	if a.GetErrorMessage().GetText() != b.GetErrorMessage().GetText() {
		sb.WriteString("\tError:\n")
		sb.WriteString(fmt.Sprintf("\t-%s\n\t+%s\n",
			a.GetErrorMessage().GetText(), b.GetErrorMessage().GetText()))
	}
	return sb.String()
}

func minmax(a, b int) (int, int) {
	if a < b {
		return a, b
	}
	return b, a
}
func addBurnsDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_Burn) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if a[i].GetAmount() != b[i].GetAmount() || !bytes.Equal(a[i].GetAssetId(), b[i].GetAssetId()) {
			lsb.WriteString(fmt.Sprintf("\t-AssetID: %s; Amount: %d\n",
				base58.Encode(a[i].GetAssetId()), a[i].Amount))
			lsb.WriteString(fmt.Sprintf("\t+AssetID: %s; Amount: %d\n",
				base58.Encode(b[i].GetAssetId()), b[i].Amount))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+AssetID: %s; Amount: %d\n",
				base58.Encode(a[i].GetAssetId()), a[i].Amount))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+AssetID: %s; Amount: %d\n",
				base58.Encode(b[i].GetAssetId()), b[i].Amount))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tBurns:\n")
		sb.WriteString(lsb.String())
	}
}

func equalDataEntries(a, b *waves.DataTransactionData_DataEntry) bool {
	return a.GetKey() == b.GetKey() && extractValue(a) == extractValue(b)
}

func extractValue(e *waves.DataTransactionData_DataEntry) string {
	switch v := e.GetValue().(type) {
	case *waves.DataTransactionData_DataEntry_BinaryValue:
		return base58.Encode(v.BinaryValue)
	case *waves.DataTransactionData_DataEntry_BoolValue:
		return fmt.Sprintf("%t", v.BoolValue)
	case *waves.DataTransactionData_DataEntry_IntValue:
		return fmt.Sprintf("%d", v.IntValue)
	case *waves.DataTransactionData_DataEntry_StringValue:
		return v.StringValue
	default:
		return fmt.Sprintf("unsupported value type %T", e.GetValue())
	}
}

func addDataDiff(sb *strings.Builder, a, b []*waves.DataTransactionData_DataEntry) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !equalDataEntries(a[i], b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-Key: %s; Value: %s\n", a[i].GetKey(), extractValue(a[i])))
			lsb.WriteString(fmt.Sprintf("\t+Key: %s; Value: %s\n", b[i].GetKey(), extractValue(b[i])))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+Key: %s; Value: %s\n", a[i].GetKey(), extractValue(a[i])))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+Key: %s; Value: %s\n", b[i].GetKey(), extractValue(b[i])))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tData Entries:\n")
		sb.WriteString(lsb.String())
	}
}

func equalIssues(a, b *waves.InvokeScriptResult_Issue) bool {
	return bytes.Equal(a.GetAssetId(), b.GetAssetId()) &&
		a.GetName() == b.GetName() &&
		a.GetDescription() == b.GetDescription() &&
		a.GetAmount() == b.GetAmount() &&
		a.GetDecimals() == b.GetDecimals() &&
		a.GetReissuable() == b.GetReissuable() &&
		bytes.Equal(a.GetScript(), b.GetScript()) &&
		a.GetNonce() == b.GetNonce()
}

func issueString(i *waves.InvokeScriptResult_Issue) string {
	return fmt.Sprintf(
		"AssetID: %s; Name: %s; Description: %s; Amount: %d; Decimals: %d; Reissuable: %t; Script: %s; Nonce: %d\n",
		base58.Encode(i.GetAssetId()), i.GetName(), i.GetDescription(), i.GetAmount(), i.GetDecimals(),
		i.GetReissuable(), base64.StdEncoding.EncodeToString(i.GetScript()), i.GetNonce())
}

func addIssuesDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_Issue) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !equalIssues(a[i], b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", issueString(a[i])))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", issueString(b[i])))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", issueString(a[i])))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", issueString(b[i])))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tIssues:\n")
		sb.WriteString(lsb.String())
	}
}

func addLeaseCancelsDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_LeaseCancel) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !bytes.Equal(a[i].GetLeaseId(), b[i].GetLeaseId()) {
			lsb.WriteString(fmt.Sprintf("\t-LeaseID: %s\n", base58.Encode(a[i].GetLeaseId())))
			lsb.WriteString(fmt.Sprintf("\t+LeaseID: %s\n", base58.Encode(b[i].GetLeaseId())))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+LeaseID: %s\n", base58.Encode(a[i].GetLeaseId())))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+LeaseID: %s\n", base58.Encode(b[i].GetLeaseId())))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tLease Cancels:\n")
		sb.WriteString(lsb.String())
	}
}

func equalRecipient(a, b *waves.Recipient) bool {
	switch ra := a.GetRecipient().(type) {
	case *waves.Recipient_Alias:
		rb, ok := b.GetRecipient().(*waves.Recipient_Alias)
		if !ok {
			return false
		}
		return ra.Alias == rb.Alias
	case *waves.Recipient_PublicKeyHash:
		rb, ok := b.GetRecipient().(*waves.Recipient_PublicKeyHash)
		if !ok {
			return false
		}
		return bytes.Equal(ra.PublicKeyHash, rb.PublicKeyHash)
	default:
		return false
	}
}

func equalLeases(a, b *waves.InvokeScriptResult_Lease) bool {
	return bytes.Equal(a.GetLeaseId(), b.GetLeaseId()) &&
		a.GetAmount() == b.GetAmount() &&
		a.GetNonce() == b.GetNonce() &&
		equalRecipient(a.GetRecipient(), b.GetRecipient())
}

func recipientString(scheme byte, r *waves.Recipient) string {
	switch tr := r.GetRecipient().(type) {
	case *waves.Recipient_Alias:
		return tr.Alias
	case *waves.Recipient_PublicKeyHash:
		a, err := proto.RebuildAddress(scheme, tr.PublicKeyHash)
		if err != nil {
			return fmt.Sprintf("invalid public key hash '%s'", base58.Encode(tr.PublicKeyHash))
		}
		return a.String()
	default:
		return fmt.Sprintf("unsupported recipient type %T", r)
	}
}

func leaseString(scheme byte, l *waves.InvokeScriptResult_Lease) string {
	return fmt.Sprintf("LeaseID: %s; Amount: %d; Nonce: %d; Recipient: %s",
		base58.Encode(l.GetLeaseId()), l.GetAmount(), l.GetNonce(), recipientString(scheme, l.GetRecipient()))
}

func addLeasesDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_Lease, scheme byte) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !equalLeases(a[i], b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", leaseString(scheme, a[i])))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", leaseString(scheme, b[i])))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", leaseString(scheme, a[i])))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", leaseString(scheme, b[i])))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tLeases:\n")
		sb.WriteString(lsb.String())
	}
}

func equalReissues(a, b *waves.InvokeScriptResult_Reissue) bool {
	return a.GetIsReissuable() == b.GetIsReissuable() &&
		a.GetAmount() == b.GetAmount() &&
		bytes.Equal(a.GetAssetId(), b.GetAssetId())
}

func reissueString(r *waves.InvokeScriptResult_Reissue) string {
	return fmt.Sprintf("AssetID: %s; Amount: %d; Reissuable: %t",
		base58.Encode(r.GetAssetId()), r.GetAmount(), r.GetIsReissuable())
}

func addReissuesDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_Reissue) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !equalReissues(a[i], b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", reissueString(a[i])))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", reissueString(b[i])))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", reissueString(a[i])))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", reissueString(b[i])))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tReissues:\n")
		sb.WriteString(lsb.String())
	}
}

func equalSponsorships(a, b *waves.InvokeScriptResult_SponsorFee) bool {
	return bytes.Equal(a.GetMinFee().GetAssetId(), b.GetMinFee().GetAssetId()) &&
		a.GetMinFee().GetAmount() == b.GetMinFee().GetAmount()
}

func sponsorshipString(s *waves.InvokeScriptResult_SponsorFee) string {
	return fmt.Sprintf("AssetID: %s; Amount: %d",
		base58.Encode(s.GetMinFee().GetAssetId()), s.GetMinFee().GetAmount())
}

func addSponsorFeesDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_SponsorFee) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !equalSponsorships(a[i], b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", sponsorshipString(a[i])))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", sponsorshipString(b[i])))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", sponsorshipString(a[i])))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", sponsorshipString(b[i])))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tSponsorships:\n")
		sb.WriteString(lsb.String())
	}
}

func equalPayments(a, b *waves.InvokeScriptResult_Payment) bool {
	return bytes.Equal(a.GetAddress(), b.GetAddress()) &&
		bytes.Equal(a.GetAmount().GetAssetId(), b.GetAmount().GetAssetId()) &&
		a.GetAmount().GetAmount() == b.GetAmount().GetAmount()
}

func paymentString(p *waves.InvokeScriptResult_Payment, scheme byte) string {
	as := ""
	a, err := proto.RebuildAddress(scheme, p.GetAddress())
	if err != nil {
		as = fmt.Sprintf("invalid address '%s'", base58.Encode(p.GetAddress()))
	} else {
		as = a.String()
	}
	return fmt.Sprintf("Address: %s; AsssetID: %s; Amount: %d",
		as, base58.Encode(p.GetAmount().GetAssetId()), p.GetAmount().GetAmount())
}

func addTransfersDiff(sb *strings.Builder, a, b []*waves.InvokeScriptResult_Payment, scheme byte) {
	la := len(a)
	lb := len(b)
	min, max := minmax(la, lb)
	lsb := new(strings.Builder)
	for i := 0; i < min; i++ {
		if !equalPayments(a[i], b[i]) {
			lsb.WriteString(fmt.Sprintf("\t-%s\n", paymentString(a[i], scheme)))
			lsb.WriteString(fmt.Sprintf("\t+%s\n", paymentString(b[i], scheme)))
		}
	}
	for i := min; i < max; i++ {
		if la > lb {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", paymentString(a[i], scheme)))
		} else {
			lsb.WriteString(fmt.Sprintf("\t+%s\n", paymentString(b[i], scheme)))
		}
	}
	if lsb.Len() > 0 {
		sb.WriteString("\tTransfers:\n")
		sb.WriteString(lsb.String())
	}
}
