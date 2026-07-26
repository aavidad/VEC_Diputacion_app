package cobertura

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido = errors.New(
		"contratacion temporal: contrato de lectura historica de cobertura invalido",
	)
	ErrHistoriaResultadoOperacionDecisionCoberturaDivergente = errors.New(
		"contratacion temporal: historia de resultado de cobertura divergente",
	)
	ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible = errors.New(
		"contratacion temporal: lectura historica de cobertura no disponible",
	)
	ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable = errors.New(
		"contratacion temporal: lectura historica de cobertura no confiable",
	)
)

// DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura reúne las filas
// terminales que el lector debe cotejar en el primario. No reconstruye ningún
// catálogo funcional.
type DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Reserva     DatosReservaTerminalOperacionDecisionCobertura
	Recibo      ReciboOperacionDecisionCobertura
	ObservadaEn time.Time
}

// ResultadoHistoricoOperacionDecisionCobertura es la unión cerrada que cruza
// el puerto lector: confirmado o no observable. No observable no prueba
// rollback y no concede permiso para repetir el efecto.
type ResultadoHistoricoOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	solicitud    *SolicitudRecuperacionResultadoOperacionDecisionCobertura
	recibo       *ReciboOperacionDecisionCobertura
	noObservable bool
	observadaEn  time.Time
}

func nuevoResultadoHistoricoConfirmadoOperacionDecisionCobertura(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	evidencia DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura,
) (ResultadoHistoricoOperacionDecisionCobertura, error) {
	if validarEvidenciaHistoricaOperacionDecisionCobertura(
		solicitud,
		evidencia,
	) != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	recibo := clonarReciboOperacionDecisionCobertura(evidencia.Recibo)
	copiaSolicitud, err := clonarSolicitudRecuperacionResultadoOperacionDecisionCobertura(
		solicitud,
	)
	if err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return ResultadoHistoricoOperacionDecisionCobertura{
		solicitud:   &copiaSolicitud,
		recibo:      &recibo,
		observadaEn: evidencia.ObservadaEn,
	}, nil
}

