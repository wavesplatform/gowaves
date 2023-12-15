package proto

import (
	"github.com/pkg/errors"

	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/ride/ast"
)

type EthereumTransactionKindType byte

const (
	EthereumTransferWavesKindType EthereumTransactionKindType = iota + 1
	EthereumTransferAssetsKindType
	EthereumInvokeKindType
)

func GuessEthereumTransactionKindType(data []byte) (EthereumTransactionKindType, error) {
	if len(data) == 0 {
		return EthereumTransferWavesKindType, nil
	}

	selectorBytes := data
	if len(data) < ethabi.SelectorSize {
		return 0, errors.Errorf("length of data from ethereum transaction is less than %d", ethabi.SelectorSize)
	}
	selector, err := ethabi.NewSelectorFromBytes(selectorBytes[:ethabi.SelectorSize])
	if err != nil {
		return 0, errors.Wrap(err, "failed to guess ethereum transaction kind")
	}

	if ethabi.IsERC20TransferSelector(selector) {
		return EthereumTransferAssetsKindType, nil
	}

	return EthereumInvokeKindType, nil
}

type EthereumTransactionKindResolver interface {
	ResolveTxKind(ethTx *EthereumTransaction, isBlockRewardDistributionActivated bool) (EthereumTransactionKind, error)
}

type ethKindResolverState interface {
	NewestScriptByAccount(address Recipient) (*ast.Tree, error)
	NewestAssetConstInfo(assetID AssetID) (*AssetConstInfo, error)
}

type ethTxKindResolver struct {
	state  ethKindResolverState
	scheme Scheme
}

func NewEthereumTransactionKindResolver(resolver ethKindResolverState, scheme Scheme) EthereumTransactionKindResolver {
	return &ethTxKindResolver{state: resolver, scheme: scheme}
}

func (e *ethTxKindResolver) ResolveTxKind(ethTx *EthereumTransaction, isBlockRewardDistributionActivated bool) (EthereumTransactionKind, error) {
	txKind, err := GuessEthereumTransactionKindType(ethTx.Data())
	if err != nil {
		return nil, errors.Wrap(err, "failed to guess ethereum tx kind")
	}

	switch txKind {
	case EthereumTransferWavesKindType:
		return NewEthereumTransferWavesTxKind(), nil
	case EthereumTransferAssetsKindType:
		db := ethabi.NewErc20MethodsMap()
		decodedData, err := db.ParseCallDataRide(ethTx.Data(), isBlockRewardDistributionActivated)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse ethereum data")
		}
		if len(decodedData.Inputs) != ethabi.NumberOfERC20TransferArguments {
			return nil, errors.Errorf("the number of arguments of erc20 function is %d, but expected it to be %d", len(decodedData.Inputs), ethabi.NumberOfERC20TransferArguments)
		}
		assetID := (*AssetID)(ethTx.To())

		assetInfo, err := e.state.NewestAssetConstInfo(*assetID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get asset info")
		}
		erc20Arguments, err := ethabi.GetERC20TransferArguments(decodedData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get erc20 arguments from decoded data")
		}
		return NewEthereumTransferAssetsErc20TxKind(*decodedData, *NewOptionalAssetFromDigest(assetInfo.ID), erc20Arguments), nil
	case EthereumInvokeKindType:
		scriptAddr, err := ethTx.WavesAddressTo(e.scheme)
		if err != nil {
			return nil, err
		}
		tree, err := e.state.NewestScriptByAccount(NewRecipientFromAddress(scriptAddr))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
		}
		db, err := ethabi.NewMethodsMapFromRideDAppMeta(tree.Meta)
		if err != nil {
			return nil, err
		}
		decodedData, err := db.ParseCallDataRide(ethTx.Data(), isBlockRewardDistributionActivated)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse ethereum data")
		}

		return NewEthereumInvokeScriptTxKind(*decodedData), nil

	default:
		return nil, errors.New("unexpected ethereum tx kind")
	}
}
