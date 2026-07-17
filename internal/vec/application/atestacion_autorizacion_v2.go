package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioAtestacionesAutorizacionV2 fija el perfil nominal VEC-AD-2 en la
// composicion. La decision y la referencia de motivo deben proceder del mismo
// ciclo de autorizacion; el usuario no puede elegir suite, clave o audiencia.
type ServicioAtestacionesAutorizacionV2 struct {
	cabecera domain.CabeceraAtestacionAutorizacionV2
	firmante ports.FirmanteAtestacionesAutorizacionV2
}

func NuevoServicioAtestacionesAutorizacionV2(
	cabecera domain.CabeceraAtestacionAutorizacionV2,
	firmante ports.FirmanteAtestacionesAutorizacionV2,
) (*ServicioAtestacionesAutorizacionV2, error) {
	if cabecera.Validar() != nil || dependenciaAtestacionNula(firmante) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ServicioAtestacionesAutorizacionV2{cabecera: cabecera, firmante: firmante}, nil
}

func (s *ServicioAtestacionesAutorizacionV2) Atestar(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
) (ports.AtestacionAutorizacionV2, error) {
	if ctx == nil {
		return ports.AtestacionAutorizacionV2{}, ports.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AtestacionAutorizacionV2{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, err)
	}
	if s == nil || s.cabecera.Validar() != nil || dependenciaAtestacionNula(s.firmante) {
		return ports.AtestacionAutorizacionV2{}, ports.ErrFirmaAtestacionNoDisponible
	}

	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV2(
		s.cabecera,
		decision,
		referenciaMotivo,
	)
	if err != nil {
		return ports.AtestacionAutorizacionV2{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, err)
	}
	resultado, err := s.firmante.FirmarAtestacionAutorizacionV2(ctx, solicitud)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return ports.AtestacionAutorizacionV2{}, errors.Join(
				ports.ErrFirmaAtestacionNoDisponible,
				contextoErr,
			)
		}
		return ports.AtestacionAutorizacionV2{}, ports.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AtestacionAutorizacionV2{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ports.AtestacionAutorizacionV2{}, ports.ErrFirmaAtestacionNoDisponible
	}
	atestacion, err := ports.NuevaAtestacionAutorizacionV2(solicitud, resultado)
	if err != nil {
		return ports.AtestacionAutorizacionV2{}, ports.ErrFirmaAtestacionNoDisponible
	}
	return atestacion, nil
}
