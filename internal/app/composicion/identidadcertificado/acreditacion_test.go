package identidadcertificado

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegistroAcreditaCadenaActualYRevocacionPosterior(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Second)
	publicaCA, privadaCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caModelo := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "AC sintética"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:     true, BasicConstraintsValid: true, SubjectKeyId: []byte{1, 2, 3, 4},
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caModelo, caModelo, publicaCA, privadaCA)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clienteModelo := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "Persona sintética"},
		NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(12 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	clienteDER, err := x509.CreateCertificate(rand.Reader, clienteModelo, ca, publica, privadaCA)
	if err != nil {
		t.Fatal(err)
	}
	cliente, err := x509.ParseCertificate(clienteDER)
	if err != nil {
		t.Fatal(err)
	}
	actual := CertificadoRegistrado{
		SujetoID: "per_aaaaaaaaaaaaaaaaaaaaaa", CuentaID: "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: ProteccionClavePKCS11, Activo: true,
	}
	suma := sha256.Sum256(cliente.Raw)
	actual.HuellaSHA256 = "sha256:" + hex.EncodeToString(suma[:])
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "certificados.json")
	escribirRegistroPrueba(t, ruta, actual)
	escribirCRLPrueba(t, filepath.Join(dir, "clientes.crl"), ca, privadaCA, ahora, nil)
	registro, err := NuevoRegistro(ruta)
	if err != nil {
		t.Fatal(err)
	}
	acreditacion, err := registro.AcreditarCadenaActual(context.Background(), []*x509.Certificate{cliente, ca}, ahora)
	if err != nil || acreditacion.SujetoID != actual.SujetoID || acreditacion.CuentaID != actual.CuentaID ||
		acreditacion.CertificadoSHA256 != actual.HuellaSHA256 ||
		!ahora.Before(acreditacion.CRLSiguienteEn) {
		t.Fatalf("acreditación de cadena actual: %+v, %v", acreditacion, err)
	}
	if _, err := registro.AcreditarCadenaActual(context.Background(), []*x509.Certificate{cliente, ca}, acreditacion.CRLSiguienteEn); err == nil {
		t.Fatal("CRL vencida admitida")
	}
	escribirCRLPrueba(t, filepath.Join(dir, "clientes.crl"), ca, privadaCA, ahora, cliente.SerialNumber)
	if _, err := registro.AcreditarCadenaActual(context.Background(), []*x509.Certificate{cliente, ca}, ahora); err == nil {
		t.Fatal("certificado revocado en CRL admitido")
	}
	escribirCRLPrueba(t, filepath.Join(dir, "clientes.crl"), ca, privadaCA, ahora, nil)
	actual.Activo = false
	escribirRegistroPrueba(t, ruta, actual)
	if _, err := registro.AcreditarCadenaActual(context.Background(), []*x509.Certificate{cliente, ca}, ahora); err == nil {
		t.Fatal("registro retirado tras construir la autoridad admitido")
	}
	if _, err := registro.Resolver(context.Background(), actual.HuellaSHA256); err == nil {
		t.Fatal("resolución del registro retirado admitida")
	}
	actual.Activo = true
	escribirRegistroPrueba(t, ruta, actual)
	if err := os.WriteFile(ruta, []byte(`{"version":1,"version":1,"certificados":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := registro.Resolver(context.Background(), actual.HuellaSHA256); err == nil {
		t.Fatal("JSON con claves duplicadas admitido")
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "clientes.crl"), ruta); err != nil {
		t.Fatal(err)
	}
	if _, err := registro.Resolver(context.Background(), actual.HuellaSHA256); err == nil {
		t.Fatal("enlace simbólico del registro admitido")
	}
}

func escribirRegistroPrueba(t *testing.T, ruta string, registro CertificadoRegistrado) {
	t.Helper()
	contenido, err := json.Marshal(DocumentoRegistro{Version: 1, Certificados: []CertificadoRegistrado{registro}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
}

func escribirCRLPrueba(t *testing.T, ruta string, ca *x509.Certificate, privada ed25519.PrivateKey,
	ahora time.Time, revocado *big.Int,
) {
	t.Helper()
	lista := &x509.RevocationList{
		Number: big.NewInt(1), ThisUpdate: ahora.Add(-time.Minute), NextUpdate: ahora.Add(time.Hour),
	}
	if revocado != nil {
		lista.RevokedCertificateEntries = []x509.RevocationListEntry{{
			SerialNumber: new(big.Int).Set(revocado), RevocationTime: ahora,
		}}
	}
	der, err := x509.CreateRevocationList(rand.Reader, lista, ca, privada)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRegistroRechazaRutasPrivadasNoConformes(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "identidad")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "certificados.json")
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoRegistro(ruta); err == nil {
		t.Fatal("directorio no privado admitido")
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoRegistro(strings.TrimPrefix(ruta, "/")); err == nil {
		t.Fatal("ruta relativa admitida")
	}
	enlace := filepath.Join(base, "alias")
	if err := os.Symlink(dir, enlace); err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoRegistro(filepath.Join(enlace, "certificados.json")); err == nil {
		t.Fatal("ancestro simbólico admitido")
	}
}
