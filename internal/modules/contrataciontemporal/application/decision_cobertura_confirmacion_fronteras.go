package application

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const tiempoMaximoReconciliacionDecisionCobertura = 2 * time.Second

func (s *ServicioConfirmacionDecisionCobertura) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.contextos) &&
		!dependenciaNula(s.motivos) && !dependenciaNula(s.sellador) &&
		!dependenciaNula(s.idempotencia) && !dependenciaNula(s.analisis) &&
		!dependenciaNula(s.reloj) && !dependenciaNula(s.gobierno) &&
		!dependenciaNula(s.coberturas) &&
		!dependenciaNula(s.autorizaciones) &&
		!dependenciaNula(s.transaccion) &&
		!dependenciaNula(s.reconciliador)
}

func (s datosSolicitudConfirmacionDecisionCobertura) validar() error {
	contexto := ports.SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: s.autenticacionRef,
		SesionRef:        s.sesionRef, PerfilRef: s.perfilRef,
	}
	if contexto.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.organizacionRef) ||
		!domain.ReferenciaOpacaValida(s.expedienteRef) ||
		s.versionEsperada == 0 ||
		s.versionEsperada >=
			cobertura.MaximoEnteroSeguroOperacionDecisionCobertura ||
		!ports.ClaveIdempotenciaValida(s.claveIdempotencia) ||
		s.identidadSemantica.Validar() != nil ||
		!s.viaElegida.Valida() ||
		(s.motivoClave != "" && !s.motivoClave.Valida()) {
		return ErrSolicitudConfirmacionDecisionCoberturaInvalida
	}
	switch s.tipo {
	case domain.DecisionCoberturaInicial:
		if s.predecesoraRef != "" || s.predecesoraHuella != "" {
			return ErrSolicitudConfirmacionDecisionCoberturaInvalida
		}
	case domain.DecisionCoberturaRectificacion:
		if !s.motivoClave.Valida() ||
			!domain.ReferenciaOpacaValida(s.predecesoraRef) ||
			!huellaConfirmacionCoberturaValida(s.predecesoraHuella) {
			return ErrSolicitudConfirmacionDecisionCoberturaInvalida
		}
	default:
		return ErrSolicitudConfirmacionDecisionCoberturaInvalida
	}
	return nil
}

func huellaConfirmacionCoberturaValida(huella string) bool {
	if len(huella) != 64 || huella == strings.Repeat("0", 64) {
		return false
	}
	_, err := hex.DecodeString(huella)
	return err == nil
}

func accionConfirmacionDecisionCobertura(
	tipo domain.TipoDecisionCoberturaGobernada,
) domain.ClaveCatalogo {
	if tipo == domain.DecisionCoberturaInicial {
		return domain.AccionDecidirCoberturaGobernada
	}
	if tipo == domain.DecisionCoberturaRectificacion {
		return domain.AccionRectificarCoberturaGobernada
	}
	return ""
}

func (s *ServicioConfirmacionDecisionCobertura) resolverMotivoConfirmacion(
	ctx context.Context,
	clave domain.ClaveCatalogo,
	instante time.Time,
) (
	cobertura.ResolucionMotivoDecisionCobertura,
	domain.MotivoGobernadoDecisionCobertura,
	error,
) {
	if clave == "" {
		return cobertura.ResolucionMotivoDecisionCobertura{},
			domain.MotivoGobernadoDecisionCobertura{},
			nil
	}
	resolucion, err := s.motivos.ResolverClave(ctx, clave, instante)
	if errContexto := ctx.Err(); errContexto != nil {
		return cobertura.ResolucionMotivoDecisionCobertura{},
			domain.MotivoGobernadoDecisionCobertura{},
			errContexto
	}
	motivo, errMotivo := resolucion.Motivo()
	resueltaEn, errInstante := resolucion.ResueltaEn()
	if err != nil {
		return cobertura.ResolucionMotivoDecisionCobertura{},
			domain.MotivoGobernadoDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoDisponible
	}
	if errMotivo != nil || errInstante != nil ||
		!resueltaEn.Equal(instante) {
		return cobertura.ResolucionMotivoDecisionCobertura{},
			domain.MotivoGobernadoDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	return resolucion, motivo, nil
}

func motivosConfirmacionCoberturaIguales(
	primero cobertura.ResolucionMotivoDecisionCobertura,
	segundo cobertura.ResolucionMotivoDecisionCobertura,
) bool {
	motivoPrimero, errPrimero := primero.Motivo()
	motivoSegundo, errSegundo := segundo.Motivo()
	if errPrimero != nil || errSegundo != nil {
		return errPrimero != nil && errSegundo != nil
	}
	return motivoPrimero == motivoSegundo
}

func solicitudGobiernoConfirmacionCobertura(
	expediente domain.Expediente,
	tipo domain.TipoDecisionCoberturaGobernada,
) (cobertura.SolicitudGobiernoOperacionCobertura, error) {
	if expediente.Validar() != nil || expediente.Analisis == nil ||
		!expediente.Analisis.HabilitaAvance() ||
		expediente.Asignacion != nil {
		return cobertura.SolicitudGobiernoOperacionCobertura{},
			ErrConfirmacionDecisionCoberturaEnConflicto
	}
	switch tipo {
	case domain.DecisionCoberturaInicial:
		if expediente.ViaCobertura != nil ||
			len(expediente.DecisionesCobertura) != 0 {
			return cobertura.SolicitudGobiernoOperacionCobertura{},
				ErrConfirmacionDecisionCoberturaEnConflicto
		}
		return cobertura.NuevaSolicitudGobiernoDecisionCobertura(
			expediente.OrganizacionRef,
			expediente.Referencia,
			expediente.Version,
		)
	case domain.DecisionCoberturaRectificacion:
		if expediente.ViaCobertura == nil ||
			expediente.ViaCobertura.DecisionGobernada == nil ||
			len(expediente.DecisionesCobertura) == 0 {
			return cobertura.SolicitudGobiernoOperacionCobertura{},
				ErrConfirmacionDecisionCoberturaEnConflicto
		}
		return cobertura.NuevaSolicitudGobiernoRectificacionCobertura(
			expediente.OrganizacionRef,
			expediente.Referencia,
			expediente.Version,
		)
	default:
		return cobertura.SolicitudGobiernoOperacionCobertura{},
			ErrConfirmacionDecisionCoberturaEnConflicto
	}
}

func nuevaSolicitudVECConfirmacionCobertura(
	ctx context.Context,
	contexto ports.ContextoAutorizacionAltaV3,
	reserva cobertura.DatosReservaPropietariaOperacionDecisionCobertura,
	gobierno cobertura.DatosGobiernoOperacionCobertura,
	recurso dominiovec.RecursoAutorizable,
) (dominiovec.SolicitudAutorizacionLigadaV3, error) {
	generador, err :=
		nuevoGeneradorCorrelacionVECReservada(reserva.CorrelacionVECRef)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	correlacion, err :=
		dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, generador)
	if err != nil {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, err
	}
	return dominiovec.NuevaSolicitudAutorizacionLigadaV3(
		dominiovec.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: contexto.Vinculo,
			ReferenciaMotivo:          gobierno.MotivoAutorizacion,
			Accion:                    string(gobierno.Accion),
			Recurso:                   recurso,
			Finalidad:                 string(gobierno.FinalidadVEC),
			Correlacion:               correlacion,
		},
	)
}

