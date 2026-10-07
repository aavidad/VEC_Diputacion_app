package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"vec-diputacion-granada/config"
)

func TestCustodiaImportacionConvocaFallaFueraDeDesarrollo(t *testing.T) {
	if _, e := custodiarImportacionConvocaDesarrollo(config.Config{}, []byte("xls")); e == nil {
		t.Fatal("custodia habilitada fuera de desarrollo")
	}
}

func TestCustodiaImportacionConvocaDistingueContenidoYConservaReferencia(t *testing.T) {
	directorio := t.TempDir()
	cfg := config.Config{
		ExecutionProfile:                   config.ExecutionProfileDevelopment,
		AuthMode:                           config.AuthModeDevelopment,
		DevelopmentGuard:                   config.DevelopmentGuardAcknowledgement,
		BolsaImportacionConvocaCustodiaDir: directorio,
	}
	for _, caso := range []struct {
		nombre, extension string
		contenido         []byte
	}{
		{"OLE2", ".xls", append([]byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}, []byte("sintetico")...)},
		{"ZIP", ".xlsx", append([]byte{'P', 'K', 3, 4}, []byte("sintetico")...)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			suma := sha256.Sum256(caso.contenido)
			huella := hex.EncodeToString(suma[:])
			for intento := 0; intento < 2; intento++ {
				referencia, err := custodiarImportacionConvocaDesarrollo(cfg, caso.contenido)
				if err != nil || referencia != "fichero:sha256:"+huella {
					t.Fatalf("custodia %d: referencia=%q error=%v", intento, referencia, err)
				}
			}
			guardado, err := os.ReadFile(filepath.Join(directorio, huella+caso.extension))
			if err != nil || sha256.Sum256(guardado) != suma {
				t.Fatalf("contenido custodiado distinto: %v", err)
			}
		})
	}
	if _, err := custodiarImportacionConvocaDesarrollo(cfg, []byte("desconocido")); err == nil {
		t.Fatal("custodia aceptó formato desconocido")
	}
}
