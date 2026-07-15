package handler

import (
	"context"
	"fmt"
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
	admitidasByConvocatoria := map[string]map[string]bool{}
	for number := 1; number <= demoPuestoFixtureCount; number++ {
		convocatoriaID := demoPuestoID(number)
		rules, err := ruleSetFor(convocatoriaID, "v1")
		if err != nil {
			return ProcedureDemoView{}, err
		}
		if _, err := usecase.EnsureConvocatoria(ctx, usecases.CrearConvocatoriaCommand{
			ID:      convocatoriaID,
			Version: "v1",
			RuleSet: rules,
		}); err != nil {
			return ProcedureDemoView{}, err
		}
		admitidasByConvocatoria[convocatoriaID] = map[string]bool{}
	}
	for _, fixture := range demoSolicitudFixtures() {
		if _, err := usecase.EnsureSolicitud(ctx, buildDemoSolicitud(fixture)); err != nil {
			return ProcedureDemoView{}, err
		}
		admitidasByConvocatoria[fixture.convocatoriaID()][fixture.id] = true
	}

	provisional := ListadoView{ConvocatoriaID: demoProcedureAggregateID, Version: "v1"}
	definitivo := ListadoView{ConvocatoriaID: demoProcedureAggregateID, Version: "v1"}
	for number := 1; number <= demoPuestoFixtureCount; number++ {
		convocatoriaID := demoPuestoID(number)
		currentListado, err := usecase.ListadoActual(ctx, convocatoriaID)
		if err != nil {
			return ProcedureDemoView{}, err
		}
		if demoListadoIsDefinitive(currentListado, admitidasByConvocatoria[convocatoriaID]) {
			currentView := listadoView(currentListado)
			provisional.Items = append(provisional.Items, currentView.Items...)
			definitivo.Items = append(definitivo.Items, currentView.Items...)
			continue
		}
		provisionalListado, err := usecase.PublicarListadoProvisional(ctx, convocatoriaID, admitidasByConvocatoria[convocatoriaID])
		if err != nil {
			return ProcedureDemoView{}, err
		}
		definitivoListado, err := usecase.PublicarListadoDefinitivo(ctx, convocatoriaID, admitidasByConvocatoria[convocatoriaID])
		if err != nil {
			return ProcedureDemoView{}, err
		}
		provisional.Items = append(provisional.Items, listadoView(provisionalListado).Items...)
		definitivo.Items = append(definitivo.Items, listadoView(definitivoListado).Items...)
	}
	return ProcedureDemoView{
		Convocatoria: ConvocatoriaView{
			ID:      demoProcedureAggregateID,
			Version: "v1",
			Estado:  domain.ProcedureStateDefinitiva,
		},
		Provisional: provisional,
		Definitivo:  definitivo,
	}, nil
}

func demoListadoIsDefinitive(listado usecases.Listado, admitidas map[string]bool) bool {
	if len(listado.Items) == 0 {
		return false
	}
	for _, item := range listado.Items {
		expected := domain.SolicitudStateExcluidaDefinitiva
		if admitidas[item.SolicitudID] {
			expected = domain.SolicitudStateAdmitidaDefinitiva
		}
		if item.Estado != expected {
			return false
		}
	}
	return true
}

type demoSolicitudFixture struct {
	id, candidateID, sorteoKey string
	mismaCategoriaMeses        int
	otraCategoriaMeses         int
	cursoHoras                 int
	tituloPuntos               float64
	otrosPuntos                float64
}

const (
	demoSolicitudFixtureCount = 138
	demoPuestoFixtureCount    = 84
	demoProcedureAggregateID  = "demo-puestos-rpt"
)

