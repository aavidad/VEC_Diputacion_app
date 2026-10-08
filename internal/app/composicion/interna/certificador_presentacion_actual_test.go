package interna

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestCertificadorPresentacionRevalidaDespuesDeAsercion(t *testing.T) {
	intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
	cadena := intercambio.estadoServidor.VerifiedChains[0]
	certificado := cadena[0]
	huella := sha256.Sum256(certificado.Raw)
	registroActual := certificadoPersonalRegistrado{
		HuellaSHA256:       "sha256:" + hex.EncodeToString(huella[:]),
		SujetoID:           "per_aaaaaaaaaaaaaaaaaaaaaa",
		CuentaID:           "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: proteccionClavePersonalPKCS11,
		Activo:             true,
	}
	dir := t.TempDir()
	ruta := filepath.Join(dir, "certificados.json")
	escribirCRLPrueba(t, dir, intercambio, nil)
	escribirRegistroCertificadoPrueba(t, ruta, registroActual)
	registro, err := nuevoRegistroCertificadosPersonales(ruta)
	if err != nil {
		t.Fatal(err)
	}
	cfg := configuracionInternaValidaPrueba()
	cfg.RetiradaPoliticaInternaEn = time.Now().UTC().Add(24 * time.Hour).Truncate(time.Second)
	_, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	extractor, _, _, err := nuevaAsercionCertificadoPersonal(cfg, "clave_desarrollo_01", privada, registro,
		"pga_certificado_desarrollo_protegido_01", "sha256:"+strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento", nil)
	estado := intercambio.estadoServidor
	r.TLS = &estado
	r = r.WithContext(context.WithValue(r.Context(), claveContextoCanalTLSInterno{},
		nuevaCapacidadCanalTLSInterno(&tokenServidorInterno{marca: 1}, estado)))
	r, err = httpseguridad.PrepararPeticionAsercionPasarela(r, 1024)
	if err != nil {
		t.Fatal(err)
	}
	protegida, err := extractor.ExtraerAsercionProtegida(r)
	if err != nil || len(protegida) == 0 {
		t.Fatal("la aserción previa no se emitió", err)
	}
	defer clear(protegida)
	certificador := &certificadorPresentacionPersonal{registro: registro}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	acreditacion, err := certificador.AcreditarCertificadoActual(r.Context(), cadena, ahora)
	if err != nil || acreditacion.SujetoID != registroActual.SujetoID ||
		acreditacion.CuentaID != registroActual.CuentaID ||
		acreditacion.CertificadoSHA256 != registroActual.HuellaSHA256 ||
		!acreditacion.CertificadoValidoHasta.Equal(certificado.NotAfter.UTC().Truncate(time.Microsecond)) ||
		!ahora.Before(acreditacion.CRLSiguienteEn) || !ahora.Before(acreditacion.CAValidaHasta) {
		t.Fatal("acreditación actual incompleta", err)
	}
	if _, err := certificador.AcreditarCertificadoActual(r.Context(), cadena, acreditacion.CRLSiguienteEn); err == nil {
		t.Fatal("CRL caducada admitida")
	}
	cadenaAjena := append([]*x509.Certificate(nil), cadena...)
	cadenaAjena[1] = nil
	if _, err := certificador.AcreditarCertificadoActual(r.Context(), cadenaAjena, ahora); err == nil {
		t.Fatal("cadena sin autoridad admitida")
	}
	// La revocación posterior a la emisión debe cerrar la presentación nueva.
	escribirCRLPrueba(t, dir, intercambio, certificado.SerialNumber)
	if _, err := certificador.AcreditarCertificadoActual(r.Context(), cadena, time.Now().UTC().Truncate(time.Microsecond)); err == nil {
		t.Fatal("CRL revocada tras emitir la aserción admitida")
	}
	escribirCRLPrueba(t, dir, intercambio, nil)
	registroActual.Activo = false
	escribirRegistroCertificadoPrueba(t, ruta, registroActual)
	if _, err := certificador.AcreditarCertificadoActual(r.Context(), cadena, time.Now().UTC().Truncate(time.Microsecond)); err == nil {
		t.Fatal("registro retirado tras emitir la aserción admitido")
	}
}
