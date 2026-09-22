package application

import (
	"context"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ServicioOrdenVigente struct {
	consulta puertosbolsa.ConsultaOrdenVigente
}

func NuevoServicioOrdenVigente(consulta puertosbolsa.ConsultaOrdenVigente) (*ServicioOrdenVigente, error) {
	if consulta == nil {
		return nil, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	return &ServicioOrdenVigente{consulta: consulta}, nil
}

func (s *ServicioOrdenVigente) Consultar(ctx context.Context, bolsaRef string) (dominiobolsa.OrdenVigenteBolsa, error) {
	if s == nil || s.consulta == nil || ctx == nil || bolsaRef == "" {
		return dominiobolsa.OrdenVigenteBolsa{}, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	orden, err := s.consulta.ConsultarOrdenVigente(ctx, bolsaRef)
	if err != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, err
	}
	if err := orden.Validar(); err != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	orden.Posiciones = append([]dominiobolsa.PosicionOrdenBolsa(nil), orden.Posiciones...)
	return orden, nil
}
