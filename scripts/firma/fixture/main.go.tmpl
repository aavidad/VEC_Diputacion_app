// Genera material desechable para ensayar el validador local de AutofirmaV2.
// Se copia al árbol temporal de AutofirmaV2 para usar sus tipos internos.
package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"autofirma.v2/internal/testsupport/pdffixture"
	pdfsign "github.com/digitorus/pdfsign/sign"
)

func fail(err error) {
	if err != nil {
		panic(err)
	}
}
func write(path string, data []byte) { fail(os.WriteFile(path, data, 0600)) }
func cert(t *x509.Certificate, issuer *x509.Certificate, key crypto.Signer, issuerKey crypto.Signer) *x509.Certificate {
	if issuer == nil {
		issuer, issuerKey = t, key
	}
	der, err := x509.CreateCertificate(rand.Reader, t, issuer, key.Public(), issuerKey)
	fail(err)
	c, err := x509.ParseCertificate(der)
	fail(err)
	return c
}
func crl(c *x509.Certificate, key crypto.Signer, n int64, revoked *big.Int) []byte {
	var entries []x509.RevocationListEntry
	if revoked != nil {
		entries = []x509.RevocationListEntry{{SerialNumber: revoked, RevocationTime: time.Now().Add(-30 * time.Minute)}}
	}
	der, err := x509.CreateRevocationList(rand.Reader, &x509.RevocationList{
		Number: big.NewInt(n), ThisUpdate: time.Now().Add(-time.Hour), NextUpdate: time.Now().Add(6 * time.Hour),
		RevokedCertificateEntries: entries,
	}, c, key)
	fail(err)
	return der
}
func main() {
	if len(os.Args) != 2 {
		panic("uso: fixture DIRECTORIO")
	}
	dir := os.Args[1]
	fail(os.MkdirAll(filepath.Join(dir, "crl"), 0700))
	now := time.Now()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	fail(err)
	root := cert(&x509.Certificate{
		SerialNumber: big.NewInt(101), Subject: pkix.Name{CommonName: "VEC E3 CA raiz sintetica"},
		NotBefore: now.Add(-2 * time.Hour), NotAfter: now.Add(48 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}, nil, rootKey, nil)
	middleKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	fail(err)
	middle := cert(&x509.Certificate{
		SerialNumber: big.NewInt(102), Subject: pkix.Name{CommonName: "VEC E3 CA intermedia sintetica"},
		NotBefore: now.Add(-2 * time.Hour), NotAfter: now.Add(48 * time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}, root, middleKey, rootKey)
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	fail(err)
	leaf := cert(&x509.Certificate{
		SerialNumber: big.NewInt(103), Subject: pkix.Name{CommonName: "VEC E3 firmante sintetico"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageContentCommitment,
		CRLDistributionPoints: []string{"http://crl.invalid/intermedia.crl"},
	}, middle, leafKey, middleKey)
	write(filepath.Join(dir, "ancla.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: root.Raw}))
	write(filepath.Join(dir, "crl", "raiz.crl"), crl(root, rootKey, 201, nil))
	write(filepath.Join(dir, "crl", "intermedia.crl"), crl(middle, middleKey, 202, nil))
	write(filepath.Join(dir, "intermedia_revocada.crl"), crl(middle, middleKey, 203, leaf.SerialNumber))
	original := filepath.Join(dir, "original.pdf")
	firmado := filepath.Join(dir, "firmado.pdf")
	write(original, pdffixture.Minimal())
	err = pdfsign.SignFile(original, firmado, pdfsign.SignData{
		Signature: pdfsign.SignDataSignature{
			Info:       pdfsign.SignDataSignatureInfo{Name: "Firmante sintetico", Date: now},
			CertType:   pdfsign.ApprovalSignature,
			DocMDPPerm: pdfsign.AllowFillingExistingFormFieldsAndSignaturesPerms,
			SubFilter:  pdfsign.SignatureSubFilterETSICAdESDetached,
		},
		Signer: leafKey, DigestAlgorithm: crypto.SHA256, Certificate: leaf,
		CertificateChains: [][]*x509.Certificate{{leaf, middle}},
	})
	fail(err)
	signed, err := os.ReadFile(firmado)
	fail(err)
	marker := []byte(pdffixture.Marker)
	pos := -1
	for i := 0; i+len(marker) <= len(signed); i++ {
		if string(signed[i:i+len(marker)]) == string(marker) {
			pos = i
			break
		}
	}
	if pos < 0 {
		panic("el PDF firmado no conserva el marcador original")
	}
	signed[pos+2] ^= 1
	write(filepath.Join(dir, "alterado.pdf"), signed)
	fmt.Println("PDF PAdES sintetico y material PKI generados")
}
