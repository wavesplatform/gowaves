package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"github.com/wavesplatform/gowaves/pkg/util/limit_listener"
)

const (
	defaultTimeout              = 30 * time.Second
	postMessageSizeLimit  int64 = 1 << 20 // 1 MB
	maxDebugMessageLength       = 100
)

type NodeApi struct {
	state state.State
	node  *node.Node
	app   *App
}

func NewNodeApi(app *App, state state.State, node *node.Node) *NodeApi {
	return &NodeApi{
		state: state,
		node:  node,
		app:   app,
	}
}

func (a *NodeApi) TransactionsBroadcast(w http.ResponseWriter, r *http.Request) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, postMessageSizeLimit))
	if err != nil {
		return errors.Wrap(err, "TransactionsBroadcast: failed to read request body")
	}
	tx, err := a.app.TransactionsBroadcast(r.Context(), b)
	if err != nil {
		return errors.Wrap(err, "TransactionsBroadcast")
	}
	err = trySendJson(w, tx)
	if err != nil {
		return errors.Wrap(err, "TransactionsBroadcast")
	}
	return nil
}

func transactionIDAtInvalidLenErr(key string) *apiErrs.InvalidTransactionIdError {
	return apiErrs.NewInvalidTransactionIDError(
		fmt.Sprintf("%s has invalid length %d. Length can either be %d or %d",
			key, // nickeskov: this part must be the last part of HTTP path
			len(key),
			crypto.DigestSize,
			crypto.SignatureSize,
		),
	)
}

func transactionIDAtInvalidCharErr(invalidChar rune, id string) *apiErrs.InvalidTransactionIdError {
	return apiErrs.NewInvalidTransactionIDError(
		fmt.Sprintf(
			"requirement failed: Wrong char %q in Base58 string '%s'",
			invalidChar,
			id,
		),
	)
}

func (a *NodeApi) TransactionInfo(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "id")

	id, err := crypto.NewDigestFromBase58(s)
	if err != nil {
		if invalidRune, isInvalid := findFirstInvalidRuneInBase58String(s); isInvalid {
			return transactionIDAtInvalidCharErr(invalidRune, s)
		}
		return transactionIDAtInvalidLenErr(s)
	}
	tx, err := a.state.TransactionByID(id.Bytes())
	if err != nil {
		origErr := errors.Cause(err)
		if state.IsNotFound(origErr) {
			return apiErrs.TransactionDoesNotExist
		}
		return errors.Wrapf(err,
			"TransactionsInfo: expected NotFound in state error, but received other error = %s", s,
		)
	}
	err = trySendJson(w, tx)
	if err != nil {
		return errors.Wrap(err, "TransactionsInfo")
	}
	return nil
}

func (a *NodeApi) BlocksLast(w http.ResponseWriter, _ *http.Request) error {
	apiBlock, err := a.app.BlocksLast()
	if err != nil {
		return errors.Wrap(err, "BlocksLast: failed to get last block")
	}
	err = trySendJson(w, apiBlock)
	if err != nil {
		return errors.Wrap(err, "BlocksLast")
	}
	return nil
}

func (a *NodeApi) BlocksFirst(w http.ResponseWriter, _ *http.Request) error {
	apiBlock, err := a.app.BlocksFirst()
	if err != nil {
		return errors.Wrap(err, "BlocksFirst: failed to get first block")
	}
	err = trySendJson(w, apiBlock)
	if err != nil {
		return errors.Wrap(err, "BlocksFirst: failed to marshal block to JSON and write to ResponseWriter")
	}
	return nil
}

func (a *NodeApi) BlocksHeadersLast(w http.ResponseWriter, _ *http.Request) error {
	lastBlockHeader, err := a.app.BlocksHeadersLast()
	if err != nil {
		return errors.Wrap(err, "BlocksHeadersLast: failed to get last block header")
	}
	err = trySendJson(w, lastBlockHeader)
	if err != nil {
		return errors.Wrap(err, "BlocksHeadersLast: failed to marshal block header to JSON and write to ResponseWriter")
	}
	return nil
}

