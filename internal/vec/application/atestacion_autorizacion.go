package application

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// ServicioAtestacionesAutorizacionV1 fija cabecera y firmante al construir la
// composicion. Ninguna peticion de usuario selecciona suite, clave o audiencia.
type ServicioAtestacionesAutorizacionV1 struct {
	cabecera domain.CabeceraAtestacionAutorizacionV1
	firmante ports.FirmanteAtestacionesAutorizacionV1
}

func NuevoServicioAtestacionesAutorizacionV1(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	firmante ports.FirmanteAtestacionesAutorizacionV1,
) (*ServicioAtestacionesAutorizacionV1, error) {
	if cabecera.Validar() != nil || dependenciaAtestacionNula(firmante) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ServicioAtestacionesAutorizacionV1{cabecera: cabecera, firmante: firmante}, nil
}

func (s *ServicioAtestacionesAutorizacionV1) Atestar(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) (ports.AtestacionAutorizacionV1, error) {
	if ctx == nil {
		return ports.AtestacionAutorizacionV1{}, ports.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AtestacionAutorizacionV1{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, err)
	}
	if s == nil || s.cabecera.Validar() != nil || dependenciaAtestacionNula(s.firmante) {
		return ports.AtestacionAutorizacionV1{}, ports.ErrFirmaAtestacionNoDisponible
	}
	solicitud, err := ports.NuevaSolicitudFirmaAtestacionAutorizacionV1(s.cabecera, decision)
	if err != nil {
		return ports.AtestacionAutorizacionV1{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, err)
	}
	resultado, err := s.firmante.FirmarAtestacionAutorizacionV1(ctx, solicitud)
	if err != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return ports.AtestacionAutorizacionV1{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, contextoErr)
		}
		return ports.AtestacionAutorizacionV1{}, ports.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AtestacionAutorizacionV1{}, errors.Join(ports.ErrFirmaAtestacionNoDisponible, err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return ports.AtestacionAutorizacionV1{}, ports.ErrFirmaAtestacionNoDisponible
	}
	atestacion, err := ports.NuevaAtestacionAutorizacionV1(solicitud, resultado)
	if err != nil {
		return ports.AtestacionAutorizacionV1{}, ports.ErrFirmaAtestacionNoDisponible
	}
	return atestacion, nil
}

func dependenciaAtestacionNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
