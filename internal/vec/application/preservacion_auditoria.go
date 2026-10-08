package application

import (
	"context"
	"reflect"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type errorFuentePreservacion struct{ causa error }

func (errorFuentePreservacion) Error() string {
	return domain.ErrPreservacionAuditoriaNoDisponible.Error()
}
func (e errorFuentePreservacion) Unwrap() error { return e.causa }
func (errorFuentePreservacion) Is(err error) bool {
	return err == domain.ErrPreservacionAuditoriaNoDisponible
}

func PublicarPreservacionAuditoria(ctx context.Context, f ports.FuentePreservacionAuditoria, s domain.SolicitudPreservacionAuditoria) (domain.ResultadoPreservacionAuditoria, error) {
	if !fuentePreservacionValida(ctx, f) || s.Validar() != nil {
		return domain.ResultadoPreservacionAuditoria{}, domain.ErrPreservacionAuditoriaInvalida
	}
	r, err := f.PublicarPreservacionAuditoria(ctx, s)
	if err != nil {
		return domain.ResultadoPreservacionAuditoria{}, errorFuentePreservacion{err}
	}
	if ctx.Err() != nil || r.Validar() != nil || r.Configuracion != s || r.Estado != "publicada" && r.Estado != "replay" {
		return domain.ResultadoPreservacionAuditoria{}, domain.ErrPreservacionAuditoriaInvalida
	}
	return r, nil
}

func ConsultarPreservacionAuditoria(ctx context.Context, f ports.FuentePreservacionAuditoria, version uint64) (domain.ResultadoPreservacionAuditoria, error) {
	if !fuentePreservacionValida(ctx, f) || version > domain.MaxVersionPreservacionAuditoria {
		return domain.ResultadoPreservacionAuditoria{}, domain.ErrPreservacionAuditoriaInvalida
	}
	r, err := f.ConsultarPreservacionAuditoria(ctx, version)
	if err != nil {
		return domain.ResultadoPreservacionAuditoria{}, errorFuentePreservacion{err}
	}
	if ctx.Err() != nil || r.Validar() != nil || r.Estado != "consultada" && r.Estado != "no_publicada" || r.Estado == "consultada" && version != 0 && r.Configuracion.Version != version {
		return domain.ResultadoPreservacionAuditoria{}, domain.ErrPreservacionAuditoriaInvalida
	}
	return r, nil
}

func fuentePreservacionValida(ctx context.Context, f ports.FuentePreservacionAuditoria) bool {
	if ctx == nil || ctx.Err() != nil || f == nil {
		return false
	}
	v := reflect.ValueOf(f)
	switch v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Func, reflect.Interface, reflect.Chan, reflect.Slice:
		return !v.IsNil()
	}
	return true
}
