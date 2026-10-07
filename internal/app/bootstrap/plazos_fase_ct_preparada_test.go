package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// La calculadora preparada (reglas leídas una vez por consulta) da los mismos
// plazos, aplicabilidad y errores que la calculadora fila a fila.
func TestPlazoFaseCTPreparadaEquivaleAFilaAFila(t *testing.T) {
	contenido, err := os.ReadFile(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	ambiguo := strings.Replace(string(contenido), `"inicio": "solicitud",`, `"inicio": "solicitud", "fases": "fiscalizacion",`, 1)
	rutaAmbigua := filepath.Join(t.TempDir(), "ct_reglas_ambiguo.demo.json")
	if err := os.WriteFile(rutaAmbigua, []byte(ambiguo), 0o600); err != nil {
		t.Fatal(err)
	}
	catalogos := map[string]struct {
		ruta        string
		calendarios *consultaCalendariosReglasPrueba
	}{
		"ejemplo":        {rutaReglasCTEjemploPrueba, calendariosPlazoFasePrueba(t)},
		"ambiguo":        {rutaAmbigua, calendariosPlazoFasePrueba(t)},
		"sin_calendario": {rutaReglasCTEjemploPrueba, nil},
	}
	fases := []domain.ClaveFase{"solicitud", "asignacion_unidad", "informe_juridico", "fiscalizacion", "subsanacion_unidad", "nombramiento", "llamamiento"}
	instantes := []time.Time{
		time.Date(2026, 9, 28, 21, 59, 0, 0, time.UTC),
		time.Date(2026, 9, 28, 22, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC),
	}
	for nombre, catalogo := range catalogos {
		original := calculadoraPlazoFaseCTPrueba(t, catalogo.ruta, catalogo.calendarios)
		preparada, err := original.(ports.PreparadorPlazosFaseRRHH).PrepararPlazosFase(t.Context())
		if err != nil {
			t.Fatalf("%s: preparar: %v", nombre, err)
		}
		for _, fase := range fases {
			for _, ahora := range instantes {
				for _, urgente := range []bool{false, true} {
					solicitud := ports.SolicitudPlazoFaseRRHH{Fase: fase, Desde: time.Date(2026, 9, 15, 9, 30, 0, 0, time.UTC), Ahora: ahora, Urgente: urgente}
					p1, a1, e1 := original.CalcularPlazoFase(t.Context(), solicitud)
					p2, a2, e2 := preparada.CalcularPlazoFase(t.Context(), solicitud)
					if !reflect.DeepEqual(p1, p2) || a1 != a2 || !errors.Is(e2, e1) && (e1 != nil || e2 != nil) {
						t.Fatalf("%s %s %v urgente=%v: fila a fila %+v %v %v; preparada %+v %v %v", nombre, fase, ahora, urgente, p1, a1, e1, p2, a2, e2)
					}
				}
			}
		}
	}
	// Una solicitud incompleta falla igual.
	preparada, _ := calculadoraPlazoFaseCTPrueba(t, rutaReglasCTEjemploPrueba, calendariosPlazoFasePrueba(t)).(ports.PreparadorPlazosFaseRRHH).PrepararPlazosFase(t.Context())
	if _, aplicable, err := preparada.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{Fase: "fiscalizacion"}); aplicable || !errors.Is(err, reglas.ErrCalculoNoDisponible) {
		t.Fatalf("solicitud sin fechas aceptada: %v %v", aplicable, err)
	}
}
