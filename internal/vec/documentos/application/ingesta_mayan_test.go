package application

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type repositorioIngestaPrueba struct {
	ports.RepositorioIngestaMayan
	plan      ports.PlanIngestaMayan
	despachos int
}

func (r *repositorioIngestaPrueba) ConsultarIngesta(_ context.Context, _ ports.OrdenConsultarIngestaMayan) (ports.PlanIngestaMayan, error) {
	return r.plan, nil
}
func (r *repositorioIngestaPrueba) DespacharIngesta(_ context.Context, _ ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, bool, error) {
	r.despachos++
	return ports.PlanIngestaMayan{}, false, nil
}

type autoridadIngestaPrueba struct {
	t     *testing.T
	ahora time.Time
}

func (a autoridadIngestaPrueba) AutorizarIngestaMayan(_ context.Context, accion string, preimagen []byte, planRef, expedienteRef string) (ports.AutorizacionV3, error) {
	return ports.AutorizacionV3{Material: materialExternaPrueba(a.t, accion, planRef, preimagen, a.ahora),
		Accion: accion, Finalidad: ports.FinalidadIngestaMayan, RecursoRef: planRef, AmbitoRef: expedienteRef,
		PrincipalID: "per:0001", PerfilActivoRef: "perfil:0001", CorrelacionRef: "corr:0001"}, nil
}

type custodioIngestaPrueba struct{ llamadas int }

func (c *custodioIngestaPrueba) IngerirOriginal(context.Context, ports.PlanIngestaMayan, io.Reader) (ports.DocumentoMayan, error) {
	c.llamadas++
	return ports.DocumentoMayan{}, errors.New("POST inesperado")
}
func (*custodioIngestaPrueba) ReconciliarOriginal(context.Context, ports.DocumentoMayan) (ports.DocumentoMayan, error) {
	panic("reconciliación inesperada")
}

type cargasIngestaPrueba struct {
	vecports.RepositorioCargasDocumentales
}

func TestIngestaIndeterminadaNuncaReenviaPOST(t *testing.T) {
	ahora := time.Date(2026, 10, 11, 18, 0, 0, 0, time.UTC)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	repo := &repositorioIngestaPrueba{plan: ports.PlanIngestaMayan{PlanRef: ref("1"), ExpedienteRef: ref("2"), Estado: ports.EstadoIngestaIndeterminada, Version: 3}}
	custodio := &custodioIngestaPrueba{}
	s := ServicioIngestaMayan{Repositorio: repo, Autoridad: autoridadIngestaPrueba{t, ahora}, Custodio: custodio, Cargas: cargasIngestaPrueba{}, Reloj: relojExternaPrueba{ahora}}
	_, err := s.Ingerir(context.Background(), repo.plan.PlanRef, repo.plan.ExpedienteRef, strings.NewReader("bytes"))
	if !errors.Is(err, ports.ErrIngestaNoReenviable) || custodio.llamadas != 0 || repo.despachos != 0 {
		t.Fatalf("replay indeterminado: err=%v POST=%d despachos=%d", err, custodio.llamadas, repo.despachos)
	}
}