func (a *NodeApi) BlocksHeadersAt(w http.ResponseWriter, r *http.Request) error {
	heightParam := chi.URLParam(r, "height")
	h, err := strconv.ParseUint(heightParam, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse 'height' url param")
	}
	header, err := a.app.BlocksHeadersAt(h)
	if err != nil {
		if state.IsInvalidInput(err) || state.IsNotFound(err) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err, "BlocksHeadersAt: failed to get block header at height %d", h)
	}
	err = trySendJson(w, header)
	if err != nil {
		return errors.Wrap(err, "BlocksHeadersAt: failed to marshal block header to JSON and write to ResponseWriter")
	}
	return nil
}

func (a *NodeApi) BlockHeadersID(w http.ResponseWriter, r *http.Request) error {
	// nickeskov: in this case id param must be non-zero length
	s := chi.URLParam(r, "id")
	id, err := proto.NewBlockIDFromBase58(s)
	if err != nil {
		if invalidRune, isInvalid := findFirstInvalidRuneInBase58String(s); isInvalid {
			return blockIDAtInvalidCharErr(invalidRune, s)
		}
		return blockIDAtInvalidLenErr(s, err)
	}
	header, err := a.app.BlocksHeadersByID(id)
	if err != nil {
		if state.IsNotFound(err) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err, "BlockHeadersID: failed to get block header by ID=%q", s)
	}
	err = trySendJson(w, header)
	if err != nil {
		return errors.Wrap(err, "BlockHeadersID: failed to marshal block header to JSON and write to ResponseWriter")
	}
	return nil
}

func (a *NodeApi) BlocksHeadersSeqFromTo(w http.ResponseWriter, r *http.Request) error {
	var (
		fromParam = chi.URLParam(r, "from")
		toParam   = chi.URLParam(r, "to")
	)
	from, err := strconv.ParseUint(fromParam, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse 'from' url param")
	}
	to, err := strconv.ParseUint(toParam, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse 'to' url param")
	}
	seq, err := a.app.BlocksHeadersFromTo(from, to)
	if err != nil {
		return errors.Wrapf(err, "BlocksHeadersSeqFromTo: failed to get block sequence from %d to %d", from, to)
	}
	err = trySendJson(w, seq)
	if err != nil {
		return errors.Wrap(err, "BlocksHeadersSeqFromTo: failed to marshal block header to JSON and write to ResponseWriter")
	}
	return nil
}

func blockIDAtInvalidLenErr(key string, err error) *apiErrs.InvalidBlockIdError {
	length := len(key)
	var incorrectLenErr crypto.IncorrectLengthError
	if err != nil && errors.As(err, &incorrectLenErr) {
		length = incorrectLenErr.Len
	}

	return apiErrs.NewInvalidBlockIDError(
		fmt.Sprintf("%s has invalid length %d. Length can either be %d or %d",
			key, // nickeskov: this part must be the last part of HTTP path
			length,
			crypto.DigestSize,
			crypto.SignatureSize,
		),
	)
}

func blockIDAtInvalidCharErr(invalidChar rune, id string) *apiErrs.InvalidBlockIdError {
	return apiErrs.NewInvalidBlockIDError(
		fmt.Sprintf(
			"requirement failed: Wrong char %q in Base58 string '%s'",
			invalidChar,
			id,
		),
	)
}

func (a *NodeApi) BlockAt(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// nickeskov: message taken from scala node
		// 	try execute `curl -X GET "https://nodes-testnet.wavesnodes.com/blocks/at/fdsfasdff" -H  "accept: application/json"`
		return blockIDAtInvalidLenErr("at", err)
	}

	block, err := a.app.BlockByHeight(height)
	if err != nil {
		if errors.Is(err, notFound) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrap(err, "BlockAt: expected NotFound in state error, but received other error")
	}

	apiBlock, err := newAPIBlock(block, a.app.services.Scheme, height)
	if err != nil {
		return errors.Wrap(err, "failed to create API block")
	}
	err = trySendJson(w, apiBlock)
	if err != nil {
		return errors.Wrap(err, "BlockEncodeJson: failed to marshal block to JSON and write to ResponseWriter")
	}
	return nil
}

func findFirstInvalidRuneInBase58String(str string) (rune, bool) {
	for _, r := range str {
		if _, ok := base58Alphabet[r]; !ok {
			return r, true
		}
	}
	return 0, false
}

