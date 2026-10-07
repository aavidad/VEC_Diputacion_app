package interna

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestAnteponerRegistroFirmaVecGobernadoNoAbreOtrasRutas(t *testing.T) {
	var base, firma int
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { base++; w.WriteHeader(http.StatusNoContent) })
	firmaHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { firma++; w.WriteHeader(http.StatusAccepted) })
	puente, err := anteponerRegistroFirmaVecGobernado(baseHandler, firmaHandler)
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{httpinterno.RutaRegistroFirmaVec, http.StatusAccepted},
		{httpinterno.RutaRegistroFirmaExterna, http.StatusNoContent},
		{httpinterno.RutaConsultaFirmasR5V2, http.StatusNoContent},
	} {
		w := httptest.NewRecorder()
		puente.ServeHTTP(w, httptest.NewRequest(http.MethodPost, caso.ruta, nil))
		if w.Code != caso.estado {
			t.Fatalf("ruta %q: %d", caso.ruta, w.Code)
		}
	}
	if firma != 1 || base != 2 {
		t.Fatalf("desvío de rutas: firma=%d base=%d", firma, base)
	}
	if _, err := anteponerRegistroFirmaVecGobernado(baseHandler, nil); err == nil {
		t.Fatal("desvío sin handler de firma")
	}
}

func TestOrigenRegistroFirmaVecGobernadoExigeNombreTLSYURLPrivada(t *testing.T) {
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := cargarOrigenFirmaVecPrivado(directorio, "vec.example.invalid"); err == nil {
		t.Fatal("sin fichero privado aceptó origen")
	}
	ruta := filepath.Join(directorio, nombreArchivoOrigenFirmaVec)
	if err := os.WriteFile(ruta, []byte("origen_firma_vec=https://vec.example.invalid:8443\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	origen, err := cargarOrigenFirmaVecPrivado(directorio, "vec.example.invalid")
	if err != nil || origen != "https://vec.example.invalid:8443" {
		t.Fatalf("origen privado: %v", err)
	}
	if _, err := cargarOrigenFirmaVecPrivado(directorio, "otro.example.invalid"); err == nil {
		t.Fatal("host ajeno al nombre TLS aceptado")
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
