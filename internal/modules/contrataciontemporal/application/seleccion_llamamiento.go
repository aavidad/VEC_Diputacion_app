package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	tiempoMaximoSeleccionLlamamiento       = 15 * time.Second
	tiempoRecuperacionSeleccionLlamamiento = 2 * time.Second
)

var (
	ErrServicioSeleccionLlamamientoInvalido = errors.New(
		"contratacion temporal: servicio de seleccion y llamamiento invalido",
	)
	ErrSolicitudSeleccionLlamamientoInvalida = errors.New(
		"contratacion temporal: solicitud de seleccion y llamamiento invalida",
	)
	ErrSeleccionLlamamientoNoDisponible = errors.New(
		"contratacion temporal: seleccion y llamamiento no disponible",
	)
	ErrResultadoSeleccionLlamamientoNoConfiable = errors.New(
		"contratacion temporal: resultado de seleccion y llamamiento no confiable",
	)
	ErrClaveSeleccionLlamamientoEnColision = errors.New(
		"contratacion temporal: clave de seleccion y llamamiento usada con otra ejecucion",
	)
	ErrEjecucionSeleccionLlamamientoConcurrente = errors.New(
		"contratacion temporal: ejecucion de seleccion y llamamiento concurrente",
	)
	ErrEjecucionSeleccionLlamamientoIndeterminada = errors.New(
		"contratacion temporal: ejecucion de seleccion y llamamiento indeterminada",
	)
)

// SolicitudSeleccionLlamamiento solo identifica un intento idempotente. La
// política, el orden y la posición seleccionada pertenecen a Bolsa y a la
// preparación confiable del servidor, nunca al canal que inicia el caso.
type SolicitudSeleccionLlamamiento struct {
	ClaveIdempotencia string
}

// DatosReciboSeleccionLlamamientoParaAdaptador es la proyección mínima que
// puede cruzar una frontera después de autenticar el recibo completo de Bolsa.
type DatosReciboSeleccionLlamamientoParaAdaptador struct {
	ReciboRef    string
	ConfirmadaEn time.Time
}

type ServicioSeleccionLlamamiento struct {
	preparador     ports.PreparadorSeleccionLlamamiento
	ejecuciones    ports.EjecucionesSeleccionLlamamiento
	disponibilidad ports.ConsultaDisponibilidadBolsa
	ordenes        ports.PreparadorOrdenBolsa
	llamamientos   ports.GestorLlamamientosBolsa
	verificador    *ports.VerificadorEvidenciaIntegracionBolsa
	reloj          ports.Reloj
}

func NuevoServicioSeleccionLlamamiento(
	preparador ports.PreparadorSeleccionLlamamiento,
	ejecuciones ports.EjecucionesSeleccionLlamamiento,
	disponibilidad ports.ConsultaDisponibilidadBolsa,
	ordenes ports.PreparadorOrdenBolsa,
	llamamientos ports.GestorLlamamientosBolsa,
	verificador *ports.VerificadorEvidenciaIntegracionBolsa,
	reloj ports.Reloj,
) (*ServicioSeleccionLlamamiento, error) {
	if dependenciaNula(preparador) || dependenciaNula(ejecuciones) ||
		dependenciaNula(disponibilidad) ||
		dependenciaNula(ordenes) || dependenciaNula(llamamientos) ||
		verificador == nil || dependenciaNula(reloj) {
		return nil, ErrServicioSeleccionLlamamientoInvalido
	}
	return &ServicioSeleccionLlamamiento{
		preparador: preparador, ejecuciones: ejecuciones,
		disponibilidad: disponibilidad, ordenes: ordenes,
		llamamientos: llamamientos, verificador: verificador, reloj: reloj,
	}, nil
}

