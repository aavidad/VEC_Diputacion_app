package interna

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vec "vec-diputacion-granada/internal/vec/domain"
)

type selectorPerfilActivoEspia struct{ llamadas int }

func (s *selectorPerfilActivoEspia) SeleccionarPerfilActivo(context.Context, vec.CuentaAutenticadaContextoActor, httpseguridad.ContextoAuditoriaAutenticada) (string, error) {
	s.llamadas++
	return "prf_no_debe_usarse", nil
}

func TestContextoActorLecturaNoConsultaPerfilSinCapsulaAutenticada(t *testing.T) {
	selector := &selectorPerfilActivoEspia{}
	a := contextoActorLecturaCT{selector: selector}
	resultado, err := a.resolver(context.Background())
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) ||
		resultado.Resultado.Validar() == nil || selector.llamadas != 0 {
		t.Fatalf("sin identidad = (%v, %v), selector llamado %d veces", resultado, err, selector.llamadas)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = a.resolver(ctx)
	if !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) || selector.llamadas != 0 {
		t.Fatalf("cancelada = %v, selector llamado %d veces", err, selector.llamadas)
	}
}
