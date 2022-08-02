package client

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/wavesplatform/gowaves/pkg/proto"
)

type TransactionInfo interface {
	proto.Transaction
	GetSpentComplexity() int
	GetHeight() proto.Height
}

// Supposed that each struct which implements this interface embeddes
// related proto.<*>Transaction struct and transactionInfoCommonImpl struct.
type transactionInfoInternal interface {
	TransactionInfo

	// Must call common function transactionInfoUnmarshalJSON and pass itself as the 2nd argument
	json.Unmarshaler

	unmarshalSpecificData(data []byte) error
	getTransactionObject() proto.Transaction
	getInfoCommonObject() *transactionInfoCommonImpl
}

func transactionInfoUnmarshalJSON(data []byte, txInfo transactionInfoInternal) error {
	if err := json.Unmarshal(data, txInfo.getTransactionObject()); err != nil {
		return errors.Wrap(err, "Unmarshal proto.Transaction failed")
	}

	if err := json.Unmarshal(data, txInfo.getInfoCommonObject()); err != nil {
		return errors.Wrap(err, "Unmarshal transactionInfoCommonImpl failed")
	}

	if err := txInfo.unmarshalSpecificData(data); err != nil {
		return errors.Wrap(err, "Unmarshal specific data failed")
	}
	return nil
}

func guessTransactionInfoType(t *proto.TransactionTypeVersion) (TransactionInfo, error) {
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

type transactionInfoCommonImpl struct {
	SpentComplexity int          `json:"spentComplexity"`
	Height          proto.Height `json:"height"`
}

func (txInfoCommon *transactionInfoCommonImpl) GetSpentComplexity() int {
	return txInfoCommon.SpentComplexity
}

func (txInfoCommon *transactionInfoCommonImpl) GetHeight() proto.Height {
	return txInfoCommon.Height
}

// noop. useful for transactions with no specific data
func (txInfoCommon *transactionInfoCommonImpl) unmarshalSpecificData(data []byte) error {
	return nil
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
	transactionInfoCommonImpl
}

func (txInfo *TransactionInfoImpl[T]) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, txInfo.T); err != nil {
		return errors.Wrap(err, "Unmarshal proto.Transaction failed")
	}

	if err := json.Unmarshal(data, &txInfo.transactionInfoCommonImpl); err != nil {
		return errors.Wrap(err, "Unmarshal transactionInfoCommonImpl failed")
	}

	return nil
}

// Two ways to implement certain transaction info.
// 1. If there's no need to parse specific data method. Type alias:
type CertainTransaction1Info = TransactionInfoImpl[proto.CertainTransaction1]

// 2. If parse specific data method is needed:
type CertainTransaction2Info struct {
	TransactionInfoImpl[proto.CertainTransaction2]
}

func (txInfo *CertainTransaction2Info[T]) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(txInfo.TransactionInfoImpl); err != nil {
		return err
	}
	// implementation of unmarshalling specific data
	return nil
}
*/

type GenesisTransactionInfo struct {
	proto.Genesis
	transactionInfoCommonImpl
}

func (txInfo *GenesisTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *GenesisTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.Genesis
}