// SeleccionarYLlamar delega en Bolsa la evaluación del orden completo y la
// generación de una propuesta. Solo devuelve el recibo minimizado después de
// autenticar los tres resultados; no acepta, formaliza ni sustituye la
// validación humana posterior.
func (s *ServicioSeleccionLlamamiento) SeleccionarYLlamar(
	ctx context.Context,
	solicitud SolicitudSeleccionLlamamiento,
) (ports.ReciboSolicitudLlamamientoBolsa, error) {
	if s == nil || ctx == nil || !s.dependenciasValidas() {
		return ports.ReciboSolicitudLlamamientoBolsa{},
			ErrServicioSeleccionLlamamientoInvalido
	}
	if !ports.ClaveIdempotenciaValida(solicitud.ClaveIdempotencia) {
		return ports.ReciboSolicitudLlamamientoBolsa{},
			ErrSolicitudSeleccionLlamamientoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	operacion, cancelar := context.WithTimeout(ctx, tiempoMaximoSeleccionLlamamiento)
	defer cancelar()
	consulta, err := s.preparador.PrepararConsultaDisponibilidad(
		operacion, solicitud.ClaveIdempotencia,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	instanteConsulta := instanteCanonico(s.reloj.Ahora())
	if consulta.ValidarEn(instanteConsulta) != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	consultaTerminal, err := ports.NuevaConsultaTerminalAutorizada(
		solicitud.ClaveIdempotencia, consulta.Contexto, instanteConsulta,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	estadoTerminal, confirmado, err := s.ejecuciones.ResolverTerminal(
		operacion, consultaTerminal, instanteConsulta,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{},
			normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if confirmado {
		recibo, err := estadoTerminal.VerificarTerminalConfirmado(
			operacion, consultaTerminal, s.verificador, instanteConsulta,
		)
		if err != nil {
			return ports.ReciboSolicitudLlamamientoBolsa{},
				ErrResultadoSeleccionLlamamientoNoConfiable
		}
		return recibo, nil
	}
	if estadoTerminal != (ports.EstadoEjecucionSeleccionLlamamiento{}) {
		return ports.ReciboSolicitudLlamamientoBolsa{},
			ErrResultadoSeleccionLlamamientoNoConfiable
	}
	resultado, err := s.disponibilidad.ConsultarDisponibilidad(operacion, consulta)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	instanteDisponibilidad := instanteCanonico(s.reloj.Ahora())
	if instanteDisponibilidad.Before(instanteConsulta) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	if _, _, err = s.verificador.VerificarDisponibilidad(
		operacion, consulta, resultado, instanteDisponibilidad,
	); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, clasificarResultadoSeleccion(operacion)
	}
	if !resultado.BolsaEncontrada || !resultado.Disponible ||
		!resultado.CantidadExacta || resultado.CantidadDisponible == 0 {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrSeleccionLlamamientoNoDisponible
	}

	comandoOrden, err := s.preparador.PrepararOrdenCompleto(
		operacion, solicitud.ClaveIdempotencia, resultado,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	instanteOrden := instanteCanonico(s.reloj.Ahora())
	if instanteOrden.Before(instanteDisponibilidad) ||
		!disponibilidadYOrdenLigados(consulta, resultado, comandoOrden, instanteOrden) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	if comandoOrden.MaximoPosiciones < resultado.CantidadDisponible {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	solicitudEjecucion, err := ports.NuevaSolicitudReservaEjecucionSeleccionLlamamiento(
		consultaTerminal, comandoOrden, resultado.CantidadDisponible, instanteOrden,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	estado, err := s.ejecuciones.Reservar(operacion, solicitudEjecucion)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	instanteTerminal := instanteOrden
	if estado.Situacion == ports.EjecucionSeleccionLlamamientoConfirmada {
		instanteTerminal = instanteCanonico(s.reloj.Ahora())
		if instanteTerminal.Before(instanteOrden) {
			return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
		}
	}
	reserva, reciboConfirmado, err := resolverReservaSeleccionLlamamiento(
		operacion, estado, solicitudEjecucion, consultaTerminal, s.verificador, instanteTerminal,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	if reciboConfirmado != (ports.ReciboSolicitudLlamamientoBolsa{}) {
		return reciboConfirmado, nil
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.liberarAntesDeEfectos(
			operacion, reserva, err,
		)
	}
	if err = s.ejecuciones.AbrirVentanaEfecto(
		operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
	); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.liberarAntesDeEfectos(
			operacion, reserva, normalizarFalloSeleccionLlamamiento(operacion, err),
		)
	}
	reciboOrden, err := s.ordenes.PrepararOrden(operacion, comandoOrden)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento, err,
		)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento, err,
		)
	}
	instanteReciboOrden := instanteCanonico(s.reloj.Ahora())
	if instanteReciboOrden.Before(instanteOrden) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
			ErrResultadoSeleccionLlamamientoNoConfiable,
		)
	}
	if _, _, err = s.verificador.VerificarReciboOrden(
		operacion, comandoOrden, reciboOrden, instanteReciboOrden,
	); err != nil || !reciboOrden.OrdenGenerada || !reciboOrden.OrdenCompleta ||
		reciboOrden.TotalPosiciones == 0 ||
		reciboOrden.TotalPosiciones < resultado.CantidadDisponible {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
			clasificarResultadoSeleccion(operacion),
		)
	}

	contextoLlamamiento, err := s.preparador.PrepararContextoLlamamiento(
		operacion, solicitud.ClaveIdempotencia, reciboOrden,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento, err,
		)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento, err,
		)
	}
	instanteLlamamiento := instanteCanonico(s.reloj.Ahora())
	if instanteLlamamiento.Before(instanteReciboOrden) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
			ErrResultadoSeleccionLlamamientoNoConfiable,
		)
	}
	comprobanteOrden, evidenciaOrden, err := s.verificador.VerificarReciboOrden(
		operacion, comandoOrden, reciboOrden, instanteLlamamiento,
	)
	if err != nil || !contextosSeleccionLigados(
		consulta.Contexto, comandoOrden.Contexto, contextoLlamamiento, instanteLlamamiento,
	) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
			clasificarResultadoSeleccion(operacion),
		)
	}
	comandoLlamamiento, err := ports.NuevoComandoSolicitarLlamamientoBolsa(
		ports.PreparacionComandoSolicitarLlamamientoBolsa{
			Contexto: contextoLlamamiento, ComandoOrden: comandoOrden,
			ReciboOrden: reciboOrden, ComprobanteOrden: comprobanteOrden,
			MaximaPosicionEvaluable: reciboOrden.TotalPosiciones,
		},
		instanteLlamamiento,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
			ErrResultadoSeleccionLlamamientoNoConfiable,
		)
	}
	if _, err = ports.NuevoArtefactoProbatorioOrdenBolsa(
		comandoOrden, reciboOrden, evidenciaOrden, comprobanteOrden,
	); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento,
			ErrResultadoSeleccionLlamamientoNoConfiable,
		)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento, err,
		)
	}
	if err = s.ejecuciones.AbrirVentanaEfecto(
		operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento,
	); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoPrepararOrdenSeleccionLlamamiento, err,
		)
	}

	recibo, err := s.llamamientos.SolicitarLlamamiento(operacion, comandoLlamamiento)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento, err,
		)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento, err,
		)
	}
	instanteRecibo := instanteCanonico(s.reloj.Ahora())
	if instanteRecibo.Before(instanteLlamamiento) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento,
			ErrResultadoSeleccionLlamamientoNoConfiable,
		)
	}
	comprobante, evidencia, err := s.verificador.VerificarReciboLlamamiento(
		operacion, comandoLlamamiento, recibo, instanteRecibo,
	)
	if err != nil || !recibo.PropuestaGenerada {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento,
			clasificarResultadoSeleccion(operacion),
		)
	}
	artefacto, err := ports.NuevoArtefactoProbatorioLlamamientoBolsa(
		comandoLlamamiento, recibo, evidencia, comprobante,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento,
			ErrResultadoSeleccionLlamamientoNoConfiable,
		)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento, err,
		)
	}
	if err = s.ejecuciones.Confirmar(operacion, reserva, recibo, artefacto); err != nil {
		recuperacion, cancelarRecuperacion := contextoRecuperacionSeleccion(operacion)
		defer cancelarRecuperacion()
		estado, falloEstado := s.ejecuciones.ConsultarEstado(recuperacion, solicitudEjecucion)
		recuperado, falloTerminal := estado.VerificarTerminalConfirmado(
			recuperacion, consultaTerminal, s.verificador, instanteRecibo,
		)
		if falloEstado == nil && falloTerminal == nil &&
			estado.Solicitud == solicitudEjecucion && recuperado == recibo {
			return recibo, nil
		}
		return ports.ReciboSolicitudLlamamientoBolsa{}, s.marcarIndeterminada(
			operacion, reserva, ports.EfectoSolicitarSeleccionLlamamiento, err,
		)
	}
	return recibo, nil
}

