package internagobierno

import (
	"context"
	"errors"
	"testing"
)

func TestOriginalR5ReutilizaF1DeLaMismaPeticion(t *testing.T) {
	e := nuevoEntornoResolucionF1Prueba(t, personaF1Prueba, cuentaF1Prueba)
	ctx, err := e.fuente.VincularContextoOriginalCT(e.ctx)
	if err != nil || ctx == nil || e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 {
		t.Fatalf("vínculo F1 original: %v", err)
	}
	peticion, r, err := e.fuente.PeticionYContextoVinculadoOriginalCT(ctx)
	if err != nil || peticion.PreparacionCT.ActorRef != personaF1Prueba ||
		r.Resultado.Contexto.PersonaRef != personaF1Prueba ||
		e.revalidador.llamadas != 1 || e.resolutor.llamadas != 1 {
		t.Fatalf("la lectura repitió o cambió F1: %v", err)
	}
	otra := *e.fuente
	if _, err := otra.ContextoVinculadoOriginalCT(ctx); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("otra instancia reutilizó el sello: %v", err)
	}
	if _, err := e.fuente.ContextoVinculadoOriginalCT(context.Background()); !errors.Is(err, ErrGobiernoInternoNoDisponible) {
		t.Fatalf("sin sello se obtuvo contexto: %v", err)
	}
}
