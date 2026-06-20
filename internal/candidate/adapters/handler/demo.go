package handler

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/usecases"
)

type ProcedureUseCase = *usecases.ProcedureUseCase

type ProcedureDemoRunner interface {
	Run(ctx context.Context) (ProcedureDemoView, error)
}

type procedureDemoRunner struct {
	usecase *usecases.ProcedureUseCase
}

func NewProcedureDemoRunner(usecase ProcedureUseCase) ProcedureDemoRunner {
	if usecase == nil {
		return nil
	}
	return procedureDemoRunner{usecase: usecase}
}

func (r procedureDemoRunner) Run(ctx context.Context) (ProcedureDemoView, error) {
	return runDemoProcedure(ctx, r.usecase)
}

func (h *Handler) handleDemo(w http.ResponseWriter, r *http.Request) {
	if h.demoRunner == nil {
		h.writeError(w, http.StatusNotFound, "api.error.not_found", errInvalidRoute)
		return
	}
	result, err := h.demoRunner.Run(r.Context())
	if err != nil {
		h.writeError(w, statusFromError(err), errorKey(err), err)
		return
	}
	h.writeJSON(w, http.StatusOK, responseEnvelope{
		Message: h.t("api.procedure.demo_completed"),
		Data:    result,
	})
}

func runDemoProcedure(ctx context.Context, usecase *usecases.ProcedureUseCase) (ProcedureDemoView, error) {
	id := "demo-convocatoria"
	rules, err := ruleSetFor(id, "v1")
	if err != nil {
		return ProcedureDemoView{}, err
	}
	record, err := usecase.CrearConvocatoria(ctx, usecases.CrearConvocatoriaCommand{
		ID:      id,
		Version: "v1",
		RuleSet: rules,
	})
	if err != nil {
		return ProcedureDemoView{}, err
	}
	if _, err := usecase.RegistrarSolicitud(ctx, buildDemoSolicitud("demo-sol-1", "demo-cand-a", "Lopez", 24, 0)); err != nil {
		return ProcedureDemoView{}, err
	}
	if _, err := usecase.RegistrarSolicitud(ctx, buildDemoSolicitud("demo-sol-2", "demo-cand-b", "Navas", 0, 20)); err != nil {
		return ProcedureDemoView{}, err
	}

	admitidas := map[string]bool{"demo-sol-1": true, "demo-sol-2": true}
	provisional, err := usecase.PublicarListadoProvisional(ctx, id, admitidas)
	if err != nil {
		return ProcedureDemoView{}, err
	}
	definitivo, err := usecase.PublicarListadoDefinitivo(ctx, id, admitidas)
	if err != nil {
		return ProcedureDemoView{}, err
	}
	return ProcedureDemoView{
		Convocatoria: convocatoriaView(record.Convocatoria),
		Provisional:  listadoView(provisional),
		Definitivo:   listadoView(definitivo),
	}, nil
}

func buildDemoSolicitud(
	id string,
	candidateID string,
	sorteoKey string,
	meses int,
	horas int,
) usecases.RegistrarSolicitudCommand {
	tipo := domain.MeritTypeExperienciaMismaCategoria
	datos := domain.MeritData{Meses: meses}
	if horas > 0 {
		tipo = domain.MeritTypeFormacionCurso
		datos = domain.MeritData{Horas: horas}
	}
	return usecases.RegistrarSolicitudCommand{
		ID:             id,
		ConvocatoriaID: "demo-convocatoria",
		CandidateID:    candidateID,
		SorteoKey:      sorteoKey,
		Merits: []domain.Merit{{
			ID:     "merit-" + id,
			Tipo:   tipo,
			Datos:  datos,
			Estado: domain.MeritStateValidado,
		}},
	}
}

func convocatoriaView(convocatoria domain.Convocatoria) ConvocatoriaView {
	return ConvocatoriaView{
		ID:      convocatoria.ID,
		Version: convocatoria.Version,
		Estado:  convocatoria.Estado,
	}
}

func listadoView(listado usecases.Listado) ListadoView {
	view := ListadoView{
		ConvocatoriaID: listado.ConvocatoriaID,
		Version:        listado.Version,
		Items:          make([]ListadoItemView, 0, len(listado.Items)),
	}
	for _, item := range listado.Items {
		view.Items = append(view.Items, ListadoItemView{
			SolicitudID: item.SolicitudID,
			CandidateID: item.CandidateID,
			Estado:      item.Estado,
			TotalPoints: item.Result.TotalPoints,
			Rank:        item.Rank,
		})
	}
	return view
}