func demoSolicitudFixtures() []demoSolicitudFixture {
	fixtures := []demoSolicitudFixture{
		{id: "demo-sol-001", candidateID: "demo-cand-a", sorteoKey: "Lopez", mismaCategoriaMeses: 220, otraCategoriaMeses: 40, cursoHoras: 240, tituloPuntos: 12, otrosPuntos: 4},
		{id: "demo-sol-002", candidateID: "cand-1", sorteoKey: "Sanchez", mismaCategoriaMeses: 176, otraCategoriaMeses: 52, cursoHoras: 210, tituloPuntos: 9, otrosPuntos: 3.5},
		{id: "demo-sol-003", candidateID: "demo.empleado", sorteoKey: "Martinez", mismaCategoriaMeses: 144, otraCategoriaMeses: 30, cursoHoras: 180, tituloPuntos: 8, otrosPuntos: 3},
		{id: "demo-sol-004", candidateID: "demo.administrativo", sorteoKey: "Garcia", mismaCategoriaMeses: 132, otraCategoriaMeses: 24, cursoHoras: 160, tituloPuntos: 8, otrosPuntos: 2.5},
		{id: "demo-sol-005", candidateID: "demo.seccion", sorteoKey: "Ramirez", mismaCategoriaMeses: 120, otraCategoriaMeses: 36, cursoHoras: 150, tituloPuntos: 7, otrosPuntos: 2},
		{id: "demo-sol-006", candidateID: "demo.servicio", sorteoKey: "Ortega", mismaCategoriaMeses: 118, otraCategoriaMeses: 20, cursoHoras: 130, tituloPuntos: 6, otrosPuntos: 2},
		{id: "demo-sol-007", candidateID: "demo.rrhh.tecnico", sorteoKey: "Navarro", mismaCategoriaMeses: 108, otraCategoriaMeses: 42, cursoHoras: 145, tituloPuntos: 7, otrosPuntos: 2.25},
		{id: "demo-sol-008", candidateID: "demo.admin", sorteoKey: "Alvarez", mismaCategoriaMeses: 96, otraCategoriaMeses: 50, cursoHoras: 120, tituloPuntos: 6, otrosPuntos: 1.5},
		{id: "demo-sol-009", candidateID: "demo-cand-b", sorteoKey: "Navas", mismaCategoriaMeses: 84, otraCategoriaMeses: 48, cursoHoras: 110, tituloPuntos: 5.5, otrosPuntos: 1.5},
		{id: "demo-sol-010", candidateID: "demo-cand-c", sorteoKey: "Vargas", mismaCategoriaMeses: 72, otraCategoriaMeses: 36, cursoHoras: 100, tituloPuntos: 5, otrosPuntos: 2},
		{id: "demo-sol-011", candidateID: "demo-cand-d", sorteoKey: "Molina", mismaCategoriaMeses: 64, otraCategoriaMeses: 40, cursoHoras: 96, tituloPuntos: 4.5, otrosPuntos: 1.25},
		{id: "demo-sol-012", candidateID: "demo-cand-e", sorteoKey: "Hernandez", mismaCategoriaMeses: 58, otraCategoriaMeses: 28, cursoHoras: 92, tituloPuntos: 4, otrosPuntos: 1.75},
		{id: "demo-sol-013", candidateID: "demo-cand-f", sorteoKey: "Ruiz", mismaCategoriaMeses: 52, otraCategoriaMeses: 32, cursoHoras: 88, tituloPuntos: 4, otrosPuntos: 1},
		{id: "demo-sol-014", candidateID: "demo-cand-g", sorteoKey: "Castillo", mismaCategoriaMeses: 48, otraCategoriaMeses: 24, cursoHoras: 72, tituloPuntos: 3.5, otrosPuntos: 1.25},
		{id: "demo-sol-015", candidateID: "demo-cand-h", sorteoKey: "Moreno", mismaCategoriaMeses: 42, otraCategoriaMeses: 30, cursoHoras: 68, tituloPuntos: 3, otrosPuntos: 1},
		{id: "demo-sol-016", candidateID: "demo-cand-i", sorteoKey: "Jimenez", mismaCategoriaMeses: 36, otraCategoriaMeses: 18, cursoHoras: 64, tituloPuntos: 2.5, otrosPuntos: 1.5},
		{id: "demo-sol-017", candidateID: "demo-cand-j", sorteoKey: "Torres", mismaCategoriaMeses: 30, otraCategoriaMeses: 20, cursoHoras: 52, tituloPuntos: 2, otrosPuntos: 1},
		{id: "demo-sol-018", candidateID: "demo-cand-k", sorteoKey: "Aguilera", mismaCategoriaMeses: 24, otraCategoriaMeses: 12, cursoHoras: 48, tituloPuntos: 2, otrosPuntos: 0.75},
		{id: "demo-sol-019", candidateID: "demo-cand-l", sorteoKey: "Benitez", mismaCategoriaMeses: 18, otraCategoriaMeses: 14, cursoHoras: 42, tituloPuntos: 1.5, otrosPuntos: 0.5},
		{id: "demo-sol-020", candidateID: "demo-cand-m", sorteoKey: "Carmona", mismaCategoriaMeses: 12, otraCategoriaMeses: 10, cursoHoras: 36, tituloPuntos: 1, otrosPuntos: 0.5},
	}
	for sequence := len(fixtures) + 1; sequence <= demoSolicitudFixtureCount; sequence++ {
		fixtures = append(fixtures, generatedDemoSolicitudFixture(sequence))
	}
	return fixtures
}

