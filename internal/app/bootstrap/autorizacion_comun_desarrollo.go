package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"time"

	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errAutorizacionComunDesarrolloNoDisponible = errors.New("vec: autorizacion comun de desarrollo no disponible")

// politicaAutorizacionSolicitudLigadaV3Desarrollo reúne los cuatro puertos de
// una política V3; no contiene acción, ruta ni conocimiento de módulos.
type politicaAutorizacionSolicitudLigadaV3Desarrollo struct {
	fuente               puertosvec.FuenteAutorizacion
	registroConcesiones  puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3
	registroDenegaciones puertosvec.RegistroDenegacionesAutorizacionLigadaV3
	validadorMotivos     puertosvec.ValidadorReferenciaMotivoAutorizacionV2
}

func nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(f puertosvec.FuenteAutorizacion, c puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3, d puertosvec.RegistroDenegacionesAutorizacionLigadaV3, m puertosvec.ValidadorReferenciaMotivoAutorizacionV2) (politicaAutorizacionSolicitudLigadaV3Desarrollo, error) {
	p := politicaAutorizacionSolicitudLigadaV3Desarrollo{f, c, d, m}
	if !p.valida() {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, errAutorizacionComunDesarrolloNoDisponible
	}
	return p, nil
}
func (p politicaAutorizacionSolicitudLigadaV3Desarrollo) valida() bool {
	return !dependenciaAutorizacionComunDesarrolloNula(p.fuente) && !dependenciaAutorizacionComunDesarrolloNula(p.registroConcesiones) && !dependenciaAutorizacionComunDesarrolloNula(p.registroDenegaciones) && !dependenciaAutorizacionComunDesarrolloNula(p.validadorMotivos)
}

// descriptorAutorizacionComunDesarrollo liga una acción a fronteras declaradas
// y a una política completa. La composición aporta todos los nombres.
type descriptorAutorizacionComunDesarrollo struct {
	Accion         string
	ClavePolitica  string
	ClaveCapacidad string
	Fronteras      []string
	Politica       politicaAutorizacionSolicitudLigadaV3Desarrollo
}
type catalogoAutorizacionComunDesarrollo struct {
	porAccion          map[string]descriptorAutorizacionComunDesarrollo
	fronteras          map[string]struct{}
	identidadFronteras *identidadCatalogoFronterasComunDesarrollo
}

func nuevoCatalogoAutorizacionComunDesarrollo(fronteras catalogoFronterasComunDesarrollo, descriptores []descriptorAutorizacionComunDesarrollo) (catalogoAutorizacionComunDesarrollo, error) {
	if fronteras.identidad == nil {
		return catalogoAutorizacionComunDesarrollo{}, errAutorizacionComunDesarrolloNoDisponible
	}
	c := catalogoAutorizacionComunDesarrollo{porAccion: make(map[string]descriptorAutorizacionComunDesarrollo, len(descriptores)), fronteras: make(map[string]struct{}, len(fronteras.porClave)), identidadFronteras: fronteras.identidad}
	for k := range fronteras.porClave {
		c.fronteras[k] = struct{}{}
	}
	for _, d := range descriptores {
		if !claveCatalogoComunValida(d.Accion) || !claveCatalogoComunValida(d.ClavePolitica) ||
			!claveCatalogoComunValida(d.ClaveCapacidad) ||
			!d.Politica.valida() || len(d.Fronteras) == 0 {
			return catalogoAutorizacionComunDesarrollo{}, errAutorizacionComunDesarrolloNoDisponible
		}
		if _, ok := c.porAccion[d.Accion]; ok {
			return catalogoAutorizacionComunDesarrollo{}, errAutorizacionComunDesarrolloNoDisponible
		}
		copia := append([]string(nil), d.Fronteras...)
		vistos := map[string]struct{}{}
		for _, f := range copia {
			frontera, ok := fronteras.porClave[f]
			if !ok || frontera.ClavePolitica != d.ClavePolitica || frontera.ClaveCapacidad != d.ClaveCapacidad {
				return catalogoAutorizacionComunDesarrollo{}, errAutorizacionComunDesarrolloNoDisponible
			}
			if _, ok := vistos[f]; ok {
				return catalogoAutorizacionComunDesarrollo{}, errAutorizacionComunDesarrolloNoDisponible
			}
			vistos[f] = struct{}{}
		}
		d.Fronteras = copia
		c.porAccion[d.Accion] = d
	}
	return c, nil
}
func (c catalogoAutorizacionComunDesarrollo) politicaPara(accion, frontera, clavePolitica, claveCapacidad string) (politicaAutorizacionSolicitudLigadaV3Desarrollo, bool) {
	d, ok := c.porAccion[accion]
	if !ok || d.ClavePolitica != clavePolitica || d.ClaveCapacidad != claveCapacidad {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, false
	}
	for _, f := range d.Fronteras {
		if f == frontera {
			return d.Politica, true
		}
	}
	return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, false
}

