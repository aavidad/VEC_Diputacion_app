package bootstrap

import (
	"context"
	"errors"
	"reflect"

	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrConfiguracionCorreoAdministracionNoDisponible = errors.New("bootstrap: configuracion de correo administrativa no disponible")

// AutoridadConfiguracionCorreoAdministracion es la única frontera admitida
// para la ruta. Debe verificar el canal, cápsula, cuenta privilegiada y
// permiso fresco antes de devolver un principal; no admite cabeceras ni demo.
type AutoridadConfiguracionCorreoAdministracion interface {
	vechttp.AutoridadRutasExactas
	PrincipalConfiguracionCorreo(context.Context) (vecdomain.Principal, error)
}

func NuevaRutaConfiguracionCorreoAdministracion(autoridad AutoridadConfiguracionCorreoAdministracion, servicio *adminapp.ServicioConfiguracionCorreo) (vechttp.RutaExacta, error) {
	if dependenciaAdministracionNula(autoridad) || servicio == nil {
		return vechttp.RutaExacta{}, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	manejador, err := adminhttp.NuevoManejadorConfiguracionCorreo(autoridad, servicio)
	if err != nil {
		return vechttp.RutaExacta{}, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	return vechttp.RutaExacta{Ruta: adminhttp.RutaConfiguracionCorreo, Manejador: manejador}, nil
}

func dependenciaAdministracionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	}
	return false
}
