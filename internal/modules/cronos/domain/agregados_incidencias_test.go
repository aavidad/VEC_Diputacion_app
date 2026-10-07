package domain

import (
	"testing"
	"time"
)

func TestAgregadosIncidenciasDistintosYCero(t *testing.T) {
	personas, dias := []string{"demo:persona:1", "demo:persona:2"}, []string{"2026-09-28", "2026-09-29"}
	var cobertura []UnidadCoberturaIncidencias
	for _, p := range personas {
		for _, d := range dias {
			cobertura = append(cobertura, UnidadCoberturaIncidencias{p, d, CoberturaIncidenciasCompleta})
		}
	}
	hechos := []HechoIncidenciaAgregada{
		{"demo:incidencia:1", personas[0], dias[0], RegistroIncompleto, IncidenciaPendiente},
		{"demo:incidencia:2", personas[0], dias[0], RegistroIncompleto, IncidenciaResuelta},
		{"demo:incidencia:3", personas[1], dias[0], RegistroIncompleto, IncidenciaPendiente},
		{"demo:incidencia:4", personas[0], dias[1], RegistroIncompleto, IncidenciaPendiente},
	}
	r, err := AgregarIncidenciasRegistradas(personas, dias, cobertura, hechos)
	if err != nil || r.IncidenciasObservadas != 4 || r.PersonasAfectadasObservadas != 2 || r.DiasAfectadosObservados != 2 || r.PersonasDiasAfectadosObservados != 3 || r.TotalIncidencias == nil || *r.TotalIncidencias != 4 || r.Grupos[0].PersonasDiasAfectadosObservados != 3 {
		t.Fatalf("aggregate: %+v %v", r, err)
	}
	r, err = AgregarIncidenciasRegistradas(personas, dias, cobertura, []HechoIncidenciaAgregada{})
	if err != nil || r.TotalIncidencias == nil || *r.TotalIncidencias != 0 || r.Estado != "completo" {
		t.Fatalf("covered zero: %+v %v", r, err)
	}
	for _, estado := range []CoberturaIncidencias{CoberturaIncidenciasIncompleta, CoberturaIncidenciasDesconocida} {
		for i := range cobertura {
			cobertura[i].Estado = estado
		}
		r, err = AgregarIncidenciasRegistradas(personas, dias, cobertura, []HechoIncidenciaAgregada{})
		if err != nil || r.TotalIncidencias != nil || r.Grupos[0].TotalIncidencias != nil || r.Estado == "completo" {
			t.Fatalf("unknown total: %+v %v", r, err)
		}
	}
}

func TestAgregadosIncidenciasRechazaAmbiguedad(t *testing.T) {
	p, d := []string{"demo:persona:1"}, []string{"2026-09-28"}
	c := []UnidadCoberturaIncidencias{{p[0], d[0], CoberturaIncidenciasCompleta}}
	h := []HechoIncidenciaAgregada{{"demo:incidencia:1", p[0], d[0], RegistroIncompleto, IncidenciaPendiente}}
	for _, v := range []struct {
		p, d []string
		c    []UnidadCoberturaIncidencias
		h    []HechoIncidenciaAgregada
	}{
		{nil, d, c, h}, {[]string{"*"}, d, c, h}, {[]string{p[0], p[0]}, d, append(c, c...), h},
		{p, d, c, append(h, h...)}, {p, d, nil, h}, {p, []string{d[0], d[0]}, append(c, c...), h},
		{p, d, c, nil}, {p, d, c, []HechoIncidenciaAgregada{{"demo:incidencia:1", p[0], "2026-09-29", RegistroIncompleto, IncidenciaPendiente}}},
		{p, d, c, []HechoIncidenciaAgregada{{"demo:incidencia:1", p[0], d[0], "absentismo", IncidenciaPendiente}}},
	} {
		if _, err := AgregarIncidenciasRegistradas(v.p, v.d, v.c, v.h); err == nil {
			t.Fatal("accepted ambiguous facts")
		}
	}
}

func TestPeriodoAgregadosLimitesYHorarioVerano(t *testing.T) {
	corte := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	dias, err := DiasPeriodoAgregados("2026-03-28", "2026-03-30", "Europe/Madrid", corte)
	if err != nil || len(dias) != 2 || dias[0] != "2026-03-28" || dias[1] != "2026-03-29" {
		t.Fatalf("DST: %v %v", dias, err)
	}
	for _, v := range []struct {
		desde, hasta, zona string
		corte              time.Time
	}{
		{"2026-03-28", "2026-03-28", "Europe/Madrid", corte},
		{"2026-03-28", "2026-03-30", "Local", corte},
		{"2026-03-28", "2026-03-30", "Europe/Madrid", corte.Add(-3 * time.Hour)},
		{"2026-3-28", "2026-03-30", "Europe/Madrid", corte},
		{"2025-03-28", "2026-03-30", "Europe/Madrid", corte},
	} {
		if _, err := DiasPeriodoAgregados(v.desde, v.hasta, v.zona, v.corte); err == nil {
			t.Fatal("accepted invalid period")
		}
	}
}
