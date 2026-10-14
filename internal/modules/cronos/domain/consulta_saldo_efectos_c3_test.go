package domain

import (
	"strings"
	"testing"
	"time"
)

func politicaSaldoEfectosC3Prueba() PoliticaSaldoEfectosC3 {
	return PoliticaSaldoEfectosC3{Referencia: "politica:cronos:efectos:v1", Version: 1, SHA256: strings.Repeat("a", 64), ActoRef: "acto:rrhh:ejemplo",
		PublicadaEn: time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC), Reglas: []ReglaSaldoEfectosC3{{
			Referencia: "regla:examen:v1", PermisoRef: "permiso:cronos:asistencia-examenes", CatalogoVersionRef: "catalogo:cronos:asistencia-examenes:sintetico-1",
			CentroRef: "centro:demo", ColectivoRef: "regimen:laboral", Efecto: EfectoCreditoTiempoAutorizadoC3, FuenteRef: "fuente:convenio:dipgra:2006:art36t",
			FuenteURL: "https://www.dipgra.es/contenidos/normativa-recursos-humanos/", FuenteVersionRef: "fuente:convenio:2006:v1",
			VigenteDesde: "2026-01-01", HastaExclusivo: "2027-01-01", FuentePublicadaEn: "2006-11-30", FuenteConsultadaEn: "2026-10-14",
		}}}
}

func TestSaldoEfectosC3CreditoParcialSinAlterarJornada(t *testing.T) {
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	p := politicaSaldoEfectosC3Prueba()
	inicio := time.Date(2026, 10, 20, 6, 0, 0, 0, time.UTC)
	permiso := PermisoConcedidoSaldoEfectosC3{SolicitudRef: "permiso:cronos:solicitud:ejemplo-0001", ResolucionRef: "permiso:cronos:resolucion:ejemplo-0001",
		PermisoRef: p.Reglas[0].PermisoRef, CatalogoVersionRef: p.Reglas[0].CatalogoVersionRef, ColectivoRef: p.Reglas[0].ColectivoRef, CentroRef: "centro:demo",
		Desde: "2026-10-20", Hasta: "2026-10-20", HoraInicio: "10:00", HoraFin: "11:00", CantidadMinutos: 60, ResolucionVersion: 3,
		ResolucionDecisionRef: "decision:ejemplo", ResolucionAuditoriaRef: "auditoria:ejemplo", ResueltaEn: time.Date(2026, 10, 19, 8, 0, 0, 0, time.UTC),
		CatalogoFuenteRef: "fuente:catalogo:ejemplo", CatalogoPublicadaEn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	d := DiaSaldoEfectosC3{Fecha: "2026-10-20", ConocidoEn: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC), Programacion: &ProgramacionSaldoEfectosC3{Fecha: "2026-10-20", CentroRef: "centro:demo", ColectivoRef: "regimen:laboral", Referencia: "programacion:cronos:ejemplo", FuenteRef: "fuente:programacion:ejemplo", Version: 1, Minutos: 450,
		InicioUTC: inicio, FinUTC: inicio.Add(450 * time.Minute)}, Completo: true, SinTrabajo: true, Permisos: []PermisoConcedidoSaldoEfectosC3{permiso}}
	r, err := CalcularSaldoEfectosC3(d, p, zona)
	if err != nil || r.Causa != "permiso_computado" || r.SaldoBaseMinutos == nil || *r.SaldoBaseMinutos != -450 || r.CreditoPermisoMinutos == nil || *r.CreditoPermisoMinutos != 60 || r.SaldoAjustadoMinutos == nil || *r.SaldoAjustadoMinutos != -390 || d.Programacion.Minutos != 450 {
		t.Fatalf("efecto=%+v, err=%v", r, err)
	}
	d.SinTrabajo = false
	d.TrabajadoMinutos = 60
	r, err = CalcularSaldoEfectosC3(d, p, zona)
	if err != nil || r.Causa != "trabajo_concurrente" || r.CreditoPermisoMinutos != nil || r.SaldoAjustadoMinutos != nil {
		t.Fatalf("solape de trabajo: %+v %v", r, err)
	}
	d.SinTrabajo = true
	d.TrabajadoMinutos = 0
	d.Permisos = nil
	r, err = CalcularSaldoEfectosC3(d, p, zona)
	if err != nil || r.SaldoAjustadoMinutos != nil || r.Causa != "fuente_no_determinable" {
		t.Fatalf("permiso sin cobertura: %+v %v", r, err)
	}
}
