package postgres

import (
	"strings"
	"testing"
)

func TestOperacionSituacionSeleccionaSQLPorGateB57(t *testing.T) {
	for _, caso := range []struct {
		catalogo bool
		version  string
		ultimo   string
	}{
		{false, "_v1(", "$25)"},
		{true, "_v2(", "$28)"},
	} {
		registro := sqlRegistrarOperacionSituacion(caso.catalogo)
		lectura := sqlListarOperacionesSituacion(caso.catalogo)
		if !strings.Contains(registro, "registrar_operacion_situacion_participacion"+caso.version) || !strings.HasSuffix(registro, caso.ultimo) {
			t.Fatalf("registro gate=%v usa SQL/argumentos inesperados", caso.catalogo)
		}
		if !strings.Contains(lectura, "listar_operaciones_situacion_participacion"+caso.version) || !strings.HasSuffix(lectura, "$12)") {
			t.Fatalf("lectura gate=%v usa SQL inesperado", caso.catalogo)
		}
	}
}
