package crypto_test

import (
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wavesplatform/gowaves/pkg/crypto"
)

func TestCertificateLoadOK(t *testing.T) {
	certificates, revocations := loadCertificatesAndRevocations(t, "testdata/tdx-cert-chain.pem")
	require.Len(t, certificates, 3)
	require.Len(t, revocations, 2)

	ts, err := time.Parse(time.DateTime, "2026-01-28 10:00:00")
	require.NoError(t, err)
	cert, err := crypto.LoadCertificate(certificates, revocations, ts)
	require.NoError(t, err)
	assert.Equal(t, "1172209109816287419860867609178890002503945525584", cert.SerialNumber.String())
}

func TestCertificateLoadFail(t *testing.T) {
	certificates, revocations := loadCertificatesAndRevocations(t, "testdata/tdx-cert-chain-revoked.pem")
	require.Len(t, certificates, 3)
	require.Len(t, revocations, 2)

	ts, err := time.Parse(time.DateTime, "2026-01-28 10:00:00")
	require.NoError(t, err)
	_, err = crypto.LoadCertificate(certificates, revocations, ts)
	require.ErrorContains(t, err, "certificate 617541246249385134247112490868883435491767640079 is revoked")
}

func BenchmarkLoadCertificateOK(b *testing.B) {
	cs, rs := loadCertificatesAndRevocations(b, "testdata/tdx-cert-chain.pem")
	ts, err := time.Parse(time.DateTime, "2026-01-28 10:00:00")
	require.NoError(b, err)
	for b.Loop() {
		cert, ldErr := crypto.LoadCertificate(cs, rs, ts)
		require.NoError(b, ldErr)
		assert.Equal(b, "1172209109816287419860867609178890002503945525584", cert.SerialNumber.String())
	}
}

func BenchmarkLoadCertificateFail(b *testing.B) {
	cs, rs := loadCertificatesAndRevocations(b, "testdata/tdx-cert-chain-revoked.pem")
	ts, err := time.Parse(time.DateTime, "2026-01-28 10:00:00")
	require.NoError(b, err)
	for b.Loop() {
		_, err = crypto.LoadCertificate(cs, rs, ts)
		require.ErrorContains(b, err, "certificate 617541246249385134247112490868883435491767640079 is revoked")
	}
}

func loadCertificatesAndRevocations(
	t testing.TB, certificatesFilename string,
) ([]*x509.Certificate, []*x509.RevocationList) {
	cf, err := os.Open(certificatesFilename)
	require.NoError(t, err)
	defer func() {
		_ = cf.Close()
	}()
	certData := readPEMCertificates(t, cf)
	certificates, err := crypto.ParseCertificates(certData)
	require.NoError(t, err)

	rf, err := os.Open("testdata/tdx-crl.pem")
	require.NoError(t, err)
	defer func() {
		_ = rf.Close()
	}()
	revocationData := readPEMRevocationLists(t, rf)
	revocations, err := crypto.ParseRevocationLists(revocationData)
	require.NoError(t, err)
	return certificates, revocations
}

func readPEMCertificates(t testing.TB, r io.Reader) [][]byte {
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	res := make([][]byte, 0)
	for len(data) > 0 {
		var b *pem.Block
		b, data = pem.Decode(data)
		require.NotEmpty(t, b)
		require.Equal(t, "CERTIFICATE", b.Type)
		res = append(res, b.Bytes)
	}
	return res
}

func readPEMRevocationLists(t testing.TB, r io.Reader) [][]byte {
	res := make([][]byte, 0)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	for len(data) > 0 {
		var b *pem.Block
		b, data = pem.Decode(data)
		require.NotEmpty(t, b)
		require.Equal(t, "X509 CRL", b.Type)
		res = append(res, b.Bytes)
	}
	return res
}
