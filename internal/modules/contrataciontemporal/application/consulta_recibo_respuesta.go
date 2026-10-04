package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioConsultaReciboRespuesta struct{ lector ports.LectorReciboRespuesta }

func NuevoServicioConsultaReciboRespuesta(lector ports.LectorReciboRespuesta) (*ServicioConsultaReciboRespuesta, error) {
	if dependenciaNula(lector) {
		return nil, ports.ErrConsultaReciboRespuestaFallo
	}
	return &ServicioConsultaReciboRespuesta{lector: lector}, nil
}

func (s *ServicioConsultaReciboRespuesta) Consultar(ctx context.Context, solicitud ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	vacio := ports.ReciboRespuestaConsultado{}
	if s == nil || ctx == nil || dependenciaNula(s.lector) {
		return vacio, ports.ErrConsultaReciboRespuestaFallo
	}
	if err := solicitud.Validar(); err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	r, err := s.lector.ConsultarReciboRespuesta(ctx, solicitud)
	var auditado ports.FalloLecturaAuditado
	conAcuse := errors.As(err, &auditado)
	if e := ctx.Err(); e != nil {
		if !conAcuse {
			return vacio, e
		}
	}
	if err != nil {
		if r != vacio {
			return vacio, ports.ErrReciboRespuestaNoConfiable
		}
		if conAcuse {
			return vacio, err
		}
		for _, conocido := range []error{ports.ErrConsultaReciboRespuestaInvalida, ports.ErrReciboRespuestaNoEncontrado, ports.ErrConsultaReciboRespuestaDenegada, ports.ErrReciboRespuestaNoConfiable, context.Canceled, context.DeadlineExceeded} {
			if errors.Is(err, conocido) {
				return vacio, conocido
			}
		}
		return vacio, ports.ErrConsultaReciboRespuestaFallo
	}
	if r.ValidarPara(solicitud) != nil {
		return vacio, ports.ErrReciboRespuestaNoConfiable
	}
	return r, nil
}
