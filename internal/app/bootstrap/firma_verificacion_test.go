package bootstrap

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma/servidorprueba"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestFirmaVerificacionApagadaNoAlteraDocumentos(t *testing.T) {
	cfg := config.Config{DocumentosEnabled: "false"}
	antes := manifiestosShellVEC(cfg)
	cfg.FirmaVerificacionURL = "https://destino-inerte.example"
	cfg.FirmaVerificacionCAFile = "/material/inexistente"
	verificador, err := nuevoVerificadorFirmaDocumentos(cfg)
	if err != nil || verificador != nil || !reflect.DeepEqual(antes, manifiestosShellVEC(cfg)) {
		t.Fatalf("apagada: verificador=%v, error=%v, manifiestos alterados=%v", verificador != nil, err, !reflect.DeepEqual(antes, manifiestosShellVEC(cfg)))
	}
	if _, err := nuevosDocumentosDesarrollo(cfg, nil, nil, nil, nil); err != nil {
		t.Fatalf("montaje apagado: %v", err)
	}
}

func TestFirmaVerificacionExigeDocumentosYConfiguracionPrivada(t *testing.T) {
	servidor := servidorprueba.Nuevo(strings.Repeat("t", 40), false)
	defer servidor.Close()
	directorio := t.TempDir()
	guardar := func(nombre string, contenido []byte) string {
		ruta := filepath.Join(directorio, nombre)
		if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	ca := guardar("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servidor.Certificate().Raw}))
	token := guardar("token", []byte(strings.Repeat("t", 40)))
	cfg := config.Config{
		FirmaVerificacionEnabled: "true", FirmaVerificacionURL: servidor.URL,
		FirmaVerificacionCAFile: ca, FirmaVerificacionTokenFile: token,
		FirmaVerificacionTimeout: "5s",
	}
	malas := map[string]func(*config.Config){
		"selector":               func(c *config.Config) { c.FirmaVerificacionEnabled = "si" },
		"sin documentos":         func(c *config.Config) {},
		"sin CA":                 func(c *config.Config) { c.FirmaVerificacionCAFile = "" },
		"CA ausente":             func(c *config.Config) { c.FirmaVerificacionCAFile = filepath.Join(directorio, "ausente.pem") },
		"sin credencial":         func(c *config.Config) { c.FirmaVerificacionTokenFile = "" },
		"plazo excesivo":         func(c *config.Config) { c.FirmaVerificacionTimeout = "61s" },
		"URL sin TLS":            func(c *config.Config) { c.FirmaVerificacionURL = "http://ejemplo.test" },
		"token insuficiente":     func(c *config.Config) { c.FirmaVerificacionTokenFile = guardar("corto", []byte("x")) },
		"nombre TLS con puerto":  func(c *config.Config) { c.FirmaVerificacionNombreServidorTLS = "example.com:443" },
		"nombre TLS punto final": func(c *config.Config) { c.FirmaVerificacionNombreServidorTLS = "example.com." },
		"nombre TLS etiqueta":    func(c *config.Config) { c.FirmaVerificacionNombreServidorTLS = "-example.com" },
		"nombre TLS caracteres":  func(c *config.Config) { c.FirmaVerificacionNombreServidorTLS = "exa mple.com" },
	}
	for nombre, mutar := range malas {
		t.Run(nombre, func(t *testing.T) {
			caso := cfg
			if nombre != "sin documentos" {
				caso.DocumentosEnabled = "true"
				caso.ExecutionProfile = config.ExecutionProfileDevelopment
				caso.AuthMode = config.AuthModeDevelopment
				caso.DevelopmentGuard = config.DevelopmentGuardAcknowledgement
			}
			mutar(&caso)
			if v, err := nuevoVerificadorFirmaDocumentos(caso); v != nil || !errors.Is(err, ErrComposicionFirmaVerificacionNoDisponible) ||
				strings.Contains(err.Error(), directorio) || strings.Contains(err.Error(), token) || strings.Contains(err.Error(), servidor.URL) {
				t.Fatalf("fallo no cerrado o detalle privado en error: %v", err)
			}
		})
	}
	cfg.DocumentosEnabled = "true"
	cfg.ExecutionProfile = config.ExecutionProfileDevelopment
	cfg.AuthMode = config.AuthModeDevelopment
	cfg.DevelopmentGuard = config.DevelopmentGuardAcknowledgement
	antes := manifiestosShellVEC(cfg)
	verificador, err := nuevoVerificadorFirmaDocumentos(cfg)
	if err != nil || verificador == nil || !reflect.DeepEqual(antes, manifiestosShellVEC(cfg)) {
		t.Fatalf("composicion válida sin rutas propias: verificador=%v error=%v", verificador != nil, err)
	}
}

func TestFirmaVerificacionRechazaCredencialesLegiblesYParIncompleto(t *testing.T) {
	servidor := servidorprueba.Nuevo(strings.Repeat("t", 40), false)
	defer servidor.Close()
	directorio := t.TempDir()
	guardar := func(nombre string, contenido []byte, modo os.FileMode) string {
		ruta := filepath.Join(directorio, nombre)
		if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ruta, modo); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servidor.Certificate().Raw})
	certificado, clave := certificadoClienteFirmaPrueba(t)
	base := config.Config{
		FirmaVerificacionEnabled: "true", FirmaVerificacionURL: servidor.URL,
		FirmaVerificacionCAFile:    guardar("ca.pem", caPEM, 0o600),
		FirmaVerificacionTokenFile: guardar("token", []byte(strings.Repeat("t", 40)), 0o600),
		FirmaVerificacionTimeout:   "5s", DocumentosEnabled: "true",
		ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
	}
	if v, err := nuevoVerificadorFirmaDocumentos(base); err != nil || v == nil {
		t.Fatalf("base privada con token: %v", err)
	}
	par := base
	par.FirmaVerificacionTokenFile = ""
	par.FirmaVerificacionCertFile = guardar("cliente.pem", certificado, 0o600)
	par.FirmaVerificacionKeyFile = guardar("cliente.key", clave, 0o600)
	if v, err := nuevoVerificadorFirmaDocumentos(par); err != nil || v == nil {
		t.Fatalf("base privada con par: %v", err)
	}
	casos := map[string]config.Config{}
	for _, modo := range []os.FileMode{0o644, 0o640} {
		sufijo := "_" + modo.String()
		caso := base
		caso.FirmaVerificacionTokenFile = guardar("token"+sufijo, []byte(strings.Repeat("t", 40)), modo)
		casos["token "+modo.String()] = caso
		caso = base
		caso.FirmaVerificacionCAFile = guardar("ca"+sufijo, caPEM, modo)
		casos["CA "+modo.String()] = caso
		caso = par
		caso.FirmaVerificacionCertFile = guardar("cliente"+sufijo+".pem", certificado, modo)
		casos["certificado "+modo.String()] = caso
		caso = par
		caso.FirmaVerificacionKeyFile = guardar("cliente"+sufijo+".key", clave, modo)
		casos["clave "+modo.String()] = caso
	}
	sinClave := par
	sinClave.FirmaVerificacionKeyFile = ""
	casos["par sin clave"] = sinClave
	sinCertificado := par
	sinCertificado.FirmaVerificacionCertFile = ""
	casos["par sin certificado"] = sinCertificado
	conTokenSinClave := base
	conTokenSinClave.FirmaVerificacionCertFile = par.FirmaVerificacionCertFile
	casos["token con par incompleto"] = conTokenSinClave
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if v, err := nuevoVerificadorFirmaDocumentos(caso); v != nil || !errors.Is(err, ErrComposicionFirmaVerificacionNoDisponible) ||
				strings.Contains(err.Error(), directorio) {
				t.Fatalf("credencial expuesta aceptada: verificador=%v error=%v", v != nil, err)
			}
			if _, err := nuevosDocumentosDesarrollo(caso, nil, nil, nil, nil); !errors.Is(err, ErrComposicionFirmaVerificacionNoDisponible) {
				t.Fatalf("montaje de Documentos no cerrado: %v", err)
			}
		})
	}
}

