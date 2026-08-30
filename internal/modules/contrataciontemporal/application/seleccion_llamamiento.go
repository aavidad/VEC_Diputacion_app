package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const tiempoMaximoSeleccionLlamamiento = 15 * time.Second

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
)

// SolicitudSeleccionLlamamiento solo identifica un intento idempotente. La
// política, el orden y la posición seleccionada pertenecen a Bolsa y a la
// preparación confiable del servidor, nunca al canal que inicia el caso.
type SolicitudSeleccionLlamamiento struct {
	ClaveIdempotencia string
}

type ServicioSeleccionLlamamiento struct {
	preparador     ports.PreparadorSeleccionLlamamiento
	disponibilidad ports.ConsultaDisponibilidadBolsa
	ordenes        ports.PreparadorOrdenBolsa
	llamamientos   ports.GestorLlamamientosBolsa
	verificador    *ports.VerificadorEvidenciaIntegracionBolsa
	reloj          ports.Reloj
}

func NuevoServicioSeleccionLlamamiento(
	preparador ports.PreparadorSeleccionLlamamiento,
	disponibilidad ports.ConsultaDisponibilidadBolsa,
	ordenes ports.PreparadorOrdenBolsa,
	llamamientos ports.GestorLlamamientosBolsa,
	verificador *ports.VerificadorEvidenciaIntegracionBolsa,
	reloj ports.Reloj,
) (*ServicioSeleccionLlamamiento, error) {
	if dependenciaNula(preparador) || dependenciaNula(disponibilidad) ||
		dependenciaNula(ordenes) || dependenciaNula(llamamientos) ||
		verificador == nil || dependenciaNula(reloj) {
		return nil, ErrServicioSeleccionLlamamientoInvalido
	}
	return &ServicioSeleccionLlamamiento{
		preparador: preparador, disponibilidad: disponibilidad, ordenes: ordenes,
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
	reciboOrden, err := s.ordenes.PrepararOrden(operacion, comandoOrden)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	instanteReciboOrden := instanteCanonico(s.reloj.Ahora())
	if instanteReciboOrden.Before(instanteOrden) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	if _, _, err = s.verificador.VerificarReciboOrden(
		operacion, comandoOrden, reciboOrden, instanteReciboOrden,
	); err != nil || !reciboOrden.OrdenGenerada || !reciboOrden.OrdenCompleta ||
		reciboOrden.TotalPosiciones == 0 {
		return ports.ReciboSolicitudLlamamientoBolsa{}, clasificarResultadoSeleccion(operacion)
	}

	contextoLlamamiento, err := s.preparador.PrepararContextoLlamamiento(
		operacion, solicitud.ClaveIdempotencia, reciboOrden,
	)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	instanteLlamamiento := instanteCanonico(s.reloj.Ahora())
	if instanteLlamamiento.Before(instanteReciboOrden) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	comprobanteOrden, evidenciaOrden, err := s.verificador.VerificarReciboOrden(
		operacion, comandoOrden, reciboOrden, instanteLlamamiento,
	)
	if err != nil || !contextosSeleccionLigados(
		consulta.Contexto, comandoOrden.Contexto, contextoLlamamiento, instanteLlamamiento,
	) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, clasificarResultadoSeleccion(operacion)
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
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	if _, err = ports.NuevoArtefactoProbatorioOrdenBolsa(
		comandoOrden, reciboOrden, evidenciaOrden, comprobanteOrden,
	); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}

	recibo, err := s.llamamientos.SolicitarLlamamiento(operacion, comandoLlamamiento)
	if err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, normalizarFalloSeleccionLlamamiento(operacion, err)
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	instanteRecibo := instanteCanonico(s.reloj.Ahora())
	if instanteRecibo.Before(instanteLlamamiento) {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	comprobante, evidencia, err := s.verificador.VerificarReciboLlamamiento(
		operacion, comandoLlamamiento, recibo, instanteRecibo,
	)
	if err != nil || !recibo.PropuestaGenerada {
		return ports.ReciboSolicitudLlamamientoBolsa{}, clasificarResultadoSeleccion(operacion)
	}
	if _, err = ports.NuevoArtefactoProbatorioLlamamientoBolsa(
		comandoLlamamiento, recibo, evidencia, comprobante,
	); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, ErrResultadoSeleccionLlamamientoNoConfiable
	}
	if err := operacion.Err(); err != nil {
		return ports.ReciboSolicitudLlamamientoBolsa{}, err
	}
	return recibo, nil
}

func (s *ServicioSeleccionLlamamiento) dependenciasValidas() bool {
	return s != nil && !dependenciaNula(s.preparador) &&
		!dependenciaNula(s.disponibilidad) && !dependenciaNula(s.ordenes) &&
		!dependenciaNula(s.llamamientos) && s.verificador != nil &&
		!dependenciaNula(s.reloj)
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
