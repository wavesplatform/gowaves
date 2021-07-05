package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/go-chi/chi"
	"github.com/mr-tron/base58"
	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
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

func (a *NodeApi) TransactionsBroadcast(_ http.ResponseWriter, r *http.Request) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "TransactionsBroadcast: failed to read request body")
	}
	err = a.app.TransactionsBroadcast(r.Context(), b)
	if err != nil {
		return errors.Wrap(err, "TransactionsBroadcast")
	}
	return nil
}

func (a *NodeApi) BlocksLast(w http.ResponseWriter, _ *http.Request) error {
	block, err := a.app.BlocksLast()
	if err != nil {
		return errors.Wrap(err, "BlocksLast: failed to get last block")
	}

	bts, err := proto.BlockEncodeJson(block)
	if err != nil {
		return errors.Wrap(err, "BlocksLast: failed to marshal block to JSON")
	}
	if _, err = w.Write(bts); err != nil {
		return errors.Wrap(err, "BlocksLast: failed to write block json to ResponseWriter")
	}
	return nil
}

func (a *NodeApi) BlocksFirst(w http.ResponseWriter, _ *http.Request) error {
	block, err := a.state.BlockByHeight(1)
	if err != nil {
		return errors.Wrap(err, "BlocksFirst")
	}
	block.Height = 1
	bts, err := proto.BlockEncodeJson(block)
	if err != nil {
		return errors.Wrap(err, "BlocksFirst: failed to marshal block to JSON")
	}
	if _, err = w.Write(bts); err != nil {
		return errors.Wrap(err, "BlocksFirst: failed to write block json to ResponseWriter")
	}
	return nil
}

func blockIDAtInvalidLenErr(key string) *apiErrs.InvalidBlockIdError {
	return apiErrs.NewInvalidBlockIDError(
		fmt.Sprintf("%s has invalid length %d. Length can either be %d or %d",
			key, // nickeskov: this part must be the last part of HTTP path
			len(key),
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
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// nickeskov: message taken from scala node
		// 	try execute `curl -X GET "https://nodes-testnet.wavesnodes.com/blocks/at/fdsfasdff" -H  "accept: application/json"`
		return blockIDAtInvalidLenErr("at")
	}

	block, err := a.state.BlockByHeight(id)
	if err != nil {
		origErr := errors.Cause(err)
		if state.IsNotFound(origErr) {
			// nickeskov: it's strange, but scala node sends empty response...
			// 	try execute `curl -X GET "https://nodes-testnet.wavesnodes.com/blocks/at/0" -H  "accept: application/json"`
			return nil
		}
		return errors.Wrap(err,
			"BlockAt: expected NotFound in state error, but received other error")
	}

	block.Height = id
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		return errors.Wrap(err,
			"BlockEncodeJson: failed to marshal block to JSON and write to ResponseWriter")
	}
	return nil
}

func findFirstInvalidRuneInBase58String(str string) *rune {
	for _, r := range str {
		if _, err := base58.Decode(string(r)); err != nil {
			return &r
		}
	}
	return nil
}

func (a *NodeApi) BlockIDAt(w http.ResponseWriter, r *http.Request) error {
	// nickeskov: in this case id param must be non zero length
	s := chi.URLParam(r, "id")
	id, err := proto.NewBlockIDFromBase58(s)
	if err != nil {
		if invalidRune := findFirstInvalidRuneInBase58String(s); invalidRune != nil {
			return blockIDAtInvalidCharErr(*invalidRune, s)
		}
		return blockIDAtInvalidLenErr(s)
	}
	block, err := a.state.Block(id)
	if err != nil {
		origErr := errors.Cause(err)
		if state.IsNotFound(origErr) {
			return apiErrs.BlockDoesNotExist
		}
		return errors.Wrapf(err,
			"BlockIDAt: expected NotFound in state error, but received other error for blockID=%s",
			s,
		)
	}

	height, err := a.state.BlockIDToHeight(id)
	if err != nil {
		// TODO(nickeskov): should handle state.IsNotFound(...)?
		return errors.Wrapf(err,
			"BlockIDAt: failed to execute state.BlockIDToHeight for blockID=%s", s)
	}
	block.Height = height
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		return errors.Wrap(err,
			"BlockIDAt: failed to marshal block to JSON and write to ResponseWriter")
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

func RunWithOpts(ctx context.Context, address string, n *NodeApi, opts *RunOptions) error {
	if opts == nil {
		opts = DefaultRunOptions()
	}

	routes, err := n.routes(opts)
	if err != nil {
		return errors.Wrap(err, "RunWithOpts")
	}

	apiServer := &http.Server{Addr: address, Handler: routes}
	go func() {
		<-ctx.Done()
		zap.S().Info("Shutting down API...")
		err := apiServer.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			zap.S().Errorf("Failed to shutdown API server: %v", err)
		}
	}()

	err = apiServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func Run(ctx context.Context, address string, n *NodeApi) error {
	// TODO(nickeskov): add run flags in CLI flags
	return RunWithOpts(ctx, address, n, nil)
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
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	rs, err := a.app.PeersConnect(r.Context(), apiKey, addr)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to new peer, addr %s", addr)
	}

	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "PeersConnect")
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

