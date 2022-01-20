package api

import (
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride"
)

func (a *App) EthereumDAppMethods(addr proto.WavesAddress) (ethabi.MethodsMap, error) {
	scriptInfo, err := a.state.ScriptInfoByAccount(proto.NewRecipientFromAddress(addr))
	if err != nil {
		return ethabi.MethodsMap{}, err
	}
	tree, err := ride.Parse(scriptInfo.Bytes)
	if err != nil {
		return ethabi.MethodsMap{}, err
	}
	return ethabi.NewMethodsMapFromRideDAppMeta(tree.Meta)
}
