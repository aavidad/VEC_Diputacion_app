package principal

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPaqueteDietasSeEnsamblaDesdeFuentesCanonicas(t *testing.T) {
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Clean(filepath.Join(directorio, "..", ".."))
	relativas := []string{
		"dietas_borradores/roles_up.sql",
		"personal/migraciones/000007_relacion_empleado_dietas.up.sql",
		"autorizacion_atestada_v3/migraciones/000049_consumidor_personal_dietas.up.sql",
		"autorizacion_atestada_v3/migraciones/000050_acceso_rutas_dietas.up.sql",
		"personal/migraciones/000008_consulta_relaciones_propias_dietas.up.sql",
		"personal/migraciones/000009_asignacion_dietas.up.sql",
		"dietas_borradores/migraciones/000001_borrador_comision_durable.up.sql",
		"dietas_borradores/migraciones/000002_tarifas_provisionales.up.sql",
		"dietas_borradores/migraciones/000003_consulta_tarifas_provisionales.up.sql",
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
	comando := exec.Command("bash", filepath.Join(directorio, "04_dietas_migraciones.sh"))
	obtenido, err := comando.CombinedOutput()
	if err != nil {
		t.Fatalf("el ensamblador Dietas falló: %v\n%s", err, obtenido)
	}
	if !bytes.Equal(obtenido, esperado.Bytes()) {
		t.Fatal("el paquete Dietas difiere de las fuentes canónicas")
	}
}