// SeleccionarYLlamarParaAdaptador minimiza únicamente el recibo que la ruta
// completa anterior ya autenticó y ligó al comando exacto de llamamiento.
func (s *ServicioSeleccionLlamamiento) SeleccionarYLlamarParaAdaptador(
	ctx context.Context,
	solicitud SolicitudSeleccionLlamamiento,
) (DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	recibo, err := s.SeleccionarYLlamar(ctx, solicitud)
	if err != nil {
		return DatosReciboSeleccionLlamamientoParaAdaptador{}, err
	}
	if !recibo.PropuestaGenerada ||
		!domain.ReferenciaOpacaValida(recibo.ReciboRef) ||
		!domain.InstanteUTCCanonico(recibo.ConfirmadaEn) {
		return DatosReciboSeleccionLlamamientoParaAdaptador{},
			ErrResultadoSeleccionLlamamientoNoConfiable
	}
	return DatosReciboSeleccionLlamamientoParaAdaptador{
		ReciboRef: recibo.ReciboRef, ConfirmadaEn: recibo.ConfirmadaEn,
	}, nil
}

func (s *ServicioSeleccionLlamamiento) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.preparador) &&
		!dependenciaNula(s.ejecuciones) && !dependenciaNula(s.disponibilidad) &&
		!dependenciaNula(s.ordenes) &&
		!dependenciaNula(s.llamamientos) && s.verificador != nil &&
		!dependenciaNula(s.reloj)
}