func (c catalogoAutorizacionComunDesarrollo) aceptaCatalogoFronteras(fronteras catalogoFronterasComunDesarrollo) bool {
	return c.identidadFronteras != nil && c.identidadFronteras == fronteras.identidad
}

type clavePoliticaAutorizacionComunDesarrollo struct{}
type selectorAutorizacionComunDesarrollo struct {
	catalogo catalogoAutorizacionComunDesarrollo
}

func (s selectorAutorizacionComunDesarrollo) politicaDesdeContexto(ctx context.Context) (politicaAutorizacionSolicitudLigadaV3Desarrollo, bool) {
	if ctx == nil {
		return politicaAutorizacionSolicitudLigadaV3Desarrollo{}, false
	}
	p, ok := ctx.Value(clavePoliticaAutorizacionComunDesarrollo{}).(politicaAutorizacionSolicitudLigadaV3Desarrollo)
	return p, ok && p.valida()
}
func (s selectorAutorizacionComunDesarrollo) ObtenerInstantaneaAutorizacion(ctx context.Context, p, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
	x, ok := s.politicaDesdeContexto(ctx)
	if !ok {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return x.fuente.ObtenerInstantaneaAutorizacion(ctx, p, perfil)
}
func (s selectorAutorizacionComunDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx context.Context, o puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	x, ok := s.politicaDesdeContexto(ctx)
	if !ok {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	return x.registroConcesiones.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, o)
}
func (s selectorAutorizacionComunDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(ctx context.Context, o puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	x, ok := s.politicaDesdeContexto(ctx)
	if !ok {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return x.registroDenegaciones.RegistrarDenegacionAutorizacionLigadaV3(ctx, o)
}
func (s selectorAutorizacionComunDesarrollo) ValidarReferenciaMotivoAutorizacionV2(ctx context.Context, r dominiovec.ReferenciaEntradaCatalogo, i time.Time) error {
	x, ok := s.politicaDesdeContexto(ctx)
	if !ok {
		return errAutorizacionComunDesarrolloNoDisponible
	}
	return x.validadorMotivos.ValidarReferenciaMotivoAutorizacionV2(ctx, r, i)
}

type autorizadorComunDesarrollo struct {
	selector selectorAutorizacionComunDesarrollo
	servicio *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
}

func nuevoAutorizadorComunDesarrollo(catalogo catalogoAutorizacionComunDesarrollo, reloj puertosvec.Reloj, g puertosvec.GeneradorReferenciaDecisionAutorizacion, cfg aplicacionvec.ConfiguracionServicioAutorizacion) (*autorizadorComunDesarrollo, error) {
	if dependenciaAutorizacionComunDesarrolloNula(reloj) || dependenciaAutorizacionComunDesarrolloNula(g) {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	s := selectorAutorizacionComunDesarrollo{catalogo}
	servicio, e := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(s, s, s, s, reloj, g, cfg)
	if e != nil {
		return nil, e
	}
	return &autorizadorComunDesarrollo{s, servicio}, nil
}
func (a *autorizadorComunDesarrollo) contextoSolicitud(ctx context.Context, sol dominiovec.SolicitudAutorizacionLigadaV3) (context.Context, error) {
	if a == nil || a.servicio == nil || ctx == nil {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	datos, e := sol.Datos()
	if e != nil {
		return nil, e
	}
	f, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !a.selector.catalogo.aceptaCatalogoFronteras(f.catalogo) {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	p, ok := a.selector.catalogo.politicaPara(datos.Accion, f.descriptor.Clave, f.descriptor.ClavePolitica, f.descriptor.ClaveCapacidad)
	if !ok {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	return context.WithValue(ctx, clavePoliticaAutorizacionComunDesarrollo{}, p), nil
}
func (a *autorizadorComunDesarrollo) ExigirSolicitudLigadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	ctx, e := a.contextoSolicitud(ctx, s)
	if e != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errors.Join(dominiovec.ErrAutorizacionDenegada, e)
	}
	f, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || r.Validar() != nil || !f.descriptor.admitePerfil(r.Contexto.PerfilActivoRef) {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errors.Join(dominiovec.ErrAutorizacionDenegada, errAutorizacionComunDesarrolloNoDisponible)
	}
	return a.servicio.ExigirSolicitudLigadaV3(ctx, s, r)
}
func (a *autorizadorComunDesarrollo) PrepararRegistroCompuestoSolicitudLigadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2, g puertosvec.GeneradorReferenciaDecisionAutorizacion) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3, error) {
	ctx, e := a.contextoSolicitud(ctx, s)
	if e != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, errors.Join(dominiovec.ErrAutorizacionDenegada, e)
	}
	f, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || r.Validar() != nil || !f.descriptor.admitePerfil(r.Contexto.PerfilActivoRef) {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{}, errors.Join(dominiovec.ErrAutorizacionDenegada, errAutorizacionComunDesarrolloNoDisponible)
	}
	return a.servicio.PrepararRegistroCompuestoSolicitudLigadaV3(ctx, s, r, g)
}
func dependenciaAutorizacionComunDesarrolloNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
