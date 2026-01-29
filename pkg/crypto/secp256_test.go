package crypto_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestVerifyECDSASignature(t *testing.T) {
	cs, rs := loadCertificatesAndRevocations(t, "testdata/tdx-cert-chain.pem")
	require.Len(t, cs, 3)
	require.Len(t, rs, 2)
	ts, err := time.Parse(time.DateTime, "2026-01-28 10:00:00")
	require.NoError(t, err)
	cert, err := crypto.LoadCertificate(cs, rs, ts)
	require.NoError(t, err)
	d, err := hex.DecodeString("6f2571102142872ec27e322e880746a97eb6e5c44aea7a64383d4b52da83e189")
	require.NoError(t, err)
	sig, err := hex.DecodeString("f7472dba5128d911617ca30b2e04fd5879f1f939e6cad38258d48dc045ac5538" +
		"e4121344314d25c8eb4fd971127704c5500951270af22245a3619479dc7e05c9")
	require.NoError(t, err)
	ok, err := crypto.VerifyECDSASignature(cert, d, sig)
	assert.NoError(t, err)
	assert.True(t, ok)
}
