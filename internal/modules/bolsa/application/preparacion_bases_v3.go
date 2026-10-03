package application

import (
	"context"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ServicioPreparacionBasesV3 struct {
	autorizador ports.ProveedorAutorizacionPreparacionBasesV3
	repositorio ports.RepositorioPreparacionBasesV3
}

func NuevoServicioPreparacionBasesV3(a ports.ProveedorAutorizacionPreparacionBasesV3, r ports.RepositorioPreparacionBasesV3) (*ServicioPreparacionBasesV3, error) {
	if dependenciaPreparacionNula(a) || dependenciaPreparacionNula(r) {
		return nil, ports.ErrPreparacionBasesNoDisponible
	}
	return &ServicioPreparacionBasesV3{a, r}, nil
}

func (s *ServicioPreparacionBasesV3) Guardar(ctx context.Context, q ports.SolicitudGuardarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	var cero ports.ResultadoPreparacionBasesV3
	if err := s.validarContextoV3(ctx); err != nil {
		return cero, err
	}
	if _, err := PrepararGuardadoPreparacionBasesV3(q); err != nil {
		return cero, err
	}
	m, err := q.Material.Canonico()
	if err != nil {
		return cero, err
	}
	q.Material = m
	a, err := s.autorizador.AutorizarGuardadoPreparacionBases(ctx, q)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, err
	}
	o := ports.OrdenGuardarPreparacionBasesV3{Solicitud: q, Autorizacion: a}
	if err := ValidarMaterialGuardarPreparacionBasesV3(o); err != nil {
		return cero, err
	}
	r, err := s.repositorio.GuardarPreparacionBasesV3(ctx, o)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, err
	}
	if err := ValidarResultadoGuardarPreparacionBasesV3(o, r); err != nil {
		return cero, err
	}
	if r.Estado == "version_en_conflicto" {
		return r, errorAccesoConfirmadoPreparacionBasesV3{ports.ErrPreparacionBasesConflicto}
	}
	if r.Estado == "clave_reutilizada" {
		return r, errorAccesoConfirmadoPreparacionBasesV3{ports.ErrPreparacionBasesClaveReutilizada}
	}
	return clonarResultadoPreparacionBasesV3(r)
}

func (s *ServicioPreparacionBasesV3) Consultar(ctx context.Context, q ports.SolicitudConsultarPreparacionBasesV3) (ports.ResultadoPreparacionBasesV3, error) {
	var cero ports.ResultadoPreparacionBasesV3
	if err := s.validarContextoV3(ctx); err != nil {
		return cero, err
	}
	if _, err := PrepararConsultaPreparacionBasesV3(q); err != nil {
		return cero, err
	}
	a, err := s.autorizador.AutorizarConsultaPreparacionBases(ctx, q)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, err
	}
	o := ports.OrdenConsultarPreparacionBasesV3{Solicitud: q, Autorizacion: a}
	if err := ValidarMaterialConsultarPreparacionBasesV3(o); err != nil {
		return cero, err
	}
	r, err := s.repositorio.ConsultarPreparacionBasesV3(ctx, o)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, err
	}
	if err := ValidarResultadoConsultarPreparacionBasesV3(o, r); err != nil {
		return cero, err
	}
	if r.Estado == "no_encontrada" {
		return r, errorAccesoConfirmadoPreparacionBasesV3{ports.ErrPreparacionBasesNoEncontrada}
	}
	return clonarResultadoPreparacionBasesV3(r)
}

func (s *ServicioPreparacionBasesV3) validarContextoV3(ctx context.Context) error {
	if ctx == nil || s == nil || dependenciaPreparacionNula(s.autorizador) || dependenciaPreparacionNula(s.repositorio) {
		return ports.ErrPreparacionBasesNoDisponible
	}
	return ctx.Err()
}

var _ ports.PreparadorBasesDurableV3 = (*ServicioPreparacionBasesV3)(nil)
