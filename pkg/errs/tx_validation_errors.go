package errs

import "fmt"

type TooBigArray struct {
	message string
}

func (a TooBigArray) Extend(message string) error {
	return NewTooBigArray(fmtExtend(a, message))
}

func NewTooBigArray(message string) *TooBigArray {
	return &TooBigArray{message: message}
}

func (a TooBigArray) Error() string {
	return a.message
}

type NonPositiveAmount struct {
	amount int64
	of     string
}

func NewNonPositiveAmount(amount int64, of string) *NonPositiveAmount {
	return &NonPositiveAmount{
		amount: amount,
		of:     of,
	}
}

func (a NonPositiveAmount) Error() string {
	return fmt.Sprintf("%d of %s", a.amount, a.of)
}

type InvalidName struct {
	message string
}

func NewInvalidName(message string) *InvalidName {
	return &InvalidName{message: message}
}

func (a InvalidName) Error() string {
	return a.message
}

func (a InvalidName) Extend(message string) error {
	return NewInvalidName(fmtExtend(a, message))
}

type AccountBalanceError struct {
	message string
}

func (a AccountBalanceError) Error() string {
	return a.message
}

func (a AccountBalanceError) Extend(s string) error {
	return NewAccountBalanceError(fmtExtend(a, s))
}

func NewAccountBalanceError(message string) *AccountBalanceError {
	return &AccountBalanceError{message: message}
}

type ToSelf struct {
	message string
}

func (a ToSelf) Error() string {
	return a.message
}

func (a ToSelf) Extend(s string) error {
	return NewToSelf(fmtExtend(a, s))
}

func NewToSelf(message string) *ToSelf {
	return &ToSelf{message: message}
}

// This struct provides message as is, without adding additional message info.
type TxValidationError struct {
	message string
}

func (a TxValidationError) Error() string {
	return a.message
}

func (a TxValidationError) Extend(s string) error {
	return NewTxValidationError(fmtExtend(a, s))
}

func NewTxValidationError(message string) *TxValidationError {
	return &TxValidationError{message: message}
}

type AssetIsNotReissuable struct {
	message string
}

func (a AssetIsNotReissuable) Error() string {
	return a.message
}

func (a AssetIsNotReissuable) Extend(s string) error {
	return NewAssetIsNotReissuable(fmtExtend(a, s))
}

func NewAssetIsNotReissuable(message string) *AssetIsNotReissuable {
	return &AssetIsNotReissuable{message: message}
}

type AliasTaken struct {
	message string
}

func (a AliasTaken) Error() string {
	return a.message
}

func (a AliasTaken) Extend(s string) error {
	return NewAliasTaken(fmtExtend(a, s))
}

func NewAliasTaken(message string) *AliasTaken {
	return &AliasTaken{message: message}
}

type Mistiming struct {
	message string
}

func (a Mistiming) Extend(message string) error {
	return NewMistiming(fmtExtend(a, message))
}

func (a Mistiming) Error() string {
	return a.message
}

func NewMistiming(message string) *Mistiming {
	return &Mistiming{message: message}
}

type EmptyDataKey struct {
	message string
}

func (a EmptyDataKey) Extend(message string) error {
	return NewEmptyDataKey(fmtExtend(a, message))
}

func (a EmptyDataKey) Error() string {
	return a.message
}

func NewEmptyDataKey(message string) *EmptyDataKey {
	return &EmptyDataKey{message: message}
}

type DuplicatedDataKeys struct {
	message string
}

func (a DuplicatedDataKeys) Extend(message string) error {
	return NewDuplicatedDataKeys(fmtExtend(a, message))
}

func (a DuplicatedDataKeys) Error() string {
	return a.message
}

func NewDuplicatedDataKeys(message string) *DuplicatedDataKeys {
	return &DuplicatedDataKeys{message: message}
}

type UnknownAsset struct {
	message string
}

func (a UnknownAsset) Error() string {
	return a.message
}

func (a UnknownAsset) Extend(message string) error {
	return NewUnknownAsset(fmtExtend(a, message))
}

func NewUnknownAsset(message string) *UnknownAsset {
	return &UnknownAsset{message: message}
}
