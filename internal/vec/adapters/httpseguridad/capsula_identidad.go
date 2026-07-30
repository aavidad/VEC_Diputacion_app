package httpseguridad

import (
	"context"
	"crypto/subtle"
	"fmt"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type CapsulaIdentidadPeticion struct {
	cuenta    dominiovec.CuentaAutenticadaContextoActor
	auditoria ContextoAuditoriaAutenticada
	servicio  *ServicioIdentidad
	instancia [32]byte
	canal     string
}

func (CapsulaIdentidadPeticion) String() string               { return "[CÁPSULA DE IDENTIDAD CONFIDENCIAL]" }
func (CapsulaIdentidadPeticion) GoString() string             { return "[CÁPSULA DE IDENTIDAD CONFIDENCIAL]" }
func (CapsulaIdentidadPeticion) MarshalJSON() ([]byte, error) { return nil, ErrIdentidadNoSerializable }
func (CapsulaIdentidadPeticion) MarshalText() ([]byte, error) { return nil, ErrIdentidadNoSerializable }
func (CapsulaIdentidadPeticion) MarshalBinary() ([]byte, error) {
	return nil, ErrIdentidadNoSerializable
}
func (CapsulaIdentidadPeticion) GobEncode() ([]byte, error) { return nil, ErrIdentidadNoSerializable }
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
	return CapsulaIdentidadPeticion{cuenta, auditoria, s, s.instanciaRef, canal.ReferenciaVinculacion()}, nil
}
func (c CapsulaIdentidadPeticion) Datos(s *ServicioIdentidad, canal CanalProxyAutenticado) (dominiovec.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if s == nil || c.servicio != s || c.instancia != s.instanciaRef || canal.validar(s) != nil || subtle.ConstantTimeCompare([]byte(c.canal), []byte(canal.ReferenciaVinculacion())) != 1 {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return c.cuenta, c.auditoria, nil
}

type claveCapsulaIdentidad struct{}

func VincularCapsulaIdentidadPeticion(ctx context.Context, c CapsulaIdentidadPeticion, s *ServicioIdentidad, canal CanalProxyAutenticado) (context.Context, error) {
	if ctx == nil || ctx.Err() != nil {
		return nil, ErrSesionNoValida
	}
	if _, _, err := c.Datos(s, canal); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, claveCapsulaIdentidad{}, c), nil
}
func ExtraerCapsulaIdentidadPeticion(ctx context.Context, s *ServicioIdentidad, canal CanalProxyAutenticado) (dominiovec.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if ctx == nil || ctx.Err() != nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	c, ok := ctx.Value(claveCapsulaIdentidad{}).(CapsulaIdentidadPeticion)
	if !ok {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return c.Datos(s, canal)
}