func (txInfo *GenesisTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type PaymentTransactionInfo struct {
	proto.Payment
	transactionInfoCommonImpl
}

func (txInfo *PaymentTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *PaymentTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.Payment
}

func (txInfo *PaymentTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type IssueWithProofsTransactionInfo struct {
	proto.IssueWithProofs
	transactionInfoCommonImpl
}

func (txInfo *IssueWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *IssueWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.IssueWithProofs
}

func (txInfo *IssueWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type IssueWithSigTransactionInfo struct {
	proto.IssueWithSig
	transactionInfoCommonImpl
}

func (txInfo *IssueWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *IssueWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.IssueWithSig
}

func (txInfo *IssueWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type TransferWithProofsTransactionInfo struct {
	proto.TransferWithProofs
	transactionInfoCommonImpl
}

func (txInfo *TransferWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *TransferWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.TransferWithProofs
}

func (txInfo *TransferWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type TransferWithSigTransactionInfo struct {
	proto.TransferWithSig
	transactionInfoCommonImpl
}

func (txInfo *TransferWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *TransferWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.TransferWithSig
}

func (txInfo *TransferWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type ReissueWithProofsTransactionInfo struct {
	proto.ReissueWithProofs
	transactionInfoCommonImpl
}

func (txInfo *ReissueWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *ReissueWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ReissueWithProofs
}

func (txInfo *ReissueWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type ReissueWithSigTransactionInfo struct {
	proto.ReissueWithSig
	transactionInfoCommonImpl
}

func (txInfo *ReissueWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *ReissueWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ReissueWithSig
}

func (txInfo *ReissueWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type BurnWithProofsTransactionInfo struct {
	proto.BurnWithProofs
	transactionInfoCommonImpl
}

func (txInfo *BurnWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *BurnWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.BurnWithProofs
}

func (txInfo *BurnWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type BurnWithSigTransactionInfo struct {
	proto.BurnWithSig
	transactionInfoCommonImpl
}

func (txInfo *BurnWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *BurnWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.BurnWithSig
}

func (txInfo *BurnWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type ExchangeWithProofsTransactionInfo struct {
	proto.ExchangeWithProofs
	transactionInfoCommonImpl
}

func (txInfo *ExchangeWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *ExchangeWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ExchangeWithProofs
}

func (txInfo *ExchangeWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type ExchangeWithSigTransactionInfo struct {
	proto.ExchangeWithSig
	transactionInfoCommonImpl
}

func (txInfo *ExchangeWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *ExchangeWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.ExchangeWithSig
}

func (txInfo *ExchangeWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type LeaseWithProofsTransactionInfo struct {
	proto.LeaseWithProofs
	transactionInfoCommonImpl
}

func (txInfo *LeaseWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *LeaseWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseWithProofs
}

func (txInfo *LeaseWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type LeaseWithSigTransactionInfo struct {
	proto.LeaseWithSig
	transactionInfoCommonImpl
}

func (txInfo *LeaseWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *LeaseWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseWithSig
}

func (txInfo *LeaseWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type LeaseCancelWithProofsTransactionInfo struct {
	proto.LeaseCancelWithProofs
	transactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseCancelWithProofs
}

func (txInfo *LeaseCancelWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type LeaseCancelWithSigTransactionInfo struct {
	proto.LeaseCancelWithSig
	transactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *LeaseCancelWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.LeaseCancelWithSig
}

func (txInfo *LeaseCancelWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type CreateAliasWithProofsTransactionInfo struct {
	proto.CreateAliasWithProofs
	transactionInfoCommonImpl
}

func (txInfo *CreateAliasWithProofsTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *CreateAliasWithProofsTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.CreateAliasWithProofs
}

func (txInfo *CreateAliasWithProofsTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type CreateAliasWithSigTransactionInfo struct {
	proto.CreateAliasWithSig
	transactionInfoCommonImpl
}

func (txInfo *CreateAliasWithSigTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *CreateAliasWithSigTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.CreateAliasWithSig
}

func (txInfo *CreateAliasWithSigTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type MassTransferTransactionInfo struct {
	proto.MassTransferWithProofs
	transactionInfoCommonImpl
}

func (txInfo *MassTransferTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *MassTransferTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.MassTransferWithProofs
}

func (txInfo *MassTransferTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type DataTransactionInfo struct {
	proto.DataWithProofs
	transactionInfoCommonImpl
}

func (txInfo *DataTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *DataTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.DataWithProofs
}

func (txInfo *DataTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type SetScriptTransactionInfo struct {
	proto.SetScriptWithProofs
	transactionInfoCommonImpl
}

func (txInfo *SetScriptTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *SetScriptTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.SetScriptWithProofs
}

func (txInfo *SetScriptTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type SponsorshipTransactionInfo struct {
	proto.SponsorshipWithProofs
	transactionInfoCommonImpl
}

func (txInfo *SponsorshipTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *SponsorshipTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.SponsorshipWithProofs
}

func (txInfo *SponsorshipTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type SetAssetScriptTransactionInfo struct {
	proto.SetAssetScriptWithProofs
	transactionInfoCommonImpl
}

func (txInfo *SetAssetScriptTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *SetAssetScriptTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.SetAssetScriptWithProofs
}

func (txInfo *SetAssetScriptTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type InvokeScriptTransactionInfo struct {
	proto.InvokeScriptWithProofs
	transactionInfoCommonImpl

	StateChanges StateChanges
}

func (txInfo *InvokeScriptTransactionInfo) unmarshalSpecificData(data []byte) error {
	tmp := struct {
		Changes *StateChanges `json:"stateChanges"`
	}{
		Changes: &txInfo.StateChanges,
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return errors.Wrap(err, "Failed to unmarshal stateChanges")
	}

	return nil
}

func (txInfo *InvokeScriptTransactionInfo) GetStateChanges() *StateChanges {
	return &txInfo.StateChanges
}

func (txInfo *InvokeScriptTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *InvokeScriptTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.InvokeScriptWithProofs
}

func (txInfo *InvokeScriptTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}

type UpdateAssetInfoTransactionInfo struct {
	proto.UpdateAssetInfoWithProofs
	transactionInfoCommonImpl
}

func (txInfo *UpdateAssetInfoTransactionInfo) getInfoCommonObject() *transactionInfoCommonImpl {
	return &txInfo.transactionInfoCommonImpl
}

func (txInfo *UpdateAssetInfoTransactionInfo) getTransactionObject() proto.Transaction {
	return &txInfo.UpdateAssetInfoWithProofs
}

func (txInfo *UpdateAssetInfoTransactionInfo) UnmarshalJSON(data []byte) error {
	return transactionInfoUnmarshalJSON(data, txInfo)
}
