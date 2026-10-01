package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type repositorioPreparacionRPTPrueba struct {
	repositorioPreparacionCERPrueba
}

func (r repositorioPreparacionRPTPrueba) ConsultarFichaRRHH(ctx context.Context, o ports.OrdenFichaEmpleadoB2) (ports.ResultadoFichaEmpleadoB2, error) {
	resultado, err := r.repositorioPreparacionCERPrueba.ConsultarFichaRRHH(ctx, o)
	if err != nil {
		return resultado, err
	}
	snapshot := func(tipo, ref string) *domain.SnapshotEntradaCatalogoEmpleadoB2 {
		return &domain.SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: o.Material.OrganismoRef(), Tipo: tipo, Ref: ref, Version: 1, Revision: 1, Denominacion: "Entrada sintética", HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: "2020-01-01", Estado: "publicada"}
	}
	anterior := domain.RelacionRegistroEmpleadoB2{RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: o.Material.OrganismoRef(), UnidadRef: "unidad:uno", RegimenRef: "regimen:uno", ModalidadRef: "modalidad:uno", Estado: "vigente",
		Traza:            domain.TrazaEmpleadoB2{Desde: "2024-01-01", RegistradaEn: o.Material.Corte().ConocidoEn.Add(-2 * time.Hour), Version: 1, ActoRef: "acto:uno", FuenteRef: "fuente:uno", FuenteVersion: 1},
		CatalogoSnapshot: domain.SnapshotCatalogoEmpleadoB2{Regimen: snapshot("regimen", "regimen:uno"), Modalidad: snapshot("modalidad", "modalidad:uno")}}
	ultima := anterior
	ultima.Estado = "finalizada"
	ultima.Traza.Version = 2
	ultima.Traza.RegistradaEn = o.Material.Corte().ConocidoEn.Add(-time.Hour)
	ultima.Traza.Hasta = o.Material.Corte().VigenteEn
	resultado.Ficha.Relaciones = []domain.RelacionRegistroEmpleadoB2{ultima, anterior}
	resultado.PreparacionRPT = &domain.PreparacionRelacionParaRPT{Cobertura: "completa", EstadoRPT: "ocupada"}
	return resultado, nil
}

func TestRegistroB2PreparacionRPTTrasV3ConservaLaUltimaRevision(t *testing.T) {
	for _, caso := range []string{"valida", "denegada", "evidencia", "ficha", "firma"} {
		t.Run(caso, func(t *testing.T) {
			var pasos []string
			actor := solicitudP(t).Actor
			s, err := NuevoServicioRegistroEmpleadoB2(autorizadorPreparacionCERPrueba{t: t, denegar: caso == "denegada", pasos: &pasos}, repositorioPreparacionRPTPrueba{repositorioPreparacionCERPrueba{pasos: &pasos, invalido: caso}})
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := s.ConsultarFicha(context.Background(), domain.SolicitudFichaEmpleadoB2{Actor: actor, OrganismoRef: "organismo:uno", EmpleadoRef: "emp_" + strings.Repeat("e", 24), Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-09-25", ConocidoEn: actor.ResueltoEn}})
			if caso != "valida" {
				if err == nil || resultado.PreparacionRPT != nil || resultado.PreparacionServicios != nil || resultado.Ficha.EmpleadoRef != "" {
					t.Fatalf("datos tras fallo: %+v, %v", resultado, err)
				}
				if caso == "denegada" && (len(pasos) != 1 || !errors.Is(err, domain.ErrRegistroEmpleadoB2Denegado)) {
					t.Fatalf("lectura tras denegación: %v, %v", pasos, err)
				}
				return
			}
			p := resultado.PreparacionRPT
			if err != nil || p == nil || resultado.PreparacionServicios == nil || len(pasos) != 2 || pasos[1] != "lectura_v3_auditada" {
				t.Fatalf("preparación sin lectura V3: %+v,%v", resultado, err)
			}
			if p.EmpleadoRef != resultado.Ficha.EmpleadoRef || p.VersionFicha != resultado.Ficha.Version || p.Corte != resultado.Ficha.Corte || p.Cobertura != "no_acreditada" || p.EstadoRPT != "pendiente_fuente_rpt" {
				t.Fatalf("fuente o vínculo incorrectos: %+v", p)
			}
			if len(p.Relaciones) != 1 || p.Relaciones[0].Estado != "finalizada" || p.Relaciones[0].EnIntervalo || p.Relaciones[0].Traza.Version != 2 {
				t.Fatalf("resucitó relación anterior: %+v", p.Relaciones)
			}
			if len(resultado.Ficha.Relaciones) != 2 || resultado.Ficha.Relaciones[1].Estado != "vigente" {
				t.Fatal("alteró historia original")
			}
		})
	}
}