func (a *NodeApi) BlockIDAt(w http.ResponseWriter, r *http.Request) error {
	// nickeskov: in this case id param must be non-zero length
	s := chi.URLParam(r, "id")
	id, err := proto.NewBlockIDFromBase58(s)
	if err != nil {
		if invalidRune, isInvalid := findFirstInvalidRuneInBase58String(s); isInvalid {
			return blockIDAtInvalidCharErr(invalidRune, s)
		}
		return blockIDAtInvalidLenErr(s, err)
	}
	block, err := a.app.Block(id)
	if err != nil {
		if errors.Is(err, notFound) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err,
			"BlockIDAt: expected NotFound in state error, but received other error for blockID=%s", s,
		)
	}

	height, err := a.state.BlockIDToHeight(id)
	if err != nil {
		// TODO(nickeskov): should handle state.IsNotFound(...)?
		return errors.Wrapf(err,
			"BlockIDAt: failed to execute state.BlockIDToHeight for blockID=%s", s)
	}
	apiBlock, err := newAPIBlock(block, a.app.services.Scheme, height)
	if err != nil {
		return errors.Wrap(err, "failed to create API block")
	}
	err = trySendJson(w, apiBlock)
	if err != nil {
		return errors.Wrap(err, "BlockIDAt: failed to marshal block to JSON and write to ResponseWriter")
	}
	return nil
}

func (a *NodeApi) BlockHeight(w http.ResponseWriter, _ *http.Request) error {
	type blockHeightResponse struct {
		Height uint64 `json:"height"`
	}

	height, err := a.state.Height()
	if err != nil {
		return errors.Wrap(err, "BlockHeight: failed to bet blocks height")
	}

	if err := trySendJson(w, blockHeightResponse{Height: height}); err != nil {
		return errors.Wrap(err, "BlockHeight")
	}
	return nil
}

func (a *NodeApi) BlockHeightByID(w http.ResponseWriter, r *http.Request) error {
	type blockHeightByIDResponse struct {
		Height uint64 `json:"height"`
	}

	s := chi.URLParam(r, "id")
	id, err := proto.NewBlockIDFromBase58(s)
	if err != nil {
		if invalidRune, isInvalid := findFirstInvalidRuneInBase58String(s); isInvalid {
			return blockIDAtInvalidCharErr(invalidRune, s)
		}
		return blockIDAtInvalidLenErr(s, err)
	}

	height, err := a.app.BlockIDToHeight(id)
	if err != nil {
		if errors.Is(err, notFound) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err,
			"BlockHeightByID: expected NotFound in state error, but received other error for blockID=%s", s,
		)
	}

	if err := trySendJson(w, blockHeightByIDResponse{Height: height}); err != nil {
		return errors.Wrap(err, "BlockHeightByID")
	}
	return nil
}

// nickeskov: in scala node this route does not exist

func (a *NodeApi) BlockScoreAt(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// TODO(nickeskov): which error it should send?
		return &BadRequestError{err}
	}
	rs, err := a.app.BlocksScoreAt(id)
	if err != nil {
		// TODO(nickeskov): which error it should send?
		return errors.Wrapf(err, "failed get blocks score at for id %d", id)
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "BlockScoreAt")
	}
	return nil
}

func Run(ctx context.Context, address string, n *NodeApi, opts *RunOptions) error {
	if opts == nil {
		opts = DefaultRunOptions()
	}

	routes, err := n.routes(opts)
	if err != nil {
		return errors.Wrap(err, "RunWithOpts")
	}

	apiServer := &http.Server{Addr: address, Handler: routes, ReadHeaderTimeout: defaultTimeout, ReadTimeout: defaultTimeout}
	go func() {
		<-ctx.Done()
		zap.S().Info("Shutting down API...")
		err := apiServer.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Errorf("Failed to shutdown API server: %v", err)
		}
	}()

	if opts.MaxConnections > 0 {
		if address == "" {
			address = ":http"
		}

		ln, lErr := net.Listen("tcp", address)
		if lErr != nil {
			return lErr
		}

		ln = limit_listener.LimitListener(ln, opts.MaxConnections)
		zap.S().Debugf("Set limit for number of simultaneous connections for REST API to %d", opts.MaxConnections)

		err = apiServer.Serve(ln)
	} else {
		err = apiServer.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *NodeApi) PeersAll(w http.ResponseWriter, _ *http.Request) error {
	rs, err := a.app.PeersAll()
	if err != nil {
		return errors.Wrap(err, "failed to fetch all peers")
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersAll")
	}
	return nil
}

