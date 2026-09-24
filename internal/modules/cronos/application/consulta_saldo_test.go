package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type repoSaldoPrueba struct {
	empleado, desde, hasta, zona string
	fuente                       ports.FuenteSaldo
	llamadas                     int
}

func (r *repoSaldoPrueba) ConsultarFuenteSaldo(_ context.Context, _ ports.OrdenConsultaSaldo, empleado, desde, hasta, zona string) (ports.FuenteSaldo, error) {
	r.llamadas++
	r.empleado, r.desde, r.hasta, r.zona = empleado, desde, hasta, zona
	r.fuente.EmpleadoRef, r.fuente.Desde, r.fuente.Hasta, r.fuente.ZonaHoraria = empleado, desde, hasta, zona
	return r.fuente, nil
}

func TestResolverPeriodoSaldoSemanaISOYFechaCivil(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	desde, hasta, err := resolverPeriodoSaldo(ports.PeriodoSaldoSemana, "", "", time.Date(2027, 1, 1, 10, 0, 0, 0, zona), zona)
	if err != nil || desde.Format("2006-01-02") != "2026-12-28" || hasta.Format("2006-01-02") != "2027-01-03" {
		t.Fatalf("desde=%s hasta=%s err=%v", desde, hasta, err)
	}
	if _, _, err := resolverPeriodoSaldo(ports.PeriodoSaldoRango, "2026-02-30", "2026-03-01", desde, zona); !errors.Is(err, ports.ErrConsultaSaldoInvalida) {
		t.Fatal(err)
	}
}

