package application

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func directorioFuentesAplicacionInternoPrueba(t *testing.T) string {
	t.Helper()
	_, fichero, _, ok := runtime.Caller(0)
	if ok && filepath.IsAbs(fichero) {
		directorio := filepath.Dir(fichero)
		if _, err := os.Stat(filepath.Join(directorio, "autorizacion.go")); err == nil {
			return directorio
		}
	}
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatalf("localizar directorio fuente de application: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directorio, "autorizacion.go")); err != nil {
		t.Fatalf("el directorio de prueba no contiene application: %v", err)
	}
	return directorio
}