func TestFirmaVerificacionNombreServidorTLSConfigurable(t *testing.T) {
	servidor := servidorprueba.Nuevo(strings.Repeat("t", 40), false)
	defer servidor.Close()
	directorio := t.TempDir()
	guardar := func(nombre string, contenido []byte) string {
		ruta := filepath.Join(directorio, nombre)
		if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	cfg := config.Config{
		FirmaVerificacionEnabled: "true", FirmaVerificacionURL: servidor.URL,
		FirmaVerificacionCAFile:    guardar("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: servidor.Certificate().Raw})),
		FirmaVerificacionTokenFile: guardar("token", []byte(strings.Repeat("t", 40))),
		FirmaVerificacionTimeout:   "5s", DocumentosEnabled: "true",
		ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
	}
	original := []byte("original sintetico")
	huella := sha256.Sum256(original)
	solicitud := docports.SolicitudVerificacionFirma{
		DocumentoID: "ref:" + strings.Repeat("1", 64), Version: 1,
		HuellaOriginalSHA256: hex.EncodeToString(huella[:]), ContenidoOriginal: original, ContenidoFirmado: []byte{0x30},
	}
	// El certificado de httptest cubre example.com y 127.0.0.1, no otro nombre.
	for nombre, esperado := range map[string]docports.MotivoVerificacionFirma{
		"":                  docports.MotivoFirmaVerificada,
		"example.com":       docports.MotivoFirmaVerificada,
		"validador.invalid": docports.MotivoValidadorNoDisponible,
	} {
		caso := cfg
		caso.FirmaVerificacionNombreServidorTLS = nombre
		verificador, err := nuevoVerificadorFirmaDocumentos(caso)
		if err != nil {
			t.Fatalf("nombre %q: %v", nombre, err)
		}
		r, err := verificador.VerificarMotivado(context.Background(), solicitud)
		if err != nil || r.Motivo != esperado {
			t.Fatalf("nombre %q: motivo=%v error=%v", nombre, r.Motivo, err)
		}
	}
}

func TestFirmaEncendidaNoOcultaErrorDeDocumentos(t *testing.T) {
	cfg := config.Config{FirmaVerificacionEnabled: "true", DocumentosEnabled: "si"}
	if _, err := nuevosDocumentosDesarrollo(cfg, nil, nil, nil, nil); !errors.Is(err, config.ErrConfiguracionDocumentosSelector) {
		t.Fatalf("selector de Documentos invalido oculto: %v", err)
	}
	cfg.DocumentosEnabled = "false"
	if _, err := nuevosDocumentosDesarrollo(cfg, nil, nil, nil, nil); !errors.Is(err, ErrComposicionFirmaVerificacionNoDisponible) {
		t.Fatalf("firma encendida sin Documentos ignorada: %v", err)
	}
}

func TestComposicionNormalRechazaFirmaEncendida(t *testing.T) {
	for _, selector := range []string{"true", "si"} {
		srv, err := NewHTTPServerWithConfig(configurarFuentesProduccionPrueba(t, config.Config{
			Address: "127.0.0.1:0", PersonalCatalogPath: "memory", StorageMode: config.StorageModeLocalDurable,
			DataDir: t.TempDir(), FirmaVerificacionEnabled: selector,
		}))
		if srv != nil || !errors.Is(err, ErrComposicionFirmaVerificacionNoDisponible) {
			t.Fatalf("selector %q ignorado en composicion normal: %v", selector, err)
		}
	}
}

func certificadoClienteFirmaPrueba(t *testing.T) ([]byte, []byte) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{
		SerialNumber: big.NewInt(9), Subject: pkix.Name{CommonName: "vec-cliente-sintetico"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	claveDER, err := x509.MarshalPKCS8PrivateKey(clave)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: claveDER})
}
