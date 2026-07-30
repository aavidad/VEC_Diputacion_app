package httpseguridad

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/xml"
	"fmt"
	"log/slog"
	"sync/atomic"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// CapsulaIdentidadPeticion retiene una identidad ya autenticada sin exponer
// sus localizadores. Solo la instancia emisora puede vincularla y consumirla.
type CapsulaIdentidadPeticion struct {
	servicio  *ServicioIdentidad
	instancia [32]byte
	canal     [32]byte
	identidad IdentidadSesion
	estado    *estadoVinculacionCapsulaIdentidad
}

type estadoVinculacionCapsulaIdentidad struct {
	vinculada atomic.Bool
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
func (CapsulaIdentidadPeticion) MarshalCBOR() ([]byte, error)  { return nil, ErrIdentidadNoSerializable }
func (*CapsulaIdentidadPeticion) UnmarshalCBOR([]byte) error   { return ErrIdentidadNoSerializable }
func (CapsulaIdentidadPeticion) MarshalYAML() (any, error)     { return nil, ErrIdentidadNoSerializable }
func (*CapsulaIdentidadPeticion) UnmarshalYAML(func(any) error) error {
	return ErrIdentidadNoSerializable
}
func (CapsulaIdentidadPeticion) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrIdentidadNoSerializable
}
func (*CapsulaIdentidadPeticion) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrIdentidadNoSerializable
}
func (CapsulaIdentidadPeticion) LogValue() slog.Value {
	return slog.StringValue("[CÁPSULA DE IDENTIDAD CONFIDENCIAL]")
}
func (CapsulaIdentidadPeticion) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte("[CÁPSULA DE IDENTIDAD CONFIDENCIAL]"))
}

// ProyectarCapsulaIdentidadPeticion crea la cápsula desde una sesión y el mismo
// canal mTLS que quedó comprometido por la aserción protegida.
func (s *ServicioIdentidad) ProyectarCapsulaIdentidadPeticion(
	ctx context.Context,
	identidad IdentidadSesion,
	canal CanalProxyAutenticado,
) (CapsulaIdentidadPeticion, error) {
	if s == nil || ctx == nil || canal.validar(s) != nil || identidad.servicio != s || subtle.ConstantTimeCompare([]byte(canal.ReferenciaVinculacion()), []byte(identidad.estado.canalVinculadoRef)) != 1 {
		return CapsulaIdentidadPeticion{}, ErrSesionNoValida
	}
	_, auditoria, err := s.ProyectarCuentaAutenticada(ctx, identidad)
	if err != nil {
		return CapsulaIdentidadPeticion{}, err
	}
	if subtle.ConstantTimeCompare([]byte(auditoria.CanalVinculadoRef()), []byte(canal.ReferenciaVinculacion())) != 1 {
		return CapsulaIdentidadPeticion{}, ErrSesionNoValida
	}
	huella := sha256.Sum256([]byte(canal.ReferenciaVinculacion()))
	return CapsulaIdentidadPeticion{
		servicio: s, instancia: s.instanciaRef, canal: huella, identidad: identidad,
		estado: &estadoVinculacionCapsulaIdentidad{},
	}, nil
}

func (c CapsulaIdentidadPeticion) datos(
	ctx context.Context,
	s *ServicioIdentidad,
	canalVinculadoRef string,
) (dominiovec.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if ctx == nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, err
	}
	if s == nil || c.servicio != s || c.estado == nil ||
		c.instancia != s.instanciaRef || c.identidad.servicio != s ||
		canalVinculadoRef == "" {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	huella := sha256.Sum256([]byte(canalVinculadoRef))
	if subtle.ConstantTimeCompare(c.canal[:], huella[:]) != 1 {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	cuenta, auditoria, err := s.ProyectarCuentaAutenticada(ctx, c.identidad)
	if err != nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, err
	}
	if subtle.ConstantTimeCompare(
		[]byte(c.identidad.estado.canalVinculadoRef), []byte(canalVinculadoRef),
	) != 1 || subtle.ConstantTimeCompare(
		[]byte(auditoria.CanalVinculadoRef()), []byte(canalVinculadoRef),
	) != 1 {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return cuenta, auditoria, nil
}

type claveCapsulaIdentidad struct{}

type capsulaIdentidadVinculada struct {
	capsula           CapsulaIdentidadPeticion
	canalVinculadoRef string
}

func (capsulaIdentidadVinculada) String() string {
	return "[CÁPSULA DE IDENTIDAD VINCULADA CONFIDENCIAL]"
}
func (capsulaIdentidadVinculada) GoString() string {
	return "[CÁPSULA DE IDENTIDAD VINCULADA CONFIDENCIAL]"
}
func (capsulaIdentidadVinculada) LogValue() slog.Value {
	return slog.StringValue("[CÁPSULA DE IDENTIDAD VINCULADA CONFIDENCIAL]")
}
func (capsulaIdentidadVinculada) Format(s fmt.State, _ rune) {
	_, _ = s.Write([]byte("[CÁPSULA DE IDENTIDAD VINCULADA CONFIDENCIAL]"))
}

// VincularCapsulaIdentidadPeticion liga una cápsula al contexto de la petición
// después de revalidar sesión, cuenta, instancia y canal actual.
func (s *ServicioIdentidad) VincularCapsulaIdentidadPeticion(
	ctx context.Context,
	c CapsulaIdentidadPeticion,
	canal CanalProxyAutenticado,
) (context.Context, error) {
	if ctx == nil {
		return nil, ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if ctx.Value(claveCapsulaIdentidad{}) != nil {
		return nil, ErrSesionNoValida
	}
	if canal.validar(s) != nil {
		return nil, ErrSesionNoValida
	}
	referenciaCanal := canal.ReferenciaVinculacion()
	if _, _, err := c.datos(ctx, s, referenciaCanal); err != nil {
		return nil, err
	}
	if !c.estado.vinculada.CompareAndSwap(false, true) {
		return nil, ErrSesionNoValida
	}
	return context.WithValue(ctx, claveCapsulaIdentidad{}, capsulaIdentidadVinculada{
		capsula: c, canalVinculadoRef: referenciaCanal,
	}), nil
}

// ExtraerCapsulaIdentidadPeticion recupera la identidad ligada y vuelve a
// comprobar su vigencia contra el registro durable.
func (s *ServicioIdentidad) ExtraerCapsulaIdentidadPeticion(
	ctx context.Context,
) (dominiovec.CuentaAutenticadaContextoActor, ContextoAuditoriaAutenticada, error) {
	if ctx == nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, err
	}
	vinculada, ok := ctx.Value(claveCapsulaIdentidad{}).(capsulaIdentidadVinculada)
	if !ok {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	if vinculada.capsula.estado == nil || !vinculada.capsula.estado.vinculada.Load() {
		return dominiovec.CuentaAutenticadaContextoActor{}, ContextoAuditoriaAutenticada{}, ErrSesionNoValida
	}
	return vinculada.capsula.datos(ctx, s, vinculada.canalVinculadoRef)
}
