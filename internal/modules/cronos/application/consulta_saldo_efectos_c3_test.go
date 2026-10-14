package application

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func fuenteSaldoEfectosC3Prueba() ports.FuenteSaldo {
	fecha := "2026-10-20"
	program := domain.ProgramacionSaldoEfectosC3{Fecha: fecha, Referencia: "programacion:cronos:ejemplo", FuenteRef: "fuente:programacion:ejemplo", Version: 1, Minutos: 450,
		InicioUTC: time.Date(2026, 10, 20, 6, 0, 0, 0, time.UTC), FinUTC: time.Date(2026, 10, 20, 13, 30, 0, 0, time.UTC), CentroRef: "centro:demo", ColectivoRef: "regimen:laboral"}
	permiso := domain.PermisoConcedidoSaldoEfectosC3{SolicitudRef: "permiso:cronos:solicitud:ejemplo-0001", ResolucionRef: "permiso:cronos:resolucion:ejemplo-0001",
		PermisoRef: "permiso:cronos:asistencia-examenes", CatalogoVersionRef: "catalogo:cronos:asistencia-examenes:sintetico-1", Desde: fecha, Hasta: fecha,
		HoraInicio: "10:00", HoraFin: "11:00", CantidadMinutos: 60, ResolucionVersion: 3,
		ResolucionDecisionRef: "decision:ejemplo", ResolucionAuditoriaRef: "auditoria:ejemplo", ResueltaEn: time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC),
		CatalogoFuenteRef: "fuente:catalogo:ejemplo", CatalogoPublicadaEn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	regla := domain.ReglaSaldoEfectosC3{Referencia: "regla:examen:v1", PermisoRef: permiso.PermisoRef, CatalogoVersionRef: permiso.CatalogoVersionRef,
		CentroRef: program.CentroRef, ColectivoRef: program.ColectivoRef, Efecto: domain.EfectoCreditoTiempoAutorizadoC3,
		FuenteRef: "fuente:convenio:dipgra:2006:art36t", FuenteURL: "https://www.dipgra.es/contenidos/normativa-recursos-humanos/",
		FuenteVersionRef: "fuente:convenio:2006:v1", VigenteDesde: "2026-01-01", HastaExclusivo: "2027-01-01", FuentePublicadaEn: "2006-11-30", FuenteConsultadaEn: "2026-10-14"}
	politica := domain.PoliticaSaldoEfectosC3{Referencia: "politica:cronos:efectos:demo", Version: 1, SHA256: strings.Repeat("a", 64), ActoRef: "acto:rrhh:ejemplo",
		PublicadaEn: time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC), Reglas: []domain.ReglaSaldoEfectosC3{regla}}
	return ports.FuenteSaldo{EmpleadoRef: "emp_prueba", Desde: fecha, Hasta: fecha, ZonaHoraria: "Europe/Madrid", Completo: true,
		Jornadas: []ports.JornadaPrevista{{Fecha: fecha, ProgramacionRef: program.Referencia, FuenteRef: program.FuenteRef, Version: 1, TurnoRef: "turno_a", PoliticaVersionRef: "politica_jornada_v1", MinutosPrevistos: 450}},
		Marcajes: []ports.MarcajeSaldo{}, MovimientosSaldo: []ports.MovimientoSaldo{{Fecha: fecha, Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{program.Referencia}}},
		EfectosC3: &ports.FuenteSaldoEfectosC3{SnapshotRef: "saldo:cronos:snapshot:ejemplo", FuenteSHA256: strings.Repeat("b", 64), CorteUTC: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
			Politicas: []domain.PoliticaSaldoEfectosC3{politica}, Programaciones: []domain.ProgramacionSaldoEfectosC3{program}, Permisos: []domain.PermisoConcedidoSaldoEfectosC3{permiso}, CoberturaPermisos: true}}
}

func TestSaldoEfectosC3CreditoConcedidoYFuenteAusentePorFecha(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	f := fuenteSaldoEfectosC3Prueba()
	desde, err := fechaCivilSaldo(f.Desde, zona)
	if err != nil {
		t.Fatal(err)
	}
	base, err := construirConsultaSaldo(ports.PeriodoSaldoRango, desde, desde, f, zona)
	if err != nil {
		t.Fatal(err)
	}
	r, err := aplicarEfectosSaldoC3(base, f, zona)
	if err != nil || r.Detalle[0].SaldoBaseMinutos == nil || *r.Detalle[0].SaldoBaseMinutos != -450 ||
		r.Detalle[0].CreditoPermisoMinutos == nil || *r.Detalle[0].CreditoPermisoMinutos != 60 ||
		r.Detalle[0].SaldoMinutos == nil || *r.Detalle[0].SaldoMinutos != -390 || r.Resumen.SaldoMinutos == nil || *r.Resumen.SaldoMinutos != -390 ||
		r.Detalle[0].ResolucionPermisoRef == "" || r.Detalle[0].PoliticaEfectosVersion != 1 || len(r.Detalle[0].MovimientosEfecto) != 1 ||
		r.Detalle[0].MovimientosEfecto[0].DeltaMinutos != 60 || r.Detalle[0].MovimientosEfecto[0].FuenteRef == "" {
		t.Fatalf("concedido=%+v, err=%v", r, err)
	}
	f.EfectosC3.Politicas = []domain.PoliticaSaldoEfectosC3{}
	r, err = aplicarEfectosSaldoC3(base, f, zona)
	if err != nil || r.Detalle[0].SaldoMinutos != nil || r.Detalle[0].SaldoBaseMinutos == nil || r.Detalle[0].CausaEfecto != "regla_no_disponible" {
		t.Fatalf("sin regla=%+v, err=%v", r, err)
	}
}

func TestSaldoEfectosC3SinReglaAfectaSoloFechaConPermiso(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	f := fuenteSaldoEfectosC3Prueba()
	f.Hasta = "2026-10-21"
	f.Jornadas = append(f.Jornadas, ports.JornadaPrevista{Fecha: f.Hasta, ProgramacionRef: "programacion:cronos:ejemplo2", FuenteRef: "fuente:programacion:ejemplo",
		Version: 1, TurnoRef: "turno_a", PoliticaVersionRef: "politica_jornada_v1", MinutosPrevistos: 450})
	f.MovimientosSaldo = append(f.MovimientosSaldo, ports.MovimientoSaldo{Fecha: f.Hasta, Tipo: "previsto", DeltaMicrosegundos: -450 * 60 * 1000000, Fuentes: []string{"programacion:cronos:ejemplo2"}})
	f.EfectosC3.Politicas = []domain.PoliticaSaldoEfectosC3{}
	desde, _ := fechaCivilSaldo(f.Desde, zona)
	hasta, _ := fechaCivilSaldo(f.Hasta, zona)
	base, err := construirConsultaSaldo(ports.PeriodoSaldoRango, desde, hasta, f, zona)
	if err != nil {
		t.Fatal(err)
	}
	r, err := aplicarEfectosSaldoC3(base, f, zona)
	if err != nil || len(r.Detalle) != 2 || r.Detalle[0].SaldoMinutos != nil || r.Detalle[0].CausaEfecto != "regla_no_disponible" ||
		r.Detalle[1].SaldoMinutos == nil || *r.Detalle[1].SaldoMinutos != -450 || r.Resumen.SaldoMinutos != nil {
		t.Fatalf("fuente ausente afecta fechas ajenas: %+v, %v", r, err)
	}
}
