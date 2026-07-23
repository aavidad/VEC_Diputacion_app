package ports

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

type relojCoberturaMutablePrueba struct {
	mu    sync.RWMutex
	ahora time.Time
}

func (r *relojCoberturaMutablePrueba) Ahora() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ahora
}

func (r *relojCoberturaMutablePrueba) fijar(ahora time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ahora = ahora
}

func TestCoberturaRechazaReciboDevueltoTrasCaducarLaOperacion(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	consumidor := consumidorCoberturaFuncion(func(
		ctx context.Context,
		orden OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error) {
		recibo, err := NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_tardio_012345",
			entorno.reloj.ahora,
		)
		if err != nil {
			return ReciboConsumoCobertura{}, err
		}
		<-ctx.Done()
		return recibo, nil
	})
	_, err := ConsultarCoberturaConFuente(
		context.Background(),
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		consumidor,
		entorno.confianza,
		entorno.reloj,
		entorno.solicitud,
		200*time.Millisecond,
	)
	if !errors.Is(err, ErrConsumoCoberturaNoDisponible) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("se aceptó un recibo devuelto tras el deadline: %v", err)
	}
}

func TestCoberturaRevalidaRelojDespuesDelConsumo(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	reloj := &relojCoberturaMutablePrueba{ahora: entorno.reloj.ahora}
	consumidor := consumidorCoberturaFuncion(func(
		_ context.Context,
		orden OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error) {
		recibo, err := NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_expirado_012345",
			entorno.reloj.ahora,
		)
		reloj.fijar(
			entorno.solicitud.SolicitadaEn.Add(
				VigenciaMaximaRespuestaCobertura,
			),
		)
		return recibo, err
	})
	_, err := ConsultarCoberturaConFuente(
		context.Background(),
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		consumidor,
		entorno.confianza,
		reloj,
		entorno.solicitud,
		time.Second,
	)
	if !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se aceptó un recibo con ventana ya vencida: %v", err)
	}
}

func TestCoberturaRechazaRelojQueRetrocedeDuranteConsumo(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	reloj := &relojCoberturaMutablePrueba{ahora: entorno.reloj.ahora}
	consumidor := consumidorCoberturaFuncion(func(
		_ context.Context,
		orden OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error) {
		recibo, err := NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_reloj_atras_0123",
			entorno.reloj.ahora,
		)
		reloj.fijar(entorno.reloj.ahora.Add(-time.Microsecond))
		return recibo, err
	})
	_, err := ConsultarCoberturaConFuente(
		context.Background(),
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		consumidor,
		entorno.confianza,
		reloj,
		entorno.solicitud,
		time.Second,
	)
	if !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se aceptó un reloj final anterior al preconsumo: %v", err)
	}
}

func TestCoberturaRechazaReciboFechadoEnElFuturo(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	consumidor := consumidorCoberturaFuncion(func(
		_ context.Context,
		orden OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error) {
		return NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_futuro_012345",
			entorno.reloj.ahora.Add(time.Second),
		)
	})
	_, err := ConsultarCoberturaConFuente(
		context.Background(),
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		consumidor,
		entorno.confianza,
		entorno.reloj,
		entorno.solicitud,
		time.Second,
	)
	if !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se aceptó un recibo posterior al reloj final: %v", err)
	}
}

func TestCoberturaCancelacionCompetitivaDespuesDelConsumidor(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	consumidor := consumidorCoberturaFuncion(func(
		_ context.Context,
		orden OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error) {
		recibo, err := NuevoReciboConsumoCobertura(
			orden,
			"consumo_cobertura_cancelado_012345",
			entorno.reloj.ahora,
		)
		cancelar()
		return recibo, err
	})
	_, err := ConsultarCoberturaConFuente(
		ctx,
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		consumidor,
		entorno.confianza,
		entorno.reloj,
		entorno.solicitud,
		time.Second,
	)
	if !errors.Is(err, ErrConsumoCoberturaNoDisponible) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("se aceptó una cancelación competitiva: %v", err)
	}
}

func TestCoberturaCredencialSAENoPuedeResponderComoBolsa(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	invocada := false
	entorno.fuente.presentador.datos.BackendRef =
		"conector_sae_cobertura_v1"
	entorno.fuente.consultar = func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		invocada = true
		return ResultadoConsultaCobertura{}, nil
	}
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) ||
		invocada {
		t.Fatalf("SAE respondió con la definición de Bolsa: %v", err)
	}
}

func TestCoberturaConstructorNominalNoSustituyeFirmaTCB(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	entorno.fuente.consultar = func(
		context.Context,
		SolicitudConsultarCobertura,
	) (ResultadoConsultaCobertura, error) {
		resultado := resultadoCoberturaFirmadoPrueba(
			t,
			entorno.solicitud,
		)
		resultado.atestacion.SelloHMAC =
			"hmac-sha256:" + dominioSelloRespuestaCobertura +
				"7:" + strings.Repeat("b", 64)
		return resultado, nil
	}
	entorno.verificador.verificar = func(
		_ context.Context,
		solicitud SolicitudVerificarRespuestaCobertura,
	) (ConfirmacionRespuestaCobertura, error) {
		return NuevaConfirmacionRespuestaCobertura(
			solicitud,
			"verificador_cobertura_tcb_012345",
			entorno.reloj.ahora,
			make([]byte, 64),
		)
	}
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("el constructor nominal aprobó un HMAC falso: %v", err)
	}
}

func TestCoberturaRechazaFirmaConfirmacionAlterada(t *testing.T) {
	entorno := nuevoEntornoCoberturaPrueba(t)
	verificar := entorno.verificador.verificar
	entorno.verificador.verificar = func(
		ctx context.Context,
		solicitud SolicitudVerificarRespuestaCobertura,
	) (ConfirmacionRespuestaCobertura, error) {
		confirmacion, err := verificar(ctx, solicitud)
		if err == nil {
			confirmacion.datos.FirmaEd25519[0] ^= 1
		}
		return confirmacion, err
	}
	if _, err := entorno.consultar(context.Background()); !errors.Is(err, ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("se aceptó una firma TCB alterada: %v", err)
	}
}
