package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

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
	// TODO: use io.LimitReader
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
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// nickeskov: message taken from scala node
		// 	try execute `curl -X GET "https://nodes-testnet.wavesnodes.com/blocks/at/fdsfasdff" -H  "accept: application/json"`
		return blockIDAtInvalidLenErr("at")
	}

	block, err := a.state.BlockByHeight(height)
	if err != nil {
		origErr := errors.Cause(err)
		if state.IsNotFound(origErr) || state.IsInvalidInput(origErr) {
			// nickeskov: it's strange, but scala node sends empty response...
			// 	try execute `curl -X GET "https://nodes-testnet.wavesnodes.com/blocks/at/0" -H  "accept: application/json"`
			return nil
		}
		return errors.Wrap(err, "BlockAt: expected NotFound in state error, but received other error")
	}

	block.Height = height
	err = json.NewEncoder(w).Encode(block)
	if err != nil {
		return errors.Wrap(err, "BlockEncodeJson: failed to marshal block to JSON and write to ResponseWriter")
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

// tryParseJson receives reader and out params. out MUST be a pointer
func tryParseJson(r io.Reader, out interface{}) error {
	// TODO(nickeskov): check empty reader
	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		return errors.Wrapf(err, "Failed to unmarshal %T as JSON into %T", r, out)
	}
	return nil
}

func trySendJson(w io.Writer, v interface{}) error {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshal %T to JSON and write it to %T", v, w)
	}
	return nil
}
