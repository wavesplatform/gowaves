package settings

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchainSettings(t *testing.T) {
	doTest := func(fileName string, bt BlockchainType) func(t *testing.T) {
		return func(t *testing.T) {
			expected, err := os.ReadFile(fileName)
			require.NoError(t, err)

			s := mustLoadEmbeddedSettings(bt) // intentionally checking must function
			actual, err := json.Marshal(s)
			require.NoError(t, err)

			assert.JSONEq(t, string(expected), string(actual))
		}
	}
	t.Run("stagenet", doTest(stagenetFile, StageNet))
	t.Run("testnet", doTest(testnetFile, TestNet))
	t.Run("mainnet", doTest(mainnetFile, MainNet))
}

func TestRewardAddressesOfEmbeddedSettings(t *testing.T) {
	for _, test := range []struct {
		blockchain BlockchainType
		dao        string
		xtn        string
	}{
		{MainNet, "3PEgG7eZHLFhcfsTSaYxgRhZsh4AxMvA4Ms", "3PFjHWuH6WXNJbwnfLHqNFBpwBS5dkYjTfv"},
		{TestNet, "3Myb6G8DkdBb8YcZzhrky65HrmiNuac3kvS", "3N13KQpdY3UU7JkWUBD9kN7t7xuUgeyYMTT"},
		{StageNet, "3MaFVH1vTv18FjBRugSRebx259D7xtRh9ic", "3MbhiRiLFLJ1EVKNP9npRszcLLQDjwnFfZM"},
	} {
		t.Run(test.dao, func(t *testing.T) {
			s := mustLoadEmbeddedSettings(test.blockchain)
			require.NotNil(t, s.DAOAddress)
			assert.Equal(t, test.dao, s.DAOAddress.String())
			require.NotNil(t, s.XTNBuybackAddress)
			assert.Equal(t, test.xtn, s.XTNBuybackAddress.String())
		})
	}
}

func TestDeprecatedRewardAddressesMigration(t *testing.T) {
	const (
		dao = "3PEgG7eZHLFhcfsTSaYxgRhZsh4AxMvA4Ms"
		xtn = "3PFjHWuH6WXNJbwnfLHqNFBpwBS5dkYjTfv"
	)
	settingsJSON := func(body string) string {
		return `{"block_reward_term": 100000, "block_reward_voting_period": 10000, ` +
			`"block_reward_term_after_20": 50000` + body + `}`
	}
	for _, test := range []struct {
		name string
		body string
		dao  string
		xtn  string
		err  string
	}{
		{
			name: "both addresses",
			body: `, "reward_addresses": ["` + dao + `", "` + xtn + `"], "reward_addresses_after_21": ["` + dao + `"]`,
			dao:  dao, xtn: xtn,
		},
		{
			name: "DAO address only",
			body: `, "reward_addresses": ["` + dao + `"], "reward_addresses_after_21": ["` + dao + `"]`,
			dao:  dao,
		},
		{
			name: "no addresses at all",
			body: `, "reward_addresses": [], "reward_addresses_after_21": []`,
		},
		{
			name: "ambiguous single address",
			body: `, "reward_addresses": ["` + xtn + `"], "reward_addresses_after_21": []`,
			err:  "use explicit 'dao_address' and 'xtn_buyback_address' settings instead",
		},
		{
			name: "ambiguous pair of addresses",
			body: `, "reward_addresses": ["` + dao + `", "` + xtn + `"], "reward_addresses_after_21": []`,
			err:  "use explicit 'dao_address' and 'xtn_buyback_address' settings instead",
		},
		{
			name: "deprecated and new settings combined",
			body: `, "reward_addresses": ["` + dao + `"], "reward_addresses_after_21": ["` + dao + `"], ` +
				`"dao_address": "` + dao + `"`,
			err: "cannot be combined with 'dao_address' or 'xtn_buyback_address' settings",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var s BlockchainSettings
			err := json.Unmarshal([]byte(settingsJSON(test.body)), &s)
			if test.err != "" {
				assert.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)

			if test.dao == "" {
				assert.Nil(t, s.DAOAddress)
			} else {
				require.NotNil(t, s.DAOAddress)
				assert.Equal(t, test.dao, s.DAOAddress.String())
			}
			if test.xtn == "" {
				assert.Nil(t, s.XTNBuybackAddress)
			} else {
				require.NotNil(t, s.XTNBuybackAddress)
				assert.Equal(t, test.xtn, s.XTNBuybackAddress.String())
			}
		})
	}
}

func TestMinXTNBuyBackPeriodDefault(t *testing.T) {
	const body = `{"block_reward_term": 100000, "block_reward_voting_period": 10000, ` +
		`"block_reward_term_after_20": 50000}`

	// An omitted period must not cease the XTN buy-back, the same way the Scala node does it.
	s, err := ReadBlockchainSettings(strings.NewReader(body))
	require.NoError(t, err)
	assert.Equal(t, uint64(defaultMinXTNBuyBackPeriod), s.MinXTNBuyBackPeriod)

	assert.Equal(t, uint64(defaultMinXTNBuyBackPeriod), MustDefaultCustomSettings().MinXTNBuyBackPeriod)
}
