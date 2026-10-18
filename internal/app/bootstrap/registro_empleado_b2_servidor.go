package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	apppersonal "vec-diputacion-granada/internal/modules/personal/application"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Las lecturas nativas reutilizan el actor nominal por acción del plan B2.
// El cliente sólo elige un recurso; organismo y perfil los fija el servidor.
type autoridadConsultaRegistroPersonalB2 struct {
	autoridad *autoridadIncorporacionPersonalB2
	accion    string
}

func contextoRegistroPersonalB2(ctx context.Context, a *autoridadIncorporacionPersonalB2, accion string) (core.ContextoActor, string, error) {
	if a == nil || a.organismoRef == "" {
		return core.ContextoActor{}, "", personal.ErrRegistroEmpleadoB2NoDisponible
	}
	actor, err := a.actor(ctx, accion)
	if errors.Is(err, ct.ErrAutorizacionDenegada) || errors.Is(err, core.ErrAutorizacionDenegada) {
		return core.ContextoActor{}, "", core.ErrAutorizacionDenegada
	}
	if err != nil {
		return core.ContextoActor{}, "", err
	}
	return actor, a.organismoRef, nil
}

func (a autoridadConsultaRegistroPersonalB2) ResolverContextoRegistroEmpleadoB2(ctx context.Context) (core.ContextoActor, string, error) {
	return contextoRegistroPersonalB2(ctx, a.autoridad, a.accion)
}

func accionConsultaRegistroPersonalB2(ruta, metodo string) (string, bool) {
	if metodo != http.MethodGet {
		return "", false
	}
	switch ruta {
	case httpapi.RutaEmpleadosOrganismoB2:
		return personal.AccionEmpleadosB2, true
	case httpapi.RutaVacantesEmpleadoB2:
		return personal.AccionVacantesB2, true
	}
	ref, presente := strings.CutPrefix(ruta, httpapi.PrefijoFichaEmpleadoB2)
	return personal.AccionFichaEmpleadoB2, presente && personal.ReferenciaEmpleadoValida(ref)
}

func rutasRegistroEmpleadoPersonalB2(ficha *apppersonal.ServicioRegistroEmpleadoB2, lista *apppersonal.ServicioEmpleadosRegistroB2,
	autoridad *autoridadIncorporacionPersonalB2, registrador vp.RegistradorAuditoriaFronteraRutaExacta,
	soporte *soporteAltaContratacionTemporalDesarrollo, fronteras catalogoFronterasComunDesarrollo) ([]httpapi.RutaExacta, []httpapi.RutaColeccion, error) {
	if ficha == nil || autoridad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(registrador) {
		return nil, nil, personal.ErrRegistroEmpleadoB2NoDisponible
	}
	auditar := auditorCatalogosEmpleadoB2{registrador: registrador}
	h, err := httpapi.NewHandlerFichaEmpleadoB2(autoridadConsultaRegistroPersonalB2{autoridad, personal.AccionFichaEmpleadoB2}, ficha, auditar)
	if err != nil {
		return nil, nil, err
	}
	colecciones := []httpapi.RutaColeccion{{Prefijo: strings.TrimSuffix(httpapi.PrefijoFichaEmpleadoB2, "/"), Manejador: ligarContextoIncorporacionPersonalB2(h, soporte, fronteras)}}
	v, err := httpapi.NewHandlerVacantesEmpleadoB2(autoridadConsultaRegistroPersonalB2{autoridad, personal.AccionVacantesB2}, ficha, auditar)
	if err != nil {
		return nil, nil, err
	}
	rutas := []httpapi.RutaExacta{{Ruta: httpapi.RutaVacantesEmpleadoB2, Manejador: ligarContextoIncorporacionPersonalB2(v, soporte, fronteras)}}
	if lista != nil {
		l, err := httpapi.NewHandlerEmpleadosOrganismoB2(autoridadConsultaRegistroPersonalB2{autoridad, personal.AccionEmpleadosB2}, lista, auditar)
		if err != nil {
			return nil, nil, err
		}
		rutas = append(rutas, httpapi.RutaExacta{Ruta: httpapi.RutaEmpleadosOrganismoB2, Manejador: ligarContextoIncorporacionPersonalB2(l, soporte, fronteras)})
	}
	return rutas, colecciones, nil
}
