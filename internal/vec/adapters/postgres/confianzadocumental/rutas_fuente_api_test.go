package confianzadocumental_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// directorioFuentesConfianzaPrueba funciona tanto con rutas absolutas como con
// las rutas de modulo que conserva go test -trimpath. Go ejecuta cada binario
// de prueba desde el directorio fuente de su paquete.
func directorioFuentesConfianzaPrueba(t *testing.T) string {
	t.Helper()
	_, fichero, _, ok := runtime.Caller(0)
	if ok && filepath.IsAbs(fichero) {
		directorio := filepath.Dir(fichero)
		if _, err := os.Stat(filepath.Join(directorio, "servicio.go")); err == nil {
			return directorio
		}
	}
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatalf("localizar directorio fuente de confianza: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directorio, "servicio.go")); err != nil {
		t.Fatalf("el directorio de prueba no contiene el paquete de confianza: %v", err)
	}
	return directorio
}
