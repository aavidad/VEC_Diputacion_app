package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestPermisosSinComposicionDenieganEfectos(t *testing.T) {
	var s *ServicioPermisos
	if _, err := s.SolicitarPermiso(context.Background(), ports.OrdenPermiso{}, ports.PeticionPermiso{}); !errors.Is(err, ErrPermisoContextoInvalido) {
		t.Fatalf("solicitud sin fuentes: %v", err)
	}
	if _, err := s.DecidirPermiso(context.Background(), ports.OrdenPermiso{}, ports.DecisionSolicitudPermiso{}); !errors.Is(err, ErrPermisoContextoInvalido) {
		t.Fatalf("decision sin autoridad: %v", err)
	}
	if _, err := s.ConsultarResumenAnual(context.Background(), ports.OrdenPermiso{}, "permiso:test", 2026); !errors.Is(err, ErrPermisoContextoInvalido) {
		t.Fatalf("resumen sin autoridad: %v", err)
	}
}
