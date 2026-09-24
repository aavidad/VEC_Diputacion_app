package interna

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialIdentidadExigeClaveYRegistroPrivados(t *testing.T) {
	base := t.TempDir()
	if err := os.Chmod(base, 0700); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "identidad")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	c := certificadoPersonalRegistrado{
		HuellaSHA256:       "sha256:" + strings.Repeat("a", 64),
		SujetoID:           "per_aaaaaaaaaaaaaaaaaaaaaa",
		CuentaID:           "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: proteccionClavePersonalPKCS11, Activo: true,
	}
	escribirRegistroCertificadoPrueba(t, filepath.Join(dir, "certificados.json"), c)
	escribirCRLPrueba(t, dir, nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13), nil)
	meta, err := json.Marshal(documentoEmisorCertificado{
		Version: 1, ClaveID: "clave_desarrollo_01",
		PoliticaRef:    "pga_certificado_desarrollo_protegido_01",
		PoliticaHuella: "sha256:" + strings.Repeat("b", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	rutaMeta := filepath.Join(dir, "emisor.json")
	if err := os.WriteFile(rutaMeta, meta, 0600); err != nil {
		t.Fatal(err)
	}
	_, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(privada)
	if err != nil {
		t.Fatal(err)
	}
	rutaClave := filepath.Join(dir, "emisor.key")
	if err := os.WriteFile(rutaClave, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	cargado, err := cargarMaterialIdentidadCertificado(base)
	if err != nil || cargado.claveID != "clave_desarrollo_01" || cargado.firmante == nil {
		t.Fatalf("material privado = (%#v, %v)", cargado, err)
	}
	if err := os.Chmod(rutaClave, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarMaterialIdentidadCertificado(base); err == nil {
		t.Fatal("aceptó clave legible por terceros")
	}
	if err := os.Chmod(rutaClave, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaMeta, []byte(`{"version":1,"version":1,"clave_id":"clave_desarrollo_01"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarMaterialIdentidadCertificado(base); err == nil {
		t.Fatal("aceptó metadatos con claves duplicadas")
	}
}
