package bootstrap

import (
	"errors"
	"testing"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestIdentidadPersonalDietasRechazaDependenciasNilTipadas(t *testing.T) {
	var seguridad *seguridadComunDesarrollo
	var reloj *relojNuloDietasPrueba
	if _, err := nuevaIdentidadPersonalDietas(seguridad, reloj); !errors.Is(err, ErrIdentidadPersonalDietasNoDisponible) {
		t.Fatalf("nil tipado admitido: %v", err)
	}
}

func TestBorradoresDietasRechazanPoolsAusentes(t *testing.T) {
	var seguridad *seguridadComunDesarrollo
	var reloj *relojNuloDietasPrueba
	rutas, colecciones, err := componerBorradoresDietas(dependenciasBorradoresDietas{seguridad: seguridad, reloj: reloj})
	if !errors.Is(err, ErrComposicionBorradoresDietasNoDisponible) || rutas != nil || colecciones != nil {
		t.Fatalf("montaje incompleto admitido: rutas=%v colecciones=%v error=%v", rutas, colecciones, err)
	}
}

type relojNuloDietasPrueba struct{ vecports.Reloj }
