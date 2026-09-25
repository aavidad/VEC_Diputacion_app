package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type repositorioCircuitoPrueba struct{ llamadas int }

func (r *repositorioCircuitoPrueba) Decidir(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudDecisionCircuito) (dietasports.ResultadoCircuitoComision, error) {
	r.llamadas++
	return dietasports.ResultadoCircuitoComision{}, nil
}
func (r *repositorioCircuitoPrueba) ListarPendientes(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.ConsultaBandejaCircuito) (dietasports.PaginaBandejaCircuito, error) {
	r.llamadas++
	return dietasports.PaginaBandejaCircuito{}, nil
}

func TestCircuitoDistingueAccionesYExigeUnidadInterna(t *testing.T) {
	ref := "dco_" + strings.Repeat("a", 22)
	decision := dietasports.SolicitudDecisionCircuito{Referencia: ref, UnidadRef: "unidad:uno", Etapa: domain.EtapaRevision, Decision: domain.DecisionDevolver, Motivo: "Falta documento", ClaveIdempotencia: "clave_0123456789abcdef", VersionEsperada: 2}
	if err := ValidarSolicitudDecisionCircuito(decision); err != nil {
		t.Fatal(err)
	}
	accion, recurso, finalidad := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionDecidirCircuito, Decision: decision})
	if accion != "dietas.documento.revisar" || recurso != ref || finalidad != "revisar_documento_dietas" {
		t.Fatalf("contrato D6: %s %s %s", accion, recurso, finalidad)
	}
	consulta := dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaRevision, UnidadRef: "unidad:uno", Limite: 20, FechaDesde: "2026-09-01", FechaHasta: "2026-09-30"}
	a, r, f := ContratoCircuito(dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionListarBandeja, Consulta: consulta})
	if a != "dietas.bandeja.revision.consultar" || r != "dietas:bandeja:revision" || f != "consultar_bandeja_revision_dietas" {
		t.Fatalf("contrato D8: %s %s %s", a, r, f)
	}
	decision.UnidadRef = ""
	if !errors.Is(ValidarSolicitudDecisionCircuito(decision), domain.ErrDecisionCircuitoInvalida) {
		t.Fatal("decisión sin unidad interna aceptada")
	}
	consulta.FechaDesde = "2026-10-01"
	if !errors.Is(ValidarConsultaBandejaCircuito(consulta), domain.ErrDecisionCircuitoInvalida) {
		t.Fatal("rango invertido aceptado")
	}
}

func TestCircuitoNoLlamaRepositorioSinIdentidadV3(t *testing.T) {
	repo := &repositorioCircuitoPrueba{}
	s, err := NuevoServicioCircuitoComision(repo)
	if err != nil {
		t.Fatal(err)
	}
	decision := dietasports.SolicitudDecisionCircuito{Referencia: "dco_" + strings.Repeat("a", 22), UnidadRef: "unidad:uno", Etapa: domain.EtapaRevision, Decision: domain.DecisionAprobar, ClaveIdempotencia: "clave_0123456789abcdef", VersionEsperada: 1}
	_, err = s.Decidir(context.Background(), dietasports.IdentidadEfectivaCircuito{}, decision)
	if !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || repo.llamadas != 0 {
		t.Fatalf("decisión sin V3: %v %d", err, repo.llamadas)
	}
	_, err = s.ListarPendientes(context.Background(), dietasports.IdentidadEfectivaCircuito{}, dietasports.ConsultaBandejaCircuito{Etapa: domain.EtapaRevision, UnidadRef: "unidad:uno", Limite: 20})
	if !errors.Is(err, dietasports.ErrAccesoCircuitoDenegado) || repo.llamadas != 0 {
		t.Fatalf("lista sin V3: %v %d", err, repo.llamadas)
	}
}