func resolverReservaSeleccionLlamamiento(
	ctx context.Context,
	estado ports.EstadoEjecucionSeleccionLlamamiento,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
	consulta ports.ConsultaTerminalAutorizada,
	verificador *ports.VerificadorEvidenciaIntegracionBolsa,
	instante time.Time,
) (
	ports.ReservaEjecucionSeleccionLlamamiento,
	ports.ReciboSolicitudLlamamientoBolsa,
	error,
) {
	vacio := ports.ReciboSolicitudLlamamientoBolsa{}
	if estado.Solicitud != solicitud {
		return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
			ErrResultadoSeleccionLlamamientoNoConfiable
	}
	switch estado.Situacion {
	case ports.EjecucionSeleccionLlamamientoPropietaria:
		if estado.ReservaRef == "" || estado.ReciboConfirmado != vacio {
			return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
				ErrResultadoSeleccionLlamamientoNoConfiable
		}
		return ports.ReservaEjecucionSeleccionLlamamiento{
			Solicitud: solicitud, ReservaRef: estado.ReservaRef,
		}, vacio, nil
	case ports.EjecucionSeleccionLlamamientoConfirmada:
		recibo, err := estado.VerificarTerminalConfirmado(
			ctx, consulta, verificador, instante,
		)
		if err != nil {
			return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
				ErrResultadoSeleccionLlamamientoNoConfiable
		}
		return ports.ReservaEjecucionSeleccionLlamamiento{}, recibo, nil
	case ports.EjecucionSeleccionLlamamientoOcupada:
		return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
			ErrEjecucionSeleccionLlamamientoConcurrente
	case ports.EjecucionSeleccionLlamamientoColision:
		return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
			ErrClaveSeleccionLlamamientoEnColision
	case ports.EjecucionSeleccionLlamamientoIndeterminada:
		if estado.EfectoPosible != ports.EfectoPrepararOrdenSeleccionLlamamiento &&
			estado.EfectoPosible != ports.EfectoSolicitarSeleccionLlamamiento {
			return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
				ErrResultadoSeleccionLlamamientoNoConfiable
		}
		return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
			ErrEjecucionSeleccionLlamamientoIndeterminada
	default:
		return ports.ReservaEjecucionSeleccionLlamamiento{}, vacio,
			ErrResultadoSeleccionLlamamientoNoConfiable
	}
}

