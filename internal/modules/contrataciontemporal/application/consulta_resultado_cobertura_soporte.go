package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func (s *ServicioConsultaResultadoCobertura) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.contextos) &&
		!dependenciaNula(s.accesos) && !dependenciaNula(s.sellador) &&
		!dependenciaNula(s.reloj) && !dependenciaNula(s.lector)
}

func (s *ServicioConsultaResultadoCobertura) consultarValidada(
	ctx context.Context,
	solicitud SolicitudConsultaResultadoCobertura,
) (ResultadoConsultaResultadoCobertura, error) {
	contexto, err :=
		s.contextos.ResolverContextoRecuperacionResultadoCobertura(ctx)
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			s.clasificarFalloAutorizacionConsulta(ctx, err)
	}
	instanteInicial, err := s.ahoraConsultaResultado(ctx)
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			s.clasificarFalloAutorizacionConsulta(ctx, err)
	}
	if _, _, _, err := contexto.DatosEn(instanteInicial); err != nil {
		return ResultadoConsultaResultadoCobertura{},
			s.denegarConsultaResultado(ctx)
	}
	if err := s.autorizarConsultaResultado(
		ctx,
		solicitud.ExpedienteRef,
		contexto,
		instanteInicial,
	); err != nil {
		return ResultadoConsultaResultadoCobertura{}, err
	}
	preimagen, err :=
		cobertura.NuevaPreimagenAmbitoRecuperacionOperacionDecisionCobertura(
			solicitud.ClaveIdempotencia,
			solicitud.ExpedienteRef,
			contexto,
			instanteInicial,
		)
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			ErrConsultaResultadoCoberturaNoConfiable
	}
	ambitos, err := s.sellador.SellarAmbitoOperacionDecisionCobertura(
		ctx,
		preimagen,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return ResultadoConsultaResultadoCobertura{}, errContexto
	}
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			ErrConsultaResultadoCoberturaNoDisponible
	}
	consulta, err :=
		cobertura.NuevaSolicitudRecuperacionResultadoOperacionDecisionCobertura(
			preimagen,
			ambitos,
		)
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			ErrConsultaResultadoCoberturaNoConfiable
	}

	instanteLectura, err := s.ahoraConsultaResultado(ctx)
	if err != nil || instanteLectura.Before(instanteInicial) {
		return ResultadoConsultaResultadoCobertura{},
			s.clasificarFalloAutorizacionConsulta(ctx, err)
	}
	if _, _, _, err := contexto.DatosEn(instanteLectura); err != nil {
		return ResultadoConsultaResultadoCobertura{},
			s.denegarConsultaResultado(ctx)
	}
	if err := s.autorizarConsultaResultado(
		ctx,
		solicitud.ExpedienteRef,
		contexto,
		instanteLectura,
	); err != nil {
		return ResultadoConsultaResultadoCobertura{}, err
	}
	historico, err :=
		s.lector.LeerResultadoHistoricoOperacionDecisionCobertura(
			ctx,
			consulta,
		)
	if errContexto := ctx.Err(); errContexto != nil {
		return ResultadoConsultaResultadoCobertura{}, errContexto
	}
	if err != nil {
		return ResultadoConsultaResultadoCobertura{},
			s.clasificarFalloLecturaResultadoCobertura(ctx, err)
	}
	if recibo, confirmado := historico.ReciboConfirmadoPara(consulta); confirmado {
		return nuevoResultadoConsultaCoberturaConfirmado(recibo)
	}
	if historico.NoObservablePara(consulta) {
		return nuevoResultadoConsultaCoberturaNoObservable(), nil
	}
	return ResultadoConsultaResultadoCobertura{},
		ErrConsultaResultadoCoberturaNoConfiable
}

func (s *ServicioConsultaResultadoCobertura) ahoraConsultaResultado(
	ctx context.Context,
) (time.Time, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	instante, err := s.reloj.AhoraGobiernoOperacionCobertura(ctx)
	if errContexto := ctx.Err(); errContexto != nil {
		return time.Time{}, errContexto
	}
	if err != nil || !domain.InstanteUTCCanonico(instante) {
		return time.Time{}, ErrConsultaResultadoCoberturaNoDisponible
	}
	return instante, nil
}

func (s *ServicioConsultaResultadoCobertura) autorizarConsultaResultado(
	ctx context.Context,
	expedienteRef string,
	contexto ports.ContextoRecuperacionResultadoCobertura,
	instante time.Time,
) error {
	peticion, err := ports.NuevaSolicitudLecturaResultadoCobertura(
		contexto,
		expedienteRef,
		instante,
	)
	if err != nil {
		return ErrConsultaResultadoCoberturaNoConfiable
	}
	resultado, err := s.accesos.AutorizarLecturaResultadoCobertura(
		ctx,
		peticion,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if err != nil {
		return s.clasificarFalloAutorizacionConsulta(ctx, err)
	}
	switch resultado {
	case ports.AutorizacionLecturaResultadoCoberturaConcedida:
		return nil
	case ports.AutorizacionLecturaResultadoCoberturaDenegada:
		return ErrConsultaResultadoCoberturaDenegada
	default:
		return ErrConsultaResultadoCoberturaNoConfiable
	}
}

func (s *ServicioConsultaResultadoCobertura) clasificarFalloAutorizacionConsulta(
	ctx context.Context,
	causa error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if errors.Is(causa, ports.ErrAutorizacionDenegada) ||
		errors.Is(causa, ports.ErrContextoAutorizacionV3Invalido) ||
		errors.Is(causa, dominiovec.ErrAutorizacionDenegada) {
		return ErrConsultaResultadoCoberturaDenegada
	}
	return ErrConsultaResultadoCoberturaNoDisponible
}

func (s *ServicioConsultaResultadoCobertura) denegarConsultaResultado(
	ctx context.Context,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrConsultaResultadoCoberturaDenegada
}

func (s *ServicioConsultaResultadoCobertura) clasificarFalloLecturaResultadoCobertura(
	ctx context.Context,
	causa error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch {
	case errors.Is(
		causa,
		cobertura.ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
	):
		return ErrConsultaResultadoCoberturaConflicto
	case errors.Is(
		causa,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	), errors.Is(
		causa,
		cobertura.ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido,
	):
		return ErrConsultaResultadoCoberturaNoConfiable
	case errors.Is(
		causa,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	):
		return ErrConsultaResultadoCoberturaNoDisponible
	default:
		return ErrConsultaResultadoCoberturaNoDisponible
	}
}
