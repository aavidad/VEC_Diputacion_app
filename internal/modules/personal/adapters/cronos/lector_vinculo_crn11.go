// Package cronos proyecta únicamente el vínculo autorizado que pide CRN11.
package cronos

import (
	"context"
	"errors"
	"reflect"

	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type LectorVinculoPropioCRN11 struct {
	lector ports.LectorVinculoPropioCRN11
}

func NuevoLectorVinculoPropioCRN11(lector ports.LectorVinculoPropioCRN11) (*LectorVinculoPropioCRN11, error) {
	if nuloLectorCRN11(lector) {
		return nil, cronosports.ErrDependenciaNoDisponible
	}
	return &LectorVinculoPropioCRN11{lector: lector}, nil
}
func (l *LectorVinculoPropioCRN11) ConsultarVinculoPropioCRN11(ctx context.Context, in cronosports.InputConsultaVinculoPropioCRN11) (cronosports.VinculoPropioHistoricoCRN11, error) {
	var cero cronosports.VinculoPropioHistoricoCRN11
	if l == nil || nuloLectorCRN11(l.lector) || ctx == nil {
		return cero, cronosports.ErrDependenciaNoDisponible
	}
	r, err := l.lector.ConsultarVinculoPropioCRN11(ctx, domain.SolicitudVinculoPropioCRN11{Actor: in.Actor, EmpleadoRef: in.EmpleadoRef})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return cero, err
		case errors.Is(err, domain.ErrVinculoCRN11Denegado), errors.Is(err, domain.ErrVinculoCRN11Invalido):
			return cero, cronosports.ErrCorreccionNoAutorizada
		default:
			return cero, cronosports.ErrDependenciaNoDisponible
		}
	}
	return cronosports.VinculoPropioHistoricoCRN11{PersonaRef: r.Vinculo.PersonaRef, EmpleadoRef: r.Vinculo.EmpleadoRef,
		VinculoRef: r.Vinculo.VinculoRef, FuenteRef: r.Vinculo.FuenteRef, Version: r.Vinculo.Version,
		Evidencia: cronosports.EvidenciaLecturaVinculoCRN11{ReciboRef: r.Evidencia.ReciboRef, DecisionRef: r.Evidencia.DecisionRef, AuditoriaRef: r.Evidencia.AuditoriaRef, ConsultadaEn: r.Evidencia.ConsultadaEn}}, nil
}
func nuloLectorCRN11(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice:
		return r.IsNil()
	}
	return false
}

var _ cronosports.LectorVinculoPropioHistoricoCRN11 = (*LectorVinculoPropioCRN11)(nil)
