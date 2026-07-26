package cobertura_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

var errPrivadoLecturaPrimariaTCBPrueba = errors.New(
	"detalle privado de lectura primaria",
)

type sesionLecturaPrimariaTCBPrueba struct {
	mu                 sync.Mutex
	recibo             cobertura.ReciboOperacionDecisionCobertura
	noEncontrado       bool
	mutar              func(*cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura)
	ultima             cobertura.DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura
	llamadas           atomic.Int32
	entrada            chan struct{}
	esperarCancelacion bool
	entradaUnaVez      sync.Once
}

func (s *sesionLecturaPrimariaTCBPrueba) LeerTerminalPrimario(
	ctx context.Context,
	consulta cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura,
) (cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura, error) {
	s.llamadas.Add(1)
	datos, err := consulta.Datos()
	if err != nil {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			err
	}
	s.mu.Lock()
	s.ultima = datos
	s.mu.Unlock()
	if s.entrada != nil {
		s.entradaUnaVez.Do(func() {
			close(s.entrada)
		})
	}
	if s.esperarCancelacion {
		<-ctx.Done()
	}
	resultado := cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{
		ObservadaEnPrimario: s.recibo.ConfirmadaEn,
	}
	if !s.noEncontrado {
		resultado.Encontrado = true
		resultado.Coordenadas = datos.Coordenadas
		resultado.HuellaOrdenSHA256 = datos.HuellaOrdenSHA256
		resultado.Recibo = clonarReciboConfirmacionOrdenC3(s.recibo)
	}
	if s.mutar != nil {
		s.mutar(&resultado)
	}
	return resultado, nil
}

type ejecutorLecturaPrimariaTCBPrueba struct {
	sesion          cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura
	errorAntes      error
	errorDespues    error
	omitirCallback  bool
	repetirCallback bool
	llamadas        atomic.Int32
}

func (e *ejecutorLecturaPrimariaTCBPrueba) EjecutarLecturaPrimariaTCB(
	ctx context.Context,
	callback func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
) error {
	e.llamadas.Add(1)
	if e.errorAntes != nil {
		return e.errorAntes
	}
	if e.omitirCallback {
		return nil
	}
	if err := callback(e.sesion); err != nil {
		return err
	}
	if e.repetirCallback {
		if err := callback(e.sesion); err != nil {
			return err
		}
	}
	return e.errorDespues
}

type ejecutorLecturaPrimariaAsincronaActivaPrueba struct {
	sesion            cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura
	entrada           <-chan struct{}
	resultadoCallback chan error
}

func (e *ejecutorLecturaPrimariaAsincronaActivaPrueba) EjecutarLecturaPrimariaTCB(
	_ context.Context,
	callback func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
) error {
	go func() {
		e.resultadoCallback <- callback(e.sesion)
	}()
	<-e.entrada
	return nil
}

type ejecutorLecturaPrimariaRetenidaPrueba struct {
	callback func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error
}

func (e *ejecutorLecturaPrimariaRetenidaPrueba) EjecutarLecturaPrimariaTCB(
	_ context.Context,
	callback func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
) error {
	e.callback = callback
	return nil
}

type ejecutorLecturaPrimariaTardiaPrueba struct {
	sesion            cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura
	disparar          chan struct{}
	resultadoCallback chan error
}

func (e *ejecutorLecturaPrimariaTardiaPrueba) EjecutarLecturaPrimariaTCB(
	_ context.Context,
	callback func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
) error {
	go func() {
		<-e.disparar
		e.resultadoCallback <- callback(e.sesion)
	}()
	return nil
}