func TestConsultaSaldoDerivaEmpleadoYNoInventaPrevisto(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	tipoOrigen := "terminal"
	r := &repoSaldoPrueba{fuente: ports.FuenteSaldo{
		Jornadas: []ports.JornadaPrevista{{Fecha: "2026-09-21", TurnoRef: "turno_a", PoliticaVersionRef: "politica_v1", MinutosPrevistos: 450}},
		Marcajes: []ports.MarcajeSaldo{
			{MarcajeRef: "m1", Movimiento: domain.PunchEntry, InstanteUTC: time.Date(2026, 9, 21, 6, 0, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1", TipoOrigen: &tipoOrigen},
			{MarcajeRef: "m2", Movimiento: domain.PunchPauseStart, InstanteUTC: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
			{MarcajeRef: "m3", Movimiento: domain.PunchPauseEnd, InstanteUTC: time.Date(2026, 9, 21, 10, 30, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
			{MarcajeRef: "m4", Movimiento: domain.PunchExit, InstanteUTC: time.Date(2026, 9, 21, 14, 0, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
		},
		MovimientosSaldo: []ports.MovimientoSaldo{
			{Fecha: "2026-09-21", Tipo: "trabajado", DeltaMicrosegundos: 450 * 60 * 1000000, Fuentes: []string{"m1", "m4"}},
			{Fecha: "2026-09-21", Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{"turno_a"}},
		},
	}}
	s, _ := NuevoServicioConsultaSaldo(r, relojMarcajePrueba{time.Now().UTC()}, zona)
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaSaldo(actor)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := s.ConsultarSaldo(context.Background(), orden, ports.PeriodoSaldoRango, "2026-09-21", "2026-09-22")
	if err != nil {
		t.Fatal(err)
	}
	if r.empleado != "emp_0123456789abcdefghijkl" || r.zona != "Europe/Madrid" || r.llamadas != 1 {
		t.Fatalf("alcance: %+v", r)
	}
	if resultado.Detalle[0].TrabajadosMinutos != 450 || resultado.Detalle[0].PausasMinutos != 30 || *resultado.Detalle[0].SaldoMinutos != 0 {
		t.Fatalf("primer dia: %+v", resultado.Detalle[0])
	}
	if resultado.Detalle[0].Marcajes[0].Origen == nil || *resultado.Detalle[0].Marcajes[0].Origen != "terminal" {
		t.Fatalf("origen no acreditado: %+v", resultado.Detalle[0].Marcajes[0])
	}
	if resultado.Detalle[1].Estado != ports.EstadoSaldoNoDisponible || resultado.Detalle[1].PrevistosMinutos != nil || resultado.Resumen.SaldoMinutos != nil || resultado.Resumen.Estado != ports.EstadoSaldoNoDisponible {
		t.Fatalf("ausencia de fuente: %+v", resultado)
	}
}

func TestConsultaSaldoRechazaLibroDivergente(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	fuente := ports.FuenteSaldo{EmpleadoRef: "emp_test", Desde: "2026-09-21", Hasta: "2026-09-21", Jornadas: []ports.JornadaPrevista{{Fecha: "2026-09-21", PoliticaVersionRef: "v1", MinutosPrevistos: 450}}, MovimientosSaldo: []ports.MovimientoSaldo{{Fecha: "2026-09-21", Tipo: "previsto", DeltaMicrosegundos: -449 * 60 * 1000000, Fuentes: []string{"turno"}}}}
	desde, _ := fechaCivilSaldo("2026-09-21", zona)
	_, err := construirConsultaSaldo(ports.PeriodoSaldoHoy, desde, desde, fuente, zona)
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
}

func TestConsultaSaldoRespetaSemanaConTurnosMayoresDeTreintaYSieteHoras(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	desde, _ := fechaCivilSaldo("2026-09-21", zona)
	hasta := desde.AddDate(0, 0, 6)
	fuente := ports.FuenteSaldo{EmpleadoRef: "emp_test", Desde: "2026-09-21", Hasta: "2026-09-27", Completo: true}
	for i := 0; i < 7; i++ {
		fecha := desde.AddDate(0, 0, i)
		clave := fecha.Format("2006-01-02")
		previsto := int64(0)
		if i < 5 {
			previsto = 600
		}
		fuente.Jornadas = append(fuente.Jornadas, ports.JornadaPrevista{Fecha: clave, TurnoRef: "turno_largo_v1", PoliticaVersionRef: "politica_v1", MinutosPrevistos: previsto})
		fuente.MovimientosSaldo = append(fuente.MovimientosSaldo, ports.MovimientoSaldo{Fecha: clave, Tipo: "previsto", DeltaMicrosegundos: -previsto * 60 * 1000000, Fuentes: []string{"turno_largo_v1"}})
		if i < 5 {
			entrada := time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 8, 0, 0, 0, zona).UTC()
			salida := entrada.Add(10 * time.Hour)
			fuente.Marcajes = append(fuente.Marcajes, ports.MarcajeSaldo{MarcajeRef: clave + "-e", Movimiento: domain.PunchEntry, InstanteUTC: entrada, Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"}, ports.MarcajeSaldo{MarcajeRef: clave + "-s", Movimiento: domain.PunchExit, InstanteUTC: salida, Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"})
			fuente.MovimientosSaldo = append(fuente.MovimientosSaldo, ports.MovimientoSaldo{Fecha: clave, Tipo: "trabajado", DeltaMicrosegundos: 600 * 60 * 1000000, Fuentes: []string{clave + "-e", clave + "-s"}})
		}
	}
	got, err := construirConsultaSaldo(ports.PeriodoSaldoSemana, desde, hasta, fuente, zona)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Resumen.PrevistosMinutos != 3000 || got.Resumen.TrabajadosMinutos != 3000 || *got.Resumen.SaldoMinutos != 0 {
		t.Fatalf("resumen=%+v", got.Resumen)
	}
}

func TestConsultaSaldoSoloDiaBNoDaSaldoDePausaNocturnaInvalida(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	r := &repoSaldoPrueba{fuente: ports.FuenteSaldo{Completo: true,
		Jornadas: []ports.JornadaPrevista{{Fecha: "2026-10-25", TurnoRef: "turno_noche", PoliticaVersionRef: "politica_v1", MinutosPrevistos: 450}},
		Marcajes: []ports.MarcajeSaldo{
			{MarcajeRef: "e", Movimiento: domain.PunchEntry, InstanteUTC: time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
			{MarcajeRef: "p", Movimiento: domain.PunchPauseStart, InstanteUTC: time.Date(2026, 10, 24, 23, 0, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
			{MarcajeRef: "s", Movimiento: domain.PunchExit, InstanteUTC: time.Date(2026, 10, 25, 1, 30, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"},
		},
		MovimientosSaldo: []ports.MovimientoSaldo{
			{Fecha: "2026-10-25", Tipo: "trabajado", DeltaMicrosegundos: 60 * 60 * 1000000, Fuentes: []string{"e/p"}},
			{Fecha: "2026-10-25", Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{"turno_noche"}},
		},
	}}
	s, _ := NuevoServicioConsultaSaldo(r, relojMarcajePrueba{time.Now().UTC()}, zona)
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaSaldo(actor)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ConsultarSaldo(context.Background(), orden, ports.PeriodoSaldoRango, "2026-10-25", "2026-10-25")
	if err != nil {
		t.Fatal(err)
	}
	if r.desde != "2026-10-25" || r.hasta != "2026-10-25" || len(got.Detalle) != 1 {
		t.Fatalf("rango parcial: %+v %+v", r, got)
	}
	dia := got.Detalle[0]
	if dia.Estado != ports.EstadoSaldoIncompleto || dia.SaldoMinutos != nil || got.Resumen.Estado != ports.EstadoSaldoIncompleto || got.Resumen.SaldoMinutos != nil {
		t.Fatalf("saldo indebido día B: %+v", got)
	}
	if dia.TrabajadosMinutos != 60 || dia.PausasMinutos != 150 {
		t.Fatalf("duración DST: %+v", dia)
	}
}

func TestConsultaSaldoSoloDiaBNoDaSaldoConEntradaAnteriorAbierta(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		name, fechaB string
		entrada      time.Time
	}{
		{"normal", "2026-09-25", time.Date(2026, 9, 24, 21, 0, 0, 0, time.UTC)},
		{"cambio_horario", "2026-10-25", time.Date(2026, 10, 24, 21, 0, 0, 0, time.UTC)},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			r := &repoSaldoPrueba{fuente: ports.FuenteSaldo{Completo: true,
				Jornadas:         []ports.JornadaPrevista{{Fecha: tc.fechaB, TurnoRef: "turno_noche", PoliticaVersionRef: "politica_v1", MinutosPrevistos: 450}},
				Marcajes:         []ports.MarcajeSaldo{{MarcajeRef: "entrada_a", Movimiento: domain.PunchEntry, InstanteUTC: tc.entrada, Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"}},
				MovimientosSaldo: []ports.MovimientoSaldo{{Fecha: tc.fechaB, Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{"turno_noche"}}},
			}}
			s, _ := NuevoServicioConsultaSaldo(r, relojMarcajePrueba{time.Now().UTC()}, zona)
			actor, err := contexto(t).OrdenConsumo.ContextoActor()
			if err != nil {
				t.Fatal(err)
			}
			orden, err := ports.NuevaOrdenConsultaSaldo(actor)
			if err != nil {
				t.Fatal(err)
			}
			got, err := s.ConsultarSaldo(context.Background(), orden, ports.PeriodoSaldoRango, tc.fechaB, tc.fechaB)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.Detalle) != 1 || got.Detalle[0].Estado != ports.EstadoSaldoIncompleto || got.Detalle[0].SaldoMinutos != nil || got.Resumen.SaldoMinutos != nil || got.Resumen.Estado != ports.EstadoSaldoIncompleto {
				t.Fatalf("saldo indebido: %+v", got)
			}
			if got.Detalle[0].TrabajadosMinutos != 0 || len(got.Detalle[0].Marcajes) != 0 {
				t.Fatalf("tiempo extrapolado: %+v", got.Detalle[0])
			}
		})
	}
}

func TestConsultaSaldoRespetaFuenteSQLIncompletaAunqueNoLlegueHechoDeBorde(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	fecha, _ := fechaCivilSaldo("2026-09-24", zona)
	fuente := ports.FuenteSaldo{Completo: false,
		Jornadas:         []ports.JornadaPrevista{{Fecha: "2026-09-24", TurnoRef: "turno", PoliticaVersionRef: "v1", MinutosPrevistos: 450}},
		MovimientosSaldo: []ports.MovimientoSaldo{{Fecha: "2026-09-24", Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{"turno"}}},
	}
	got, err := construirConsultaSaldo(ports.PeriodoSaldoRango, fecha, fecha, fuente, zona)
	if err != nil {
		t.Fatal(err)
	}
	if got.Detalle[0].Estado != ports.EstadoSaldoIncompleto || got.Detalle[0].SaldoMinutos != nil || got.Resumen.Estado != ports.EstadoSaldoIncompleto || got.Resumen.SaldoMinutos != nil {
		t.Fatalf("fuente incompleta dio saldo: %+v", got)
	}
}

func TestConsultaSaldoEntradaAntiguaMarcaTodoRangoMultidiario(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	r := &repoSaldoPrueba{fuente: ports.FuenteSaldo{Completo: true,
		Marcajes: []ports.MarcajeSaldo{{MarcajeRef: "entrada_antigua", Movimiento: domain.PunchEntry, InstanteUTC: time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC), Canal: canalSaldoPrueba(), OrigenRef: "terminal_1"}},
	}}
	for _, fecha := range []string{"2026-09-24", "2026-09-25", "2026-09-26"} {
		r.fuente.Jornadas = append(r.fuente.Jornadas, ports.JornadaPrevista{Fecha: fecha, TurnoRef: "turno", PoliticaVersionRef: "v1", MinutosPrevistos: 450})
		r.fuente.MovimientosSaldo = append(r.fuente.MovimientosSaldo, ports.MovimientoSaldo{Fecha: fecha, Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{"turno"}})
	}
	s, _ := NuevoServicioConsultaSaldo(r, relojMarcajePrueba{time.Now().UTC()}, zona)
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsultaSaldo(actor)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.ConsultarSaldo(context.Background(), orden, ports.PeriodoSaldoRango, "2026-09-24", "2026-09-26")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Detalle) != 3 || got.Resumen.Estado != ports.EstadoSaldoIncompleto || got.Resumen.SaldoMinutos != nil {
		t.Fatalf("resumen indebido: %+v", got)
	}
	for _, dia := range got.Detalle {
		if dia.Estado != ports.EstadoSaldoIncompleto || dia.SaldoMinutos != nil || dia.TrabajadosMinutos != 0 {
			t.Fatalf("día interior disponible: %+v", dia)
		}
	}
}

func canalSaldoPrueba() ports.CanalSaldo {
	return ports.CanalSaldo{PoliticaVersionRef: "politica_v1", CanalRef: "canal_terminal", OrigenRef: "terminal_1", CalidadRef: "calidad_v1"}
}

func TestConsultaSaldoRechazaOrdenVaciaAntesDeLeer(t *testing.T) {
	r := &repoSaldoPrueba{}
	s, _ := NuevoServicioConsultaSaldo(r, relojMarcajePrueba{time.Now().UTC()}, time.UTC)
	_, err := s.ConsultarSaldo(context.Background(), ports.OrdenConsultaSaldo{}, ports.PeriodoSaldoHoy, "", "")
	if !errors.Is(err, ports.ErrConsultaSaldoInvalida) || r.llamadas != 0 {
		t.Fatal(err, r.llamadas)
	}
}
