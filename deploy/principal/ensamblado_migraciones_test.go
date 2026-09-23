package principal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaqueteMigracionesSeEnsamblaDesdeFuentesCanonicas(t *testing.T) {
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Clean(filepath.Join(directorio, "..", ".."))
	relativas := []string{
		"autorizacion_atestada_v3/migraciones/000047_consumidor_datos_contacto_participacion",
		"bolsa_llamamientos/migraciones/000016_datos_contacto_participacion",
		"autorizacion_atestada_v3/migraciones/000048_consumidor_emision_llamamiento",
		"bolsa_llamamientos/migraciones/000017_emision_llamamiento",
	}

	var esperado bytes.Buffer
	esperado.WriteString("\\set ON_ERROR_STOP on\nBEGIN;\n")
	for _, relativa := range relativas {
		ruta := filepath.Join(repo, "deploy", "postgresql", relativa+".up.sql")
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		esperado.WriteString("-- INICIO deploy/postgresql/" + relativa + ".up.sql\n")
		for _, linea := range strings.Split(strings.TrimSuffix(string(contenido), "\n"), "\n") {
			if linea == "\\set ON_ERROR_STOP on" || linea == "BEGIN;" || linea == "COMMIT;" {
				continue
			}
			esperado.WriteString(linea + "\n")
		}
		esperado.WriteString("-- FIN deploy/postgresql/" + relativa + ".up.sql\n")
	}
	esperado.WriteString(":finalizar;\n")

	comando := exec.Command("bash", filepath.Join(directorio, "02_migraciones.sh"))
	obtenido, err := comando.CombinedOutput()
	if err != nil {
		t.Fatalf("el ensamblador falló: %v\n%s", err, obtenido)
	}
	if !bytes.Equal(obtenido, esperado.Bytes()) {
		t.Fatal("el paquete desplegable difiere de las cuatro migraciones canónicas")
	}
	if bytes.Count(obtenido, []byte("p_perfil_mutacion IS NOT DISTINCT FROM 'emision_llamamiento_bolsa'")) < 2 {
		t.Fatal("el paquete perdió la admisión B7 del núcleo AD3-48")
	}
}
