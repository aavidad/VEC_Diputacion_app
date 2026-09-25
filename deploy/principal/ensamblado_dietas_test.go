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
		"dietas_borradores/migraciones/000004_calculo_comision.up.sql",
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

// El paquete incremental D2-D7 conserva el orden causal verificado, incluye
// cada migración íntegra (sin su BEGIN/COMMIT) tras su marca de instalada y
// termina en una sola transacción pendiente de `:finalizar`.
func TestPaqueteDietasIncrementalOrdenYMarcas(t *testing.T) {
	directorio, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Clean(filepath.Join(directorio, "..", ".."))
	orden := []string{
		"dietas_borradores/migraciones/000005_auditoria_frontera.up.sql",
		"autorizacion_atestada_v3/migraciones/000059_consumidor_documento_dietas.up.sql",
		"personal/migraciones/000012_asignacion_dietas.up.sql",
		"personal/migraciones/000013_auditoria_frontera_asignacion_dietas.up.sql",
		"dietas_borradores/migraciones/000006_documento_comision.up.sql",
		"dietas_borradores/migraciones/000007_circuito_comision.up.sql",
		"autorizacion_atestada_v3/migraciones/000080_consumidor_revisor_documento_dietas.up.sql",
		"dietas_borradores/migraciones/000008_revision_circuito_comision.up.sql",
		"autorizacion_atestada_v3/migraciones/000075_campo_devolucion_documento_dietas.up.sql",
		"dietas_borradores/migraciones/000009_otros_gastos_justificados.up.sql",
		"dietas_borradores/migraciones/000010_devolucion_reenvio_comision.up.sql",
		"dietas_borradores/migraciones/000011_campo_devolucion_decision.up.sql",
	}
	obtenido, err := exec.Command("bash", filepath.Join(directorio, "04_dietas_migraciones.sh"), "--incremental").Output()
	if err != nil {
		t.Fatalf("el ensamblador incremental falló: %v", err)
	}
	salida := string(obtenido)
	if !strings.HasPrefix(salida, "\\set ON_ERROR_STOP on\nBEGIN;\n") || !strings.HasSuffix(salida, "\n:finalizar;\n") ||
		strings.Count(salida, "\nBEGIN;\n") != 1 || strings.Count(salida, "\nCOMMIT;\n") != 0 {
		t.Fatal("el paquete incremental no es una sola transacción pendiente de :finalizar")
	}
	if strings.Count(salida, "\\if :vec_dietas_instalada\n") != len(orden) || strings.Count(salida, "\n\\endif\n") != len(orden) {
		t.Fatal("cada migración debe ir tras su marca de instalada")
	}
	posicion := 0
	for _, relativa := range orden {
		contenido, err := os.ReadFile(filepath.Join(repo, "deploy", "postgresql", relativa))
		if err != nil {
			t.Fatal(err)
		}
		var cuerpo strings.Builder
		cuerpo.WriteString("\\else\n-- INICIO deploy/postgresql/" + relativa + "\n")
		for _, linea := range strings.Split(strings.TrimSuffix(string(contenido), "\n"), "\n") {
			if linea == "\\set ON_ERROR_STOP on" || linea == "BEGIN;" || linea == "COMMIT;" {
				continue
			}
			cuerpo.WriteString(linea + "\n")
		}
		cuerpo.WriteString("-- FIN deploy/postgresql/" + relativa + "\n\\endif\n")
		indice := strings.Index(salida[posicion:], cuerpo.String())
		if indice < 0 {
			t.Fatalf("%s falta, está alterada o fuera de orden", relativa)
		}
		posicion += indice + cuerpo.Len()
	}
	if err := exec.Command("bash", filepath.Join(directorio, "04_dietas_migraciones.sh"), "--otro").Run(); err == nil {
		t.Fatal("un modo desconocido debe rechazarse")
	}
}
