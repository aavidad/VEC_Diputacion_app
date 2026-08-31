package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioOperacionGINPIXRecuperableInvalido = errors.New(
		"contratacion temporal: servicio de operacion ginpix recuperable invalido",
	)
	ErrSolicitudOperacionGINPIXRecuperableInvalida = errors.New(
		"contratacion temporal: solicitud de operacion ginpix recuperable invalida",
	)
	ErrColisionOperacionGINPIXRecuperable = errors.New(
		"contratacion temporal: colision de operacion ginpix recuperable",
	)
	ErrOperacionGINPIXNoDisponible = errors.New(
		"contratacion temporal: operacion ginpix no disponible",
	)
	ErrOperacionGINPIXIndeterminada = errors.New(
		"contratacion temporal: operacion ginpix indeterminada",
	)
	ErrResultadoOperacionGINPIXNoConfiable = errors.New(
		"contratacion temporal: resultado de operacion ginpix no confiable",
	)
)

// SolicitudOperacionGINPIXRecuperable recibe exclusivamente contratos ya
// autenticados y validados. No contiene selectores de adaptador ni autoridad
// reconstruible por un cliente.
type SolicitudOperacionGINPIXRecuperable struct {
	Mapeo         ports.SolicitudMapeoGINPIX
	Orden         ports.OrdenConfirmarIncorporacion
	Incorporacion ports.ReciboConfirmacionIncorporacion
}

func (s SolicitudOperacionGINPIXRecuperable) puerto() (
	ports.SolicitudOperacionGINPIX,
	error,
) {
	solicitud, err := ports.NuevaSolicitudOperacionGINPIX(
		s.Mapeo,
		s.Orden,
		s.Incorporacion,
	)
	if err != nil {
		return ports.SolicitudOperacionGINPIX{},
			ErrSolicitudOperacionGINPIXRecuperableInvalida
	}
	return solicitud, nil
}

type ServicioOperacionGINPIXRecuperable struct {
	registro  ports.RegistroOperacionGINPIX
	emisor    ports.EmisorOperacionGINPIX
	consultor ports.ConsultorOperacionGINPIX
}

func NuevoServicioOperacionGINPIXRecuperable(
	registro ports.RegistroOperacionGINPIX,
	emisor ports.EmisorOperacionGINPIX,
	consultor ports.ConsultorOperacionGINPIX,
) (*ServicioOperacionGINPIXRecuperable, error) {
	if dependenciaNula(registro) || dependenciaNula(emisor) || dependenciaNula(consultor) {
		return nil, ErrServicioOperacionGINPIXRecuperableInvalido
	}
	return &ServicioOperacionGINPIXRecuperable{
		registro: registro, emisor: emisor, consultor: consultor,
	}, nil
}

// Ejecutar reserva antes de emitir. Desde EmisionAutorizada, cualquier
// abandono queda indeterminado: un replay nunca obtiene una segunda emision.
func (s *ServicioOperacionGINPIXRecuperable) Ejecutar(
	ctx context.Context,
	entrada SolicitudOperacionGINPIXRecuperable,
) (ports.ResultadoOperacionGINPIX, error) {
	if err := s.validar(ctx); err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	solicitud, err := entrada.puerto()
	if err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	reserva, err := s.registro.ReservarOperacionGINPIX(ctx, solicitud)
	if err != nil {
		return ports.ResultadoOperacionGINPIX{}, normalizarFalloRegistroGINPIX(ctx, err)
	}
	if reserva.ValidarPara(solicitud) != nil {
		return ports.ResultadoOperacionGINPIX{}, ErrResultadoOperacionGINPIXNoConfiable
	}
	if errContexto := ctx.Err(); errContexto != nil {
		if reserva.Situacion == ports.ReservaOperacionGINPIXEmisionAutorizada {
			s.marcarIndeterminada(ctx, reserva)
			return ports.ResultadoOperacionGINPIX{},
				errors.Join(ErrOperacionGINPIXIndeterminada, errContexto)
		}
		return ports.ResultadoOperacionGINPIX{}, errContexto
	}
	switch reserva.Situacion {
	case ports.ReservaOperacionGINPIXConfirmada:
		return resultadoOperacionGINPIXValidado(solicitud, reserva.Resultado)
	case ports.ReservaOperacionGINPIXPendienteConciliacion:
		return ports.ResultadoOperacionGINPIX{}, ErrOperacionGINPIXIndeterminada
	case ports.ReservaOperacionGINPIXEmisionAutorizada:
		return s.emitir(ctx, solicitud, reserva)
	default:
		return ports.ResultadoOperacionGINPIX{}, ErrResultadoOperacionGINPIXNoConfiable
	}
}

