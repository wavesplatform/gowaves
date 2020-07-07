package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURL(t *testing.T) {
	for _, test := range []struct {
		db  string
		url string
		err string
	}{
		{"test-db", "http://user:password@localhost:8086/test-db", ""},
		{"test-db", "http://localhost:8086/test-db", ""},
		{"test-db", "http://localhost:8086/test-db", ""},
		{"test-db", "http://localhost:1234567890/test-db", "invalid port number 1234567890"},
		{"test-db", "http://localhost:8086", "empty database"},
		{"test-db", "http://localhost:8086/", "empty database"},
		{"db", "http://localhost/db", ""},
	} {
		cfg, s, err := parseURL(test.url)
		if test.err != "" {
			assert.EqualError(t, err, test.err)
		} else {
			require.NoError(t, err)
			assert.NotNil(t, cfg)
			assert.Equal(t, test.db, s)
		}
	}
}