func (a *NodeApi) PeersKnown(w http.ResponseWriter, _ *http.Request) error {
	rs, err := a.app.PeersKnown()
	if err != nil {
		return errors.Wrap(err, "failed to fetch known peers")
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersKnown")
	}
	return nil
}

func (a *NodeApi) PeersSpawned(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.PeersSpawned()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersSpawned")
	}
	return nil
}

type PeersConnectRequest struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func (a *NodeApi) PeersConnect(w http.ResponseWriter, r *http.Request) error {
	req := &PeersConnectRequest{}
	if err := tryParseJson(r.Body, req); err != nil {
		return errors.Wrap(err, "failed to parse PeersConnect request body as JSON")
	}
	// TODO(nickeskov): remove this and use auth middleware
	apiKey := r.Header.Get("X-API-Key")
	addr := net.JoinHostPort(req.Host, strconv.FormatUint(uint64(req.Port), 10))
	rs, err := a.app.PeersConnect(r.Context(), apiKey, addr)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to new peer, addr %s", addr)
	}

	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersConnect")
	}
	return nil
}

func (a *NodeApi) AddrByAlias(w http.ResponseWriter, r *http.Request) error {
	type addrResponse struct {
		Address string `json:"address"`
	}

	aliasShort := chi.URLParam(r, "alias")

	alias := proto.NewAlias(a.app.scheme(), aliasShort)
	if _, err := alias.Valid(a.app.scheme()); err != nil {
		msg := err.Error()
		return apiErrs.NewCustomValidationError(msg)
	}

	addr, err := a.app.AddrByAlias(*alias)
	if err != nil {
		origErr := errors.Cause(err)
		if state.IsNotFound(origErr) {
			return apiErrs.NewAliasDoesNotExistError(alias.String())
		}
		return errors.Wrapf(err, "failed to find addr by short alias %q", aliasShort)
	}

	resp := addrResponse{Address: addr.String()}
	if err := trySendJson(w, resp); err != nil {
		return errors.Wrap(err, "AddrByAlias")
	}
	return nil
}

func (a *NodeApi) AliasesByAddr(w http.ResponseWriter, r *http.Request) error {
	addrBase58 := chi.URLParam(r, "address")

	addr, err := proto.NewAddressFromString(addrBase58)
	if err != nil {
		return &apiErrs.InvalidAddressError{}
	}

	aliases, err := a.app.AliasesByAddr(addr)
	if err != nil {
		if state.IsNotFound(err) {
			aliases = nil
		} else {
			return errors.Wrapf(err, "failed to find aliases by addr")
		}
	}

	if aliases == nil {
		aliases = []proto.Alias{} // ensure that empty array will be return instead of nil
	}
	if err := trySendJson(w, aliases); err != nil {
		return errors.Wrap(err, "AliasesByAddr")
	}
	return nil
}

func (a *NodeApi) NodeStatus(w http.ResponseWriter, r *http.Request) error {
	type resp struct {
		BlockchainHeight uint64 `json:"blockchainHeight"`
		StateHeight      uint64 `json:"stateHeight"`
		UpdatedTimestamp int64  `json:"updatedTimestamp"`
		UpdatedDate      string `json:"updatedDate"`
	}

	stateHeight, err := a.app.state.Height()
	if err != nil {
		return errors.Wrap(err, "failed to get state height in NodeStatus HTTP endpoint")
	}

	blockHeader := a.state.TopBlock()
	updatedTimestampMillis := int64(blockHeader.Timestamp)

	// TODO: meaning of 'UpdatedDate' in scala node  differs from ours
	out := resp{
		BlockchainHeight: stateHeight,
		StateHeight:      stateHeight,
		UpdatedTimestamp: updatedTimestampMillis,
		UpdatedDate:      time.UnixMilli(updatedTimestampMillis).UTC().Format(time.RFC3339Nano),
	}
	if err := trySendJson(w, out); err != nil {
		return errors.Wrap(err, "NodeStatus")
	}
	return nil
}

