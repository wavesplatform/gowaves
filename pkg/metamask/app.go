package metamask

import (
	"github.com/wavesplatform/gowaves/pkg/services"
)

type nodeRPCApp struct {
	*services.Services
}

//type ethTxKind byte
//
//const (
//	wavesTransferEthTxKind = iota + 1
//	assetTransferEthTxKind
//	invokeEthTxKind
//)
//
//func (app nodeRPCApp) guessEthTxKind(value *big.Int, data []byte) ethTxKind {
//	return 0
//}
