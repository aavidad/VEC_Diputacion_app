package application

import (
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
)

func TestNewProfessionalPortalViewHasUsefulAdministrativeModules(t *testing.T) {
	view := NewProfessionalPortalView(sampleProfessionalExpediente())

	if view.Locale != "es-ES" || view.Candidate.ID == "" || view.Header.TitleKey == "" {
		t.Fatalf("professional portal header = %#v, candidate = %#v", view.Header, view.Candidate)
	}
	if len(view.Modules) < 8 {
		t.Fatalf("modules count = %d, want at least 8", len(view.Modules))
	}
	for _, mode := range []string{"real", "demo"} {
		if !hasProfessionalCapabilityMode(view.Capabilities, mode) {
			t.Fatalf("capability mode %q missing in %#v", mode, view.Capabilities)
		}
	}
	for _, id := range []string{"claims", "notifications", "notification_send", "notification_read", "audit_scope"} {
		if !hasProfessionalCapability(view.Capabilities, id, "real") {
			t.Fatalf("real capability %q missing in %#v", id, view.Capabilities)
		}
	}

	required := []string{
		"dashboard", "convocatorias", "expediente", "meritos", "documentos",
		"autobaremo", "alegaciones", "notificaciones", "mensajes", "noticias",
		"perfil", "ayuda",
	}
	seen := map[string]ProfessionalModuleView{}
	for _, module := range view.Modules {
		seen[module.ID] = module
		if module.Group == "" || module.Accent == "" || module.LabelKey == "" || module.PrimaryActionKey == "" {
			t.Fatalf("empty module metadata: %#v", module)
		}
	}
	for _, id := range required {
		if _, ok := seen[id]; !ok {
			t.Fatalf("module %q missing in %#v", id, view.Modules)
		}
	}
}

func TestNewProfessionalPortalViewCoversCandidateVecNavigation(t *testing.T) {
	view := NewProfessionalPortalView(sampleProfessionalExpediente())
	seen := professionalModuleIDs(view.Modules)

	for _, id := range []string{"mensajes", "noticias", "ayuda"} {
		if !seen[id] {
			t.Fatalf("candidate VEC module %q missing in %#v", id, view.Modules)
		}
	}
	for _, action := range view.PendingActions {
		if !seen[action.ModuleID] {
			t.Fatalf("action %q points to missing module %q", action.ID, action.ModuleID)
		}
	}
}

func TestNewProfessionalPortalViewHighlightsEmptyAutobaremoWork(t *testing.T) {
	expediente := sampleProfessionalExpediente()
	expediente.Merits = nil
	expediente.Baremo.Details = nil

	view := NewProfessionalPortalView(expediente)

	if !hasProfessionalAction(view.PendingActions, "add_merit") {
		t.Fatalf("pending actions = %#v, want add_merit", view.PendingActions)
	}
	for _, module := range view.Modules {
		if (module.ID == "meritos" || module.ID == "autobaremo") && module.AlertCount == 0 {
			t.Fatalf("module %q alert count = 0, want missing-work alert", module.ID)
		}
	}
	if len(view.Autobaremo.Warnings) == 0 || view.Autobaremo.Warnings[0].Code != "baremo_without_details" {
		t.Fatalf("warnings = %#v, want baremo_without_details", view.Autobaremo.Warnings)
	}
}

func TestNewProfessionalPortalViewExplainsAutobaremo(t *testing.T) {
	view := NewProfessionalPortalView(sampleProfessionalExpediente())

	if view.Autobaremo.TotalPoints != 8.5 || view.Autobaremo.RuleSetVersion != "v1" {
		t.Fatalf("autobaremo summary = %#v", view.Autobaremo)
	}
	if len(view.Autobaremo.Sections) < 3 || len(view.Autobaremo.Details) == 0 || len(view.Autobaremo.Explanation) < 3 {
		t.Fatalf("autobaremo lacks breakdown, details or explanation: %#v", view.Autobaremo)
	}
	if len(view.Autobaremo.Warnings) == 0 {
		t.Fatalf("autobaremo warnings empty, want capped-section notice")
	}
	if view.Autobaremo.Sections[0].ID != string(domain.BaremoSectionExperiencia) {
		t.Fatalf("first section = %#v, want experiencia", view.Autobaremo.Sections[0])
	}
}

func TestNewProfessionalPortalViewDoesNotReturnEmptyWorkspace(t *testing.T) {
	view := NewProfessionalPortalView(sampleProfessionalExpediente())

	if len(view.Summary) == 0 || len(view.PendingActions) == 0 || view.Audit.ReceiptKey == "" {
		t.Fatalf("workspace is empty: summary=%#v actions=%#v audit=%#v", view.Summary, view.PendingActions, view.Audit)
	}
	for _, metric := range view.Summary {
		if metric.ID == "" || metric.LabelKey == "" || metric.StateKey == "" {
			t.Fatalf("empty metric metadata: %#v", metric)
		}
	}
	for _, action := range view.PendingActions {
		if action.ID == "" || action.LabelKey == "" || action.ModuleID == "" {
			t.Fatalf("empty action metadata: %#v", action)
		}
	}
}

func TestProfessionalPortalApplicationLayerStaysAdapterFree(t *testing.T) {
	source, err := os.ReadFile("professional_portal.go")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if strings.Contains(string(source), "net/http") {
		t.Fatalf("professional_portal.go imports or references net/http")
	}
}

func professionalModuleIDs(modules []ProfessionalModuleView) map[string]bool {
	seen := map[string]bool{}
	for _, module := range modules {
		seen[module.ID] = true
	}
	return seen
}

func hasProfessionalAction(actions []ProfessionalActionView, want string) bool {
	for _, action := range actions {
		if action.ID == want {
			return true
		}
	}
	return false
}

func hasProfessionalCapabilityMode(capabilities []ProfessionalCapabilityView, mode string) bool {
	for _, capability := range capabilities {
		if capability.Mode == mode {
			return true
		}
	}
	return false
}

func hasProfessionalCapability(capabilities []ProfessionalCapabilityView, id, mode string) bool {
	for _, capability := range capabilities {
		if capability.ID == id && capability.Mode == mode && capability.Route != "" {
			return true
		}
	}
	return false
}

func sampleProfessionalExpediente() ExpedienteView {
	return ExpedienteView{
		Candidate: CandidateView{ID: "cand-1", DNI: "12345678A", Nombre: "Ana Perez", Email: "ana@example.test", CallID: "conv-2026"},
		Merits: []MeritView{
			{ID: "merit-1", Tipo: domain.MeritTypeExperienciaMismaCategoria, Estado: domain.MeritStatePresentado, Datos: MeritDataCommand{Meses: 30}},
			{ID: "merit-2", Tipo: domain.MeritTypeFormacionCurso, Estado: domain.MeritStatePresentado, Datos: MeritDataCommand{Horas: 50}},
		},
		Baremo: BaremoView{
			TotalPoints: 8.5, RuleSetID: "conv-2026", RuleSetVersion: "v1",
			SectionPoints: map[string]float64{
				string(domain.BaremoSectionExperiencia): 6,
				string(domain.BaremoSectionFormacion):   2.5,
			},
			Details: []BaremoDetailView{
				{MeritID: "merit-1", MeritType: string(domain.MeritTypeExperienciaMismaCategoria), Section: string(domain.BaremoSectionExperiencia), RawPoints: 7, AppliedPoints: 6, Capped: true},
				{MeritID: "merit-2", MeritType: string(domain.MeritTypeFormacionCurso), Section: string(domain.BaremoSectionFormacion), RawPoints: 2.5, AppliedPoints: 2.5},
			},
		},
	}
}