func (a *NodeApi) walletSeed(w http.ResponseWriter, _ *http.Request) error {
	type seed struct {
		Seed string `json:"seed"`
	}

	seeds58 := a.app.WalletSeeds()
	seeds := make([]seed, 0, len(seeds58))
	for _, seed58 := range seeds58 {
		seeds = append(seeds, seed{Seed: seed58})
	}

	if err := trySendJson(w, seeds); err != nil {
		return errors.Wrap(err, "walletSeed")
	}
	return nil
}

func (a *NodeApi) PeersConnected(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.PeersConnected()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersConnected")
	}
	return nil
}

func (a *NodeApi) PeersSuspended(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.PeersSuspended()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersSuspended")
	}
	return nil
}

func (a *NodeApi) PeersBlackListed(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.PeersBlackListed()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersBlackListed")
	}
	return nil
}

func (a *NodeApi) PeersClearBlackList(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.PeersClearBlackList()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersBlackListed")
	}
	return nil
}

func (a *NodeApi) BlocksGenerators(w http.ResponseWriter, _ *http.Request) error {
	rs, err := a.app.BlocksGenerators()
	if err != nil {
		return errors.Wrap(err, "failed to get BlocksGenerators")
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "BlocksGenerators")
	}
	return nil
}

func (a *NodeApi) poolTransactions(w http.ResponseWriter, _ *http.Request) error {
	type poolTransactions struct {
		Count int `json:"count"`
	}

	rs := poolTransactions{
		Count: a.app.PoolTransactions(),
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "poolTransactions")
	}
	return nil
}

func (a *NodeApi) unconfirmedSize(w http.ResponseWriter, _ *http.Request) error {
	type unconfirmedSize struct {
		Size int `json:"size"`
	}

	rs := unconfirmedSize{
		Size: a.app.PoolTransactions(),
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "unconfirmedSize")
	}
	return nil
}

type rollbackResponse struct {
	BlockID proto.BlockID `json:"blockId"`
}

func (a *NodeApi) RollbackToHeight(w http.ResponseWriter, r *http.Request) error {
	type rollbackRequest struct {
		Height                  uint64 `json:"rollbackTo"`
		ReturnTransactionsToUtx bool   `json:"returnTransactionsToUtx"`
	}

	rollbackReq := &rollbackRequest{}
	if err := tryParseJson(r.Body, rollbackReq); err != nil {
		return errors.Wrap(err, "failed to parse RollbackToHeight body as JSON")
	}
	err := a.state.RollbackToHeight(rollbackReq.Height)
	if err != nil {
		origErr := errors.Cause(err)
		if state.IsNotFound(origErr) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err, "failed to rollback to height %d", rollbackReq.Height)
	}
	block, err := a.app.BlockByHeight(rollbackReq.Height)
	if err != nil {
		if errors.Is(err, notFound) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrap(err, "expected NotFound in state error, but received other error")
	}
	if err = trySendJson(w, rollbackResponse{block.BlockID()}); err != nil {
		return errors.Wrap(err, "RollbackToHeight")
	}
	return nil
}

func (a *NodeApi) RollbackTo(w http.ResponseWriter, r *http.Request) error {
	type rollbackResponse struct {
		BlockID proto.BlockID `json:"blockId"`
	}
	idBase58 := chi.URLParam(r, "id")
	id, err := proto.NewBlockIDFromBase58(idBase58)
	if err != nil {
		return err
	}
	if err = a.state.RollbackTo(id); err != nil {
		return errors.Wrapf(err, "failed to rollback to block %s", id)
	}
	if err = trySendJson(w, rollbackResponse{id}); err != nil {
		return errors.Wrap(err, "RollbackTo")
	}
	return nil
}

type walletLoadKeysRequest struct {
	Password string `json:"password"`
}

type walletLoadKeys interface {
	LoadKeys(apiKey string, password []byte) error
}

