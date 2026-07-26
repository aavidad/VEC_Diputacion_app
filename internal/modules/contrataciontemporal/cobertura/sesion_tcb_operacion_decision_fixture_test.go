package cobertura_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

var errPrivadoSesionTCBPrueba = errors.New(
	"error privado del ejecutor TCB de prueba",
)

func datosReciboSesionTCBPrueba(
	recibo cobertura.ReciboOperacionDecisionCobertura,
) cobertura.DatosReciboSesionTCBOperacionDecisionCobertura {
	datos := cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{
		ReciboRef:               recibo.ReciboRef,
		ReservaRef:              recibo.ReservaRef,
		AuditoriaRef:            recibo.AuditoriaRef,
		CorrelacionVECRef:       recibo.CorrelacionVECRef,
		DecisionVECRef:          recibo.DecisionVECRef,
		DecisionVECHuellaSHA256: recibo.DecisionVECHuellaSHA256,
		CodigoProbatorioVEC:     recibo.CodigoProbatorioVEC,
		ConcedidaVEC:            recibo.ConcedidaVEC,
		RevisionCercado:         recibo.RevisionCercado,
		AmbitoIdempotenciaHMAC:  recibo.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     recibo.HuellaSemanticaHMAC,
		ConfirmadaEn:            recibo.ConfirmadaEn,
	}
	if recibo.Aplicada != nil {
		datos.Aplicada = true
		datos.DecisionCoberturaRef = recibo.Aplicada.DecisionCoberturaRef
		datos.DecisionCoberturaHuella =
			recibo.Aplicada.DecisionCoberturaHuella
		datos.VersionResultante = recibo.Aplicada.VersionResultante
		datos.EventoRef = recibo.Aplicada.EventoRef
		datos.ActuacionRef = recibo.Aplicada.ActuacionRef
	} else {
		datos.DenegadaVEC = true
	}
	return datos
}

type sesionTCBOperacionDecisionPrueba struct {
	mu               sync.Mutex
	senalBloqueo     sync.Once
	pasos            []string
	recibo           cobertura.DatosReciboSesionTCBOperacionDecisionCobertura
	errorEn          string
	cabecera         cobertura.DatosCabeceraSesionTCBOperacionDecisionCobertura
	gobierno         cobertura.DatosGobiernoSesionTCBOperacionDecisionCobertura
	vec              cobertura.DatosDecisionVECSesionTCBOperacionDecisionCobertura
	consumos         []cobertura.DatosConsumoC1SesionTCBOperacionDecisionCobertura
	concesion        cobertura.DatosEfectoConcedidoSesionTCBOperacionDecisionCobertura
	denegacion       cobertura.DatosTerminalDenegadoSesionTCBOperacionDecisionCobertura
	cabeceraOpaca    cobertura.CabeceraSesionTCBOperacionDecisionCobertura
	gobiernoOpaco    cobertura.GobiernoSesionTCBOperacionDecisionCobertura
	consumosOpacos   []cobertura.ConsumoC1SesionTCBOperacionDecisionCobertura
	concesionOpaca   cobertura.EfectoConcedidoSesionTCBOperacionDecisionCobertura
	denegacionOpaca  cobertura.TerminalDenegadoSesionTCBOperacionDecisionCobertura
	bloquearEn       string
	entradaBloqueo   chan struct{}
	continuarBloqueo chan struct{}
}

func (s *sesionTCBOperacionDecisionPrueba) registrar(
	paso string,
) error {
	s.pasos = append(s.pasos, paso)
	if s.bloquearEn == paso {
		s.senalBloqueo.Do(func() {
			close(s.entradaBloqueo)
		})
		<-s.continuarBloqueo
	}
	if s.errorEn == paso {
		return errPrivadoSesionTCBPrueba
	}
	return nil
}

func (s *sesionTCBOperacionDecisionPrueba) Abrir(
	cabecera cobertura.CabeceraSesionTCBOperacionDecisionCobertura,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	datos, err := cabecera.Datos()
	if err != nil {
		return err
	}
	s.cabecera = datos
	s.cabeceraOpaca = cabecera
	return s.registrar("abrir")
}

func (s *sesionTCBOperacionDecisionPrueba) Gobierno(
	gobierno cobertura.GobiernoSesionTCBOperacionDecisionCobertura,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	datos, err := gobierno.Datos()
	if err != nil {
		return err
	}
	s.gobierno = datos
	s.gobiernoOpaco = gobierno
	return s.registrar("gobierno")
}

func (s *sesionTCBOperacionDecisionPrueba) DecisionVEC(
	decision cobertura.DecisionVECSesionTCBOperacionDecisionCobertura,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	datos, err := decision.Datos()
	if err != nil {
		return err
	}
	s.vec = datos
	return s.registrar("vec")
}

func (s *sesionTCBOperacionDecisionPrueba) ConsumoC1(
	consumo cobertura.ConsumoC1SesionTCBOperacionDecisionCobertura,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	datos, err := consumo.Datos()
	if err != nil {
		return err
	}
	s.consumos = append(s.consumos, datos)
	s.consumosOpacos = append(s.consumosOpacos, consumo)
	return s.registrar("c1")
}

