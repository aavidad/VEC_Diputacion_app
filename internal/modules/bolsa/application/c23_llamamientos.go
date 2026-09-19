package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ServicioLlamamientosOperativos struct {
	repositorio ports.RepositorioLlamamientoOperativo
}

func NuevoServicioLlamamientosOperativos(r ports.RepositorioLlamamientoOperativo) (*ServicioLlamamientosOperativos, error) {
	if r == nil {
		return nil, ports.ErrLlamamientoOperativoNoDisponible
	}
	return &ServicioLlamamientosOperativos{repositorio: r}, nil
}
func (s *ServicioLlamamientosOperativos) Contactos(ctx context.Context, participacion string) ([]ports.ContactoLlamamientoOperativo, error) {
	if ctx == nil || s == nil || s.repositorio == nil || !ports.ReferenciaOpacaLlamamientoValida(participacion) {
		return nil, ports.ErrLlamamientoOperativoInvalido
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.repositorio.ContactosParticipacion(ctx, participacion)
}
func (s *ServicioLlamamientosOperativos) Abrir(ctx context.Context, entrada ports.AperturaLlamamientoOperativo) (ports.ReciboLlamamientoOperativo, error) {
	if ctx == nil || s == nil || s.repositorio == nil || entrada.Validar() != nil {
		return ports.ReciboLlamamientoOperativo{}, ports.ErrLlamamientoOperativoInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboLlamamientoOperativo{}, err
	}
	return s.repositorio.AbrirLlamamiento(ctx, entrada)
}
func (s *ServicioLlamamientosOperativos) RegistrarResultado(ctx context.Context, entrada ports.ResultadoLlamamientoOperativo) (ports.ReciboLlamamientoOperativo, error) {
	if ctx == nil || s == nil || s.repositorio == nil || entrada.Validar() != nil {
		return ports.ReciboLlamamientoOperativo{}, ports.ErrLlamamientoOperativoInvalido
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboLlamamientoOperativo{}, err
	}
	recibo, err := s.repositorio.RegistrarResultadoLlamamiento(ctx, entrada)
	if errors.Is(err, ports.ErrLlamamientoOperativoNoDisponible) {
		return ports.ReciboLlamamientoOperativo{}, err
	}
	return recibo, err
}
