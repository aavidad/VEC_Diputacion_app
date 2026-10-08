package application

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestProyeccionEstadoCeseSolapadoYRetornoALista(t *testing.T) {
	madrid, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	// Cese 31/01→31/10 (9 meses) y otro 30/06→30/07 (1 mes):
	// la fecha efectiva es el último cese, el límite sigue siendo octubre.
	efecto := time.Date(2026, 6, 30, 0, 0, 0, 0, madrid)
	limite := time.Date(2026, 10, 31, 0, 0, 0, 0, madrid)
	base := ports.SituacionParticipacion{Situacion: "trabajando", Desde: time.Date(2026, 1, 1, 0, 0, 0, 0, madrid)}
	corte := time.Date(2026, 8, 1, 12, 0, 0, 0, madrid)
	actual, err := ProyectarSituacionConEstadoCese(base, ports.EstadoCese{
		FechaEfecto: efecto, DisponibleDesde: limite, EnRestriccion: true, TrabajoCesado: true,
	}, true, corte)
	if err != nil || actual.Situacion != "disponible_desde" || !actual.Desde.Equal(efecto) ||
		actual.FechaDisponible == nil || !actual.FechaDisponible.Equal(limite) ||
		EstadoPublicoRestriccionCese(actual, corte) != "no_disponible" {
		t.Fatalf("dos ceses solapados proyectados mal: %+v, %v", actual, err)
	}
	corte = time.Date(2026, 11, 1, 12, 0, 0, 0, madrid)
	actual, err = ProyectarSituacionConEstadoCese(base, ports.EstadoCese{
		FechaEfecto: efecto, DisponibleDesde: limite, TrabajoCesado: true,
	}, true, corte)
	if err != nil || actual.Situacion != "disponible" || actual.FechaDisponible != nil ||
		EstadoPublicoRestriccionCese(actual, corte) != "disponible" {
		t.Fatalf("retorno al vencer no aplicado: %+v, %v", actual, err)
	}
	actual, err = ProyectarSituacionConEstadoCese(base, ports.EstadoCese{
		FechaEfecto: efecto, DisponibleDesde: limite, TrabajoCesado: false,
	}, true, corte)
	if err != nil || actual.Situacion != "trabajando" {
		t.Fatalf("otra relación abierta perdió ocupado: %+v, %v", actual, err)
	}
	base.Situacion = "disponible_desde"
	base.FechaDisponible = &limite
	actual, err = ProyectarSituacionConEstadoCese(base, ports.EstadoCese{
		FechaEfecto: efecto, DisponibleDesde: limite, TrabajoCesado: true,
	}, true, corte)
	if err != nil || actual.Situacion != "disponible" || actual.FechaDisponible != nil {
		t.Fatalf("fecha vencida sigue rotulada como espera: %+v, %v", actual, err)
	}
}

func TestProyeccionEstadoCeseConservaPausaB2Posterior(t *testing.T) {
	madrid, _ := time.LoadLocation("Europe/Madrid")
	efecto := time.Date(2026, 6, 1, 0, 0, 0, 0, madrid)
	cese := time.Date(2026, 10, 1, 0, 0, 0, 0, madrid)
	pausa := time.Date(2026, 12, 1, 0, 0, 0, 0, madrid)
	corte := time.Date(2026, 7, 1, 12, 0, 0, 0, madrid)
	base := ports.SituacionParticipacion{Situacion: "disponible_desde", Desde: efecto.Add(-time.Hour), FechaDisponible: &pausa}
	actual, err := ProyectarSituacionConEstadoCese(base, ports.EstadoCese{
		FechaEfecto: efecto, DisponibleDesde: cese, EnRestriccion: true,
	}, true, corte)
	if err != nil || actual.FechaDisponible == nil || !actual.FechaDisponible.Equal(pausa) {
		t.Fatalf("pausa posterior adelantada: %+v, %v", actual, err)
	}
	for _, estado := range []string{"no_disponible", "excluido", "renuncia"} {
		base.Situacion, base.FechaDisponible = estado, nil
		actual, err = ProyectarSituacionConEstadoCese(base, ports.EstadoCese{
			FechaEfecto: efecto, DisponibleDesde: cese, EnRestriccion: true,
		}, true, corte)
		if err != nil || actual.Situacion != estado {
			t.Fatalf("estado más restrictivo alterado: %+v, %v", actual, err)
		}
	}
}

func TestProyeccionCesePendienteImpideTurnoSinInventarFecha(t *testing.T) {
	madrid, _ := time.LoadLocation("Europe/Madrid")
	corte := time.Date(2026, 10, 8, 12, 0, 0, 0, madrid)
	desde := time.Date(2026, 1, 5, 0, 0, 0, 0, madrid)
	pendienteDesde := time.Date(2026, 10, 7, 15, 0, 0, 0, madrid)
	fecha := time.Date(2026, 11, 1, 0, 0, 0, 0, madrid)
	for _, anterior := range []string{"disponible", "trabajando", "disponible_desde"} {
		base := ports.SituacionParticipacion{Situacion: anterior, Desde: desde, FechaDisponible: &fecha}
		actual, err := ProyectarSituacionConEstadoCese(base, ports.EstadoCese{CesePendiente: true, PendienteDesde: pendienteDesde}, true, corte)
		if err != nil || actual.Situacion != "no_disponible" || actual.FechaDisponible != nil || !actual.Desde.Equal(pendienteDesde) {
			t.Fatalf("pendiente con situación %s: %+v %v", anterior, actual, err)
		}
		if EstadoPublicoRestriccionCese(actual, corte) != "no_disponible" {
			t.Fatalf("pendiente visible como elegible: %+v", actual)
		}
	}
	base := ports.SituacionParticipacion{Situacion: "disponible", Desde: desde}
	for _, previo := range []ports.EstadoCese{{CesePendiente: true, PendienteDesde: pendienteDesde},
		{CesePendiente: true, PendienteDesde: pendienteDesde, FechaEfecto: desde, DisponibleDesde: fecha, EnRestriccion: true, TrabajoCesado: true}} {
		actual, err := ProyectarSituacionConEstadoCese(base, previo, true, corte)
		if err != nil || actual.Situacion != "no_disponible" || actual.FechaDisponible != nil {
			t.Fatalf("pendiente prevalece sobre B45 previo: %+v %v", actual, err)
		}
	}
	if _, err := ProyectarSituacionConEstadoCese(base, ports.EstadoCese{CesePendiente: true}, true, corte); err == nil {
		t.Fatal("pendiente sin instante B13 aceptado")
	}
	for _, anterior := range []string{"excluido", "renuncia", "no_disponible"} {
		base := ports.SituacionParticipacion{Situacion: anterior, Desde: desde}
		actual, err := ProyectarSituacionConEstadoCese(base, ports.EstadoCese{CesePendiente: true}, true, corte)
		if err != nil || actual.Situacion != anterior || !actual.Desde.Equal(desde) {
			t.Fatalf("pendiente alteró situación anterior %s: %+v %v", anterior, actual, err)
		}
	}
	sinCese, err := ProyectarSituacionConEstadoCese(base, ports.EstadoCese{}, false, corte)
	if err != nil || sinCese.Situacion != "disponible" {
		t.Fatalf("ausencia de cese alteró situación: %+v %v", sinCese, err)
	}
}
