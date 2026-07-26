package cobertura_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestOrdenOperacionDecisionRecorreConcesionYDenegacionCompletas(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenC3(t)
	casos := []struct {
		nombre    string
		candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3
	}{
		{"concesión", escenario.candidataConcedida},
		{"denegación", escenario.candidataDenegada},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			orden, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
				context.Background(),
				relojGobiernoOrdenC3{
					ahora: escenario.base.Add(4 * time.Millisecond),
				},
				escenario.preparacion,
				caso.candidata,
			)
			if err != nil {
				t.Fatalf("recorrido completo rechazado: %v", err)
			}
			if _, err = json.Marshal(orden); !errors.Is(
				err,
				cobertura.ErrSerializacionOperacionDecisionCoberturaProhibida,
			) {
				t.Fatalf("orden final no opaca: %v", err)
			}
		})
	}
}

func TestOrdenOperacionDecisionRechazaLimiteExclusivoRollbackYCancelacion(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenC3(t)
	casos := []struct {
		nombre string
		ctx    context.Context
		ahora  time.Time
	}{
		{
			nombre: "límite exacto excluido",
			ctx:    context.Background(),
			ahora:  escenario.limite,
		},
		{
			nombre: "rollback anterior a preparación",
			ctx:    context.Background(),
			ahora:  escenario.base.Add(2500 * time.Microsecond),
		},
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	casos = append(casos, struct {
		nombre string
		ctx    context.Context
		ahora  time.Time
	}{
		nombre: "contexto cancelado",
		ctx:    cancelado,
		ahora:  escenario.base.Add(4 * time.Millisecond),
	})
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
				caso.ctx,
				relojGobiernoOrdenC3{ahora: caso.ahora},
				escenario.preparacion,
				escenario.candidataDenegada,
			); !errors.Is(
				err,
				cobertura.ErrOrdenOperacionDecisionCoberturaInvalida,
			) {
				t.Fatalf("ataque temporal aceptado: %v", err)
			}
		})
	}
}

func TestOrdenOperacionDecisionRechazaCandidataMismasReferenciasOtraSemantica(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenC3(t)
	if _, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
		context.Background(),
		relojGobiernoOrdenC3{
			ahora: escenario.base.Add(4 * time.Millisecond),
		},
		escenario.preparacion,
		escenario.candidataAjena,
	); !errors.Is(
		err,
		cobertura.ErrOrdenOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("candidata A/B con otra semántica aceptada: %v", err)
	}
}

func TestOrdenOperacionDecisionConcurrenciaNominalSinEstadoCompartido(
	t *testing.T,
) {
	escenario := nuevoEscenarioOrdenC3(t)
	const trabajadores = 24
	var grupo sync.WaitGroup
	errores := make(chan error, trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func(indice int) {
			defer grupo.Done()
			candidata := escenario.candidataConcedida
			if indice%2 == 1 {
				candidata = escenario.candidataDenegada
			}
			_, err := cobertura.NuevaOrdenOperacionDecisionCobertura(
				context.Background(),
				relojGobiernoOrdenC3{
					ahora: escenario.base.Add(4 * time.Millisecond),
				},
				escenario.preparacion,
				candidata,
			)
			errores <- err
		}(indice)
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("creación concurrente falló: %v", err)
		}
	}
}