func TestLecturaPrimariaTCBEmpujaHuellaPrivadaYElevaReciboExacto(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, err :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			escenario.ordenConcedida,
		)
	if err != nil {
		t.Fatal(err)
	}
	if _, existe := reflect.TypeOf(solicitud).MethodByName(
		"HuellaOrdenSHA256",
	); existe {
		t.Fatal("la solicitud expuso un getter de huella")
	}
	sesion := &sesionLecturaPrimariaTCBPrueba{
		recibo: escenario.reciboConcedido,
	}
	ejecutor := &ejecutorLecturaPrimariaTCBPrueba{sesion: sesion}
	reconciliador, err :=
		cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
			ejecutor,
		)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err :=
		reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
			context.Background(),
			solicitud,
		)
	if err != nil || ejecutor.llamadas.Load() != 1 ||
		sesion.llamadas.Load() != 1 {
		t.Fatalf("lectura primaria exacta rechazada: %v", err)
	}
	if _, valida := resultado.ConfirmacionPara(
		escenario.ordenConcedida,
	); !valida {
		t.Fatal("el recibo exacto no fue elevado")
	}
	sesion.mu.Lock()
	consulta := sesion.ultima
	sesion.mu.Unlock()
	if len(consulta.HuellaOrdenSHA256) != 64 {
		t.Fatal("la sesión técnica no recibió la huella opaca")
	}
	coordenadas, err := solicitud.CoordenadasPrimarias()
	if err != nil || consulta.Coordenadas != coordenadas {
		t.Fatal("la consulta técnica alteró las coordenadas minimizadas")
	}
}

func TestLecturaPrimariaTCBNoConcluyenteYDivergenteFallaCerrado(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, _ :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			escenario.ordenConcedida,
		)
	t.Run("ausente", func(t *testing.T) {
		sesion := &sesionLecturaPrimariaTCBPrueba{
			recibo:       escenario.reciboConcedido,
			noEncontrado: true,
		}
		reconciliador, _ :=
			cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
				&ejecutorLecturaPrimariaTCBPrueba{sesion: sesion},
			)
		resultado, err :=
			reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
				context.Background(),
				solicitud,
			)
		if err != nil {
			t.Fatalf("ausencia no concluyente rechazada: %v", err)
		}
		if _, valida := resultado.ConfirmacionPara(
			escenario.ordenConcedida,
		); valida {
			t.Fatal("la ausencia fabricó una confirmación")
		}
	})
	t.Run("huella divergente", func(t *testing.T) {
		sesion := &sesionLecturaPrimariaTCBPrueba{
			recibo: escenario.reciboConcedido,
			mutar: func(
				r *cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura,
			) {
				r.HuellaOrdenSHA256 = strings.Repeat("f", 64)
			},
		}
		reconciliador, _ :=
			cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
				&ejecutorLecturaPrimariaTCBPrueba{sesion: sesion},
			)
		if _, err :=
			reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
				context.Background(),
				solicitud,
			); !errors.Is(
			err,
			cobertura.ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible,
		) {
			t.Fatalf("huella divergente no falló cerrada: %v", err)
		}
	})
}

func TestLecturaPrimariaTCBRechazaCicloInfractorYSaneaErrores(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, _ :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			escenario.ordenConcedida,
		)
	for _, caso := range []struct {
		nombre   string
		ejecutor *ejecutorLecturaPrimariaTCBPrueba
	}{
		{
			nombre: "omitido",
			ejecutor: &ejecutorLecturaPrimariaTCBPrueba{
				omitirCallback: true,
			},
		},
		{
			nombre: "repetido",
			ejecutor: &ejecutorLecturaPrimariaTCBPrueba{
				sesion: &sesionLecturaPrimariaTCBPrueba{
					recibo: escenario.reciboConcedido,
				},
				repetirCallback: true,
			},
		},
		{
			nombre: "error privado posterior",
			ejecutor: &ejecutorLecturaPrimariaTCBPrueba{
				sesion: &sesionLecturaPrimariaTCBPrueba{
					recibo: escenario.reciboConcedido,
				},
				errorDespues: errPrivadoLecturaPrimariaTCBPrueba,
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			reconciliador, _ :=
				cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
					caso.ejecutor,
				)
			_, err :=
				reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
					context.Background(),
					solicitud,
				)
			if !errors.Is(
				err,
				cobertura.ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible,
			) ||
				errors.Is(err, errPrivadoLecturaPrimariaTCBPrueba) ||
				errors.Unwrap(err) != nil {
				t.Fatalf("ciclo infractor no quedó saneado: %v", err)
			}
		})
	}
	var ejecutorNulo *ejecutorLecturaPrimariaTCBPrueba
	if _, err :=
		cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
			ejecutorNulo,
		); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("ejecutor nulo tipado aceptado: %v", err)
	}
}

