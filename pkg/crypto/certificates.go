package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
	"fmt"
	"time"
)

func ParseCertificates(data [][]byte) ([]*x509.Certificate, error) {
	r := make([]*x509.Certificate, 0, len(data))
	for _, d := range data {
		c, err := x509.ParseCertificate(d)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificates: %w", err)
		}
		r = append(r, c)
	}
	return r, nil
}

func ParseRevocationLists(data [][]byte) ([]*x509.RevocationList, error) {
	r := make([]*x509.RevocationList, 0, len(data))
	for _, d := range data {
		rl, err := x509.ParseRevocationList(d)
		if err != nil {
			return nil, fmt.Errorf("failed to parse revocation lists: %w", err)
		}
		r = append(r, rl)
	}
	return r, nil
}

func VerifyRevocationList(revocationList *x509.RevocationList, certificates []*x509.Certificate, ts time.Time) error {
	if revocationList == nil {
		return errors.New("revocation list is empty")
	}
	// Check CRL freshness.
	if revocationList.ThisUpdate.After(ts) {
		return errors.New("inactive CRL")
	}
	if !revocationList.NextUpdate.IsZero() && revocationList.NextUpdate.Before(ts) {
		return errors.New("stale CRL")
	}
	// Find signer certificate and verify CRL signature.
	var lastErr error
	for _, cert := range certificates {
		if cert == nil || !cert.IsCA {
			continue
		}
		if (cert.KeyUsage & x509.KeyUsageCRLSign) == 0 {
			continue
		}
		if len(cert.SubjectKeyId) > 0 && len(revocationList.AuthorityKeyId) > 0 &&
			!bytes.Equal(cert.SubjectKeyId, revocationList.AuthorityKeyId) {
			continue
		}
		if err := revocationList.CheckSignatureFrom(cert); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("unverified CRL: %w", lastErr)
	}
	return errors.New("unverified CRL")
}

func IsRevoked(cert *x509.Certificate, revocationLists []*x509.RevocationList) bool {
	for _, rl := range revocationLists {
		for _, re := range rl.RevokedCertificateEntries {
			if re.SerialNumber != nil && re.SerialNumber.Cmp(cert.SerialNumber) == 0 {
				return true
			}
		}
	}
	return false
}

func LoadCertificate(
	certificates []*x509.Certificate, revocations []*x509.RevocationList, ts time.Time,
) (*x509.Certificate, error) {
	if len(certificates) == 0 || len(revocations) == 0 {
		return nil, errors.New("no certificates or revocation lists provided")
	}
	if len(revocations) != len(certificates)-1 {
		return nil, errors.New("number of revocation lists must be one less than number of certificates")
	}
	// Verify revocation lists.
	for _, rl := range revocations {
		if err := VerifyRevocationList(rl, certificates, ts); err != nil {
			return nil, fmt.Errorf("failed to verify revocation list: %w", err)
		}
	}
	// Verify certificate chain.
	for _, cert := range certificates {
		if IsRevoked(cert, revocations) {
			return nil, fmt.Errorf("certificate %d is revoked", cert.SerialNumber)
		}
	}
	leaf := certificates[0]                   // Leaf is the first in the chain.
	root := certificates[len(certificates)-1] // Root is the last in the chain.
	roots := x509.NewCertPool()
	roots.AddCert(root)
	intermediates := x509.NewCertPool()
	for _, c := range certificates {
		if c.Equal(root) || c.Equal(leaf) {
			continue
		}
		if c.IsCA {
			intermediates.AddCert(c)
		}
	}
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   ts,
	}
	if _, err := leaf.Verify(opts); err != nil {
		return nil, fmt.Errorf("failed to verify certificate chain: %w", err)
	}
	if !hasExtension(leaf) {
		return nil, errors.New("certificate is missing SGX Extensions")
	}
	return leaf, nil
}

func hasExtension(cert *x509.Certificate) bool {
	const extensionOID = "1.2.840.113741.1.13.1" // SGX extension OID
	for _, ext := range cert.Extensions {
		if ext.Id.String() == extensionOID {
			return true
		}
	}
	return false
}

// CertificatePublicKeyToBytes extracts the public key from the provided X.509 certificate.
// Certificate must contain an ECDSA public key on the NIST P-256 curve.
func CertificatePublicKeyToBytes(cert *x509.Certificate) ([]byte, error) {
	switch pk := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		res := make([]byte, P256RawPubKeySize)
		pk.X.FillBytes(res[:32])
		pk.Y.FillBytes(res[32:])
		return res, nil
	default:
		return nil, fmt.Errorf("unsupported public key type %T", pk)
	}
}
