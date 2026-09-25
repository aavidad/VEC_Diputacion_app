package interna

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestCertificadoTLSRealEmiteAsercionLigadaAPeticionYRevocable(t *testing.T) {
	intercambio := nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13)
	certificado := intercambio.estadoServidor.VerifiedChains[0][0]
	suma := sha256.Sum256(certificado.Raw)
	huella := "sha256:" + hex.EncodeToString(suma[:])
	c := certificadoPersonalRegistrado{
		HuellaSHA256: huella, SujetoID: "per_aaaaaaaaaaaaaaaaaaaaaa",
		CuentaID:           "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: proteccionClavePersonalPKCS11, Activo: true,
	}
	ruta := filepath.Join(t.TempDir(), "certificados.json")
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, nil)
	escribirRegistroCertificadoPrueba(t, ruta, c)
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
	extractor, verificador, _, err := nuevaAsercionCertificadoPersonal(
		cfg, "clave_desarrollo_01", privada, registro,
		"pga_certificado_desarrollo_protegido_01", "sha256:"+strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	original := httptest.NewRequest(http.MethodGet,
		"/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=exp_sintetico", nil)
	estado := intercambio.estadoServidor
	original.TLS = &estado
	original = original.WithContext(context.WithValue(original.Context(), claveContextoCanalTLSInterno{},
		nuevaCapacidadCanalTLSInterno(&tokenServidorInterno{marca: 1}, estado)))
	peticion, err := httpseguridad.PrepararPeticionAsercionPasarela(original, 1024)
	if err != nil {
		t.Fatal(err)
	}
	asercion, err := extractor.ExtraerAsercionProtegida(peticion)
	if err != nil {
		t.Fatalf("emitir desde TLS real: %v", err)
	}
	verificada, err := verificador.Verificar(peticion.Context(), asercion)
	if err != nil || verificada.SujetoID != c.SujetoID ||
		verificada.Cuenta.ID != c.CuentaID || len(verificada.Factores) != 1 {
		t.Fatalf("aserción verificada = (%#v, %v)", verificada, err)
	}
	otraAsercion, err := extractor.ExtraerAsercionProtegida(peticion)
	if err != nil {
		t.Fatalf("segunda petición: %v", err)
	}
	segunda, err := verificador.Verificar(peticion.Context(), otraAsercion)
	if err != nil || segunda.SesionID == verificada.SesionID || segunda.ID == verificada.ID {
		t.Fatalf("sesión o nonce reutilizados: %v", err)
	}
	otra := httptest.NewRequest(http.MethodGet, "/api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=otro", nil)
	otra, err = httpseguridad.PrepararPeticionAsercionPasarela(otra, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := verificador.Verificar(otra.Context(), asercion); err == nil {
		t.Fatal("reutilizó aserción en otra consulta")
	}
	peticion.Header.Set("Authorization", "Bearer hostil")
	if _, err := extractor.ExtraerAsercionProtegida(peticion); err == nil {
		t.Fatal("aceptó cabecera del navegador")
	}
	peticion.Header.Del("Authorization")
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, certificado.SerialNumber)
	if _, err := extractor.ExtraerAsercionProtegida(peticion); err == nil {
		t.Fatal("emitió tras revocación firmada por AC")
	}
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, nil)
	if err := os.Remove(filepath.Join(filepath.Dir(ruta), "clientes.crl")); err != nil {
		t.Fatal(err)
	}
	if _, err := extractor.ExtraerAsercionProtegida(peticion); err == nil {
		t.Fatal("emitió sin estado de revocación de la AC")
	}
	escribirCRLPrueba(t, filepath.Dir(ruta), intercambio, nil)
	c.Activo = false
	escribirRegistroCertificadoPrueba(t, ruta, c)
	if _, err := extractor.ExtraerAsercionProtegida(peticion); err == nil {
		t.Fatal("emitió tras revocar certificado")
	}
}

