package client

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

/*
** TransactionInfo
 */

type TransactionInfo interface {
	// Supposed that each struct which implements this interface embeddes
	// related proto.<*>Transaction struct and TransactionInfoCommon struct.

	proto.Transaction

	// Must call common function TransactionInfoUnmarshalJSON and pass itself as the 2nd argument
	json.Unmarshaler

	TransactionInfoCommon

	unmarshalSpecificData(data []byte) error
	getTransactionObject() proto.Transaction
	getInfoCommonObject() TransactionInfoCommon
}

func TransactionInfoUnmarshalJSON(data []byte, txInfo TransactionInfo) error {
	if err := json.Unmarshal(data, txInfo.getTransactionObject()); err != nil {
		return errors.Wrap(err, "Unmarshal proto.Transaction failed")
	}

	if err := json.Unmarshal(data, txInfo.getInfoCommonObject()); err != nil {
		return errors.Wrap(err, "Unmarshal TransactionInfoCommon failed")
	}

	if err := txInfo.unmarshalSpecificData(data); err != nil {
		return errors.Wrap(err, "Unmarshal specific data failed")
	}
	return nil
}

func GuessTransactionInfoType(t *proto.TransactionTypeVersion) (TransactionInfo, error) {
	var out TransactionInfo
	switch t.Type {
	case proto.GenesisTransaction: // 1
		out = &GenesisTransactionInfo{}
	case proto.PaymentTransaction: // 2
		out = &PaymentTransactionInfo{}
	case proto.IssueTransaction: // 3
		if t.Version >= 2 {
			out = &IssueWithProofsTransactionInfo{}
		} else {
			// TODO: impl IssueWithSigTransactionInfo
			out = &IssueWithSigTransactionInfo{}
		}
	case proto.TransferTransaction: // 4
		if t.Version >= 2 {
			out = &TransferWithProofsTransactionInfo{}
		} else {
			out = &TransferWithSigTransactionInfo{}
		}
	case proto.ReissueTransaction: // 5
		if t.Version >= 2 {
			out = &ReissueWithProofsTransactionInfo{}
		} else {
			out = &ReissueWithSigTransactionInfo{}
		}
	case proto.BurnTransaction: // 6
		if t.Version >= 2 {
			out = &BurnWithProofsTransactionInfo{}
		} else {
			out = &BurnWithSigTransactionInfo{}
		}
	case proto.ExchangeTransaction: // 7
		if t.Version >= 2 {
			out = &ExchangeWithProofsTransactionInfo{}
		} else {
			out = &ExchangeWithSigTransactionInfo{}
		}
	case proto.LeaseTransaction: // 8
		if t.Version >= 2 {
			out = &LeaseWithProofsTransactionInfo{}
		} else {
			out = &LeaseWithSigTransactionInfo{}
		}
	case proto.LeaseCancelTransaction: // 9
		if t.Version >= 2 {
			out = &LeaseCancelWithProofsTransactionInfo{}
		} else {
			out = &LeaseCancelWithSigTransactionInfo{}
		}
	case proto.CreateAliasTransaction: // 10
		if t.Version >= 2 {
			out = &CreateAliasWithProofsTransactionInfo{}
		} else {
			out = &CreateAliasWithSigTransactionInfo{}
		}
	case proto.MassTransferTransaction: // 11
		out = &MassTransferTransactionInfo{}
	case proto.DataTransaction: // 12
		out = &DataTransactionInfo{}
	case proto.SetScriptTransaction: // 13
		out = &SetScriptTransactionInfo{}
	case proto.SponsorshipTransaction: // 14
		out = &SponsorshipTransactionInfo{}
	case proto.SetAssetScriptTransaction: // 15
		out = &SetAssetScriptTransactionInfo{}
	case proto.InvokeScriptTransaction: // 16
		out = &InvokeScriptTransactionInfo{}
	case proto.UpdateAssetInfoTransaction: // 17
		out = &UpdateAssetInfoTransactionInfo{}
	case proto.EthereumMetamaskTransaction: // 18
		out = &EthereumTransactionInfo{}
	}
	if out == nil {
		return nil, errors.Errorf("unknown transaction type %d version %d", t.Type, t.Version)
	}
	return out, nil
}

/*
** TransactionInfoCommon
 */

type TransactionInfoCommon interface {
	GetSpentComplexity() int
	GetHeight() proto.Height
}