type rollbackRequest struct {
	Height uint64 `json:"height"`
}

type rollbackToHeight interface {
	RollbackToHeight(string, proto.Height) error
}

func RollbackToHeight(app rollbackToHeight) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		rollbackReq := &rollbackRequest{}
		if err := tryParseJson(r.Body, rollbackReq); err != nil {
			return errors.Wrap(err, "failed to parse RollbackToHeight body as JSON")
		}
		// TODO(nickeskov): remove this and use auth middleware
		apiKey := r.Header.Get("X-API-Key")
		if err := app.RollbackToHeight(apiKey, rollbackReq.Height); err != nil {
			return errors.Wrapf(err, "failed to rollback to height %d", rollbackReq.Height)
		}
		// TODO(nickeskov): looks like bug...
		if err := trySendJson(w, nil); err != nil {
			return errors.Wrap(err, "RollbackToHeight")
		}
		return nil
	}
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

	// TODO(nickeskov): create new method in state (like TopBlock, but for TopBlockHeader)
	blockHeader, err := a.state.HeaderByHeight(stateHeight)
	if err != nil {
		return errors.Wrapf(err, "failed to get block header from state by height %d", stateHeight)
	}
	updatedTimestampMillis := int64(blockHeader.Timestamp)

	out := resp{
		BlockchainHeight: stateHeight,
		StateHeight:      stateHeight,
		UpdatedTimestamp: updatedTimestampMillis,
		UpdatedDate:      fromUnixMillis(updatedTimestampMillis).Format(time.RFC3339Nano),
	}
	if err := trySendJson(w, out); err != nil {
		return errors.Wrap(err, "NodeStatus")
	}
	return nil
}

func (a *NodeApi) BuildVersion(w http.ResponseWriter, _ *http.Request) error {
	type ver struct {
		Version string `json:"version"`
	}

	buildVersion := a.app.Config().BuildVersion

	out := ver{Version: fmt.Sprintf("GoWaves %s", buildVersion)}
	if err := trySendJson(w, out); err != nil {
		return errors.Wrap(err, "BuildVersion")
	}
	return nil
}

func (a *NodeApi) AddrByAlias(w http.ResponseWriter, r *http.Request) error {
	type addrResponse struct {
		Address string `json:"address"`
	}

	// nickeskov: alias as plain text without an 'alias' prefix and chain ID (scheme)
	aliasShort := chi.URLParam(r, "alias")

	chainID := proto.SchemeFromString(a.app.Config().BlockchainType)

	alias := proto.NewAlias(chainID, aliasShort)
	if _, err := alias.Valid(); err != nil {
		// TODO(nickeskov): check that error msg looks like in scala
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

func (a *NodeApi) nodeProcesses(w http.ResponseWriter, _ *http.Request) error {
	rs := a.app.NodeProcesses()
	if err := trySendJson(w, rs); err != nil {
		return errors.Wrap(err, "nodeProcesses")
	}
	return nil
}

func (a *NodeApi) stateHash(w http.ResponseWriter, r *http.Request) error {
	s := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// TODO(nickeskov): which error it should send?
		return &BadRequestError{err}
	}

	stateHash, err := a.state.StateHashAtHeight(height)
	if err != nil {
		return errors.Wrapf(err, "failed to get state hash at height %d", height)
	}
	if err := trySendJson(w, stateHash); err != nil {
		return errors.Wrap(err, "stateHash")
	}
	return nil
}

func (a *NodeApi) sendSelfInterrupt(w http.ResponseWriter, _ *http.Request) error {
	type resp struct {
		Stopped bool `json:"stopped"`
	}

	selfPid := os.Getpid()
	p, err := os.FindProcess(selfPid)
	if err != nil {
		return errors.Wrapf(err, "failed to find process (self) with pid %d", selfPid)
	}
	interrupt := os.Interrupt
	if err := p.Signal(interrupt); err != nil {
		return errors.Wrapf(err,
			"failed to send signal %q to self process with pid %d", interrupt, selfPid)
	}
	if err := trySendJson(w, resp{Stopped: true}); err != nil {
		return errors.Wrap(err, "sendSelfInterrupt")
	}
	zap.S().Infof("Sent by node HTTP API to self process %q signal", interrupt)
	return nil
}

func (a *NodeApi) walletSeed(w http.ResponseWriter, _ *http.Request) error {
	type seed struct {
		Seed string `json:"seed"`
	}

	// TODO(nickeskov): This works not like in scala node.
	// 	Scala node don't have multiple wallets, it have only one wallet.

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
