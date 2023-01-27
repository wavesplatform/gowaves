package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type performerInfo struct {
	height  uint64
	blockID proto.BlockID
}

type transactionPerformer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor, settings}, nil
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, assetID crypto.Digest, info *performerInfo) error {
	blockHeight := info.height + 1
	// Create new asset.
	assetInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			tail:     proto.DigestTail(assetID),
			issuer:   tx.SenderPK,
			decimals: int8(tx.Decimals),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:                 *big.NewInt(int64(tx.Quantity)),
			name:                     tx.Name,
			description:              tx.Description,
			lastNameDescChangeHeight: blockHeight,
			reissuable:               tx.Reissuable,
		},
	}
	if err := tp.stor.assets.issueAsset(proto.AssetIDFromDigest(assetID), assetInfo, info.blockID); err != nil {
		return errors.Wrap(err, "failed to issue asset")
	}
	return nil
}

func (tp *transactionPerformer) performIssueWithSig(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.IssueWithSig)
	if !ok {
		return errors.New("failed to convert interface to IssueWithSig transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return err
	}
	if err := tp.stor.scriptsStorage.setAssetScript(assetID, proto.Script{}, tx.SenderPK, info.blockID); err != nil {
		return err
	}
	return tp.performIssue(&tx.Issue, assetID, info)
}

func (tp *transactionPerformer) performIssueWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.IssueWithProofs)
	if !ok {
		return errors.New("failed to convert interface to IssueWithProofs transaction")
	}
	txID, err := tx.GetID(tp.settings.AddressSchemeCharacter)
	if err != nil {
		return errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	assetID, err := crypto.NewDigestFromBytes(txID)
	if err != nil {
		return err
	}
	if err := tp.stor.scriptsStorage.setAssetScript(assetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return err
	}
	return tp.performIssue(&tx.Issue, assetID, info)
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, info *performerInfo) error {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
	}
	if err := tp.stor.assets.reissueAsset(proto.AssetIDFromDigest(tx.AssetID), change, info.blockID); err != nil {
		return errors.Wrap(err, "failed to reissue asset")
	}
	return nil
}

func (tp *transactionPerformer) performReissueWithSig(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.ReissueWithSig)
	if !ok {
		return errors.New("failed to convert interface to ReissueWithSig transaction")
	}
	return tp.performReissue(&tx.Reissue, info)
}

func (tp *transactionPerformer) performReissueWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.ReissueWithProofs)
	if !ok {
		return errors.New("failed to convert interface to ReissueWithProofs transaction")
	}
	return tp.performReissue(&tx.Reissue, info)
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, info *performerInfo) error {
	// Modify asset.
	change := &assetBurnChange{
		diff: int64(tx.Amount),
	}
	if err := tp.stor.assets.burnAsset(proto.AssetIDFromDigest(tx.AssetID), change, info.blockID); err != nil {
		return errors.Wrap(err, "failed to burn asset")
	}
	return nil
}

func (tp *transactionPerformer) performBurnWithSig(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.BurnWithSig)
	if !ok {
		return errors.New("failed to convert interface to BurnWithSig transaction")
	}
	return tp.performBurn(&tx.Burn, info)
}

func (tp *transactionPerformer) performBurnWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.BurnWithProofs)
	if !ok {
		return errors.New("failed to convert interface to BurnWithProofs transaction")
	}
	return tp.performBurn(&tx.Burn, info)
}

func (tp *transactionPerformer) increaseOrderVolume(order proto.Order, tx proto.Exchange, info *performerInfo) error {
	orderId, err := order.GetID()
	if err != nil {
		return err
	}
	fee := tx.GetBuyMatcherFee()
	if order.GetOrderType() == proto.Sell {
		fee = tx.GetSellMatcherFee()
	}
	if err := tp.stor.ordersVolumes.increaseFilledFee(orderId, fee, info.blockID); err != nil {
		return err
	}
	if err := tp.stor.ordersVolumes.increaseFilledAmount(orderId, tx.GetAmount(), info.blockID); err != nil {
		return err
	}
	return nil
}

func (tp *transactionPerformer) performExchange(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(proto.Exchange)
	if !ok {
		return errors.New("failed to convert interface to Exchange transaction")
	}
	so, err := tx.GetSellOrder()
	if err != nil {
		return errors.Wrap(err, "no sell order")
	}
	if err := tp.increaseOrderVolume(so, tx, info); err != nil {
		return err
	}
	bo, err := tx.GetBuyOrder()
	if err != nil {
		return errors.Wrap(err, "no buy order")
	}
	if err := tp.increaseOrderVolume(bo, tx, info); err != nil {
		return err
	}
	return nil
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, id *crypto.Digest, info *performerInfo) error {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	var recipientAddr *proto.WavesAddress
	if addr := tx.Recipient.Address(); addr == nil {
		recipientAddr, err = tp.stor.aliases.newestAddrByAlias(tx.Recipient.Alias().Alias)
		if err != nil {
			return errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = addr
	}
	// Add leasing to lease state.
	l := &leasing{
		Sender:         senderAddr,
		Recipient:      *recipientAddr,
		Amount:         tx.Amount,
		Height:         info.height,
		Status:         LeaseActive,
		RecipientAlias: tx.Recipient.Alias(),
	}
	if err := tp.stor.leases.addLeasing(*id, l, info.blockID); err != nil {
		return errors.Wrap(err, "failed to add leasing")
	}
	return nil
}

func (tp *transactionPerformer) performLeaseWithSig(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseWithSig)
	if !ok {
		return errors.New("failed to convert interface to LeaseWithSig transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info)
}

func (tp *transactionPerformer) performLeaseWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseWithProofs)
	if !ok {
		return errors.New("failed to convert interface to LeaseWithProofs transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, txID *crypto.Digest, info *performerInfo) error {
	if err := tp.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, info.height, txID); err != nil {
		return errors.Wrap(err, "failed to cancel leasing")
	}
	return nil
}

func (tp *transactionPerformer) performLeaseCancelWithSig(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelWithSig)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelWithSig transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info)
}

