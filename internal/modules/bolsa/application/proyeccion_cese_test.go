package application

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestProyeccionCeseRRHHYPublicaConservaLimites(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	corte := time.Date(2026, 10, 1, 12, 0, 0, 0, madrid)
	disponible := time.Date(2027, 3, 28, 0, 0, 0, 0, madrid)
	efecto := time.Date(2026, 9, 28, 0, 0, 0, 0, madrid)
	restriccion := ports.RestriccionCese{DisponibleDesde: disponible, FechaEfecto: efecto, ReciboRef: "recibo:cese:1", PoliticaVersion: 1}
	for _, estado := range []string{"disponible", "trabajando", "disponible_desde"} {
		base := ports.SituacionParticipacion{Situacion: estado, Desde: corte.Add(-time.Hour)}
		actual := ProyectarRestriccionCese(base, restriccion, true)
		if actual.Situacion != "disponible_desde" || actual.FechaDisponible == nil || !actual.FechaDisponible.Equal(disponible) || !actual.Desde.Equal(efecto) {
			t.Fatalf("RRHH %s: %+v", estado, actual)
		}
		if publico := EstadoPublicoRestriccionCese(actual, corte); publico != "no_disponible" {
			t.Fatalf("B10 debe exponer solo estado minimizado: %q", publico)
		}
	}
	for _, estado := range []string{"no_disponible", "excluido", "renuncia"} {
		base := ports.SituacionParticipacion{Situacion: estado, Desde: corte.Add(-time.Hour)}
		if actual := ProyectarRestriccionCese(base, restriccion, true); actual.Situacion != estado || actual.FechaDisponible != nil {
			t.Fatalf("estado más restrictivo alterado %s: %+v", estado, actual)
		}
	}
	posterior := disponible.AddDate(0, 0, 1)
	base := ports.SituacionParticipacion{Situacion: "disponible_desde", Desde: efecto.Add(-time.Hour), FechaDisponible: &posterior}
	if actual := ProyectarRestriccionCese(base, restriccion, true); actual.FechaDisponible == nil || !actual.FechaDisponible.Equal(posterior) {
		t.Fatalf("se adelantó fecha ordinaria posterior: %+v", actual)
	}
}