func WalletLoadKeys(app walletLoadKeys) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		js := &walletLoadKeysRequest{}
		if err := tryParseJson(r.Body, js); err != nil {
			return errors.Wrap(err, "failed to parse WalletLoadKeys body as JSON")
		}
		// TODO(nickeskov): remove this and use auth middleware
		apiKey := r.Header.Get("X-API-Key")
		if err := app.LoadKeys(apiKey, []byte(js.Password)); err != nil {
			return errors.Wrap(err, "failed to execute LoadKeys")
		}
		return nil
	}
}

func (a *NodeApi) WalletAccounts(w http.ResponseWriter, _ *http.Request) error {
	rs, err := a.app.Accounts()
	if err != nil {
		return errors.Wrap(err, "failed to get Accounts")
	}
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "WalletAccounts")
	}
	return nil
}

func (a *NodeApi) GoMinerInfo(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.Miner()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "GoMinerInfo")
	}
	return nil
}

func (a *NodeApi) Addresses(w http.ResponseWriter, _ *http.Request) error {
	addresses, err := a.app.Addresses()
	if err != nil {
		return errors.Wrap(err, "failed to get Addresses")
	}
	if err := trySendJson(w, addresses); err != nil {
		return errors.Wrap(err, "Addresses")
	}
	return nil
}

func (a *NodeApi) nodeProcesses(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.NodeProcesses()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "nodeProcesses")
	}
	return nil
}

func (a *NodeApi) stateHashDebug(height proto.Height) (*proto.StateHashDebug, error) {
	stateHash, err := a.state.LegacyStateHashAtHeight(height)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get state hash at height %d", height)
	}
	stateHashDebug := proto.NewStateHashJSDebug(*stateHash, height, a.app.version().Version)
	return &stateHashDebug, nil
}

func (a *NodeApi) stateHash(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// TODO(nickeskov): which error it should send?
		return &BadRequestError{err}
	}
	stateHashDebug, err := a.stateHashDebug(height)
	if err != nil {
		return errors.Wrap(err, "failed to get state hash debug")
	}

	if err := trySendJson(w, stateHashDebug); err != nil {
		return errors.Wrap(err, "stateHash")
	}
	return nil
}

func (a *NodeApi) stateHashLast(w http.ResponseWriter, _ *http.Request) error {
	height, err := a.state.Height()
	if err != nil {
		return errors.Wrap(err, "failed to get last height")
	}
	stateHashDebug, err := a.stateHashDebug(height - 1)
	if err != nil {
		return errors.Wrap(err, "failed to get last state hash")
	}
	if err := trySendJson(w, stateHashDebug); err != nil {
		return errors.Wrap(err, "stateHash")
	}
	return nil
}

func (a *NodeApi) snapshotStateHash(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// TODO(nickeskov): which error it should send?
		return &BadRequestError{err}
	}
	sh, err := a.state.SnapshotStateHashAtHeight(height)
	if err != nil {
		if state.IsNotFound(err) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err, "failed to get snapshot state hash at height %d", height)
	}
	type out struct {
		StateHash proto.HexBytes `json:"stateHash"`
	}
	if sendErr := trySendJson(w, out{StateHash: sh.Bytes()}); sendErr != nil {
		return errors.Wrap(sendErr, "snapshotStateHash")
	}
	return nil
}

func wavesAddressInvalidCharErr(invalidChar rune, id string) *apiErrs.CustomValidationError {
	return apiErrs.NewCustomValidationError(
		fmt.Sprintf(
			"requirement failed: Wrong char %q in Base58 string '%s'",
			invalidChar,
			id,
		),
	)
}

func (a *NodeApi) EthereumDAppABI(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "address")
	addr, err := proto.NewAddressFromString(s)
	if err != nil {
		if invalidRune, isInvalid := findFirstInvalidRuneInBase58String(s); isInvalid {
			return wavesAddressInvalidCharErr(invalidRune, s)
		}
		return apiErrs.InvalidAddress
	}
	methods, err := a.app.EthereumDAppMethods(addr)
	if err != nil {
		if errors.Is(err, notFound) {
			return nil // empty output if script is not found (according to the scala node)
		}
		return errors.Wrapf(err, "failed to get EthereumDAppMethods by address=%q", addr.String())
	}
	if err := trySendJson(w, methods); err != nil {
		return errors.Wrap(err, "EthereumDAppABI")
	}
	return nil
}

