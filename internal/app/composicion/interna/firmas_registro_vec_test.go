package interna

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOrigenRegistroFirmaVecGobernadoExigeNombreTLSYURLPrivada(t *testing.T) {
	cfg := Configuracion{NombreServidorTLS: "vec.example.invalid"}
	for _, c := range []struct {
		origen string
		valido bool
	}{
		{"https://vec.example.invalid", true},
		{"https://vec.example.invalid:8443", true},
		{"", false},
		{"http://vec.example.invalid", false},
		{"https://127.0.0.1:8443", false},
		{"https://otro.example.invalid", false},
		{"https://vec.example.invalid/ruta", false},
		{"https://vec.example.invalid?identidad=persona", false},
		{"https://usuario@vec.example.invalid", false},
	} {
		if obtuvo := origenRegistroFirmaVecGobernadoValido(cfg, c.origen); obtuvo != c.valido {
			t.Fatalf("origen %q: %t", c.origen, obtuvo)
		}
	}
	cfg.NombreServidorTLS = ""
	if origenRegistroFirmaVecGobernadoValido(cfg, "https://vec.example.invalid") {
		t.Fatal("sin nombre TLS gobernado aceptó origen")
	}
}

func TestAcreditadorRegistroFirmaVecReconsultaCRLYRegistro(t *testing.T) {
	intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
	hoja, ca := intercambio.estadoServidor.VerifiedChains[0][0], intercambio.estadoServidor.VerifiedChains[0][1]
	suma := sha256.Sum256(hoja.Raw)
	huella := "sha256:" + hex.EncodeToString(suma[:])
	c := certificadoPersonalRegistrado{HuellaSHA256: huella,
		SujetoID: "per_aaaaaaaaaaaaaaaaaaaaaa", CuentaID: "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: proteccionClavePersonalPKCS11, Activo: true}
	ruta := filepath.Join(t.TempDir(), "certificados.json")
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, nil)
	escribirRegistroCertificadoPrueba(t, ruta, c)
	registro, err := nuevoRegistroCertificadosPersonales(ruta)
	if err != nil {
		t.Fatal(err)
	}
	a := acreditadorRegistroCertificadoFirmaVec{registro: registro}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	identidad, err := a.AcreditarCertificadoFirmaVecV2(context.Background(), hoja, ca, ahora)
	if err != nil || identidad.PersonaRef != c.SujetoID || identidad.CuentaRef != c.CuentaID {
		t.Fatalf("certificado registrado: %v", err)
	}
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, hoja.SerialNumber)
	if _, err := a.AcreditarCertificadoFirmaVecV2(context.Background(), hoja, ca, ahora); err == nil {
		t.Fatal("CRL revocada admitida")
	}
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, nil)
	c.Activo = false
	escribirRegistroCertificadoPrueba(t, ruta, c)
	if _, err := a.AcreditarCertificadoFirmaVecV2(context.Background(), hoja, ca, ahora); err == nil {
		t.Fatal("registro retirado admitido")
	}
	c.Activo = true
	escribirRegistroCertificadoPrueba(t, ruta, c)
	if _, err := a.AcreditarCertificadoFirmaVecV2(context.Background(), hoja, ca, hoja.NotAfter); err == nil {
		t.Fatal("certificado caducado admitido")
	}
	if err := os.Remove(filepath.Join(filepath.Dir(ruta), "clientes.crl")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AcreditarCertificadoFirmaVecV2(context.Background(), hoja, ca, ahora); err == nil {
		t.Fatal("CRL ausente admitida")
	}
}
