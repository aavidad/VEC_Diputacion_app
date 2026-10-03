package domain

import (
	"strings"
	"testing"
)

func planFirmaPruebaV2() PlanCompetenciaFirmaV2 {
	v := VersionPlanFirmaV2{"catalogo:prueba", 1, strings.Repeat("a", 64)}
	return PlanCompetenciaFirmaV2{Version: v, FuenteRef: "fuente:prueba", Pasos: []CompetenciaPasoFirmaV2{{EntradaClave: "entrada.uno", Circuito: v, Documento: "resolucion", PasoRef: "paso.uno", PasoOrden: 1, PerfilEsperadoRef: "perfil:logico", RolID: "rol.central", CargoRef: "cargo:prueba", OrganizacionRef: "org:prueba", UnidadRef: "unidad:prueba", Accion: "documento.firmar", Finalidad: "formalizar", TipoRecurso: "documento", EsquemaContexto: "contexto.v1", MapeoVersion: 1, MapeoFuenteRef: "fuente:prueba"}}}
}

func TestPlanFirmaV2SeleccionExacta(t *testing.T) {
	p := planFirmaPruebaV2()
	e := p.Pasos[0]
	got, err := p.Seleccionar(e.Circuito, e.Documento, e.PasoRef, e.PerfilEsperadoRef, e.OrganizacionRef, e.UnidadRef)
	if err != nil || got != e {
		t.Fatalf("seleccion exacta: %v", err)
	}
	for _, cambio := range []func(*CompetenciaPasoFirmaV2){func(v *CompetenciaPasoFirmaV2) { v.Circuito.Version++ }, func(v *CompetenciaPasoFirmaV2) { v.Circuito.HuellaSHA256 = strings.Repeat("b", 64) }, func(v *CompetenciaPasoFirmaV2) { v.PerfilEsperadoRef = "perfil:otro" }, func(v *CompetenciaPasoFirmaV2) { v.UnidadRef = "unidad:otra" }} {
		otro := e
		cambio(&otro)
		if _, err := p.Seleccionar(otro.Circuito, otro.Documento, otro.PasoRef, otro.PerfilEsperadoRef, otro.OrganizacionRef, otro.UnidadRef); err == nil {
			t.Fatal("acepto selector diferente")
		}
	}
}

func TestPlanFirmaV2DeniegaMapeoAusenteYAmbiguo(t *testing.T) {
	for _, mutar := range []func(*PlanCompetenciaFirmaV2){func(p *PlanCompetenciaFirmaV2) { p.Pasos[0].RolID = "" }, func(p *PlanCompetenciaFirmaV2) { p.Pasos[0].CargoRef = "*" }, func(p *PlanCompetenciaFirmaV2) { p.Pasos[0].MapeoFuenteRef = "" }, func(p *PlanCompetenciaFirmaV2) { p.Pasos[0].MapeoVersion = 0 }, func(p *PlanCompetenciaFirmaV2) {
		e := p.Pasos[0]
		e.EntradaClave = "entrada.dos"
		e.RolID = "otro.rol"
		p.Pasos = append(p.Pasos, e)
	}} {
		p := planFirmaPruebaV2()
		mutar(&p)
		if p.Validar() == nil {
			t.Fatal("acepto mapeo invalido o ambiguo")
		}
	}
}
