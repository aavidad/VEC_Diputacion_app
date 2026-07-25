package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func (t *transaccionConfirmacionPrueba) EjecutarSesionTCB(
	ctx context.Context,
	callback func(cobertura.SesionTCBOperacionDecisionCobertura) error,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.llamadas++
	if t.falloAntesCommit {
		return errors.New("fallo privado anterior al callback")
	}
	sesion := &sesionTCBConfirmacionAplicacionPrueba{transaccion: t}
	if err := callback(sesion); err != nil {
		return err
	}
	if t.cancelar != nil {
		t.cancelar()
	}
	if t.ambigua || t.errorRetorno != nil {
		return errors.New("respuesta perdida después del callback")
	}
	return nil
}

type sesionTCBConfirmacionAplicacionPrueba struct {
	transaccion *transaccionConfirmacionPrueba
}

func (*sesionTCBConfirmacionAplicacionPrueba) Abrir(
	cobertura.CabeceraSesionTCBOperacionDecisionCobertura,
) error {
	return nil
}

func (*sesionTCBConfirmacionAplicacionPrueba) Gobierno(
	cobertura.GobiernoSesionTCBOperacionDecisionCobertura,
) error {
	return nil
}

func (*sesionTCBConfirmacionAplicacionPrueba) DecisionVEC(
	cobertura.DecisionVECSesionTCBOperacionDecisionCobertura,
) error {
	return nil
}

func (*sesionTCBConfirmacionAplicacionPrueba) ConsumoC1(
	cobertura.ConsumoC1SesionTCBOperacionDecisionCobertura,
) error {
	return nil
}

func (*sesionTCBConfirmacionAplicacionPrueba) Concesion(
	cobertura.EfectoConcedidoSesionTCBOperacionDecisionCobertura,
) error {
	return nil
}

func (*sesionTCBConfirmacionAplicacionPrueba) Denegacion(
	cobertura.TerminalDenegadoSesionTCBOperacionDecisionCobertura,
) error {
	return nil
}

func (s *sesionTCBConfirmacionAplicacionPrueba) Confirmar(
	_ context.Context,
) (cobertura.DatosReciboSesionTCBOperacionDecisionCobertura, error) {
	t := s.transaccion
	if t.salidaInvalida {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, nil
	}
	recibo, err := reciboConfirmacionCoberturaPrueba(
		t.idempotencia,
		t.vec,
		t.reloj.Ahora(),
		t.aplicada,
	)
	if err != nil {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
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
	if aplicada, existe := recibo.ResultadoAplicado(); existe {
		datos.Aplicada = true
		datos.DecisionCoberturaRef = aplicada.DecisionCoberturaRef
		datos.DecisionCoberturaHuella = aplicada.DecisionCoberturaHuella
		datos.VersionResultante = aplicada.VersionResultante
		datos.EventoRef = aplicada.EventoRef
		datos.ActuacionRef = aplicada.ActuacionRef
	}
	if _, existe := recibo.ResultadoDenegadoVEC(); existe {
		datos.DenegadaVEC = true
	}
	return datos, nil
}
