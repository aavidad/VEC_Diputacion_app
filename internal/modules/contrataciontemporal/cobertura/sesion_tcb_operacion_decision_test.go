package cobertura_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestSesionTCBDespliegaConcesionYDenegacionSinCruzarRamas(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	casos := []struct {
		nombre       string
		orden        cobertura.OrdenOperacionDecisionCobertura
		recibo       cobertura.ReciboOperacionDecisionCobertura
		rama         cobertura.RamaSesionTCBOperacionDecisionCobertura
		concedidaVEC bool
	}{
		{
			nombre:       "concesión",
			orden:        escenario.ordenConcedida,
			recibo:       escenario.reciboConcedido,
			rama:         cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida,
			concedidaVEC: true,
		},
		{
			nombre: "denegación",
			orden:  escenario.ordenDenegada,
			recibo: escenario.reciboDenegado,
			rama:   cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
				recibo: datosReciboSesionTCBPrueba(caso.recibo),
			}
			transaccion, err :=
				cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
					ejecutor,
				)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err :=
				cobertura.ConfirmarOperacionDecisionCobertura(
					context.Background(),
					transaccion,
					caso.orden,
				)
			if err != nil {
				t.Fatalf("confirmación TCB rechazada: %v", err)
			}
			recibo, err := resultado.ReciboPara(caso.orden)
			if err != nil || recibo.DecisionVECRef != caso.recibo.DecisionVECRef {
				t.Fatalf("recibo nominal inesperado: %#v, %v", recibo, err)
			}
			sesion := ejecutor.ultimaSesion()
			if sesion == nil {
				t.Fatal("el ejecutor no recibió sesión")
			}
			pasos, cabecera, vec, consumos := sesion.instantanea()
			if cabecera.Rama != caso.rama ||
				vec.Concedida != caso.concedidaVEC ||
				ejecutor.llamadas.Load() != 1 {
				t.Fatalf(
					"despliegue incoherente: rama=%s vec=%t llamadas=%d",
					cabecera.Rama,
					vec.Concedida,
					ejecutor.llamadas.Load(),
				)
			}
			if caso.concedidaVEC {
				if cabecera.NumeroConsumosC1 == 0 ||
					uint64(len(consumos)) != cabecera.NumeroConsumosC1 ||
					pasos[0] != "abrir" ||
					pasos[1] != "gobierno" ||
					pasos[2] != "vec" ||
					pasos[len(pasos)-2] != "concesion" ||
					pasos[len(pasos)-1] != "confirmar" {
					t.Fatalf("secuencia grant inválida: %v", pasos)
				}
				for indice, consumo := range consumos {
					if consumo.Posicion != uint64(indice)+1 ||
						consumo.Total != uint64(len(consumos)) ||
						pasos[indice+3] != "c1" {
						t.Fatalf(
							"consumo C1 fuera de orden: %#v / %v",
							consumo,
							pasos,
						)
					}
				}
				return
			}
			esperados := []string{"abrir", "vec", "denegacion", "confirmar"}
			if !reflect.DeepEqual(pasos, esperados) ||
				len(consumos) != 0 ||
				cabecera.NumeroConsumosC1 != 0 ||
				cabecera.PreparacionC1Ref != "" ||
				cabecera.AnalisisRef != "" {
				t.Fatalf(
					"deny transportó gobierno/C1/C2: %v, %#v",
					pasos,
					cabecera,
				)
			}
			sesion.mu.Lock()
			gobiernoVacio := reflect.ValueOf(sesion.gobierno).IsZero()
			concesionVacia := reflect.ValueOf(sesion.concesion).IsZero()
			denegacionValida := sesion.denegacion.ReservaRef ==
				caso.recibo.ReservaRef
			sesion.mu.Unlock()
			if !gobiernoVacio || !concesionVacia || !denegacionValida {
				t.Fatal("la rama deny recibió un fragmento positivo")
			}
		})
	}
}