// Recuperar no reserva una operacion nueva y nunca llama al emisor. Solo
// consulta una reserva pendiente y confirma localmente un recibo completo.
func (s *ServicioOperacionGINPIXRecuperable) Recuperar(
	ctx context.Context,
	entrada SolicitudOperacionGINPIXRecuperable,
) (ports.ResultadoOperacionGINPIX, error) {
	if err := s.validar(ctx); err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	solicitud, err := entrada.puerto()
	if err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionGINPIX{}, err
	}
	reserva, err := s.registro.ConsultarReservaOperacionGINPIX(ctx, solicitud)
	if err != nil {
		return ports.ResultadoOperacionGINPIX{}, normalizarFalloRegistroGINPIX(ctx, err)
	}
	if reserva.ValidarPara(solicitud) != nil {
		return ports.ResultadoOperacionGINPIX{}, ErrResultadoOperacionGINPIXNoConfiable
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ResultadoOperacionGINPIX{}, errContexto
	}
	switch reserva.Situacion {
	case ports.ReservaOperacionGINPIXConfirmada:
		return resultadoOperacionGINPIXValidado(solicitud, reserva.Resultado)
	case ports.ReservaOperacionGINPIXEmisionAutorizada:
		if !s.marcarIndeterminada(ctx, reserva) {
			return ports.ResultadoOperacionGINPIX{}, ErrOperacionGINPIXIndeterminada
		}
		reserva.Situacion = ports.ReservaOperacionGINPIXPendienteConciliacion
		return s.consultar(ctx, solicitud, reserva)
	case ports.ReservaOperacionGINPIXPendienteConciliacion:
		return s.consultar(ctx, solicitud, reserva)
	default:
		return ports.ResultadoOperacionGINPIX{}, ErrResultadoOperacionGINPIXNoConfiable
	}
}

func (s *ServicioOperacionGINPIXRecuperable) emitir(
	ctx context.Context,
	solicitud ports.SolicitudOperacionGINPIX,
	reserva ports.ReservaOperacionGINPIX,
) (ports.ResultadoOperacionGINPIX, error) {
	if err := ctx.Err(); err != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{}, errors.Join(ErrOperacionGINPIXIndeterminada, err)
	}
	recibo, err := s.emisor.EmitirOperacionGINPIX(ctx, solicitud, reserva)
	tieneResultado := recibo != (ports.ReciboExternoOperacionGINPIX{})
	if err != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{}, errorIndeterminadoGINPIX(ctx)
	}
	if !tieneResultado || recibo.ValidarPara(solicitud) != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{}, ErrOperacionGINPIXIndeterminada
	}
	if errContexto := ctx.Err(); errContexto != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{},
			errors.Join(ErrOperacionGINPIXIndeterminada, errContexto)
	}
	return s.confirmar(ctx, solicitud, reserva, recibo)
}

func (s *ServicioOperacionGINPIXRecuperable) consultar(
	ctx context.Context,
	solicitud ports.SolicitudOperacionGINPIX,
	reserva ports.ReservaOperacionGINPIX,
) (ports.ResultadoOperacionGINPIX, error) {
	if err := ctx.Err(); err != nil {
		return ports.ResultadoOperacionGINPIX{}, errors.Join(ErrOperacionGINPIXIndeterminada, err)
	}
	recibo, err := s.consultor.ConsultarOperacionGINPIX(ctx, solicitud, reserva)
	if err != nil || recibo == (ports.ReciboExternoOperacionGINPIX{}) ||
		recibo.ValidarPara(solicitud) != nil {
		return ports.ResultadoOperacionGINPIX{}, errorIndeterminadoGINPIX(ctx)
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.ResultadoOperacionGINPIX{},
			errors.Join(ErrOperacionGINPIXIndeterminada, errContexto)
	}
	return s.confirmar(ctx, solicitud, reserva, recibo)
}

func (s *ServicioOperacionGINPIXRecuperable) confirmar(
	ctx context.Context,
	solicitud ports.SolicitudOperacionGINPIX,
	reserva ports.ReservaOperacionGINPIX,
	recibo ports.ReciboExternoOperacionGINPIX,
) (ports.ResultadoOperacionGINPIX, error) {
	resultado, err := s.registro.ConfirmarOperacionGINPIX(ctx, reserva, recibo)
	if err != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{}, errorIndeterminadoGINPIX(ctx)
	}
	if errContexto := ctx.Err(); errContexto != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{},
			errors.Join(ErrOperacionGINPIXIndeterminada, errContexto)
	}
	if resultado.ValidarPara(solicitud) != nil {
		s.marcarIndeterminada(ctx, reserva)
		return ports.ResultadoOperacionGINPIX{}, errors.Join(
			ErrOperacionGINPIXIndeterminada,
			ErrResultadoOperacionGINPIXNoConfiable,
		)
	}
	return resultado, nil
}

func (s *ServicioOperacionGINPIXRecuperable) marcarIndeterminada(
	ctx context.Context,
	reserva ports.ReservaOperacionGINPIX,
) bool {
	return s.registro.MarcarOperacionGINPIXIndeterminada(
		contextoRecuperacionGINPIX(ctx),
		reserva,
	) == nil
}

func (s *ServicioOperacionGINPIXRecuperable) validar(ctx context.Context) error {
	if s == nil || ctx == nil || dependenciaNula(s.registro) || dependenciaNula(s.emisor) ||
		dependenciaNula(s.consultor) {
		return ErrServicioOperacionGINPIXRecuperableInvalido
	}
	return nil
}

func resultadoOperacionGINPIXValidado(
	solicitud ports.SolicitudOperacionGINPIX,
	resultado ports.ResultadoOperacionGINPIX,
) (ports.ResultadoOperacionGINPIX, error) {
	if resultado.ValidarPara(solicitud) != nil {
		return ports.ResultadoOperacionGINPIX{}, ErrResultadoOperacionGINPIXNoConfiable
	}
	return resultado, nil
}

func normalizarFalloRegistroGINPIX(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ports.ErrColisionOperacionGINPIX) {
		return ErrColisionOperacionGINPIXRecuperable
	}
	return ErrOperacionGINPIXNoDisponible
}

func errorIndeterminadoGINPIX(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return errors.Join(ErrOperacionGINPIXIndeterminada, ctx.Err())
	}
	return ErrOperacionGINPIXIndeterminada
}

func contextoRecuperacionGINPIX(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}
