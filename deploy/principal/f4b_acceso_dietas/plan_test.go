package main

import (
	"testing"
	"time"

	dominio "vec-diputacion-granada/internal/vec/domain"
)

func TestPlanF4bPreimagenYVigencia(t *testing.T) {
	emision := time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC)
	ahora := emision.Add(24 * time.Hour)
	original := dominio.AsignacionPerfil{
		AsignacionID: "dietas_r1d_0123456789abcdef", Version: 1,
		PerfilActivoRef: "prf_0123456789abcdef0123456789abcdef",
		PrincipalID:     "per_0123456789abcdef0123456789abcdef",
		VersionRolRef:   "rol:dietas_r1d_provisional:v1",
		Estado:          dominio.EstadoAsignacionPerfilActiva,
		Ambitos:         []dominio.AmbitoPerfil{{Clave: "empleado_ref", Valores: []string{"emp_0123456789abcdef"}}},
		VigenteDesde:    emision, VigenteHasta: emision.Add(180 * 24 * time.Hour),
		EmitidaPor: "desarrollo:dietas-r1d:provisional", EmitidaEn: emision,
	}
	revocada := original
	revocada.Version = 2
	revocada.Estado = dominio.EstadoAsignacionPerfilRevocada
	revocada.RevocadaPor = actorP6
	revocada.RevocadaEn = emision.Add(time.Hour)
	revocada.RevocacionRef = actoP6
	h1, err := original.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := revocada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	p := preimagen{
		Original:   fila{Ref: original.Referencia(), Huella: h1, Documento: original},
		Revocada:   fila{Ref: revocada.Referencia(), Huella: h2, Documento: revocada},
		PunteroPor: actorP6, PunteroActo: actoP6,
	}
	plan, err := construir(p, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Documento.Version != 3 || !plan.Documento.VigenteDesde.Equal(ahora) ||
		plan.Documento.EmitidaPor != actorF4 || plan.Documento.Estado != dominio.EstadoAsignacionPerfilActiva ||
		plan.NuevaHuella == h2 {
		t.Fatal("version activa F4b incorrecta")
	}
	alterada := p
	alterada.Revocada.Huella = h1
	if _, err := construir(alterada, ahora); err == nil {
		t.Fatal("acepto huella de P6 distinta")
	}
	alterada = p
	alterada.PunteroActo = "acto:ajeno"
	if _, err := construir(alterada, ahora); err == nil {
		t.Fatal("acepto puntero ajeno")
	}
	if _, err := construir(p, original.VigenteHasta); err == nil {
		t.Fatal("acepto vigencia vencida")
	}

	retirada, err := construirRetirada(preimagenRetirada{
		Activa:     fila{Ref: plan.NuevaRef, Huella: plan.NuevaHuella, Documento: plan.Documento},
		PunteroPor: actorF4, PunteroActo: actoF4,
	}, ahora.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if retirada.Documento.Version != 4 || retirada.Documento.Estado != dominio.EstadoAsignacionPerfilRevocada ||
		retirada.Documento.RevocadaPor != actorRetiradaF4 || retirada.Actor != actorRetiradaF4 {
		t.Fatal("retirada F4b incorrecta")
	}
	if _, err := construirRetirada(preimagenRetirada{
		Activa:     fila{Ref: plan.NuevaRef, Huella: plan.NuevaHuella, Documento: plan.Documento},
		PunteroPor: "administracion:f4:reactivacion-dietas-r1d", PunteroActo: "acto:f4:reactivacion-dietas-r1d:20260924",
	}, ahora.Add(time.Hour)); err == nil {
		t.Fatal("retirada F4b acepto puntero de F4")
	}
}