func (a *NodeApi) AssetsDetailsByID(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "id")
	fullAssetID, err := crypto.NewDigestFromBase58(s)
	if err != nil {
		return apiErrs.InvalidAssetId
	}

	var full bool
	if f := r.URL.Query().Get("full"); f != "" {
		if full, err = strconv.ParseBool(f); err != nil {
			return apiErrs.InvalidAssetId
		}
	}

	assetDetails, err := a.app.AssetsDetailsByID(fullAssetID, full)
	if err != nil {
		if errors.Is(err, errs.UnknownAsset{}) {
			return apiErrs.NewAssetDoesNotExistError(fullAssetID)
		}
		return errors.Wrapf(err, "failed to get asset details by assetID=%q", fullAssetID)
	}
	if err := trySendJson(w, assetDetails); err != nil {
		return errors.Wrap(err, "AssetsDetailsByID")
	}
	return nil
}

func (a *NodeApi) AssetsDetailsByIDsGet(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	return a.assetsDetailsByIDs(w, query.Get("full"), query["id"])

}

func (a *NodeApi) AssetsDetailsByIDsPost(w http.ResponseWriter, r *http.Request) error {
	var data struct {
		IDs []string `json:"ids"`
	}
	if err := tryParseJson(r.Body, &data); err != nil {
		return err
	}
	query := r.URL.Query()
	return a.assetsDetailsByIDs(w, query.Get("full"), data.IDs)
}

func (a *NodeApi) assetsDetailsByIDs(w http.ResponseWriter, fullQueryParam string, ids []string) (err error) {
	var full bool
	if fullQueryParam != "" {
		full, err = strconv.ParseBool(fullQueryParam)
		if err != nil {
			return apiErrs.InvalidAssetId
		}
	}
	if len(ids) == 0 {
		return apiErrs.AssetIdNotSpecified
	}
	if limit := a.app.settings.AssetDetailsLimit; len(ids) > limit {
		return apiErrs.NewTooBigArrayAllocationError(limit)
	}
	var (
		fullAssetsIDs = make([]crypto.Digest, 0, len(ids))
		invalidIDs    []string
	)
	for _, id := range ids {
		d, err := crypto.NewDigestFromBase58(id)
		if err != nil {
			invalidIDs = append(invalidIDs, id)
		} else {
			fullAssetsIDs = append(fullAssetsIDs, d)
		}
	}
	if len(invalidIDs) != 0 {
		return apiErrs.NewInvalidIDsError(invalidIDs)
	}

	assetsDetails, err := a.app.AssetsDetails(fullAssetsIDs, full)
	if err != nil {
		return errors.Wrapf(err, "failed to get asset details by list of assets")
	}
	if err := trySendJson(w, assetsDetails); err != nil {
		return errors.Wrap(err, "AssetsDetails")
	}
	return nil
}

func (a *NodeApi) version(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.version()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "Version")
	}
	return nil
}

// tryParseJson receives reader and out params. out MUST be a pointer
func tryParseJson(r io.Reader, out interface{}) error {
	// TODO(nickeskov): check empty reader
	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal %T as JSON into %T", r, out)
	}
	return nil
}

func trySendJson(w io.Writer, v interface{}) error {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %T to JSON and write it to %T", v, w)
	}
	return nil
}

func (a *NodeApi) debugPrint(_ http.ResponseWriter, r *http.Request) error {
	type debugPrintRequest struct {
		Message string `json:"message"`
	}

	req := &debugPrintRequest{}
	if err := tryParseJson(r.Body, req); err != nil {
		return errors.Wrap(err, "failed to parse DebugPrint request body as JSON")
	}
	trimmedStr := req.Message
	if len(req.Message) > maxDebugMessageLength {
		trimmedStr = req.Message[:maxDebugMessageLength]
	}
	safeStr := strings.NewReplacer("\n", "", "\r", "").Replace(trimmedStr)
	zap.S().Debug(safeStr)
	return nil
}