func TestSesionTCBExigeCallbackSincronoExactamenteUnaVez(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	casos := []struct {
		nombre    string
		configura func(*ejecutorSesionTCBOperacionDecisionPrueba)
	}{
		{
			nombre: "callback omitido",
			configura: func(e *ejecutorSesionTCBOperacionDecisionPrueba) {
				e.omitirCallback = true
			},
		},
		{
			nombre: "callback repetido",
			configura: func(e *ejecutorSesionTCBOperacionDecisionPrueba) {
				e.repetirCallback = true
			},
		},
		{
			nombre: "callback retenido",
			configura: func(e *ejecutorSesionTCBOperacionDecisionPrueba) {
				e.retenerCallback = true
			},
		},
		{
			nombre: "sesión typed nil",
			configura: func(e *ejecutorSesionTCBOperacionDecisionPrueba) {
				e.typedNilSesion = true
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
				recibo: datosReciboSesionTCBPrueba(
					escenario.reciboConcedido,
				),
			}
			caso.configura(ejecutor)
			transaccion, err :=
				cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
					ejecutor,
				)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = cobertura.ConfirmarOperacionDecisionCobertura(
				context.Background(),
				transaccion,
				escenario.ordenConcedida,
			); err == nil {
				t.Fatal("el ejecutor violó el ciclo de vida sin rechazo")
			}
			if ejecutor.retenerCallback {
				if errRetenido := ejecutor.ejecutarRetenido(); !errors.Is(
					errRetenido,
					cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
				) {
					t.Fatalf(
						"callback escapado todavía ejecutable: %v",
						errRetenido,
					)
				}
			}
		})
	}

	var ejecutorNulo *ejecutorSesionTCBOperacionDecisionPrueba
	if _, err := cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
		ejecutorNulo,
	); !errors.Is(
		err,
		cobertura.ErrContratoConfirmacionOperacionDecisionCoberturaInvalido,
	) {
		t.Fatalf("ejecutor typed nil aceptado por composición: %v", err)
	}
}

func TestSesionTCBCallbackRetenidoTrasExitoNoPuedeReabrirEfectos(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:                 datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		retenerDespuesCallback: true,
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
		context.Background(),
		transaccion,
		escenario.ordenConcedida,
	)
	if err != nil {
		t.Fatalf("la primera ejecución válida falló: %v", err)
	}
	if _, err = resultado.ReciboPara(escenario.ordenConcedida); err != nil {
		t.Fatal(err)
	}
	sesion := ejecutor.ultimaSesion()
	pasosAntes, _, _, _ := sesion.instantanea()
	if err = ejecutor.ejecutarRetenido(); !errors.Is(
		err,
		cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("el callback retenido volvió a ser ejecutable: %v", err)
	}
	pasosDespues, _, _, _ := sesion.instantanea()
	ejecutor.mu.Lock()
	numeroSesiones := len(ejecutor.sesiones)
	ejecutor.mu.Unlock()
	if numeroSesiones != 1 ||
		!reflect.DeepEqual(pasosAntes, pasosDespues) {
		t.Fatalf(
			"el escape creó sesión o efectos: sesiones=%d antes=%v después=%v",
			numeroSesiones,
			pasosAntes,
			pasosDespues,
		)
	}
	if _, err = resultado.ReciboPara(escenario.ordenConcedida); err != nil {
		t.Fatalf("el escape alteró el resultado ya emitido: %v", err)
	}
}

func TestSesionTCBCommitAmbiguoNuncaPublicaNiReintenta(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:       datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		errorDespues: errPrivadoSesionTCBPrueba,
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
		context.Background(),
		transaccion,
		escenario.ordenConcedida,
	)
	if errors.Is(err, errPrivadoSesionTCBPrueba) ||
		errors.Unwrap(err) != nil ||
		!errors.Is(
			err,
			cobertura.ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible,
		) ||
		err.Error() !=
			cobertura.ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible.Error() {
		t.Fatalf("error ambiguo no quedó saneado: %v", err)
	}
	formatoError := fmt.Sprintf("%v %+v", err, err)
	if strings.Contains(formatoError, errPrivadoSesionTCBPrueba.Error()) {
		t.Fatalf("el formato filtró la causa privada: %q", formatoError)
	}
	var logError bytes.Buffer
	slog.New(slog.NewJSONHandler(&logError, nil)).Error(
		"confirmación",
		"error",
		err,
	)
	if strings.Contains(logError.String(), errPrivadoSesionTCBPrueba.Error()) {
		t.Fatalf("el log filtró la causa privada: %s", logError.String())
	}
	if _, errResultado := resultado.ReciboPara(
		escenario.ordenConcedida,
	); errResultado == nil {
		t.Fatal("un error de COMMIT publicó resultado nominal")
	}

	ejecutorIntento := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:       datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		errorDespues: errPrivadoSesionTCBPrueba,
	}
	transaccionIntento, _ :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
			ejecutorIntento,
		)
	intento, err := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		context.Background(),
		transaccionIntento,
		escenario.ordenConcedida,
	)
	if !errors.Is(
		err,
		cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo,
	) ||
		ejecutorIntento.llamadas.Load() != 1 {
		t.Fatalf("la ambigüedad reintentó o se perdió: %v", err)
	}
	if _, confirmada := intento.ConfirmacionPara(
		escenario.ordenConcedida,
	); confirmada {
		t.Fatal("la ambigüedad fabricó confirmación")
	}
	if _, reconciliar := intento.ReconciliacionPara(
		escenario.ordenConcedida,
	); !reconciliar {
		t.Fatal("la ambigüedad no produjo reconciliación primaria")
	}
}

