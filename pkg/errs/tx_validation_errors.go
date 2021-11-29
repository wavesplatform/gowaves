package errs

import "fmt"

type TooBigArray struct {
	message string
}

func NewTooBigArray(message string) *TooBigArray {
	return &TooBigArray{message: message}
}

func (a TooBigArray) Error() string {
	return a.message
}

func (a TooBigArray) Extend(message string) error {
	return NewTooBigArray(fmtExtend(a, message))
}

func (a TooBigArray) Is(target error) bool {
	_, ok := target.(TooBigArray)
	return ok
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

func (a NonPositiveAmount) Is(target error) bool {
	_, ok := target.(NonPositiveAmount)
	return ok
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

func (a InvalidName) Is(target error) bool {
	_, ok := target.(InvalidName)
	return ok
}

type AccountBalanceError struct {
	message string
}

func NewAccountBalanceError(message string) *AccountBalanceError {
	return &AccountBalanceError{message: message}
}

func (a AccountBalanceError) Error() string {
	return a.message
}

func (a AccountBalanceError) Extend(s string) error {
	return NewAccountBalanceError(fmtExtend(a, s))
}

func (a AccountBalanceError) Is(target error) bool {
	_, ok := target.(AccountBalanceError)
	return ok
}

type ToSelf struct {
	message string
}

func NewToSelf(message string) *ToSelf {
	return &ToSelf{message: message}
}

func (a ToSelf) Error() string {
	return a.message
}

func (a ToSelf) Extend(s string) error {
	return NewToSelf(fmtExtend(a, s))
}

func (a ToSelf) Is(target error) bool {
	_, ok := target.(ToSelf)
	return ok
}

// TxValidationError provides message as is, without adding additional message info.
type TxValidationError struct {
	ValidationErrorImpl
	message string
}

func NewTxValidationError(message string) *TxValidationError {
	return &TxValidationError{message: message}
}

func (a TxValidationError) Error() string {
	return a.message
}

func (a TxValidationError) Extend(s string) error {
	return NewTxValidationError(fmtExtend(a, s))
}

func (a TxValidationError) Is(target error) bool {
	_, ok := target.(TxValidationError)
	return ok
}

type AssetIsNotReissuable struct {
	ValidationErrorImpl
	message string
}

func NewAssetIsNotReissuable(message string) *AssetIsNotReissuable {
	return &AssetIsNotReissuable{message: message}
}

func (a AssetIsNotReissuable) Error() string {
	return a.message
}

func (a AssetIsNotReissuable) Extend(s string) error {
	return NewAssetIsNotReissuable(fmtExtend(a, s))
}

func (a AssetIsNotReissuable) Is(target error) bool {
	_, ok := target.(AssetIsNotReissuable)
	return ok
}

type AliasTaken struct {
	ValidationErrorImpl
	message string
}

func NewAliasTaken(message string) *AliasTaken {
	return &AliasTaken{message: message}
}

func (a AliasTaken) Error() string {
	return a.message
}

func (a AliasTaken) Extend(s string) error {
	return NewAliasTaken(fmtExtend(a, s))
}

func (a AliasTaken) Is(target error) bool {
	_, ok := target.(AliasTaken)
	return ok
}

type Mistiming struct {
	ValidationErrorImpl
	message string
}

func NewMistiming(message string) *Mistiming {
	return &Mistiming{message: message}
}

func (a Mistiming) Error() string {
	return a.message
}

func (a Mistiming) Extend(message string) error {
	return NewMistiming(fmtExtend(a, message))
}

func (a Mistiming) Is(target error) bool {
	_, ok := target.(Mistiming)
	return ok
}

type EmptyDataKey struct {
	ValidationErrorImpl
	message string
}

func NewEmptyDataKey(message string) *EmptyDataKey {
	return &EmptyDataKey{message: message}
}

func (a EmptyDataKey) Error() string {
	return a.message
}

func (a EmptyDataKey) Extend(message string) error {
	return NewEmptyDataKey(fmtExtend(a, message))
}

func (a EmptyDataKey) Is(target error) bool {
	_, ok := target.(EmptyDataKey)
	return ok
}

type DuplicatedDataKeys struct {
	message string
}

func NewDuplicatedDataKeys(message string) *DuplicatedDataKeys {
	return &DuplicatedDataKeys{message: message}
}

func (a DuplicatedDataKeys) Error() string {
	return a.message
}

func (a DuplicatedDataKeys) Extend(message string) error {
	return NewDuplicatedDataKeys(fmtExtend(a, message))
}

func (a DuplicatedDataKeys) Is(target error) bool {
	_, ok := target.(DuplicatedDataKeys)
	return ok
}

type UnknownAsset struct {
	message string
}

func NewUnknownAsset(message string) *UnknownAsset {
	return &UnknownAsset{message: message}
}

func (a UnknownAsset) Error() string {
	return a.message
}

func (a UnknownAsset) Extend(message string) error {
	return NewUnknownAsset(fmtExtend(a, message))
}

func (a UnknownAsset) Is(target error) bool {
	_, ok := target.(UnknownAsset)
	return ok
}

type AssetIssuedByOtherAddress struct {
	message string
}

func NewAssetIssuedByOtherAddress(message string) *AssetIssuedByOtherAddress {
	return &AssetIssuedByOtherAddress{
		message: message,
	}
}

func (a AssetIssuedByOtherAddress) Error() string {
	return a.message
}

func (a AssetIssuedByOtherAddress) Extend(message string) error {
	return NewAssetIssuedByOtherAddress(fmtExtend(a, message))
}

func (a AssetIssuedByOtherAddress) Is(target error) bool {
	_, ok := target.(AssetIssuedByOtherAddress)
	return ok
}

type FeeValidation struct {
	message string
}

func NewFeeValidation(message string) *FeeValidation {
	return &FeeValidation{
		message: message,
	}
}

func (a FeeValidation) Error() string {
	return a.message
}

func (a FeeValidation) Extend(message string) error {
	return NewFeeValidation(fmtExtend(a, message))
}

func (a FeeValidation) Is(target error) bool {
	_, ok := target.(FeeValidation)
	return ok
}

type AssetUpdateInterval struct {
	message string
}

func NewAssetUpdateInterval(message string) *AssetUpdateInterval {
	return &AssetUpdateInterval{
		message: message,
	}
}

func (a AssetUpdateInterval) Error() string {
	return a.message
}

func (a AssetUpdateInterval) Extend(message string) error {
	return NewAssetUpdateInterval(fmtExtend(a, message))
}

func (a AssetUpdateInterval) Is(target error) bool {
	_, ok := target.(AssetUpdateInterval)
	return ok
}

type TransactionNotAllowedByScript struct {
	message string
	asset   []byte
}

func NewTransactionNotAllowedByScript(message string, asset []byte) *TransactionNotAllowedByScript {
	return &TransactionNotAllowedByScript{
		message: message,
		asset:   asset,
	}
}

func (a TransactionNotAllowedByScript) Asset() []byte {
	return a.asset
}

func (a TransactionNotAllowedByScript) IsAssetScript() bool {
	return len(a.asset) > 0
}

func (a TransactionNotAllowedByScript) Error() string {
	return a.message
}

func (a TransactionNotAllowedByScript) Extend(message string) error {
	return NewTransactionNotAllowedByScript(fmtExtend(a, message), a.asset)
}

func (a TransactionNotAllowedByScript) Is(target error) bool {
	_, ok := target.(TransactionNotAllowedByScript)
	return ok
}
