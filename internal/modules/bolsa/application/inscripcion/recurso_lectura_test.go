package inscripcion

import (
	"strings"
	"testing"
)

func TestRecursoLecturaLigaFiltroEIdiomaSinReferenciaPersonalEnClaro(t *testing.T) {
	filtro := Filtro{Estado: EstadoPendiente, ConvocatoriaRef: "cv1_YXV4aWxpYXI_v1", Limite: 20, Cursor: "solicitud_inscripcion_" + strings.Repeat("a", 64)}
	a, err := RecursoLectura(AccionListarRRHH, "per_sintetica_001", "es", filtro, "")
	if err != nil || len(a) != len("inscripciones_rrhh_")+64 || strings.Contains(a, "per_sintetica_001") {
		t.Fatalf("recurso=%q err=%v", a, err)
	}
	filtro.Cursor = "solicitud_inscripcion_" + strings.Repeat("b", 64)
	b, err := RecursoLectura(AccionListarRRHH, "per_sintetica_001", "es", filtro, "")
	if err != nil || a == b {
		t.Fatal("cursor distinto no cambio recurso")
	}
	c, err := RecursoLectura(AccionListarRRHH, "per_sintetica_001", "en", filtro, "")
	if err != nil || b == c {
		t.Fatal("idioma distinto no cambio recurso")
	}
	if _, err := RecursoLectura(AccionDetalleAbierta, "per_sintetica_001", "es", Filtro{}, "cv1_YXV4aWxpYXI_v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := RecursoLectura(AccionDetalleAbierta, "per_sintetica_001", "es", Filtro{}, "bolsa:sin-version"); err == nil {
		t.Fatal("referencia de bolsa aceptada como convocatoria")
	}
	if _, err := RecursoLectura(AccionListarRRHH, "per_sintetica_001", "es", Filtro{Estado: EstadoPendiente, Limite: 20, Cursor: "cv1_YXV4aWxpYXI_v1"}, ""); err == nil {
		t.Fatal("cursor de convocatorias aceptado en solicitudes")
	}
}

func TestRecursoSelectorRRHHYBandejaExigenAmbitoDeConvocatoria(t *testing.T) {
	persona := "per_sintetica_001"
	selector, err := RecursoLectura(AccionConvocatoriasRRHH, persona, "es", Filtro{Limite: 20}, "")
	if err != nil || !strings.HasPrefix(selector, "inscripciones_rrhh_convocatorias_") || strings.Contains(selector, persona) {
		t.Fatalf("selector sin recurso opaco: %q %v", selector, err)
	}
	otro, err := RecursoLectura(AccionConvocatoriasRRHH, persona, "es", Filtro{Limite: 20, Cursor: "cv1_YXV4aWxpYXI_v1"}, "")
	if err != nil || selector == otro {
		t.Fatal("cursor del selector no queda ligado al permiso")
	}
	if _, err := RecursoLectura(AccionConvocatoriasRRHH, persona, "es", Filtro{Limite: 20, Cursor: "solicitud_inscripcion_" + strings.Repeat("a", 64)}, ""); err == nil {
		t.Fatal("cursor de solicitud aceptado como convocatoria")
	}
	if _, err := RecursoLectura(AccionListarRRHH, persona, "es", Filtro{Estado: EstadoPendiente, Limite: 20}, ""); err == nil {
		t.Fatal("bandeja sin convocatoria daría permiso global")
	}
}