func TestSesionTCBFalloAntesCommitNoExponeCausaNiReconcilia(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	for _, caso := range []struct {
		nombre   string
		ejecutor *ejecutorSesionTCBOperacionDecisionPrueba
	}{
		{
			nombre: "ejecutor falla antes del callback",
			ejecutor: &ejecutorSesionTCBOperacionDecisionPrueba{
				errorAntes: errPrivadoSesionTCBPrueba,
			},
		},
		{
			nombre: "callback falla antes de recibo valido",
			ejecutor: &ejecutorSesionTCBOperacionDecisionPrueba{
				recibo: datosReciboSesionTCBPrueba(
					escenario.reciboConcedido,
				),
				errorEnSesion: "gobierno",
			},
		},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			transaccion, err :=
				cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
					caso.ejecutor,
				)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err :=
				cobertura.ConfirmarOperacionDecisionCobertura(
					context.Background(),
					transaccion,
					escenario.ordenConcedida,
				)
			if !errors.Is(
				err,
				cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaNoDisponible,
			) ||
				errors.Is(err, errPrivadoSesionTCBPrueba) ||
				errors.Unwrap(err) != nil {
				t.Fatalf("fallo pre-COMMIT mal clasificado: %v", err)
			}
			if strings.Contains(
				fmt.Sprintf("%v %+v", err, err),
				errPrivadoSesionTCBPrueba.Error(),
			) {
				t.Fatalf("el error filtró la causa privada: %v", err)
			}
			if _, errResultado := resultado.ReciboPara(
				escenario.ordenConcedida,
			); errResultado == nil {
				t.Fatal("el fallo pre-COMMIT publicó resultado")
			}

			intento, err :=
				cobertura.IntentarConfirmacionOperacionDecisionCobertura(
					context.Background(),
					transaccion,
					escenario.ordenConcedida,
				)
			if !errors.Is(
				err,
				cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaNoDisponible,
			) {
				t.Fatalf("el intento perdió el fallo pre-COMMIT: %v", err)
			}
			if _, reconciliar := intento.ReconciliacionPara(
				escenario.ordenConcedida,
			); reconciliar {
				t.Fatal("el fallo pre-COMMIT fabricó reconciliación")
			}
			if !intento.FalloAntesCommitPara(escenario.ordenConcedida) ||
				intento.FalloAntesCommitPara(escenario.ordenDenegada) {
				t.Fatal("la prueba pre-COMMIT no quedó ligada a la orden exacta")
			}
		})
	}
}

func TestSesionTCBEsperaCallbackCruzadoYBloqueaConfirmacionPosterior(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	entrada := make(chan struct{})
	continuar := make(chan struct{})
	porRetornar := make(chan struct{})
	resultadoCallback := make(chan error, 1)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:                 datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		callbackAsincrono:      true,
		retenerDespuesCallback: true,
		bloquearEnSesion:       "abrir",
		entradaBloqueo:         entrada,
		continuarBloqueo:       continuar,
		ejecutorPorRetornar:    porRetornar,
		resultadoCallback:      resultadoCallback,
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	type salida struct {
		resultado cobertura.ResultadoConfirmacionOperacionDecisionCobertura
		err       error
	}
	salidaOperacion := make(chan salida, 1)
	go func() {
		resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
			context.Background(),
			transaccion,
			escenario.ordenConcedida,
		)
		salidaOperacion <- salida{resultado: resultado, err: err}
	}()

	<-entrada
	<-porRetornar
	select {
	case resultado := <-salidaOperacion:
		t.Fatalf(
			"el wrapper retornó con el callback activo: %v",
			resultado.err,
		)
	case <-time.After(25 * time.Millisecond):
	}
	close(continuar)
	if errCallback := <-resultadoCallback; !errors.Is(
		errCallback,
		cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("el callback cruzado no fue abortado: %v", errCallback)
	}
	salidaFinal := <-salidaOperacion
	if !errors.Is(
		salidaFinal.err,
		cobertura.ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("carrera mal clasificada: %v", salidaFinal.err)
	}
	sesion := ejecutor.ultimaSesion()
	pasosAntes, _, _, _ := sesion.instantanea()
	if !reflect.DeepEqual(pasosAntes, []string{"abrir"}) {
		t.Fatalf("hubo efectos posteriores al retorno: %v", pasosAntes)
	}
	if err = ejecutor.ejecutarRetenido(); !errors.Is(
		err,
		cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("el callback escapado volvió a ser útil: %v", err)
	}
	pasosDespues, _, _, _ := sesion.instantanea()
	if !reflect.DeepEqual(pasosAntes, pasosDespues) {
		t.Fatalf(
			"la reutilización alteró el adaptador: antes=%v después=%v",
			pasosAntes,
			pasosDespues,
		)
	}
}