func TestCertificadoPersonalRevocadoCortaEmisionYGarantiaSinReinicio(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "certificados.json")
	escribirCRLPrueba(t, filepath.Dir(ruta), nuevoIntercambioTLSIdentidadOfflinePrueba(t, tls.VersionTLS13), nil)
	const huella = "sha256:" + "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
	c := certificadoPersonalRegistrado{
		HuellaSHA256:       huella,
		SujetoID:           "per_aaaaaaaaaaaaaaaaaaaaaa",
		CuentaID:           "cta_bbbbbbbbbbbbbbbbbbbbbb",
		ProteccionClaveRef: proteccionClavePersonalPKCS11,
		Activo:             true,
	}
	escribirRegistroCertificadoPrueba(t, ruta, c)
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
	_, _, evaluador, err := nuevaAsercionCertificadoPersonal(cfg, "clave_desarrollo_01", privada, registro,
		"pga_certificado_desarrollo_protegido_01", "sha256:"+strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("montar emisor de certificado: %v", err)
	}
	entrada := httpseguridad.EntradaEvaluacionGarantia{
		ACRVerificado: httpseguridad.ACRCertificadoPersonalDesarrolloProtegido,
		Emisor:        cfg.EmisorIdentidad, Superficie: httpseguridad.SuperficieInternaCorporativa,
		SujetoID: c.SujetoID, CuentaID: c.CuentaID,
		MetodoPrimario: httpseguridad.MetodoCertificado,
		Factores: []httpseguridad.FactorAutenticacion{{
			Metodo: httpseguridad.MetodoCertificado, SujetoVinculadoID: c.SujetoID,
			CredencialRef: "cert:" + huella, EvidenciaRef: "tls:verified:" + huella,
			GrupoCriptograficoRef: "key:" + huella, VerificadoEn: time.Now().UTC(),
		}},
	}
	resultado, err := evaluador.Evaluar(context.Background(), entrada)
	if err != nil || resultado.Garantia != dominiovec.AuthAssuranceSubstantial {
		t.Fatalf("garantía real = (%#v, %v)", resultado, err)
	}
	c.Activo = false
	escribirRegistroCertificadoPrueba(t, ruta, c)
	if _, err := registro.resolver(context.Background(), huella); !errors.Is(err, ErrCertificadoPersonalNoDisponible) {
		t.Fatalf("certificado retirado resuelto: %v", err)
	}
	if _, err := evaluador.Evaluar(context.Background(), entrada); !errors.Is(err, ErrCertificadoPersonalNoDisponible) {
		t.Fatalf("garantía tras retirada: %v", err)
	}
}

func escribirCRLPrueba(t *testing.T, dir string, i *intercambioTLSIdentidadOfflinePrueba, revocado *big.Int) {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Second)
	lista := &x509.RevocationList{
		Number: big.NewInt(1), ThisUpdate: ahora.Add(-time.Minute), NextUpdate: ahora.Add(time.Hour),
	}
	if revocado != nil {
		lista.RevokedCertificateEntries = []x509.RevocationListEntry{{
			SerialNumber: new(big.Int).Set(revocado), RevocationTime: ahora,
		}}
	}
	der, err := x509.CreateRevocationList(rand.Reader, lista, i.autoridad, i.claveAutoridad)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "clientes.crl")
	temporal := ruta + ".nuevo"
	if err := os.WriteFile(temporal, pem.EncodeToMemory(&pem.Block{Type: "X509 CRL", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporal, ruta); err != nil {
		t.Fatal(err)
	}
}

func escribirRegistroCertificadoPrueba(t *testing.T, ruta string, c certificadoPersonalRegistrado) {
	t.Helper()
	contenido, err := json.Marshal(documentoCertificadosPersonales{
		Version: 1, Certificados: []certificadoPersonalRegistrado{c},
	})
	if err != nil {
		t.Fatal(err)
	}
	temporal := ruta + ".nuevo"
	if err := os.WriteFile(temporal, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporal, ruta); err != nil {
		t.Fatal(err)
	}
}
