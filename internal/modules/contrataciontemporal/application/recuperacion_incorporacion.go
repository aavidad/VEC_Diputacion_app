package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioRecuperacionIncorporacionInvalido  = errors.New("contratacion temporal: servicio de recuperacion de incorporacion invalido")
	ErrSolicitudRecuperacionIncorporacionInvalida = errors.New("contratacion temporal: solicitud de recuperacion de incorporacion invalida")
	ErrRecuperacionIncorporacionDenegada          = errors.New("contratacion temporal: recuperacion de incorporacion denegada")
	ErrRecuperacionIncorporacionNoDisponible      = errors.New("contratacion temporal: recuperacion de incorporacion no disponible")
	ErrRegistroIncorporacionNoConfiable           = errors.New("contratacion temporal: registro de incorporacion no confiable")
)

// ServicioRecuperacionIncorporacion no recibe ninguna frontera de efecto.
// El registro inicial y su semántica permanecen en un servicio independiente.
type ServicioRecuperacionIncorporacion struct {
	resolutor ports.ResolutorRecuperacionIncorporacion
	lector    ports.LectorRegistroIncorporacion
	reloj     ports.Reloj
}

func NuevoServicioRecuperacionIncorporacion(resolutor ports.ResolutorRecuperacionIncorporacion,
	lector ports.LectorRegistroIncorporacion, reloj ports.Reloj,
) (*ServicioRecuperacionIncorporacion, error) {
	if dependenciaNula(resolutor) || dependenciaNula(lector) || dependenciaNula(reloj) {
		return nil, ErrServicioRecuperacionIncorporacionInvalido
	}
	return &ServicioRecuperacionIncorporacion{resolutor: resolutor, lector: lector, reloj: reloj}, nil
}

func (s *ServicioRecuperacionIncorporacion) Recuperar(ctx context.Context, solicitud ports.SolicitudRecuperacionIncorporacion) (ports.ReciboConfirmacionIncorporacion, error) {
	if s == nil || ctx == nil || dependenciaNula(s.resolutor) || dependenciaNula(s.lector) || dependenciaNula(s.reloj) {
		return ports.ReciboConfirmacionIncorporacion{}, ErrServicioRecuperacionIncorporacionInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	if solicitud.Validar() != nil {
		return ports.ReciboConfirmacionIncorporacion{}, ErrSolicitudRecuperacionIncorporacionInvalida
	}
	orden, err := s.resolutor.ResolverRecuperacionIncorporacion(ctx, solicitud)
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, falloRecuperacionIncorporacion(ctx, err, ErrRecuperacionIncorporacionDenegada)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	antes := s.reloj.Ahora()
	if solicitud.ValidarPara(orden, antes) != nil {
		return ports.ReciboConfirmacionIncorporacion{}, ErrRecuperacionIncorporacionDenegada
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	recibo, historia, err := s.lector.LeerRegistroIncorporacion(ctx, orden)
	if err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, falloRecuperacionIncorporacion(ctx, err, ErrRecuperacionIncorporacionNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	// No publica una respuesta si el reloj retrocedió o el permiso venció
	// mientras la dependencia estaba trabajando.
	despues := s.reloj.Ahora()
	if despues.Before(antes) || solicitud.ValidarPara(orden, despues) != nil {
		return ports.ReciboConfirmacionIncorporacion{}, ErrRecuperacionIncorporacionDenegada
	}
	if recibo.ValidarRecuperacion(historia, orden, despues) != nil {
		return ports.ReciboConfirmacionIncorporacion{}, ErrRegistroIncorporacionNoConfiable
	}
	// Copia solo tras validar la cardinalidad y el contenido contra la historia.
	recibo.Documentos = append([]domain.DocumentoSeguimiento(nil), recibo.Documentos...)
	if err := ctx.Err(); err != nil {
		return ports.ReciboConfirmacionIncorporacion{}, err
	}
	return recibo, nil
}

func falloRecuperacionIncorporacion(ctx context.Context, err, opaco error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return opaco
}
