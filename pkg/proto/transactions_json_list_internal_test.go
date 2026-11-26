package proto

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oldTransactionsUnmarshalJSON is a verbatim copy of the double-parse Transactions.UnmarshalJSON that was replaced by
// UnmarshalTransactionsListJSON. It is kept as the oracle for decode equality.
func oldTransactionsUnmarshalJSON(data []byte) (Transactions, error) {
	var tt []*TransactionTypeVersion
	err := json.Unmarshal(data, &tt)
	if err != nil {
		return nil, errors.Wrap(err, "TransactionType unmarshal")
	}
	transactions := make([]Transaction, len(tt))
	for i, row := range tt {
		realType, gErr := GuessTransactionType(row)
		if gErr != nil {
			return nil, gErr
		}
		transactions[i] = realType
	}
	err = json.Unmarshal(data, &transactions)
	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func jsonListFixtures(t testing.TB) map[string][]byte {
	kp := benchKeyPair(t, "json-list-sender")
	var txs []Transaction
	for _, c := range simpleTxCases(t) {
		txs = append(txs, c.tx)
	}
	for _, v := range []byte{1, 2, 3} {
		txs = append(txs, benchTransfer(t, v, kp))
	}
	for _, v := range []byte{1, 2} {
		txs = append(txs, benchData(t, v, kp), benchInvoke(t, v, kp))
	}
	for _, v := range []byte{1, 2, 3} {
		txs = append(txs, benchExchange(t, v, kp))
	}
	mixed := make([]json.RawMessage, 0, len(txs))
	for _, tx := range txs {
		js, err := json.Marshal(tx)
		if err != nil {
			continue // Some synthetic vectors are not representable in JSON, skip them.
		}
		mixed = append(mixed, js)
	}
	require.NotEmpty(t, mixed)
	mixedJS, err := json.Marshal(mixed)
	require.NoError(t, err)

	fixtures := map[string][]byte{
		"mixed":  mixedJS,
		"empty":  []byte("[]"),
		"null":   []byte("null"),
		"spaces": []byte(" [ " + string(mixed[0]) + " ,\n\t" + string(mixed[len(mixed)-1]) + " ] "),
	}
	for i, bt := range blockTests {
		var b struct {
			Transactions json.RawMessage `json:"transactions"`
		}
		require.NoError(t, json.Unmarshal([]byte(bt.jsonEncoded), &b))
		if len(b.Transactions) > 0 {
			fixtures[fmt.Sprintf("block-%d", i)] = b.Transactions
		}
	}
	return fixtures
}

func TestUnmarshalTransactionsListJSONEqualsOld(t *testing.T) {
	for name, data := range jsonListFixtures(t) {
		t.Run(name, func(t *testing.T) {
			want, err := oldTransactionsUnmarshalJSON(data)
			require.NoError(t, err)
			var got Transactions
			require.NoError(t, json.Unmarshal(data, &got))
			assert.Equal(t, want, got)
			assert.Equal(t, want == nil, got == nil)
			wantJS, err := json.Marshal(want)
			require.NoError(t, err)
			gotJS, err := json.Marshal(got)
			require.NoError(t, err)
			assert.Equal(t, wantJS, gotJS)
		})
	}
}

func TestUnmarshalTransactionsListJSONErrors(t *testing.T) {
	transfer, err := json.Marshal(benchTransfer(t, 2, benchKeyPair(t, "json-list-sender")))
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		data string
		err  string
	}{
		{"unknown type", `[{"type":200,"version":1}]`, "unknown transaction type"},
		{"zero type", `[{"version":1}]`, "unknown transaction type"},
		{"bad version type", `[{"type":4,"version":"x"}]`, "TransactionType unmarshal"},
		{"version overflow", `[{"type":4,"version":256}]`, "TransactionType unmarshal"},
		{"bad type type", `[{"type":"transfer"}]`, "TransactionType unmarshal"},
		{"not an object", `[1]`, "TransactionType unmarshal"},
		{"null element", `[null]`, "null transaction"},
		{"not an array", `{"type":4}`, "TransactionType unmarshal"},
		{"malformed", `[{"type":4`, "unexpected end of JSON input"},
		{"bad body", `[{"type":4,"version":2,"amount":"x"}]`, "TransactionType unmarshal"},
		{"second element", `[` + string(transfer) + `,{"type":200}]`, "unknown transaction type"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got Transactions
			uErr := json.Unmarshal([]byte(tc.data), &got)
			require.Error(t, uErr)
			assert.ErrorContains(t, uErr, tc.err)
			assert.Nil(t, got)
		})
	}
}

func BenchmarkTransactionsUnmarshalJSON(b *testing.B) {
	data := jsonListFixtures(b)["mixed"]
	b.Run("old", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := oldTransactionsUnmarshalJSON(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("new", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := UnmarshalTransactionsListJSON(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