func (s *sesionTCBOperacionDecisionPrueba) Concesion(
	efecto cobertura.EfectoConcedidoSesionTCBOperacionDecisionCobertura,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	datos, err := efecto.Datos()
	if err != nil {
		return err
	}
	s.concesion = datos
	s.concesionOpaca = efecto
	return s.registrar("concesion")
}

func (s *sesionTCBOperacionDecisionPrueba) Denegacion(
	terminal cobertura.TerminalDenegadoSesionTCBOperacionDecisionCobertura,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	datos, err := terminal.Datos()
	if err != nil {
		return err
	}
	s.denegacion = datos
	s.denegacionOpaca = terminal
	return s.registrar("denegacion")
}

func (s *sesionTCBOperacionDecisionPrueba) Confirmar(
	ctx context.Context,
) (cobertura.DatosReciboSesionTCBOperacionDecisionCobertura, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	if err := s.registrar("confirmar"); err != nil {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	return s.recibo, nil
}

func (s *sesionTCBOperacionDecisionPrueba) instantanea() (
	[]string,
	cobertura.DatosCabeceraSesionTCBOperacionDecisionCobertura,
	cobertura.DatosDecisionVECSesionTCBOperacionDecisionCobertura,
	[]cobertura.DatosConsumoC1SesionTCBOperacionDecisionCobertura,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.pasos...),
		s.cabecera,
		s.vec,
		append(
			[]cobertura.DatosConsumoC1SesionTCBOperacionDecisionCobertura(nil),
			s.consumos...,
		)
}

type ejecutorSesionTCBOperacionDecisionPrueba struct {
	mu                     sync.Mutex
	recibo                 cobertura.DatosReciboSesionTCBOperacionDecisionCobertura
	errorAntes             error
	errorDespues           error
	errorEnSesion          string
	omitirCallback         bool
	repetirCallback        bool
	retenerCallback        bool
	retenerDespuesCallback bool
	despuesCallback        func()
	typedNilSesion         bool
	callbackAsincrono      bool
	bloquearEnSesion       string
	entradaBloqueo         chan struct{}
	continuarBloqueo       chan struct{}
	ejecutorPorRetornar    chan struct{}
	resultadoCallback      chan error
	llamadas               atomic.Int32
	sesiones               []*sesionTCBOperacionDecisionPrueba
	callbackRetenido       func(cobertura.SesionTCBOperacionDecisionCobertura) error
	sesionRetenida         cobertura.SesionTCBOperacionDecisionCobertura
}

func (e *ejecutorSesionTCBOperacionDecisionPrueba) nuevaSesion() cobertura.SesionTCBOperacionDecisionCobertura {
	if e.typedNilSesion {
		var nula *sesionTCBOperacionDecisionPrueba
		return nula
	}
	sesion := &sesionTCBOperacionDecisionPrueba{
		recibo:           e.recibo,
		errorEn:          e.errorEnSesion,
		bloquearEn:       e.bloquearEnSesion,
		entradaBloqueo:   e.entradaBloqueo,
		continuarBloqueo: e.continuarBloqueo,
	}
	e.mu.Lock()
	e.sesiones = append(e.sesiones, sesion)
	e.mu.Unlock()
	return sesion
}

func (e *ejecutorSesionTCBOperacionDecisionPrueba) EjecutarSesionTCB(
	_ context.Context,
	callback func(cobertura.SesionTCBOperacionDecisionCobertura) error,
) error {
	e.llamadas.Add(1)
	if e.errorAntes != nil {
		return e.errorAntes
	}
	if e.retenerCallback {
		e.mu.Lock()
		e.callbackRetenido = callback
		e.mu.Unlock()
		return nil
	}
	if e.omitirCallback {
		return nil
	}
	sesion := e.nuevaSesion()
	if e.callbackAsincrono {
		if e.retenerDespuesCallback {
			e.mu.Lock()
			e.callbackRetenido = callback
			e.sesionRetenida = sesion
			e.mu.Unlock()
		}
		go func() {
			e.resultadoCallback <- callback(sesion)
		}()
		<-e.entradaBloqueo
		close(e.ejecutorPorRetornar)
		return e.errorDespues
	}
	if err := callback(sesion); err != nil {
		return err
	}
	if e.retenerDespuesCallback {
		e.mu.Lock()
		e.callbackRetenido = callback
		e.sesionRetenida = sesion
		e.mu.Unlock()
	}
	if e.repetirCallback {
		_ = callback(e.nuevaSesion())
	}
	if e.despuesCallback != nil {
		e.despuesCallback()
	}
	return e.errorDespues
}

func (e *ejecutorSesionTCBOperacionDecisionPrueba) ultimaSesion() *sesionTCBOperacionDecisionPrueba {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.sesiones) == 0 {
		return nil
	}
	return e.sesiones[len(e.sesiones)-1]
}

func (e *ejecutorSesionTCBOperacionDecisionPrueba) ejecutarRetenido() error {
	e.mu.Lock()
	callback := e.callbackRetenido
	sesion := e.sesionRetenida
	e.mu.Unlock()
	if callback == nil {
		return errors.New("callback de prueba ausente")
	}
	if sesion == nil {
		sesion = e.nuevaSesion()
	}
	return callback(sesion)
}
