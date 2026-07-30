package httpseguridad

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"log/slog"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type CapsulaIdentidadPeticion struct {
	cuenta    dominiovec.CuentaAutenticadaContextoActor
	auditoria ContextoAuditoriaAutenticada
	servicio  *ServicioIdentidad
	instancia [32]byte
	canal     [32]byte
	identidad IdentidadSesion
}

func (CapsulaIdentidadPeticion) String() string               { return "[CÁPSULA DE IDENTIDAD CONFIDENCIAL]" }
func (CapsulaIdentidadPeticion) GoString() string             { return "[CÁPSULA DE IDENTIDAD CONFIDENCIAL]" }
func (CapsulaIdentidadPeticion) MarshalJSON() ([]byte, error) { return nil, ErrIdentidadNoSerializable }
func (CapsulaIdentidadPeticion) MarshalText() ([]byte, error) { return nil, ErrIdentidadNoSerializable }
func (CapsulaIdentidadPeticion) MarshalBinary() ([]byte, error) {
	return nil, ErrIdentidadNoSerializable
}
func (CapsulaIdentidadPeticion) GobEncode() ([]byte, error)    { return nil, ErrIdentidadNoSerializable }
func (*CapsulaIdentidadPeticion) UnmarshalJSON([]byte) error   { return ErrIdentidadNoSerializable }
func (*CapsulaIdentidadPeticion) UnmarshalText([]byte) error   { return ErrIdentidadNoSerializable }
func (*CapsulaIdentidadPeticion) UnmarshalBinary([]byte) error { return ErrIdentidadNoSerializable }
func (*CapsulaIdentidadPeticion) GobDecode([]byte) error       { return ErrIdentidadNoSerializable }
func (CapsulaIdentidadPeticion) LogValue() slog.Value {
	return slog.StringValue("[CÁPSULA DE IDENTIDAD CONFIDENCIAL]")
}
func (CapsulaIdentidadPeticion) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte("[CÁPSULA DE IDENTIDAD CONFIDENCIAL]"))
}
func (s *ServicioIdentidad) ProyectarCapsulaIdentidadPeticion(ctx context.Context, identidad IdentidadSesion, canal CanalProxyAutenticado) (CapsulaIdentidadPeticion, error) {
	if s == nil || ctx == nil || canal.validar(s) != nil || identidad.servicio != s || subtle.ConstantTimeCompare([]byte(canal.ReferenciaVinculacion()), []byte(identidad.estado.canalVinculadoRef)) != 1 {
		return CapsulaIdentidadPeticion{}, ErrSesionNoValida
	}
	cuenta, auditoria, err := s.ProyectarCuentaAutenticada(ctx, identidad)
	if err != nil {
		return CapsulaIdentidadPeticion{}, err
	}
	if subtle.ConstantTimeCompare([]byte(auditoria.CanalVinculadoRef()), []byte(canal.ReferenciaVinculacion())) != 1 {
		return CapsulaIdentidadPeticion{}, ErrSesionNoValida
	}
	huella := sha256.Sum256([]byte(canal.ReferenciaVinculacion()))
	return CapsulaIdentidadPeticion{cuenta, auditoria, s, s.instanciaRef, huella, identidad}, nil
}
func (c CapsulaIdentidadPeticion) datos(ctx context.Context, s *ServicioIdentidad, canal CanalProxyAutenticado) (dominiovec.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if ctx == nil || s == nil || c.servicio != s || c.instancia != s.instanciaRef || canal.validar(s) != nil || c.identidad.servicio != s {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	huella := sha256.Sum256([]byte(canal.ReferenciaVinculacion()))
	if subtle.ConstantTimeCompare(c.canal[:], huella[:]) != 1 {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return s.ProyectarCuentaAutenticada(ctx, c.identidad)
}

type claveCapsulaIdentidad struct{}

func (s *ServicioIdentidad) VincularCapsulaIdentidadPeticion(ctx context.Context, c CapsulaIdentidadPeticion, canal CanalProxyAutenticado) (context.Context, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrSesionNoValida
	}
	if _, _, err := c.datos(ctx, s, canal); err != nil {
		return nil, err
	}
	if ctx.Value(claveCapsulaIdentidad{}) != nil {
		return nil, ErrSesionNoValida
	}
	return context.WithValue(ctx, claveCapsulaIdentidad{}, c), nil
}
func (s *ServicioIdentidad) ExtraerCapsulaIdentidadPeticion(ctx context.Context, canal CanalProxyAutenticado) (dominiovec.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if ctx == nil || ctx.Err() != nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	c, ok := ctx.Value(claveCapsulaIdentidad{}).(CapsulaIdentidadPeticion)
	if !ok {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return c.datos(ctx, s, canal)
}
