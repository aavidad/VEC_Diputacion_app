package xlsconvoca_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
)

func TestFormatoContenidoDistingueLibrosRenombrados(t *testing.T) {
	casos := []struct {
		ruta, nombre, fisico string
	}{
		{filepath.Join("..", "..", "application", "testdata", "carga_convoca", "carga_convoca_ejemplo.xlsx"), "carga_convoca_ejemplo.xls", "xlsx"},
		{filepath.Join("testdata", "xls_sinteticos", "resumen.xls"), "resumen.xlsx", "xls"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			contenido, err := os.ReadFile(caso.ruta)
			if err != nil {
				t.Fatal(err)
			}
			formato, err := xlsconvoca.FormatoContenido(contenido)
			if err != nil || formato != caso.fisico || formato == strings.TrimPrefix(filepath.Ext(caso.nombre), ".") {
				t.Fatalf("formato físico %q, nombre %q, error %v", formato, caso.nombre, err)
			}
		})
	}
	if _, err := xlsconvoca.FormatoContenido([]byte("libro falso")); !errors.Is(err, xlsconvoca.ErrXLSInvalido) {
		t.Fatalf("cabecera desconocida: %v", err)
	}
}