func contextoRecuperacionSeleccion(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(
		context.WithoutCancel(ctx), tiempoRecuperacionSeleccionLlamamiento,
	)
}

func (s *ServicioSeleccionLlamamiento) liberarAntesDeEfectos(
	ctx context.Context,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	causa error,
) error {
	recuperacion, cancelar := contextoRecuperacionSeleccion(ctx)
	defer cancelar()
	if err := s.ejecuciones.LiberarAntesDeEfectos(recuperacion, reserva); err != nil {
		return errorSeleccionIndeterminada(ctx, causa)
	}
	return causa
}

func (s *ServicioSeleccionLlamamiento) marcarIndeterminada(
	ctx context.Context,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	efecto ports.EfectoSeleccionLlamamiento,
	causa error,
) error {
	recuperacion, cancelar := contextoRecuperacionSeleccion(ctx)
	defer cancelar()
	_ = s.ejecuciones.MarcarIndeterminada(recuperacion, reserva, efecto)
	return errorSeleccionIndeterminada(ctx, causa)
}

func errorSeleccionIndeterminada(ctx context.Context, causa error) error {
	if ctx != nil && ctx.Err() != nil {
		return errors.Join(ErrEjecucionSeleccionLlamamientoIndeterminada, ctx.Err())
	}
	if errors.Is(causa, context.Canceled) || errors.Is(causa, context.DeadlineExceeded) {
		return errors.Join(ErrEjecucionSeleccionLlamamientoIndeterminada, causa)
	}
	return ErrEjecucionSeleccionLlamamientoIndeterminada
}

func disponibilidadYOrdenLigados(
	consulta ports.SolicitudDisponibilidadBolsa,
	resultado ports.ResultadoDisponibilidadBolsa,
	orden ports.ComandoPrepararOrdenBolsa,
	instante time.Time,
) bool {
	datosConsulta, errConsulta := consulta.Contexto.DatosEn(instante)
	datosOrden, errOrden := orden.Contexto.DatosEn(instante)
	return errConsulta == nil && errOrden == nil && orden.ValidarEn(instante) == nil &&
		resultado.ValidarParaEn(consulta, instante) == nil &&
		orden.Necesidad == resultado.Necesidad && orden.Bolsa == resultado.Bolsa &&
		datosConsulta.OrganizacionRef == datosOrden.OrganizacionRef &&
		datosConsulta.ExpedienteRef == datosOrden.ExpedienteRef &&
		datosConsulta.VersionExpediente == datosOrden.VersionExpediente &&
		datosConsulta.CorrelacionRef == datosOrden.CorrelacionRef &&
		datosConsulta.Finalidad == datosOrden.Finalidad &&
		datosConsulta.OperacionRef != datosOrden.OperacionRef
}

func contextosSeleccionLigados(
	consulta ports.ContextoPeticionIntegracionBolsa,
	orden ports.ContextoPeticionIntegracionBolsa,
	llamamiento ports.ContextoPeticionIntegracionBolsa,
	instante time.Time,
) bool {
	a, errA := consulta.DatosEn(instante)
	b, errB := orden.DatosEn(instante)
	c, errC := llamamiento.DatosEn(instante)
	return errA == nil && errB == nil && errC == nil &&
		a.OrganizacionRef == b.OrganizacionRef && b.OrganizacionRef == c.OrganizacionRef &&
		a.ExpedienteRef == b.ExpedienteRef && b.ExpedienteRef == c.ExpedienteRef &&
		a.VersionExpediente == b.VersionExpediente && b.VersionExpediente == c.VersionExpediente &&
		a.CorrelacionRef == b.CorrelacionRef && b.CorrelacionRef == c.CorrelacionRef &&
		a.Finalidad == b.Finalidad && b.Finalidad == c.Finalidad &&
		a.OperacionRef != b.OperacionRef && a.OperacionRef != c.OperacionRef &&
		b.OperacionRef != c.OperacionRef
}

func normalizarFalloSeleccionLlamamiento(ctx context.Context, err error) error {
	if ctx != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return errContexto
		}
	}
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	default:
		return ErrSeleccionLlamamientoNoDisponible
	}
}

func clasificarResultadoSeleccion(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return ErrResultadoSeleccionLlamamientoNoConfiable
}
