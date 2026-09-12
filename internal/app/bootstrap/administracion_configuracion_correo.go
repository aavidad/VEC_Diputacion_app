package bootstrap

import (
	"context"
	"errors"
	"reflect"

	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
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

type servicioConfiguracionCorreoHTTP interface {
	Consultar(context.Context, vecdomain.Principal) (admindomain.VistaConfiguracionCorreo, error)
	Actualizar(context.Context, vecdomain.Principal, admindomain.ActualizacionConfiguracionCorreo) (admindomain.VistaConfiguracionCorreo, error)
}

// La adaptación conserva una API de vista, con consulta auditada y cambio
// separados en aplicación. El recibo de lectura permanece en la autoridad T13.
type servicioConfiguracionCorreoCompuesto struct {
	consulta *adminapp.ServicioConsultaConfiguracionCorreo
	cambio   *adminapp.ServicioConfiguracionCorreo
}

func (s *servicioConfiguracionCorreoCompuesto) Consultar(ctx context.Context, p vecdomain.Principal) (admindomain.VistaConfiguracionCorreo, error) {
	if s == nil || s.consulta == nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	resultado, err := s.consulta.Consultar(ctx, p)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, err
	}
	return resultado.Vista, nil
}
func (s *servicioConfiguracionCorreoCompuesto) Actualizar(ctx context.Context, p vecdomain.Principal, e admindomain.ActualizacionConfiguracionCorreo) (admindomain.VistaConfiguracionCorreo, error) {
	if s == nil || s.cambio == nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	return s.cambio.Actualizar(ctx, p, e)
}

func NuevaRutaConfiguracionCorreoAdministracion(autoridad AutoridadConfiguracionCorreoAdministracion, servicio servicioConfiguracionCorreoHTTP) (vechttp.RutaExacta, error) {
	if dependenciaAdministracionNula(autoridad) || dependenciaAdministracionNula(servicio) {
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
