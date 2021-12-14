package state

import (
	"github.com/pkg/errors"
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

func (e *ethInfo) fillRequiredTxFields(ethTx *proto.EthereumTransaction, params *appendTxParams) error {
	EthSenderAddr, err := ethTx.From()
	if err != nil {
		return err
	}
	senderAddress, err := EthSenderAddr.ToWavesAddress(e.settings.AddressSchemeCharacter)
	if err != nil {
		return err
	}

	switch kind := ethTx.TxKind.(type) {
	case *proto.EthereumTransferWavesTxKind:
		kind.From = senderAddress

	case *proto.EthereumTransferAssetsErc20TxKind:
		db := ethabi.NewErc20MethodsMap()
		decodedData, err := db.ParseCallData(ethTx.Data())
		if err != nil {
			return errors.Errorf("failed to parse ethereum data")
		}
		if len(decodedData.Inputs) != ethabi.NumberOfERC20TransferArguments {
			return errors.Errorf("the number of arguments of erc20 function is %d, but expected it to be %d", len(decodedData.Inputs), ethabi.NumberOfERC20TransferArguments)
		}
		assetID := (*proto.AssetID)(ethTx.To())
		assetInfo, err := e.stor.assets.newestAssetInfo(*assetID, true)
		if err != nil {
			return errors.Wrap(err, "failed to get asset info")
		}
		fullAssetID := proto.ReconstructDigest(*assetID, assetInfo.tail)

		kind.Asset = proto.NewOptionalAssetFromDigest(fullAssetID)
		kind.From = senderAddress
	case *proto.EthereumInvokeScriptTxKind:
		scriptAddr, err := ethTx.WavesAddressTo(e.settings.AddressSchemeCharacter)
		if err != nil {
			return err
		}
		tree, err := e.stor.scriptsStorage.newestScriptByAddr(*scriptAddr, !params.initialisation)
		if err != nil {
			return errors.Wrapf(err, "failed to instantiate script on address '%s'", scriptAddr.String())
		}
		db, err := ethabi.NewMethodsMapFromRideDAppMeta(tree.Meta)
		if err != nil {
			return err
		}
		decodedData, err := db.ParseCallData(ethTx.Data())
		if err != nil {
			return errors.Wrap(err, "failed to parse ethereum data")
		}

		kind.DecodedCallData = decodedData
		kind.From = senderAddress
	default:
		return errors.New("unexpected ethereum tx kind")
	}
	return nil
}
