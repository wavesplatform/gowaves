package errs

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNonPositiveAmount(t *testing.T) {
	rs := NewNonPositiveAmount(1, "some")
	require.Equal(t, "1 of some", rs.Error())
}

func TestEmptyDataKey(t *testing.T) {
	require.EqualError(t, Extend(NewEmptyDataKey("a"), "b"), "b: a")
}

func TestDuplicatedDataKeys(t *testing.T) {
	require.EqualError(t, Extend(NewDuplicatedDataKeys("a"), "b"), "b: a")
}

func TestMistiming(t *testing.T) {
	require.EqualError(t, Extend(NewMistiming("a"), "b"), "b: a")
}

func TestUnknownAsset(t *testing.T) {
	require.EqualError(t, NewUnknownAsset("a").Extend("b"), "b: a")
}

func TestNewTooBigArray(t *testing.T) {
	require.EqualError(t, NewTooBigArray("a").Extend("b"), "b: a")
}

func TestNewInvalidName(t *testing.T) {
	require.EqualError(t, NewInvalidName("a").Extend("b"), "b: a")
}

func TestNewAccountBalanceError(t *testing.T) {
	require.EqualError(t, NewAccountBalanceError("a").Extend("b"), "b: a")
}

func TestNewToSelf(t *testing.T) {
	require.EqualError(t, NewToSelf("a").Extend("b"), "b: a")
}

func TestNewAliasTaken(t *testing.T) {
	require.EqualError(t, NewAliasTaken("a").Extend("b"), "b: a")
}

func TestNewAssetIsNotReissuable(t *testing.T) {
	require.EqualError(t, NewAssetIsNotReissuable("a").Extend("b"), "b: a")
}

func TestNewTxValidationError(t *testing.T) {
	require.EqualError(t, NewTxValidationError("a").Extend("b"), "b: a")
	require.True(t, IsValidationError(NewTxValidationError("")))
}

func TestNewAssetIssuedByOtherAddress(t *testing.T) {
	require.EqualError(t, NewAssetIssuedByOtherAddress("a").Extend("b"), "b: a")
}

func TestFeeValidation(t *testing.T) {
	require.EqualError(t, NewFeeValidation("a").Extend("b"), "b: a")
}

func TestNewAssetUpdateInterval(t *testing.T) {
	require.EqualError(t, NewAssetUpdateInterval("a").Extend("b"), "b: a")
}

func TestNewTransactionNotAllowedByScript(t *testing.T) {
	err := NewTransactionNotAllowedByScript("a", nil)
	require.EqualError(t, err.Extend("b"), "b: a")
	require.False(t, err.IsAssetScript())
	require.Len(t, err.Asset(), 0)
}

func TestErrorIsCompatibility(t *testing.T) {
	assert.True(t, errors.Is(NewTooBigArray("test"), TooBigArray{}))
	assert.False(t, errors.Is(NewTooBigArray("test"), NonPositiveAmount{}))

	assert.True(t, errors.Is(NewNonPositiveAmount(0, "test"), NonPositiveAmount{}))
	assert.False(t, errors.Is(NewNonPositiveAmount(0, "test"), InvalidName{}))

	assert.True(t, errors.Is(NewInvalidName("test"), InvalidName{}))
	assert.False(t, errors.Is(NewInvalidName("test"), AccountBalanceError{}))

	assert.True(t, errors.Is(NewAccountBalanceError("test"), AccountBalanceError{}))
	assert.False(t, errors.Is(NewAccountBalanceError("test"), ToSelf{}))

	assert.True(t, errors.Is(NewToSelf("test"), ToSelf{}))
	assert.False(t, errors.Is(NewToSelf("test"), TxValidationError{}))

	assert.True(t, errors.Is(NewTxValidationError("test"), TxValidationError{}))
	assert.False(t, errors.Is(NewTxValidationError("test"), AssetIsNotReissuable{}))

	assert.True(t, errors.Is(NewAssetIsNotReissuable("test"), AssetIsNotReissuable{}))
	assert.False(t, errors.Is(NewAssetIsNotReissuable("test"), AliasTaken{}))

	assert.True(t, errors.Is(NewAliasTaken("test"), AliasTaken{}))
	assert.False(t, errors.Is(NewAliasTaken("test"), Mistiming{}))

	assert.True(t, errors.Is(NewMistiming("test"), Mistiming{}))
	assert.False(t, errors.Is(NewMistiming("test"), EmptyDataKey{}))

	assert.True(t, errors.Is(NewEmptyDataKey("test"), EmptyDataKey{}))
	assert.False(t, errors.Is(NewEmptyDataKey("test"), DuplicatedDataKeys{}))

	assert.True(t, errors.Is(NewDuplicatedDataKeys("test"), DuplicatedDataKeys{}))
	assert.False(t, errors.Is(NewDuplicatedDataKeys("test"), UnknownAsset{}))

	assert.True(t, errors.Is(NewUnknownAsset("test"), UnknownAsset{}))
	assert.False(t, errors.Is(NewUnknownAsset("test"), AssetIssuedByOtherAddress{}))

	assert.True(t, errors.Is(NewAssetIssuedByOtherAddress("test"), AssetIssuedByOtherAddress{}))
	assert.False(t, errors.Is(NewAssetIssuedByOtherAddress("test"), FeeValidation{}))

	assert.True(t, errors.Is(NewFeeValidation("test"), FeeValidation{}))
	assert.False(t, errors.Is(NewFeeValidation("test"), AssetUpdateInterval{}))

	assert.True(t, errors.Is(NewAssetUpdateInterval("test"), AssetUpdateInterval{}))
	assert.False(t, errors.Is(NewAssetUpdateInterval("test"), TransactionNotAllowedByScript{}))

	assert.True(t, errors.Is(NewTransactionNotAllowedByScript("test", nil), TransactionNotAllowedByScript{}))
	assert.False(t, errors.Is(NewTransactionNotAllowedByScript("test", nil), TooBigArray{}))
}
