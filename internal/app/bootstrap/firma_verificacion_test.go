package bootstrap

import (
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/documentos/adapters/validadorautofirma/servidorprueba"
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
		"selector":           func(c *config.Config) { c.FirmaVerificacionEnabled = "si" },
		"sin documentos":     func(c *config.Config) {},
		"sin CA":             func(c *config.Config) { c.FirmaVerificacionCAFile = "" },
		"CA ausente":         func(c *config.Config) { c.FirmaVerificacionCAFile = filepath.Join(directorio, "ausente.pem") },
		"sin credencial":     func(c *config.Config) { c.FirmaVerificacionTokenFile = "" },
		"plazo excesivo":     func(c *config.Config) { c.FirmaVerificacionTimeout = "61s" },
		"URL sin TLS":        func(c *config.Config) { c.FirmaVerificacionURL = "http://ejemplo.test" },
		"token insuficiente": func(c *config.Config) { c.FirmaVerificacionTokenFile = guardar("corto", []byte("x")) },
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
