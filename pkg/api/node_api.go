package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/mr-tron/base58"
	"github.com/pkg/errors"
	apiErrs "github.com/wavesplatform/gowaves/pkg/api/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/node"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/state"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
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
	// TODO(nickeskov): it looks like a bug, maybe need call proto.BlockEncodeJson?
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
	// TODO(nickeskov): check
	id, err := proto.NewBlockIDFromBase58(s)
	if err != nil {
		if invalidRune := findFirstInvalidRuneInBase58String(s); invalidRune != nil {
			return blockIDAtInvalidCharErr(*invalidRune, s)
		}
		return errors.Wrapf(err, "failed to decode id %q as base58 and failed to find firs", s)
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
	// nickeskov:
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
	err = json.NewEncoder(w).Encode(&blockHeightResponse{Height: height})
	if err != nil {
		return errors.Wrap(err,
			"BlockHeight: failed to marshal block to JSON and write to ResponseWriter")
	}
	return nil
}

// nickeskov: in scala node this route does not exist

func (a *NodeApi) BlockScoreAt(w http.ResponseWriter, r *http.Request) {
	s := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		handleError(w, &BadRequestError{err})
		return
	}
	rs, err := a.app.BlocksScoreAt(id)
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
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

func (a *NodeApi) PeersAll(w http.ResponseWriter, _ *http.Request) {
	rs, err := a.app.PeersAll()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) PeersSpawned(w http.ResponseWriter, _ *http.Request) {
	rs := a.app.PeersSpawned()
	sendJson(w, rs)
}

type PeersConnectRequest struct {
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func (a *NodeApi) PeersConnect(w http.ResponseWriter, r *http.Request) {
	req := new(PeersConnectRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		handleError(w, err)
		return
	}
	apiKey := r.Header.Get("X-API-Key")
	rs, err := a.app.PeersConnect(r.Context(), apiKey, fmt.Sprintf("%s:%d", req.Host, req.Port))
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) PeersConnected(w http.ResponseWriter, _ *http.Request) {
	rs, err := a.app.PeersConnected()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) PeersSuspended(w http.ResponseWriter, _ *http.Request) {
	rs, err := a.app.PeersSuspended()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) BlocksGenerators(w http.ResponseWriter, _ *http.Request) {
	rs, err := a.app.BlocksGenerators()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) poolTransactions(w http.ResponseWriter, _ *http.Request) {
	rs := a.app.PoolTransactions()
	sendJson(w, rs)
}

func (a *NodeApi) unconfirmedSize(w http.ResponseWriter, _ *http.Request) {
	sendJson(w, map[string]int{
		"size": a.app.PoolTransactions(),
	})
}

type rollbackRequest struct {
	Height uint64 `json:"height"`
}

type rollbackToHeight interface {
	RollbackToHeight(string, proto.Height) error
}

func RollbackToHeight(app rollbackToHeight) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		js := &rollbackRequest{}
		err := json.NewDecoder(r.Body).Decode(js)
		if err != nil {
			handleError(w, err)
			return
		}
		apiKey := r.Header.Get("X-API-Key")
		err = app.RollbackToHeight(apiKey, js.Height)
		if err != nil {
			handleError(w, err)
			return
		}
		sendJson(w, nil)
	}
}

type walletLoadKeysRequest struct {
	Password string `json:"password"`
}

type walletLoadKeys interface {
	LoadKeys(apiKey string, password []byte) error
}

func WalletLoadKeys(app walletLoadKeys) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		js := &walletLoadKeysRequest{}
		err := json.NewDecoder(r.Body).Decode(js)
		if err != nil {
			handleError(w, err)
			return
		}
		apiKey := r.Header.Get("X-API-Key")
		err = app.LoadKeys(apiKey, []byte(js.Password))
		if err != nil {
			handleError(w, err)
			return
		}
		sendJson(w, nil)
	}
}

func (a *NodeApi) WalletAccounts(w http.ResponseWriter, _ *http.Request) {
	rs, err := a.app.Accounts()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) MinerInfo(w http.ResponseWriter, _ *http.Request) {
	rs, err := a.app.Miner()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, rs)
}

func (a *NodeApi) Addresses(w http.ResponseWriter, _ *http.Request) {
	addresses, err := a.app.Addresses()
	if err != nil {
		handleError(w, err)
		return
	}
	sendJson(w, addresses)
}

func (a *NodeApi) nodeProcesses(w http.ResponseWriter, _ *http.Request) {
	rs := a.app.NodeProcesses()
	sendJson(w, rs)
}

func (a *NodeApi) stateHash(w http.ResponseWriter, r *http.Request) {
	s := chi.URLParam(r, "height")
	height, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stateHash, err := a.state.StateHashAtHeight(height)
	if err != nil {
		handleError(w, err)
		return
	}
	err = json.NewEncoder(w).Encode(stateHash)
	if err != nil {
		handleError(w, err)
		return
	}
}

// TODO(nickeskov): use ApiError type and send JSON body
// 	remove this
func handleError(w http.ResponseWriter, err error) {
	switch err.(type) {
	case *AuthError:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusForbidden)
	case *BadRequestError:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusBadRequest)
	default:
		http.Error(w, fmt.Sprintf("Failed to complete request: %s", err.Error()), http.StatusInternalServerError)
	}
}

// TODO(nickeskov): use ApiError type and send JSON body
// 	remove this
func sendJson(w http.ResponseWriter, v interface{}) {
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %s", err.Error()), http.StatusInternalServerError)
	}
}
