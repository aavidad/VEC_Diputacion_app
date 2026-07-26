package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioAtestacionesAutorizacionV3 fija el perfil VEC-AD-3 en la
// composición confiable. Suite, clave y audiencia no proceden del cliente.
type ServicioAtestacionesAutorizacionV3 struct {
	cabecera domain.CabeceraAtestacionAutorizacionV3
	firmante ports.FirmanteAtestacionesAutorizacionV3
}

func NuevoServicioAtestacionesAutorizacionV3(
	cabecera domain.CabeceraAtestacionAutorizacionV3,
	firmante ports.FirmanteAtestacionesAutorizacionV3,
) (*ServicioAtestacionesAutorizacionV3, error) {
	if cabecera.Validar() != nil || dependenciaAtestacionNula(firmante) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ServicioAtestacionesAutorizacionV3{
		cabecera: cabecera,
		firmante: firmante,
	}, nil
}

func (s *ServicioAtestacionesAutorizacionV3) Atestar(
	ctx context.Context,
	decision domain.DecisionAutorizacionLigadaV3,
	referenciaMotivo domain.ReferenciaEntradaCatalogo,
	resultadoContexto domain.ResultadoContextoActorRegistradoV2,
) (ports.AtestacionAutorizacionV3, error) {
	if ctx == nil {
		return ports.AtestacionAutorizacionV3{},
			ports.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AtestacionAutorizacionV3{}, errors.Join(
			ports.ErrFirmaAtestacionNoDisponible,
			err,
		)
	}
	if s == nil || s.cabecera.Validar() != nil ||
		dependenciaAtestacionNula(s.firmante) {
		return ports.AtestacionAutorizacionV3{},
			ports.ErrFirmaAtestacionNoDisponible
	}
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(
		s.cabecera,
		decision,
		referenciaMotivo,
		resultadoContexto,
	)
	if err != nil {
		return ports.AtestacionAutorizacionV3{}, errors.Join(
			ports.ErrFirmaAtestacionNoDisponible,
			err,
		)
	}
	resultado, err := s.firmante.FirmarAtestacionAutorizacionV3(ctx, solicitud)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return ports.AtestacionAutorizacionV3{}, errors.Join(
				ports.ErrFirmaAtestacionNoDisponible,
				contextoErr,
			)
		}
		return ports.AtestacionAutorizacionV3{},
			ports.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AtestacionAutorizacionV3{}, errors.Join(
			ports.ErrFirmaAtestacionNoDisponible,
			err,
		)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ports.AtestacionAutorizacionV3{},
			ports.ErrFirmaAtestacionNoDisponible
	}
	atestacion, err := ports.NuevaAtestacionAutorizacionV3(solicitud, resultado)
	if err != nil {
		return ports.AtestacionAutorizacionV3{},
			ports.ErrFirmaAtestacionNoDisponible
	}
	return atestacion, nil
}
