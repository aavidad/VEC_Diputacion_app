package identidadcertificado

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectorioAusenteConservaCausaSinRutaPrivada(t *testing.T) {
	directorio := t.TempDir()
	ruta := filepath.Join(directorio, "ausente", "registro.json")
	casos := map[string]func() error{
		"registro": func() error { _, err := NuevoRegistro(ruta); return err },
		"archivo":  func() error { _, err := LeerArchivoPrivado(ruta, 1024); return err },
	}
	for nombre, ejecutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			err := ejecutar()
			if !errors.Is(err, ErrNoDisponible) || !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("causa no conservada: %v", err)
			}
			if strings.Contains(err.Error(), directorio) {
				t.Fatal("el error expone la ruta privada")
			}
		})
	}
}