func (s *ServicioConfirmacionDecisionCobertura) confirmarOrden(
	ctx context.Context,
	orden cobertura.OrdenOperacionDecisionCobertura,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	intento, err := cobertura.IntentarConfirmacionOperacionDecisionCobertura(
		ctx,
		s.transaccion,
		orden,
	)
	if confirmacion, valida := intento.ConfirmacionPara(orden); valida {
		recibo, errRecibo := confirmacion.ReciboPara(orden)
		if errRecibo == nil {
			return recibo, nil
		}
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	if intento.FalloAntesCommitPara(orden) {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoDisponible
	}
	solicitud, requiereReconciliar := intento.ReconciliacionPara(orden)
	if !requiereReconciliar ||
		!errors.Is(
			err,
			cobertura.ErrResultadoConfirmacionOperacionDecisionCoberturaAmbiguo,
		) {
		if errContexto := ctx.Err(); errContexto != nil {
			return cobertura.ReciboOperacionDecisionCobertura{}, errContexto
		}
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoConfiable
	}
	// Tras una respuesta perdida no se repite la transacción. La lectura
	// primaria usa un presupuesto interno corto que conserva valores de traza,
	// pero no hereda una cancelación tardía del transporte original.
	baseReconciliacion := context.WithoutCancel(ctx)
	ctxReconciliacion, cancelar := context.WithTimeout(
		baseReconciliacion,
		tiempoMaximoReconciliacionDecisionCobertura,
	)
	defer cancelar()
	confirmacion, err := cobertura.
		ReconciliarConfirmacionOperacionDecisionCobertura(
			ctxReconciliacion,
			s.reconciliador,
			solicitud,
			orden,
		)
	if confirmacionValida, valida :=
		confirmacion.ReciboPara(orden); valida == nil {
		return confirmacionValida, nil
	}
	if errContexto := ctxReconciliacion.Err(); errContexto != nil {
		return cobertura.ReciboOperacionDecisionCobertura{},
			ErrConfirmacionDecisionCoberturaNoDisponible
	}
	_ = err
	return cobertura.ReciboOperacionDecisionCobertura{},
		ErrConfirmacionDecisionCoberturaNoDisponible
}

func (s *ServicioConfirmacionDecisionCobertura) ahoraConfirmacion(
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
		return time.Time{}, ErrConfirmacionDecisionCoberturaNoDisponible
	}
	return instante, nil
}

func (s *ServicioConfirmacionDecisionCobertura) errorDependencia(
	ctx context.Context,
	_ error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrConfirmacionDecisionCoberturaNoDisponible
}

func (s *ServicioConfirmacionDecisionCobertura) errorContexto(
	ctx context.Context,
	err error,
) error {
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if errors.Is(err, ports.ErrAutorizacionDenegada) ||
		errors.Is(err, ports.ErrContextoAutorizacionV3Invalido) ||
		errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return ErrConfirmacionDecisionCoberturaDenegada
	}
	return ErrConfirmacionDecisionCoberturaNoDisponible
}

func (s *ServicioConfirmacionDecisionCobertura) errorPersistencia(
	ctx context.Context,
	err error,
) error {
	if errContexto := ctx.Err(); errContexto != nil {
		return errContexto
	}
	if errors.Is(err, cobertura.ErrOperacionDecisionCoberturaOcupada) {
		return ErrConfirmacionDecisionCoberturaOcupada
	}
	if errors.Is(
		err,
		cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) {
		return ErrConfirmacionDecisionCoberturaEnConflicto
	}
	return ErrConfirmacionDecisionCoberturaNoDisponible
}

func (s *ServicioConfirmacionDecisionCobertura) errorDenegacion(
	ctx context.Context,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrConfirmacionDecisionCoberturaDenegada
}
