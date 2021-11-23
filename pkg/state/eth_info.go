package state

import (
	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/errs"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/proto/ethabi"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type ethInfo struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newEthInfo(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) *ethInfo {
	return &ethInfo{stor: stor, settings: settings}
}

func GuessEthereumTransactionKind(
	data []byte,
	to *proto.EthereumAddress,
	newestAssetInfo func(id proto.AssetID, filter bool) (*assetInfo, error)) (int64, error) {
	if data == nil {
		return EthereumTransferWavesKind, nil
	}

	selectorBytes := data
	if len(data) < ethabi.SelectorSize {
		return 0, errors.Errorf("length of data from ethereum transaction is less than %d", ethabi.SelectorSize)
	}
	selector, err := ethabi.NewSelectorFromBytes(selectorBytes[:ethabi.SelectorSize])
	if err != nil {
		return 0, errors.Errorf("failed to guess ethereum transaction kind, %v", err)
	}

	assetID := (*proto.AssetID)(to)

	_, err = newestAssetInfo(*assetID, true)
	if err != nil && !errors.Is(err, errs.UnknownAsset{}) {
		return 0, errors.Errorf("failed to get asset info by ethereum recipient address %s, %v", to.String(), err)
	}

	if ethabi.IsERC20TransferSelector(selector) && err == nil {
		return EthereumTransferAssetsKind, nil
	}

	return EthereumInvokeKind, nil
}

func (e *ethInfo) ethereumTransactionKind(ethTx *proto.EthereumTransaction, params *appendTxParams) (proto.EthereumTransactionKind, error) {
	txKind, err := GuessEthereumTransactionKind(ethTx.Data(), ethTx.To(), e.stor.assets.newestAssetInfo)
	if err != nil {
		return nil, errors.Errorf("failed to guess ethereum tx kind, %v", err)
	}

	switch txKind {
	case EthereumTransferWavesKind:
		return proto.NewEthereumTransferWavesTxKind(), nil
	case EthereumTransferAssetsKind:

		db := ethabi.NewErc20MethodsMap()
		decodedData, err := db.ParseCallDataRide(ethTx.Data())
		if err != nil {
			return nil, errors.Errorf("failed to parse ethereum data")
		}
		if len(decodedData.Inputs) != ethabi.NumberOfERC20TransferArguments {
			return nil, errors.Errorf("the number of arguments of erc20 function is %d, but expected it to be %d", len(decodedData.Inputs), ethabi.NumberOfERC20TransferArguments)
		}
		assetID := (*proto.AssetID)(ethTx.To())

		assetInfo, err := e.stor.assets.newestAssetInfo(*assetID, true)
		if err != nil {
			return nil, errors.Errorf("failed to get asset info %v", err)
		}
		fullAssetID := proto.ReconstructDigest(*assetID, assetInfo.tail)
		return proto.NewEthereumTransferAssetsErc20TxKind(*decodedData, *proto.NewOptionalAssetFromDigest(fullAssetID)), nil
	case EthereumInvokeKind:
		scriptAddr, err := ethTx.WavesAddressTo(e.settings.AddressSchemeCharacter)
		if err != nil {
			return nil, err
		}
		tree, err := e.stor.scriptsStorage.newestScriptByAddr(*scriptAddr, !params.initialisation)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
		}
		db, err := ethabi.NewMethodsMapFromRideDAppMeta(tree.Meta)
		if err != nil {
			return nil, err
		}
		decodedData, err := db.ParseCallDataRide(ethTx.Data())
		if err != nil {
			return nil, errors.Errorf("failed to parse ethereum data, %v", err)
		}

		return proto.NewEthereumInvokeScriptTxKind(*decodedData), nil

	default:
		return nil, errors.Errorf("unexpected ethereum tx kind")
	}
}
