package domain

import (
	"errors"
	"testing"
)

func TestNewMeritDefaultsDraftAndValidatesData(t *testing.T) {
	merit, err := NewMerit(" merit-1 ", MeritTypeExperienciaMismaCategoria, MeritData{Meses: 12})
	if err != nil {
		t.Fatalf("NewMerit() error = %v", err)
	}

	if merit.ID != "merit-1" {
		t.Fatalf("ID = %q, want %q", merit.ID, "merit-1")
	}
	if merit.Estado != MeritStateBorrador {
		t.Fatalf("Estado = %q, want %q", merit.Estado, MeritStateBorrador)
	}
	if err := merit.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestMeritTypesAndStatesValidateKnownValues(t *testing.T) {
	validTypes := []MeritType{
		MeritTypeExperienciaMismaCategoria,
		MeritTypeExperienciaOtraCategoria,
		MeritTypeFormacionTitulo,
		MeritTypeFormacionCurso,
		MeritTypeOtros,
	}
	for _, meritType := range validTypes {
		if !meritType.IsValid() {
			t.Fatalf("MeritType(%q).IsValid() = false", meritType)
		}
	}
	if MeritType("desconocido").IsValid() {
		t.Fatalf("unknown MeritType reported valid")
	}

	validStates := []MeritState{
		MeritStateBorrador,
		MeritStatePresentado,
		MeritStateValidado,
		MeritStateRechazado,
		MeritStateSubsanacion,
	}
	for _, state := range validStates {
		if !state.IsValid() {
			t.Fatalf("MeritState(%q).IsValid() = false", state)
		}
	}
	if MeritState("Archivado").IsValid() {
		t.Fatalf("unknown MeritState reported valid")
	}
}

func TestMeritValidateRejectsInvalidFields(t *testing.T) {
	valid := Merit{
		ID:     "merit-1",
		Tipo:   MeritTypeFormacionCurso,
		Datos:  MeritData{Horas: 20},
		Estado: MeritStateBorrador,
	}
	tests := []struct {
		name    string
		merit   Merit
		wantErr error
	}{
		{name: "id required", merit: Merit{Tipo: valid.Tipo, Datos: valid.Datos, Estado: valid.Estado}, wantErr: ErrMeritIDRequired},
		{name: "type invalid", merit: Merit{ID: valid.ID, Tipo: MeritType("x"), Datos: valid.Datos, Estado: valid.Estado}, wantErr: ErrMeritTypeInvalid},
		{name: "state invalid", merit: Merit{ID: valid.ID, Tipo: valid.Tipo, Datos: valid.Datos, Estado: MeritState("x")}, wantErr: ErrMeritStateInvalid},
		{name: "negative months", merit: Merit{ID: valid.ID, Tipo: valid.Tipo, Datos: MeritData{Meses: -1}, Estado: valid.Estado}, wantErr: ErrMeritDataInvalid},
		{name: "negative hours", merit: Merit{ID: valid.ID, Tipo: valid.Tipo, Datos: MeritData{Horas: -1}, Estado: valid.Estado}, wantErr: ErrMeritDataInvalid},
		{name: "negative fixed points", merit: Merit{ID: valid.ID, Tipo: valid.Tipo, Datos: MeritData{PuntosFijos: -0.01}, Estado: valid.Estado}, wantErr: ErrMeritDataInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.merit.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMeritTransitionRules(t *testing.T) {
	merit := Merit{
		ID:     "merit-1",
		Tipo:   MeritTypeFormacionCurso,
		Datos:  MeritData{Horas: 20},
		Estado: MeritStateBorrador,
	}

	if !merit.CanTransition(MeritStatePresentado) {
		t.Fatalf("draft merit should transition to submitted")
	}
	if merit.CanTransition(MeritStateValidado) {
		t.Fatalf("draft merit should not transition directly to validated")
	}
	if err := merit.Transition(MeritStatePresentado); err != nil {
		t.Fatalf("Transition(Presentado) error = %v", err)
	}
	if merit.Estado != MeritStatePresentado {
		t.Fatalf("Estado = %q, want %q", merit.Estado, MeritStatePresentado)
	}
	if err := merit.Transition(MeritStateSubsanacion); err != nil {
		t.Fatalf("Transition(Subsanacion) error = %v", err)
	}
	if err := merit.Transition(MeritStatePresentado); err != nil {
		t.Fatalf("Transition(Presentado from Subsanacion) error = %v", err)
	}
	if err := merit.Transition(MeritStateBorrador); !errors.Is(err, ErrMeritTransition) {
		t.Fatalf("Transition(Borrador) error = %v, want %v", err, ErrMeritTransition)
	}
	if err := merit.Transition(MeritState("x")); !errors.Is(err, ErrMeritStateInvalid) {
		t.Fatalf("Transition(invalid) error = %v, want %v", err, ErrMeritStateInvalid)
	}

	var nilMerit *Merit
	if err := nilMerit.Transition(MeritStatePresentado); !errors.Is(err, ErrMeritTransition) {
		t.Fatalf("nil Transition() error = %v, want %v", err, ErrMeritTransition)
	}
}
