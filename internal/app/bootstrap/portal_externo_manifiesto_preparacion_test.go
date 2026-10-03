package bootstrap

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
)

func TestPrepararManifiestoPortalExternoComparaDERYSPKI(t *testing.T) {
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otraClave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	emitir := func(clave *ecdsa.PrivateKey, serie int64) []byte {
		t.Helper()
		cert := &x509.Certificate{
			SerialNumber: big.NewInt(serie),
			Subject:      pkix.Name{CommonName: "CA sintética de preparación"},
			NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
			IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
		}
		der, err := x509.CreateCertificate(rand.Reader, cert, cert, &clave.PublicKey, clave)
		if err != nil {
			t.Fatal(err)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	}
	caInterna, caReemitida, caDistinta := emitir(clave, 1), emitir(clave, 2), emitir(otraClave, 3)
	certInterno, err := decodificarCertificadoUnico(caInterna)
	if err != nil {
		t.Fatal(err)
	}
	certReemitido, err := decodificarCertificadoUnico(caReemitida)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(certInterno.Raw, certReemitido.Raw) || !bytes.Equal(certInterno.RawSubjectPublicKeyInfo, certReemitido.RawSubjectPublicKeyInfo) {
		t.Fatal("la prueba exige certificados DER distintos con la misma SPKI")
	}
	material := func(ca []byte) string {
		t.Helper()
		raiz := t.TempDir()
		if err := os.Mkdir(filepath.Join(raiz, "ca"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(raiz, "ca", "ca.crt"), ca, 0o600); err != nil {
			t.Fatal(err)
		}
		return raiz
	}
	for _, caso := range []struct {
		nombre  string
		ca      []byte
		rechazo bool
	}{
		{"mismo DER con otro PEM", append([]byte("\n"), caInterna...), true},
		{"DER distinto con la misma SPKI", caReemitida, true},
		{"CA distintas", caDistinta, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			interno, externo := material(caInterna), material(caso.ca)
			certExterno, err := decodificarCertificadoUnico(caso.ca)
			if err != nil {
				t.Fatal(err)
			}
			huellaExterna := sha256.Sum256(certExterno.Raw)
			manifiesto, err := json.Marshal(map[string]any{
				"version": 1, "huella_ca_sha256": hex.EncodeToString(huellaExterna[:]),
			})
			if err != nil {
				t.Fatal(err)
			}
			ruta := filepath.Join(externo, "manifiesto.json")
			if err := os.WriteFile(ruta, manifiesto, 0o600); err != nil {
				t.Fatal(err)
			}
			err = prepararManifiestoCAPropiaPortalExterno(interno, externo)
			posterior, errLectura := os.ReadFile(ruta)
			if errLectura != nil {
				t.Fatal(errLectura)
			}
			if caso.rechazo {
				if !errors.Is(err, ErrPreparacionPortalExternoInvalida) || !bytes.Equal(manifiesto, posterior) {
					t.Fatalf("CA compartida o manifiesto alterado: %v", err)
				}
				if caso.nombre == "DER distinto con la misma SPKI" {
					comprobarPreparacionCompletaRechazaCAMismaSPKI(t, interno, externo, manifiesto)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var publicado struct {
				Version int    `json:"version"`
				Interna string `json:"huella_ca_interna_sha256"`
				Propia  string `json:"huella_ca_sha256"`
			}
			huellaInterna := sha256.Sum256(certInterno.Raw)
			if json.Unmarshal(posterior, &publicado) != nil || publicado.Version != 2 ||
				publicado.Interna != hex.EncodeToString(huellaInterna[:]) ||
				publicado.Propia != hex.EncodeToString(huellaExterna[:]) {
				t.Fatal("el manifiesto v2 no conserva las huellas DER exactas")
			}
		})
	}
}

func comprobarPreparacionCompletaRechazaCAMismaSPKI(t *testing.T, interno, externo string, manifiesto []byte) {
	t.Helper()
	for _, caso := range []struct{ raiz, contenido string }{
		{interno, "idempotencia interna sintética"},
		{externo, "idempotencia externa sintética"},
	} {
		dir := filepath.Join(caso.raiz, "idempotencia")
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "localizador.bin"), []byte(caso.contenido), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(externo, separacionportales.FicheroMarcaPortal), []byte(`{"version":1,"portal":"externo"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		PortalProceso: "interno", ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir: interno,
	}.Normalize()
	_, err := PrepararMaterialPortalExterno(context.Background(), cfg, OpcionesPreparacionPortalExterno{
		Destino: externo, Consumidores: []string{"mi_bolsa"},
		HuellaAprobacionSHA256: strings.Repeat("a", 64), PreimagenSHA256: strings.Repeat("b", 64),
	})
	if !errors.Is(err, ErrPreparacionPortalExternoInvalida) {
		t.Fatalf("la preparación aceptó la CA reemitida: %v", err)
	}
	posterior, err := os.ReadFile(filepath.Join(externo, "manifiesto.json"))
	if err != nil || !bytes.Equal(manifiesto, posterior) {
		t.Fatal("la preparación completa alteró el manifiesto")
	}
	if _, err := os.Stat(filepath.Join(externo, "externo", "v3")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("la preparación llegó a crear raíz o publicación: %v", err)
	}
}
