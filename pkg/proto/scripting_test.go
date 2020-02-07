package proto

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScriptResultV3BinaryRoundTrip(t *testing.T) {
	waves, err := NewOptionalAssetFromString("WAVES")
	assert.NoError(t, err)
	asset0, err := NewOptionalAssetFromString("Ft8X1v1LTa1ABafufpaCWyVj8KkaxUWE6xBhW6sNFJck")
	assert.NoError(t, err)
	asset1, err := NewOptionalAssetFromString("Ft5X1v1LTa1ABafufpaCWyVj7KkaxUWE6xBhW6sNFJck")
	assert.NoError(t, err)
	addr0, err := NewAddressFromString("3PQ8bp1aoqHQo3icNqFv6VM36V1jzPeaG1v")
	assert.NoError(t, err)
	rcp := NewRecipientFromAddress(addr0)
	tests := []ScriptResultV3{
		{
			Writes: []DataEntry{
				&IntegerDataEntry{"some key", 12345},
				&BooleanDataEntry{"negative value", false},
				&StringDataEntry{"some key", "some value string"},
				&BinaryDataEntry{Key: "k3", Value: []byte{0x24, 0x7f, 0x71, 0x14, 0x1d}},
				&IntegerDataEntry{"some key2", -12345},
				&BooleanDataEntry{"negative value2", true},
				&StringDataEntry{"some key143", "some value2 string"},
				&BinaryDataEntry{Key: "k5", Value: []byte{0x24, 0x7f, 0x71, 0x10, 0x1d}},
			},
			Transfers: []ScriptResultTransfer{
				{Amount: math.MaxInt64, Asset: *waves, Recipient: rcp},
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: -10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
		},
		{
			Writes: []DataEntry{
				&IntegerDataEntry{"some key", 12345},
			},
		},
		{
			Transfers: []ScriptResultTransfer{
				{Amount: 100500, Asset: *waves, Recipient: rcp},
				{Amount: -10, Asset: *asset0, Recipient: rcp},
				{Amount: 0, Asset: *asset1, Recipient: rcp},
			},
		},
	}
	for _, tc := range tests {
		if b, err := tc.MarshalWithAddresses(); assert.NoError(t, err) {
			sr := ScriptResultV3{}
			if err := sr.UnmarshalWithAddresses(b); assert.NoError(t, err) {
				assert.Equal(t, tc, sr)
			}
		}
	}
	// Should not work with alias recipients.
	alias, err := NewAliasFromString("alias:T:blah-blah-blah")
	assert.NoError(t, err)
	sr := tests[0]
	sr.Transfers[0].Recipient = NewRecipientFromAlias(*alias)
	_, err = sr.MarshalWithAddresses()
	assert.Error(t, err)
}