type TransactionInfoCommonImpl struct {
	SpentComplexity int          `json:"spentComplexity"`
	Height          proto.Height `json:"height"`
}

func (txInfoCommon *TransactionInfoCommonImpl) GetSpentComplexity() int {
	return txInfoCommon.SpentComplexity
}

func (txInfoCommon *TransactionInfoCommonImpl) GetHeight() proto.Height {
	return txInfoCommon.Height
}

/*
 ** TODO: consider this implementation when embedded type parameter is implemented in Go.
 **
 ** Here is possible future implementation using generics (draft). This may reduce
 ** hundreds lines of code and eliminate copy-paste pattern.
 ** In Go 1.18 this approach is not possible due to not allowed embedded type parameter.
 ** But this feature seems to be implemented in the next versions of Go. See Go proposal:
 ** https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#embedded-type-parameter

type TransactionInfoImpl[T proto.Transaction] struct {
	T
	TransactionInfoCommonImpl
}

func (txInfo *TransactionInfoImpl[T]) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *TransactionInfoImpl[T]) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *TransactionInfoImpl[T]) getTransactionObject() proto.Transaction {
	return &txInfo.T
}

func (txInfo *TransactionInfoImpl[T]) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

// Two ways to implement certain transaction info.
// 1. If there's no need to override unmarshalSpecificData method. Type alias:
type CertainTransaction1Info = TransactionInfoImpl[proto.CertainTransaction1]

// 2. If unmarshalSpecificData method is needed to override:
type CertainTransaction2Info struct {
	TransactionInfoImpl[proto.CertainTransaction2]
}

func (txInfo *CertainTransaction2Info[T]) unmarshalSpecificData(data []byte) error {
	// implementation of unmarshalling specific data
	return nil
}
*/

/*
 ** Genesis Transaction
 */

type GenesisTransactionInfo struct {
	proto.Genesis
	TransactionInfoCommonImpl
}

func (txInfo *GenesisTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *GenesisTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *GenesisTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.Genesis
}

