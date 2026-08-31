package interna

import (
	"context"
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
	servicio    *httpseguridad.ServicioIdentidad
	propietario *tokenServidorInterno
}

// NuevaFachadaIdentidadOffline recibe un servicio cuyos puertos productivos ya
// han sido constituidos por su autoridad y lo liga al propietario C4 exacto.
// No crea proveedores ni alternativas de memoria.
func NuevaFachadaIdentidadOffline(
	servicio *httpseguridad.ServicioIdentidad,
	servidor *ServidorInterno,
) (*FachadaIdentidadOffline, error) {
	if servicio == nil || servidor == nil || servidor.propietario != servidor ||
		servidor.token == nil {
		return nil, ErrIdentidadOfflineNoDisponible
	}
	return &FachadaIdentidadOffline{
		servicio:    servicio,
		propietario: servidor.token,
	}, nil
}

// Autenticar consume exclusivamente la capacidad de canal emitida por C4 en
// el contexto y una asercion protegida. Exigir la superficie interna obliga al
// ServicioIdentidad ya validado a aplicar Kerberos mas certificado, dos grupos
// criptograficos y garantia alta.
func (f *FachadaIdentidadOffline) Autenticar(
	ctx context.Context,
	asercionProtegida []byte,
) (httpseguridad.CapsulaIdentidadPeticion, error) {
	capsula, _, err := f.autenticar(ctx, asercionProtegida)
	return capsula, err
}

// AutenticarYVincular consume el canal TLS interno una sola vez, autentica la
// asercion y liga inmediatamente la capsula opaca al contexto de la misma
// peticion. El contexto original nunca se devuelve ante un fallo.
func (f *FachadaIdentidadOffline) AutenticarYVincular(
	ctx context.Context,
	asercionProtegida []byte,
) (context.Context, error) {
	capsula, canal, err := f.autenticar(ctx, asercionProtegida)
	if err != nil {
		return nil, err
	}
	ctxVinculado, err := f.servicio.VincularCapsulaIdentidadPeticion(ctx, capsula, canal)
	if err != nil {
		return nil, err
	}
	return ctxVinculado, nil
}

func (f *FachadaIdentidadOffline) autenticar(
	ctx context.Context,
	asercionProtegida []byte,
) (httpseguridad.CapsulaIdentidadPeticion, httpseguridad.CanalProxyAutenticado, error) {
	if f == nil || f.servicio == nil || f.propietario == nil ||
		interfazNulaIdentidadOffline(ctx) {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, ErrIdentidadOfflineNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, err
	}
	capacidad, valida := ctx.Value(
		claveContextoCanalTLSInterno{},
	).(*capacidadCanalTLSInterno)
	if !valida {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, httpseguridad.ErrCanalProxyNoAutenticado
	}
	estadoTLS, consumida := capacidad.consumir(f.propietario)
	if !consumida {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, httpseguridad.ErrCanalProxyNoAutenticado
	}

	canal, err := f.servicio.AutenticarCanalTLSMutuo(estadoTLS)
	if err != nil || canal.Tipo() != httpseguridad.CanalProxyTLSMutuo ||
		canal.Superficie() != httpseguridad.SuperficieInternaCorporativa {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, httpseguridad.ErrCanalProxyNoAutenticado
	}
	credencial, err := httpseguridad.NuevaCredencialProxy(asercionProtegida, canal)
	if err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, err
	}
	identidad, err := f.servicio.Resolver(ctx, credencial)
	if err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, err
	}
	capsula, err := f.servicio.ProyectarCapsulaIdentidadPeticion(ctx, identidad, canal)
	if err != nil {
		return httpseguridad.CapsulaIdentidadPeticion{}, httpseguridad.CanalProxyAutenticado{}, err
	}
	return capsula, canal, nil
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
