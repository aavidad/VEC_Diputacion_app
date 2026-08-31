package interna

import (
	"context"
	"crypto/tls"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

var ErrIdentidadOfflineNoDisponible = errors.New(
	"composicion interna: identidad offline no disponible",
)

// FachadaIdentidadOffline encadena la identidad ya compuesta sin poseer
// listeners, configuracion ni credenciales. ServicioIdentidad conserva la
// autoridad exclusiva sobre canal, asercion, alta durable y revalidacion.
type FachadaIdentidadOffline struct {
	servicio *httpseguridad.ServicioIdentidad
}

// NuevaFachadaIdentidadOffline recibe un servicio cuyos puertos productivos ya
// han sido constituidos por su autoridad. No crea proveedores ni alternativas
// de memoria.
func NuevaFachadaIdentidadOffline(
	servicio *httpseguridad.ServicioIdentidad,
) (*FachadaIdentidadOffline, error) {
	if servicio == nil {
		return nil, ErrIdentidadOfflineNoDisponible
	}
	return &FachadaIdentidadOffline{servicio: servicio}, nil
}

// Autenticar transforma exclusivamente un ConnectionState mTLS real y una
// asercion protegida en la capsula opaca existente. Exigir la superficie
// interna obliga al ServicioIdentidad ya validado a aplicar Kerberos mas
// certificado, dos grupos criptograficos y garantia alta.
func (f *FachadaIdentidadOffline) Autenticar(
	ctx context.Context,
	estadoTLS tls.ConnectionState,
	asercionProtegida []byte,
) (httpseguridad.CapsulaIdentidadPeticion, error) {
	if f == nil || f.servicio == nil || interfazNulaIdentidadOffline(ctx) {
		return httpseguridad.CapsulaIdentidadPeticion{}, ErrIdentidadOfflineNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, err
	}

	canal, err := f.servicio.AutenticarCanalTLSMutuo(estadoTLS)
	if err != nil || canal.Tipo() != httpseguridad.CanalProxyTLSMutuo ||
		canal.Superficie() != httpseguridad.SuperficieInternaCorporativa {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.ErrCanalProxyNoAutenticado
	}
	credencial, err := httpseguridad.NuevaCredencialProxy(asercionProtegida, canal)
	if err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, err
	}
	identidad, err := f.servicio.Resolver(ctx, credencial)
	if err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, err
	}
	return f.servicio.ProyectarCapsulaIdentidadPeticion(ctx, identidad, canal)
}

func interfazNulaIdentidadOffline(valor any) bool {
	if valor == nil {
		return true
	}
	vista := reflect.ValueOf(valor)
	switch vista.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return vista.IsNil()
	default:
		return false
	}
}