func TestSesionTCBReciboTerminadoTrasRetornoEjecutorNuncaSePublica(
	t *testing.T,
) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	entrada := make(chan struct{})
	continuar := make(chan struct{})
	porRetornar := make(chan struct{})
	resultadoCallback := make(chan error, 1)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:              datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		callbackAsincrono:   true,
		bloquearEnSesion:    "confirmar",
		entradaBloqueo:      entrada,
		continuarBloqueo:    continuar,
		ejecutorPorRetornar: porRetornar,
		resultadoCallback:   resultadoCallback,
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	type salida struct {
		resultado cobertura.ResultadoConfirmacionOperacionDecisionCobertura
		err       error
	}
	salidaOperacion := make(chan salida, 1)
	go func() {
		resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
			context.Background(),
			transaccion,
			escenario.ordenConcedida,
		)
		salidaOperacion <- salida{resultado: resultado, err: err}
	}()

	<-entrada
	<-porRetornar
	close(continuar)
	if errCallback := <-resultadoCallback; !errors.Is(
		errCallback,
		cobertura.ErrSesionTCBOperacionDecisionCoberturaInvalida,
	) {
		t.Fatalf("el callback tardío no fue invalidado: %v", errCallback)
	}
	salidaFinal := <-salidaOperacion
	if !errors.Is(
		salidaFinal.err,
		cobertura.ErrEjecucionSesionTCBOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("el recibo tardío no quedó ambiguo: %v", salidaFinal.err)
	}
	if _, err = salidaFinal.resultado.ReciboPara(
		escenario.ordenConcedida,
	); err == nil {
		t.Fatal("el recibo terminado tras el retorno se publicó")
	}
}

func TestSesionTCBCancelacionPreviaYTardia(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ctxPrevio, cancelarPrevio := context.WithCancel(context.Background())
	cancelarPrevio()
	ejecutorPrevio := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(escenario.reciboConcedido),
	}
	transaccionPrevia, _ :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
			ejecutorPrevio,
		)
	if _, err := cobertura.ConfirmarOperacionDecisionCobertura(
		ctxPrevio,
		transaccionPrevia,
		escenario.ordenConcedida,
	); !errors.Is(err, context.Canceled) ||
		ejecutorPrevio.llamadas.Load() != 0 {
		t.Fatalf("cancelación previa abrió la sesión: %v", err)
	}

	ctxTardio, cancelarTardio := context.WithCancel(context.Background())
	ejecutorTardio := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo:          datosReciboSesionTCBPrueba(escenario.reciboConcedido),
		despuesCallback: cancelarTardio,
	}
	transaccionTardia, _ :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
			ejecutorTardio,
		)
	resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
		ctxTardio,
		transaccionTardia,
		escenario.ordenConcedida,
	)
	if err != nil {
		t.Fatalf("cancelación posterior al COMMIT anuló recibo: %v", err)
	}
	if _, err = resultado.ReciboPara(escenario.ordenConcedida); err != nil {
		t.Fatal(err)
	}
}

func TestSesionTCBRechazaReciboAdulteradoYErrorIntermedio(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	adulterado := datosReciboSesionTCBPrueba(escenario.reciboConcedido)
	adulterado.ReservaRef = "reserva_ajena"
	for _, ejecutor := range []*ejecutorSesionTCBOperacionDecisionPrueba{
		{recibo: adulterado},
		{
			recibo:        datosReciboSesionTCBPrueba(escenario.reciboConcedido),
			errorEnSesion: "gobierno",
		},
	} {
		transaccion, err :=
			cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(
				ejecutor,
			)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = cobertura.ConfirmarOperacionDecisionCobertura(
			context.Background(),
			transaccion,
			escenario.ordenConcedida,
		); err == nil {
			t.Fatal("recibo o paso intermedio inválido aceptado")
		}
	}
}

func TestSesionTCBTransaccionEsSeguraParaUsoConcurrente(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionOrdenC3(t)
	ejecutor := &ejecutorSesionTCBOperacionDecisionPrueba{
		recibo: datosReciboSesionTCBPrueba(escenario.reciboConcedido),
	}
	transaccion, err :=
		cobertura.NuevaTransaccionOperacionDecisionCoberturaTCB(ejecutor)
	if err != nil {
		t.Fatal(err)
	}
	const paralelismo = 32
	var wg sync.WaitGroup
	errores := make(chan error, paralelismo)
	for indice := 0; indice < paralelismo; indice++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultado, err := cobertura.ConfirmarOperacionDecisionCobertura(
				context.Background(),
				transaccion,
				escenario.ordenConcedida,
			)
			if err == nil {
				_, err = resultado.ReciboPara(escenario.ordenConcedida)
			}
			errores <- err
		}()
	}
	wg.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("confirmación concurrente falló: %v", err)
		}
	}
	if ejecutor.llamadas.Load() != paralelismo {
		t.Fatalf("llamadas perdidas: %d", ejecutor.llamadas.Load())
	}
}
