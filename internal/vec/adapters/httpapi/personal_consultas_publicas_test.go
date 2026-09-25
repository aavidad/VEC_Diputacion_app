package httpapi

import (
	"errors"
	"testing"
)

func TestConsultasPublicasPersonalExigenFuente(t *testing.T) {
	if _, err := NewHandlerCategoriasProfesionalesPublicas(nil); !errors.Is(err, ErrConsultaPublicaPersonalInvalida) {
		t.Fatalf("categorias sin fuente: %v", err)
	}
	if _, err := NewHandlerRPTPublica(nil); !errors.Is(err, ErrConsultaPublicaPersonalInvalida) {
		t.Fatalf("rpt sin fuente: %v", err)
	}
	if _, err := NewHandlerEstructuraOrganizativaPublica(nil); !errors.Is(err, ErrConsultaPublicaPersonalInvalida) {
		t.Fatalf("estructura sin fuente: %v", err)
	}
}