func nuevoResultadoHistoricoNoObservableOperacionDecisionCobertura(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	observadaEn time.Time,
) (ResultadoHistoricoOperacionDecisionCobertura, error) {
	if _, err := solicitud.DatosLectura(); err != nil ||
		!instanteOperacionDecisionCoberturaValido(observadaEn) {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	copiaSolicitud, err :=
		clonarSolicitudRecuperacionResultadoOperacionDecisionCobertura(
			solicitud,
		)
	if err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return ResultadoHistoricoOperacionDecisionCobertura{
		solicitud:    &copiaSolicitud,
		noObservable: true,
		observadaEn:  observadaEn,
	}, nil
}

func (r ResultadoHistoricoOperacionDecisionCobertura) ReciboConfirmadoPara(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
) (ReciboOperacionDecisionCobertura, bool) {
	if r.solicitud == nil || r.recibo == nil || r.noObservable ||
		!solicitudesRecuperacionResultadoOperacionDecisionCoberturaIguales(
			*r.solicitud,
			solicitud,
		) ||
		!instanteOperacionDecisionCoberturaValido(r.observadaEn) {
		return ReciboOperacionDecisionCobertura{}, false
	}
	if validarReciboHistoricoContraSolicitud(solicitud, *r.recibo) != nil {
		return ReciboOperacionDecisionCobertura{}, false
	}
	return clonarReciboOperacionDecisionCobertura(*r.recibo), true
}

func (r ResultadoHistoricoOperacionDecisionCobertura) NoObservablePara(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
) bool {
	_, err := solicitud.DatosLectura()
	return err == nil && r.solicitud != nil &&
		solicitudesRecuperacionResultadoOperacionDecisionCoberturaIguales(
			*r.solicitud,
			solicitud,
		) &&
		r.recibo == nil && r.noObservable &&
		instanteOperacionDecisionCoberturaValido(r.observadaEn)
}

func validarEvidenciaHistoricaOperacionDecisionCobertura(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	evidencia DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura,
) error {
	datos, err := solicitud.DatosLectura()
	reserva := evidencia.Reserva
	if err != nil ||
		reserva.OrganizacionRef != datos.OrganizacionRef ||
		reserva.ExpedienteRef != datos.ExpedienteRef ||
		!domain.ReferenciaOpacaValida(reserva.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(reserva.ExpedienteRef) ||
		reserva.VersionExpediente < 2 ||
		reserva.VersionExpediente >=
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		!domain.ReferenciaOpacaValida(reserva.ReservaRef) ||
		!domain.ReferenciaOpacaValida(reserva.ReciboRef) ||
		!domain.ReferenciaOpacaValida(reserva.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(reserva.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(reserva.EventoRef) ||
		!domain.ReferenciaOpacaValida(reserva.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(reserva.DecisionVECRef) ||
		!solicitud.contieneAmbito(reserva.AmbitoIdempotenciaHMAC) ||
		!parSellosPersistidosOperacionDecisionCoberturaValido(
			reserva.AmbitoIdempotenciaHMAC,
			reserva.HuellaSemanticaHMAC,
		) ||
		reserva.RevisionCercado == 0 ||
		reserva.RevisionCercado >
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(reserva.ObservadaEnDB) ||
		!instanteOperacionDecisionCoberturaValido(evidencia.ObservadaEn) ||
		evidencia.ObservadaEn.Before(evidencia.Recibo.ConfirmadaEn) ||
		validarReciboHistoricoContraReserva(
			solicitud,
			reserva,
			evidencia.Recibo,
		) != nil {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func validarReciboHistoricoContraReserva(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	reserva DatosReservaTerminalOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) error {
	if validarReciboHistoricoContraSolicitud(solicitud, recibo) != nil ||
		recibo.ReservaRef != reserva.ReservaRef ||
		recibo.ReciboRef != reserva.ReciboRef ||
		recibo.AuditoriaRef != reserva.AuditoriaRef ||
		recibo.CorrelacionVECRef != reserva.CorrelacionVECRef ||
		recibo.DecisionVECRef != reserva.DecisionVECRef ||
		recibo.AmbitoIdempotenciaHMAC != reserva.AmbitoIdempotenciaHMAC ||
		recibo.HuellaSemanticaHMAC != reserva.HuellaSemanticaHMAC ||
		recibo.RevisionCercado != reserva.RevisionCercado ||
		recibo.ConfirmadaEn.Before(reserva.ObservadaEnDB) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if aplicada, existe := recibo.ResultadoAplicado(); existe {
		if aplicada.VersionResultante != reserva.VersionExpediente+1 ||
			aplicada.EventoRef != reserva.EventoRef ||
			aplicada.ActuacionRef != reserva.ActuacionRef {
			return ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
	}
	return nil
}

func validarReciboHistoricoContraSolicitud(
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) error {
	if !solicitud.contieneAmbito(recibo.AmbitoIdempotenciaHMAC) ||
		!parSellosPersistidosOperacionDecisionCoberturaValido(
			recibo.AmbitoIdempotenciaHMAC,
			recibo.HuellaSemanticaHMAC,
		) ||
		!domain.ReferenciaOpacaValida(recibo.ReciboRef) ||
		!domain.ReferenciaOpacaValida(recibo.ReservaRef) ||
		!domain.ReferenciaOpacaValida(recibo.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(recibo.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(recibo.DecisionVECRef) ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			recibo.DecisionVECHuellaSHA256,
		) ||
		!dominiovec.CodigoResultadoEvaluacionAutorizacionV3Valido(
			recibo.CodigoProbatorioVEC,
			recibo.ConcedidaVEC,
		) ||
		recibo.RevisionCercado == 0 ||
		recibo.RevisionCercado >
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(recibo.ConfirmadaEn) ||
		(recibo.Aplicada == nil) == (recibo.DenegadaVEC == nil) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if recibo.Aplicada != nil {
		if !recibo.ConcedidaVEC ||
			!referenciaDecisionCoberturaLigadaAHuella(
				recibo.Aplicada.DecisionCoberturaRef,
				recibo.Aplicada.DecisionCoberturaHuella,
			) ||
			!domain.ReferenciaOpacaValida(recibo.Aplicada.EventoRef) ||
			!domain.ReferenciaOpacaValida(recibo.Aplicada.ActuacionRef) {
			return ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
		return nil
	}
	if recibo.ConcedidaVEC {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func parSellosPersistidosOperacionDecisionCoberturaValido(
	ambito string,
	semantica string,
) bool {
	ambitos, errAmbito := ports.NuevaColeccionSellosHMAC(ambito, nil)
	semanticas, errSemantica := ports.NuevaColeccionSellosHMAC(
		semantica,
		nil,
	)
	if errAmbito != nil || errSemantica != nil {
		return false
	}
	_, _, err := ports.ParActivoColeccionesHMAC(
		ambitos,
		dominioAmbitoOperacionDecisionCobertura,
		semanticas,
		dominioSemanticaOperacionDecisionCobertura,
	)
	return err == nil
}

// LectorResultadoHistoricoOperacionDecisionCobertura solo observa terminales;
// no reserva, reapropia, confirma ni reconcilia efectos.
type LectorResultadoHistoricoOperacionDecisionCobertura interface {
	LeerResultadoHistoricoOperacionDecisionCobertura(
		context.Context,
		SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	) (ResultadoHistoricoOperacionDecisionCobertura, error)
	lectorResultadoHistoricoOperacionDecisionCoberturaSellado()
}
