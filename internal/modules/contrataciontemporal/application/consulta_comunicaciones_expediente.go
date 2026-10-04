package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ServicioConsultaComunicacionesExpediente struct {
	lector ports.LectorComunicacionesExpediente
}

func NuevoServicioConsultaComunicacionesExpediente(lector ports.LectorComunicacionesExpediente) (*ServicioConsultaComunicacionesExpediente, error) {
	if dependenciaNula(lector) {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	return &ServicioConsultaComunicacionesExpediente{lector: lector}, nil
}

func (s *ServicioConsultaComunicacionesExpediente) ConsultarComunicacionesExpediente(ctx context.Context, c ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error) {
	vacia := ports.PaginaComunicacionesExpediente{}
	if s == nil || ctx == nil || dependenciaNula(s.lector) {
		return vacia, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	if c.Validar() != nil {
		return vacia, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	p, err := s.lector.ConsultarComunicacionesExpediente(ctx, c)
	if e := ctx.Err(); e != nil {
		var auditado ports.FalloLecturaAuditado
		if !errors.As(err, &auditado) {
			return vacia, e
		}
	}
	if err != nil {
		if p.ExpedienteRef != "" || p.Comunicaciones != nil || p.SiguienteCursor != "" {
			return vacia, ports.ErrResultadoComunicacionesExpedienteNoConfiable
		}
		return vacia, err
	}
	if p.ValidarPara(c) != nil {
		return vacia, ports.ErrResultadoComunicacionesExpedienteNoConfiable
	}
	return p, nil
}