func generatedDemoSolicitudFixture(sequence int) demoSolicitudFixture {
	sortKeys := []string{
		"Alonso", "Baena", "Campos", "Delgado", "Espinosa", "Fernandez", "Garrido", "Herrera",
		"Ibañez", "Jurado", "Lara", "Mesa", "Nieto", "Olivares", "Pardo", "Quiles",
		"Roldan", "Salcedo", "Ubeda", "Valverde", "Yuste", "Zamora",
	}
	return demoSolicitudFixture{
		id:                  fmt.Sprintf("demo-sol-%03d", sequence),
		candidateID:         fmt.Sprintf("EMP-%04d", sequence),
		sorteoKey:           sortKeys[sequence%len(sortKeys)],
		mismaCategoriaMeses: 18 + (sequence*7)%126,
		otraCategoriaMeses:  (sequence * 5) % 56,
		cursoHoras:          40 + (sequence*11)%150,
		tituloPuntos:        float64(2 + sequence%9),
		otrosPuntos:         float64(sequence%7) + 0.5,
	}
}

func buildDemoSolicitud(fixture demoSolicitudFixture) usecases.RegistrarSolicitudCommand {
	return usecases.RegistrarSolicitudCommand{
		ID:             fixture.id,
		ConvocatoriaID: fixture.convocatoriaID(),
		CandidateID:    fixture.candidateID,
		SorteoKey:      fixture.sorteoKey,
		Merits: []domain.Merit{
			demoMerit(fixture.id+"-exp-misma", domain.MeritTypeExperienciaMismaCategoria, domain.MeritData{Meses: fixture.mismaCategoriaMeses}),
			demoMerit(fixture.id+"-exp-otra", domain.MeritTypeExperienciaOtraCategoria, domain.MeritData{Meses: fixture.otraCategoriaMeses}),
			demoMerit(fixture.id+"-curso", domain.MeritTypeFormacionCurso, domain.MeritData{Horas: fixture.cursoHoras}),
			demoMerit(fixture.id+"-titulo", domain.MeritTypeFormacionTitulo, domain.MeritData{PuntosFijos: fixture.tituloPuntos}),
			demoMerit(fixture.id+"-otros", domain.MeritTypeOtros, domain.MeritData{PuntosFijos: fixture.otrosPuntos}),
		},
	}
}

func (fixture demoSolicitudFixture) convocatoriaID() string {
	return demoPuestoID(positionNumberForSolicitudSequence(fixture.sequence()))
}

func (fixture demoSolicitudFixture) sequence() int {
	var sequence int
	if _, err := fmt.Sscanf(fixture.id, "demo-sol-%d", &sequence); err != nil || sequence <= 0 {
		return 1
	}
	return sequence
}

func positionNumberForSolicitudSequence(sequence int) int {
	if sequence <= 0 {
		return 1
	}
	return ((sequence - 1) % demoPuestoFixtureCount) + 1
}

func demoPuestoID(number int) string {
	if number <= 0 {
		number = 1
	}
	return fmt.Sprintf("demo-puesto-%03d", number)
}

func demoMerit(id string, tipo domain.MeritType, datos domain.MeritData) domain.Merit {
	return domain.Merit{
		ID:     "merit-" + id,
		Tipo:   tipo,
		Datos:  datos,
		Estado: domain.MeritStateValidado,
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
			ConvocatoriaID: listado.ConvocatoriaID,
			SolicitudID:    item.SolicitudID,
			CandidateID:    item.CandidateID,
			Estado:         item.Estado,
			TotalPoints:    item.Result.TotalPoints,
			SectionPoints:  baremoSectionPointsView(item.Result),
			RuleSetID:      item.Result.RuleSetID,
			RuleSetVersion: item.Result.RuleSetVersion,
			Details:        baremoDetailsView(item.Result),
			Rank:           item.Rank,
		})
	}
	return view
}

func baremoSectionPointsView(result domain.BaremoResult) map[string]float64 {
	points := make(map[string]float64, len(result.SectionPoints))
	for section, value := range result.SectionPoints {
		points[string(section)] = value
	}
	return points
}

func baremoDetailsView(result domain.BaremoResult) []BaremoDetailView {
	details := make([]BaremoDetailView, 0, len(result.Details))
	for _, detail := range result.Details {
		details = append(details, BaremoDetailView{
			MeritID:       detail.MeritID,
			MeritType:     string(detail.MeritType),
			Section:       string(detail.Section),
			RawPoints:     detail.RawPoints,
			AppliedPoints: detail.AppliedPoints,
			Capped:        detail.Capped,
		})
	}
	return details
}
