package main

import (
	"testing"
	"time"

	dominio "vec-diputacion-granada/internal/vec/domain"
)

func TestConstruirRevocacionConservaPreimagenYUsaHuellaGo(t *testing.T) {
	base := time.Date(2026, 9, 23, 21, 0, 0, 0, time.UTC)
	a := dominio.AsignacionPerfil{
		AsignacionID: "dietas_r1d_0123456789abcdef", Version: 1,
		PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef",
		PrincipalID:     "per_0123456789abcdef0123456789abcdef",
		VersionRolRef:   "rol:dietas_r1d_provisional:v1",
		Estado:          dominio.EstadoAsignacionPerfilActiva,
		Ambitos: []dominio.AmbitoPerfil{
			{Clave: "empleado_ref", Valores: []string{"emp_0123456789abcdef0123456789abcdef"}},
			{Clave: "persona_ref", Valores: []string{"per_0123456789abcdef0123456789abcdef"}},
		},
		VigenteDesde: base.Add(-time.Hour), VigenteHasta: base.Add(time.Hour),
		EmitidaPor: "desarrollo:dietas-r1d:provisional", EmitidaEn: base.Add(-time.Hour),
	}
	h, err := a.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	p, err := construir(preimagen{a.Referencia(), h, a}, base)
	if err != nil {
		t.Fatal(err)
	}
	if p.NuevaRef != "asignacion:dietas_r1d_0123456789abcdef:v2" ||
		p.Documento.Estado != dominio.EstadoAsignacionPerfilRevocada ||
		p.Documento.PerfilActivoRef != a.PerfilActivoRef ||
		p.Documento.PrincipalID != a.PrincipalID ||
		p.Documento.VersionRolRef != a.VersionRolRef ||
		p.Documento.VigenteDesde != a.VigenteDesde ||
		p.Documento.VigenteHasta != a.VigenteHasta ||
		p.Documento.EmitidaEn != a.EmitidaEn ||
		p.Documento.RevocacionRef != acto {
		t.Fatal("revocacion no conserva identidad/fechas")
	}
	calculada, err := p.Documento.HuellaSHA256()
	if err != nil || calculada != p.NuevaHuella {
		t.Fatal("huella no coincide con dominio")
	}
	if _, err := construir(preimagen{a.Referencia(), "0", a}, base); err == nil {
		t.Fatal("preimagen con huella falsa aceptada")
	}
}
