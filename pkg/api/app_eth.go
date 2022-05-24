package api

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride/serialization"
	"github.com/wavesplatform/gowaves/pkg/state"
)

func (a *App) EthereumDAppMethods(addr proto.WavesAddress) (ethabi.MethodsMap, error) {
	scriptInfo, err := a.state.ScriptInfoByAccount(proto.NewRecipientFromAddress(addr))
	if err != nil {
		if state.IsNotFound(err) {
			return ethabi.MethodsMap{}, errors.Wrap(notFound, "script is not found")
		}
		return ethabi.MethodsMap{}, err
	}
	if len(scriptInfo.Bytes) == 0 {
		return ethabi.MethodsMap{}, errors.Wrap(notFound, "script is empty")
	}
	tree, err := serialization.Parse(scriptInfo.Bytes)
	if err != nil {
		return ethabi.MethodsMap{}, err
	}
	return ethabi.NewMethodsMapFromRideDAppMeta(tree.Meta)
}
