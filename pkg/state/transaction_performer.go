package state

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/crypto"
	"github.com/wavesplatform/gowaves/pkg/proto"
	"github.com/wavesplatform/gowaves/pkg/settings"
)

type performerInfo struct {
	initialisation bool
	blockID        crypto.Signature
}

type transactionPerformer struct {
	stor     *blockchainEntitiesStorage
	settings *settings.BlockchainSettings
}

func newTransactionPerformer(stor *blockchainEntitiesStorage, settings *settings.BlockchainSettings) (*transactionPerformer, error) {
	return &transactionPerformer{stor, settings}, nil
}

func (tp *transactionPerformer) performIssue(tx *proto.Issue, id []byte, info *performerInfo) error {
	assetID, err := crypto.NewDigestFromBytes(id)
	if err != nil {
		return err
	}
	// Create new asset.
	assetInfo := &assetInfo{
		assetConstInfo: assetConstInfo{
			issuer:      tx.SenderPK,
			name:        tx.Name,
			description: tx.Description,
			decimals:    int8(tx.Decimals),
		},
		assetChangeableInfo: assetChangeableInfo{
			quantity:   *big.NewInt(int64(tx.Quantity)),
			reissuable: tx.Reissuable,
		},
	}
	if err := tp.stor.assets.issueAsset(assetID, assetInfo, info.blockID); err != nil {
		return errors.Wrap(err, "failed to issue asset")
	}
	return nil
}

func (tp *transactionPerformer) performIssueV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.IssueV1)
	if !ok {
		return errors.New("failed to convert interface to IssueV1 transaction")
	}
	txID, err := tx.GetID()
	if err != nil {
		return errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	return tp.performIssue(&tx.Issue, txID, info)
}

func (tp *transactionPerformer) performIssueV2(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.IssueV2)
	if !ok {
		return errors.New("failed to convert interface to IssueV2 transaction")
	}
	txID, err := tx.GetID()
	if err != nil {
		return errors.Errorf("failed to get transaction ID: %v\n", err)
	}
	return tp.performIssue(&tx.Issue, txID, info)
}

func (tp *transactionPerformer) performReissue(tx *proto.Reissue, info *performerInfo) error {
	// Modify asset.
	change := &assetReissueChange{
		reissuable: tx.Reissuable,
		diff:       int64(tx.Quantity),
		blockID:    info.blockID,
	}
	if err := tp.stor.assets.reissueAsset(tx.AssetID, change, !info.initialisation); err != nil {
		return errors.Wrap(err, "failed to reissue asset")
	}
	return nil
}

func (tp *transactionPerformer) performReissueV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.ReissueV1)
	if !ok {
		return errors.New("failed to convert interface to ReissueV1 transaction")
	}
	return tp.performReissue(&tx.Reissue, info)
}

func (tp *transactionPerformer) performReissueV2(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.ReissueV2)
	if !ok {
		return errors.New("failed to convert interface to ReissueV2 transaction")
	}
	return tp.performReissue(&tx.Reissue, info)
}

func (tp *transactionPerformer) performBurn(tx *proto.Burn, info *performerInfo) error {
	// Modify asset.
	change := &assetBurnChange{
		diff:    int64(tx.Amount),
		blockID: info.blockID,
	}
	if err := tp.stor.assets.burnAsset(tx.AssetID, change, !info.initialisation); err != nil {
		return errors.Wrap(err, "failed to burn asset")
	}
	return nil
}

func (tp *transactionPerformer) performBurnV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.BurnV1)
	if !ok {
		return errors.New("failed to convert interface to BurnV1 transaction")
	}
	return tp.performBurn(&tx.Burn, info)
}

func (tp *transactionPerformer) performBurnV2(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.BurnV2)
	if !ok {
		return errors.New("failed to convert interface to BurnV2 transaction")
	}
	return tp.performBurn(&tx.Burn, info)
}

func (tp *transactionPerformer) performLease(tx *proto.Lease, id *crypto.Digest, info *performerInfo) error {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	var recipientAddr *proto.Address
	if tx.Recipient.Address == nil {
		recipientAddr, err = tp.stor.aliases.newestAddrByAlias(tx.Recipient.Alias.Alias, !info.initialisation)
		if err != nil {
			return errors.Errorf("invalid alias: %v\n", err)
		}
	} else {
		recipientAddr = tx.Recipient.Address
	}
	// Add leasing to lease state.
	l := &leasing{true, tx.Amount, *recipientAddr, senderAddr}
	if err := tp.stor.leases.addLeasing(*id, l, info.blockID); err != nil {
		return errors.Wrap(err, "failed to add leasing")
	}
	return nil
}

func (tp *transactionPerformer) performLeaseV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseV1 transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info)
}

func (tp *transactionPerformer) performLeaseV2(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseV2 transaction")
	}
	return tp.performLease(&tx.Lease, tx.ID, info)
}

func (tp *transactionPerformer) performLeaseCancel(tx *proto.LeaseCancel, info *performerInfo) error {
	if err := tp.stor.leases.cancelLeasing(tx.LeaseID, info.blockID, !info.initialisation); err != nil {
		return errors.Wrap(err, "failed to cancel leasing")
	}
	return nil
}

func (tp *transactionPerformer) performLeaseCancelV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelV1)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV1 transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, info)
}

func (tp *transactionPerformer) performLeaseCancelV2(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.LeaseCancelV2)
	if !ok {
		return errors.New("failed to convert interface to LeaseCancelV2 transaction")
	}
	return tp.performLeaseCancel(&tx.LeaseCancel, info)
}

func (tp *transactionPerformer) performCreateAlias(tx *proto.CreateAlias, info *performerInfo) error {
	senderAddr, err := proto.NewAddressFromPublicKey(tp.settings.AddressSchemeCharacter, tx.SenderPK)
	if err != nil {
		return err
	}
	// Save alias to aliases storage.
	inf := &aliasInfo{
		stolen: tp.stor.aliases.exists(tx.Alias.Alias, !info.initialisation),
		addr:   senderAddr,
	}
	if err := tp.stor.aliases.createAlias(tx.Alias.Alias, inf, info.blockID); err != nil {
		return err
	}
	return nil
}

func (tp *transactionPerformer) performCreateAliasV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasV1)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV1 transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info)
}

func (tp *transactionPerformer) performCreateAliasV2(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.CreateAliasV2)
	if !ok {
		return errors.New("failed to convert interface to CreateAliasV2 transaction")
	}
	return tp.performCreateAlias(&tx.CreateAlias, info)
}

func (tp *transactionPerformer) performDataV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.DataV1)
	if !ok {
		return errors.New("failed to convert interface to DataV1 transaction")
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

func (tp *transactionPerformer) performSponsorshipV1(transaction proto.Transaction, info *performerInfo) error {
	tx, ok := transaction.(*proto.SponsorshipV1)
	if !ok {
		return errors.New("failed to convert interface to SponsorshipV1 transaction")
	}
	if err := tp.stor.sponsoredAssets.sponsorAsset(tx.AssetID, tx.MinAssetFee, info.blockID); err != nil {
		return errors.Wrap(err, "failed to sponsor asset")
	}
	return nil
}