func (txInfo *GenesisTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** Payment Transaction
 */

type PaymentTransactionInfo struct {
	proto.Payment
	TransactionInfoCommonImpl
}

func (txInfo *PaymentTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *PaymentTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *PaymentTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.Payment
}

func (txInfo *PaymentTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** Issue Transaction
 */

type IssueWithProofsTransactionInfo struct {
	proto.IssueWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *IssueWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *IssueWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *IssueWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.IssueWithProofs
}

func (txInfo *IssueWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** Issue Transaction
 */

type IssueWithSigTransactionInfo struct {
	proto.IssueWithSig
	TransactionInfoCommonImpl
}

func (txInfo *IssueWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *IssueWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *IssueWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.IssueWithSig
}

func (txInfo *IssueWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** TransferWithProofs Transaction
 */

type TransferWithProofsTransactionInfo struct {
	proto.TransferWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *TransferWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *TransferWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *TransferWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.TransferWithProofs
}

func (txInfo *TransferWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** TransferWithSig Transaction
 */

type TransferWithSigTransactionInfo struct {
	proto.TransferWithSig
	TransactionInfoCommonImpl
}

func (txInfo *TransferWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *TransferWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *TransferWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.TransferWithSig
}

func (txInfo *TransferWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** ReissueWithProofs Transaction
 */

type ReissueWithProofsTransactionInfo struct {
	proto.ReissueWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *ReissueWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *ReissueWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *ReissueWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ReissueWithProofs
}

func (txInfo *ReissueWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
 ** ReissueWithSig Transaction
 */

type ReissueWithSigTransactionInfo struct {
	proto.ReissueWithSig
	TransactionInfoCommonImpl
}

func (txInfo *ReissueWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *ReissueWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *ReissueWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ReissueWithSig
}

func (txInfo *ReissueWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** BurnWithProofs Transaction
 */

type BurnWithProofsTransactionInfo struct {
	proto.BurnWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *BurnWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *BurnWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *BurnWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.BurnWithProofs
}

func (txInfo *BurnWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** BurnWithSig Transaction
 */

type BurnWithSigTransactionInfo struct {
	proto.BurnWithSig
	TransactionInfoCommonImpl
}

func (txInfo *BurnWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *BurnWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *BurnWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.BurnWithSig
}

func (txInfo *BurnWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** ExchangeWithProofs Transaction
 */

type ExchangeWithProofsTransactionInfo struct {
	proto.ExchangeWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *ExchangeWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *ExchangeWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *ExchangeWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ExchangeWithProofs
}

func (txInfo *ExchangeWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** ExchangeWithSig Transaction
 */

type ExchangeWithSigTransactionInfo struct {
	proto.ExchangeWithSig
	TransactionInfoCommonImpl
}

func (txInfo *ExchangeWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *ExchangeWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *ExchangeWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ExchangeWithSig
}

func (txInfo *ExchangeWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** LeaseWithProofs Transaction
 */

type LeaseWithProofsTransactionInfo struct {
	proto.LeaseWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *LeaseWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *LeaseWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *LeaseWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseWithProofs
}

func (txInfo *LeaseWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** LeaseWithSig Transaction
 */

type LeaseWithSigTransactionInfo struct {
	proto.LeaseWithSig
	TransactionInfoCommonImpl
}

func (txInfo *LeaseWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *LeaseWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *LeaseWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseWithSig
}

func (txInfo *LeaseWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** LeaseCancelWithProofs Transaction
 */

type LeaseCancelWithProofsTransactionInfo struct {
	proto.LeaseCancelWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseCancelWithProofs
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** LeaseCancelWithSig Transaction
 */

type LeaseCancelWithSigTransactionInfo struct {
	proto.LeaseCancelWithSig
	TransactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *LeaseCancelWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseCancelWithSig
}

func (txInfo *LeaseCancelWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** CreateAliasWithProofs Transaction
 */

type CreateAliasWithProofsTransactionInfo struct {
	proto.CreateAliasWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *CreateAliasWithProofsTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *CreateAliasWithProofsTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *CreateAliasWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.CreateAliasWithProofs
}

func (txInfo *CreateAliasWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** CreateAliasWithSig Transaction
 */

type CreateAliasWithSigTransactionInfo struct {
	proto.CreateAliasWithSig
	TransactionInfoCommonImpl
}

func (txInfo *CreateAliasWithSigTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *CreateAliasWithSigTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *CreateAliasWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.CreateAliasWithSig
}

func (txInfo *CreateAliasWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** MassTransfer Transaction
 */

type MassTransferTransactionInfo struct {
	proto.MassTransferWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *MassTransferTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *MassTransferTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *MassTransferTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.MassTransferWithProofs
}

func (txInfo *MassTransferTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** Data Transaction
 */

type DataTransactionInfo struct {
	proto.DataWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *DataTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *DataTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *DataTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.DataWithProofs
}

func (txInfo *DataTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** SetScript Transaction
 */

type SetScriptTransactionInfo struct {
	proto.SetScriptWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *SetScriptTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *SetScriptTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *SetScriptTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.SetScriptWithProofs
}

func (txInfo *SetScriptTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** Sponsorship Transaction
 */

type SponsorshipTransactionInfo struct {
	proto.SponsorshipWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *SponsorshipTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *SponsorshipTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *SponsorshipTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.SponsorshipWithProofs
}

func (txInfo *SponsorshipTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** SetAssetScript Transaction
 */

type SetAssetScriptTransactionInfo struct {
	proto.SetAssetScriptWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *SetAssetScriptTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *SetAssetScriptTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *SetAssetScriptTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.SetAssetScriptWithProofs
}

func (txInfo *SetAssetScriptTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** InvokeScript Transaction
 */

type InvokeScriptTransactionInfo struct {
	proto.InvokeScriptWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *InvokeScriptTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *InvokeScriptTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *InvokeScriptTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.InvokeScriptWithProofs
}

func (txInfo *InvokeScriptTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}

/*
** UpdateAssetInfo Transaction
 */

type UpdateAssetInfoTransactionInfo struct {
	proto.UpdateAssetInfoWithProofs
	TransactionInfoCommonImpl
}

func (txInfo *UpdateAssetInfoTransactionInfo) unmarshalSpecificData(data []byte) error {
	return nil
}

func (txInfo *UpdateAssetInfoTransactionInfo) getInfoCommonObject() TransactionInfoCommon {
	return &txInfo.TransactionInfoCommonImpl
}

func (txInfo *UpdateAssetInfoTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.UpdateAssetInfoWithProofs
}

func (txInfo *UpdateAssetInfoTransactionInfo) UnmarshalJSON(data []byte) error {
	return TransactionInfoUnmarshalJSON(data, txInfo)
}