func TestLecturaPrimariaTCBRechazaCallbackAsincronoActivoAlRetorno(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, _ :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			escenario.ordenConcedida,
		)
	entrada := make(chan struct{})
	sesion := &sesionLecturaPrimariaTCBPrueba{
		recibo:             escenario.reciboConcedido,
		entrada:            entrada,
		esperarCancelacion: true,
	}
	ejecutor := &ejecutorLecturaPrimariaAsincronaActivaPrueba{
		sesion:            sesion,
		entrada:           entrada,
		resultadoCallback: make(chan error, 1),
	}
	reconciliador, _ :=
		cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
			ejecutor,
		)
	salida := make(chan error, 1)
	go func() {
		_, err :=
			reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
				context.Background(),
				solicitud,
			)
		salida <- err
	}()
	<-entrada
	if err := <-salida; !errors.Is(
		err,
		cobertura.ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("callback activo al retorno no falló cerrado: %v", err)
	}
	if err := <-ejecutor.resultadoCallback; !errors.Is(
		err,
		cobertura.ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("callback activo conservó validez: %v", err)
	}
	if sesion.llamadas.Load() != 1 {
		t.Fatalf("lecturas tras retorno: %d", sesion.llamadas.Load())
	}
}

func TestLecturaPrimariaTCBInvalidaCallbackRetenidoYTardio(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	solicitud, _ :=
		cobertura.NuevaSolicitudReconciliacionOperacionDecisionCobertura(
			escenario.ordenConcedida,
		)
	t.Run("retenido", func(t *testing.T) {
		sesion := &sesionLecturaPrimariaTCBPrueba{
			recibo: escenario.reciboConcedido,
		}
		ejecutor := &ejecutorLecturaPrimariaRetenidaPrueba{}
		reconciliador, _ :=
			cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
				ejecutor,
			)
		_, err :=
			reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
				context.Background(),
				solicitud,
			)
		if !errors.Is(
			err,
			cobertura.ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible,
		) {
			t.Fatalf("callback retenido no invalidó la lectura: %v", err)
		}
		if err = ejecutor.callback(sesion); !errors.Is(
			err,
			cobertura.ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida,
		) {
			t.Fatalf("callback retenido volvió a ser útil: %v", err)
		}
		if sesion.llamadas.Load() != 0 {
			t.Fatal("el callback retenido alcanzó el primario")
		}
	})
	t.Run("tardío", func(t *testing.T) {
		sesion := &sesionLecturaPrimariaTCBPrueba{
			recibo: escenario.reciboConcedido,
		}
		ejecutor := &ejecutorLecturaPrimariaTardiaPrueba{
			sesion:            sesion,
			disparar:          make(chan struct{}),
			resultadoCallback: make(chan error, 1),
		}
		reconciliador, _ :=
			cobertura.NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
				ejecutor,
			)
		_, err :=
			reconciliador.ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
				context.Background(),
				solicitud,
			)
		if !errors.Is(
			err,
			cobertura.ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible,
		) {
			t.Fatalf("callback tardío no invalidó la lectura: %v", err)
		}
		close(ejecutor.disparar)
		if err = <-ejecutor.resultadoCallback; !errors.Is(
			err,
			cobertura.ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida,
		) {
			t.Fatalf("callback tardío volvió a ser útil: %v", err)
		}
		if sesion.llamadas.Load() != 0 {
			t.Fatal("el callback tardío alcanzó el primario")
		}
	})
}
