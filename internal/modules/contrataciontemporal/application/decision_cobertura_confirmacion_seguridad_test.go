package application

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestDecidirCoberturaUsaReferenciasVECPreasignadasExactas(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	observador := &preparadorObservadorConfirmacionPrueba{
		delegado: escenario.vec,
	}
	escenario.servicio.autorizaciones = observador
	recibo, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, decision := observador.referencias()
	if correlacion != recibo.CorrelacionVECRef ||
		decision != recibo.DecisionVECRef ||
		correlacion != "correlacion_11111111111111111111111111111111" ||
		decision != "dec_11111111111111111111111111111111" {
		t.Fatalf(
			"VEC no usó las referencias reservadas: %s, %s",
			correlacion,
			decision,
		)
	}
}

func TestDecidirCoberturaRechazaCandidataVECAdulteradaAntesDeTx(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	observador := &preparadorObservadorConfirmacionPrueba{
		delegado:  escenario.vec,
		adulterar: true,
	}
	escenario.servicio.autorizaciones = observador
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(
		err,
		ErrConfirmacionDecisionCoberturaNoConfiable,
	) || escenario.transaccion.total() != 0 {
		t.Fatalf("la candidata VEC adulterada alcanzó Tx: %v", err)
	}
}

func TestDecidirCoberturaSalidaTCBInvalidaNoAlcanzaCommitNiPrimario(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	escenario.transaccion.salidaInvalida = true
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(
		err,
		ErrConfirmacionDecisionCoberturaNoDisponible,
	) || escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 0 {
		t.Fatalf("la salida TCB inválida alcanzó COMMIT o primario: %v", err)
	}
}

func TestDecidirCoberturaFalloAntesCommitNoReconciliaNiReintenta(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	escenario.transaccion.falloAntesCommit = true
	_, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if !errors.Is(
		err,
		ErrConfirmacionDecisionCoberturaNoDisponible,
	) ||
		escenario.transaccion.total() != 1 ||
		escenario.reconciliador.total() != 0 {
		t.Fatalf(
			"el fallo anterior al COMMIT se reconcilió o reintentó: %v",
			err,
		)
	}
}

func TestDecidirCoberturaConcurrenteConcedeUnSoloPropietario(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, false)
	escenario.idempotencia.modo = idempotenciaExclusion
	inicio := make(chan struct{})
	const participantes = 8
	errores := make(chan error, participantes)
	var grupo sync.WaitGroup
	for indice := 0; indice < participantes; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			_, err := escenario.servicio.Decidir(
				context.Background(),
				escenario.solicitud,
			)
			errores <- err
		}()
	}
	close(inicio)
	grupo.Wait()
	close(errores)
	exitos := 0
	ocupadas := 0
	for err := range errores {
		switch {
		case err == nil:
			exitos++
		case errors.Is(err, ErrConfirmacionDecisionCoberturaOcupada):
			ocupadas++
		default:
			t.Fatalf("resultado concurrente inesperado: %v", err)
		}
	}
	if exitos != 1 || ocupadas != participantes-1 ||
		escenario.transaccion.total() != 1 ||
		escenario.vec.total() != 1 {
		t.Fatalf(
			"propiedad concurrente inválida: éxitos=%d ocupadas=%d",
			exitos,
			ocupadas,
		)
	}
}
