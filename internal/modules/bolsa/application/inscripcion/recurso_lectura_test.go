package inscripcion

import (
	"strings"
	"testing"
)

func TestRecursoLecturaLigaFiltroEIdiomaSinReferenciaPersonalEnClaro(t *testing.T) {
	filtro := Filtro{Estado: EstadoPendiente, ConvocatoriaRef: "cv1_YXV4aWxpYXI_v1", Limite: 20, Cursor: "cursor-opaco"}
	a, err := RecursoLectura(AccionListarRRHH, "per_sintetica_001", "es", filtro, "")
	if err != nil || len(a) != len("inscripciones_rrhh_")+64 || strings.Contains(a, "per_sintetica_001") {
		t.Fatalf("recurso=%q err=%v", a, err)
	}
	filtro.Cursor = "otro-cursor"
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
}
