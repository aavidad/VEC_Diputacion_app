package application

import (
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

func TestPaginaRPTPublicaV2CuentaAntesDePaginarYConservaIncertidumbre(t *testing.T) {
	s := domain.SnapshotRPTPublicaV2{
		PublicacionRef: "rpt-publicada:2026-05-07", Corte: "2026-05-07", HuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Catalogo: domain.CatalogoRPTPublicaV2{
			Estado:     domain.EstadoCandidatoRPTPublicaV2,
			Categorias: []domain.CategoriaRPTPublicaV2{{Clave: "tecnico", Denominacion: "TÉCNICO"}},
			Puestos: []domain.PuestoRPTPublicoV2{
				{PuestoRPTPublico: domain.PuestoRPTPublico{Codigo: "430-101-001", Denominacion: "TÉCNICO DE GESTIÓN", CategoriaClave: "tecnico", CentroCodigo: "101"}, CategoriasClaves: []string{"tecnico"}, CategoriasPendientes: []domain.CategoriaPendienteRPTPublicaV2{{Denominacion: "AUXILIAR", Origen: "categoria"}}},
				{PuestoRPTPublico: domain.PuestoRPTPublico{Codigo: "430-101-002", Denominacion: "TÉCNICO DE INFORMÁTICA", CentroCodigo: "102"}, CategoriasClaves: []string{"tecnico"}},
				{PuestoRPTPublico: domain.PuestoRPTPublico{Codigo: "430-101-003", Denominacion: "ADMINISTRATIVO"}},
			},
			CategoriasPendientesGrupo: []string{"AUXILIAR"},
		},
	}
	p := paginaRPTPublicaV2(s, domain.FiltroRPTPublicaV2{Vista: "puestos", Q: "tecnico", Limite: 1, Offset: 1}, ports.EvidenciaLecturaRPTPublicaV2{})
	if p.Total != 2 || len(p.Puestos) != 1 || p.Puestos[0].Codigo != "430-101-002" || p.Estado != domain.EstadoCandidatoRPTPublicaV2 || len(p.CategoriasPendientesGrupo) != 1 {
		t.Fatalf("pagina=%+v", p)
	}
	p = paginaRPTPublicaV2(s, domain.FiltroRPTPublicaV2{Vista: "categorias", Q: "tecnico", Limite: 1, Offset: 4}, ports.EvidenciaLecturaRPTPublicaV2{})
	if p.Total != 1 || len(p.Categorias) != 0 || p.Categorias == nil {
		t.Fatalf("categoria fuera de página=%+v", p)
	}
	p = paginaRPTPublicaV2(s, domain.FiltroRPTPublicaV2{Vista: "puestos", CategoriaClave: "tecnico", Limite: 25}, ports.EvidenciaLecturaRPTPublicaV2{})
	if p.Total != 1 || len(p.Puestos) != 1 || p.Puestos[0].Codigo != "430-101-001" {
		t.Fatalf("una alternativa sin asignación singular duplicó la lista=%+v", p)
	}
	p = paginaRPTPublicaV2(s, domain.FiltroRPTPublicaV2{Vista: "puestos", CentroCodigo: "102", Limite: 25}, ports.EvidenciaLecturaRPTPublicaV2{})
	if p.Total != 1 || len(p.Puestos) != 1 || p.Puestos[0].Codigo != "430-101-002" {
		t.Fatalf("centro filtrado=%+v", p)
	}
}
