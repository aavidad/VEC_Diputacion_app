package principal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaqueteCronosSeEnsamblaDesdeFuentesCanonicas(t *testing.T) {
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Clean(filepath.Join(directorio, "..", ".."))
	relativas := []string{
		"cronos_v1/migraciones/000001_esquema_marcajes.up.sql",
		"autorizacion_atestada_v3/migraciones/000051_consumidor_marcaje_propio_cronos.up.sql",
		"cronos_v1/migraciones/000002_registrar_marcaje_propio.up.sql",
	}
	var esperado bytes.Buffer
	esperado.WriteString("\\set ON_ERROR_STOP on\nBEGIN;\n")
	for _, relativa := range relativas {
		contenido, err := os.ReadFile(filepath.Join(repo, "deploy", "postgresql", relativa))
		if err != nil {
			t.Fatal(err)
		}
		esperado.WriteString("-- INICIO deploy/postgresql/" + relativa + "\n")
		for _, linea := range strings.Split(strings.TrimSuffix(string(contenido), "\n"), "\n") {
			if linea == "\\set ON_ERROR_STOP on" || linea == "BEGIN;" || linea == "COMMIT;" {
				continue
			}
			esperado.WriteString(linea + "\n")
		}
		esperado.WriteString("-- FIN deploy/postgresql/" + relativa + "\n")
	}
	esperado.WriteString(":finalizar;\n")
	comando := exec.Command("bash", filepath.Join(directorio, "05_cronos_migraciones.sh"))
	obtenido, err := comando.CombinedOutput()
	if err != nil {
		t.Fatalf("el ensamblador Cronos falló: %v\n%s", err, obtenido)
	}
	if !bytes.Equal(obtenido, esperado.Bytes()) {
		t.Fatal("el paquete Dietas difiere de las fuentes canónicas")
	}
}