func (tp *transactionPerformer) performLeaseCancelWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelWithProofs)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelWithProofs transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, tx.ID, info)
}

func (tp *transactionPerformer) performCreateAlias(tx *proto.CreateAlias, info *performerInfo) error {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	// Save alias to aliases storage.
	inf := &aliasInfo{
		stolen: tp.stor.aliases.exists(tx.Alias.Alias),
		addr:   senderAddr,
	}
	if err := tp.stor.aliases.createAlias(tx.Alias.Alias, inf, info.blockID); err != nil {
		return err
	}
	return nil
}

func (tp *transactionPerformer) performCreateAliasWithSig(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasWithSig)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasWithSig transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info)
}

func (tp *transactionPerformer) performCreateAliasWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasWithProofs)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasWithProofs transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info)
}

func (tp *transactionPerformer) performDataWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.DataWithProofs)
	if !ok {
		return errors.New("failed to convert interface to DataWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	for _, entry := range tx.Entries {
		if err := tp.stor.accountsDataStor.appendEntry(senderAddr, entry, info.blockID); err != nil {
			return err
		}
	}
	return nil
}

func (tp *transactionPerformer) performSponsorshipWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.SponsorshipWithProofs)
	if !ok {
		return errors.New("failed to convert interface to SponsorshipWithProofs transaction")
	}
	if err := tp.stor.sponsoredAssets.sponsorAsset(tx.AssetID, tx.MinAssetFee, info.blockID); err != nil {
		return errors.Wrap(err, "failed to sponsor asset")
	}
	return nil
}

func (tp *transactionPerformer) performSetScriptWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.SetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert interface to SetScriptWithProofs transaction")
	}
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	if err := tp.stor.scriptsStorage.setAccountScript(senderAddr, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return errors.Wrap(err, "failed to set account script")
	}
	return nil
}

func (tp *transactionPerformer) performSetAssetScriptWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.SetAssetScriptWithProofs)
	if !ok {
		return errors.New("failed to convert interface to SetAssetScriptWithProofs transaction")
	}
	if err := tp.stor.scriptsStorage.setAssetScript(tx.AssetID, tx.Script, tx.SenderPK, info.blockID); err != nil {
		return errors.Wrap(err, "failed to set asset script")
	}
	return nil
}

func (tp *transactionPerformer) performInvokeScriptWithProofs(transaction proto.Transaction, info *performerInfo) error {
	if _, ok := transaction.(*proto.InvokeScriptWithProofs); !ok {
		return errors.New("failed to convert interface to InvokeScriptWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return errors.Wrap(err, "failed to commit invoke changes")
	}
	return nil
}

func (tp *transactionPerformer) performInvokeExpressionWithProofs(transaction proto.Transaction, info *performerInfo) error {
	if _, ok := transaction.(*proto.InvokeExpressionTransactionWithProofs); !ok {
		return errors.New("failed to convert interface to InvokeExpressionWithProofs transaction")
	}
	if err := tp.stor.commitUncertain(info.blockID); err != nil {
		return errors.Wrap(err, "failed to commit invoke changes")
	}
	return nil
}

func (tp *transactionPerformer) performEthereumTransactionWithProofs(transaction proto.Transaction, info *performerInfo) error {
	ethTx, ok := transaction.(*proto.EthereumTransaction)
	if !ok {
		return errors.New("failed to convert interface to EthereumTransaction transaction")
	}
	if _, ok := ethTx.TxKind.(*proto.EthereumInvokeScriptTxKind); ok {
		if err := tp.stor.commitUncertain(info.blockID); err != nil {
			return errors.Wrap(err, "failed to commit invoke changes")
		}
	}
	// nothing to do for proto.EthereumTransferWavesTxKind and proto.EthereumTransferAssetsErc20TxKind
	return nil
}

func (tp *transactionPerformer) performUpdateAssetInfoWithProofs(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.UpdateAssetInfoWithProofs)
	if !ok {
		return errors.New("failed to convert interface to UpdateAssetInfoWithProofs transaction")
	}
	blockHeight := info.height + 1
	ch := &assetInfoChange{
		newName:        tx.Name,
		newDescription: tx.Description,
		newHeight:      blockHeight,
	}
	if err := tp.stor.assets.updateAssetInfo(tx.AssetID, ch, info.blockID); err != nil {
		return errors.Wrap(err, "failed to update asset info")
	}
	return nil
}
