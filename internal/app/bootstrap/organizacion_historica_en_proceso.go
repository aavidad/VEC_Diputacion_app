package bootstrap

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const fronteraPersonalDesdeCT = "organizacion-historica-desde-ct"

// Sólo reconoce la suboperación privada, con el método y ruta originales.
// No registra otra ruta HTTP ni concede permisos.
func fronteraConsultaPersonal(f fronteraSeguridadComunDesarrollo) bool {
	return f.ruta == httpapi.RutaOrganizacionHistoricaPersonal && f.metodo == http.MethodGet ||
		f.descriptor.Clave == fronteraPersonalDesdeCT && f.ruta == httpinterno.RutaSeleccionLlamamiento && f.metodo == http.MethodPost
}

type consultaPersonalConActor interface {
	Consultar(context.Context, personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)
}

type consultaPersonalLigadaCT struct {
	origen   context.Context
	contexto context.Context
	actor    ct.ContextoAutorizacionAltaV3
	consulta consultaPersonalConActor
	barrera  *barreraPeticionesOrganizacionHistorica
}

func (c consultaPersonalLigadaCT) Consultar(ctx context.Context, selector personal.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	if ctx == nil || ctx != c.origen || ctx.Err() != nil || c.consulta == nil || c.barrera == nil || !c.barrera.entrar() {
		return personalports.ResultadoConsultaOrganizacionHistorica{}, personal.ErrOrganizacionHistoricaNoDisponible
	}
	defer c.barrera.activas.Done()
	return c.consulta.Consultar(c.contexto, personal.SolicitudConsultaOrganizacionHistorica{Actor: c.actor.Resultado.Contexto, Selector: selector})
}

func (m *montajeOrganizacionHistoricaCombinada) consultaParaCT(ctx context.Context, puente *puenteBolsaLlamamientoDesarrollo, d preparacionLlamamientoDesarrollo) (consultaOrganizacionHistoricaPuestoCT, string, error) {
	if m == nil || m.fuente == nil || m.consulta == nil || puente == nil || puente.operacionNativa == nil {
		return nil, "", ct.ErrIntegracionBolsaNoDisponible
	}
	captura, err := puente.capturaOperacionNativaDesdeHijo(ctx, d)
	if err != nil {
		return nil, "", err
	}
	// El hijo Bolsa conserva la misma hoja y ruta CT. Sólo su frontera privada
	// puede iniciar esta consulta; la captura CT se coteja de nuevo al entrar.
	frontera, ok := fronteraSeguridadComunDesdeContexto(ctx)
	capacidad, valida := puente.alta.soporte.capacidadValida(ctx)
	if !ok || !valida || !puente.operacionNativa.privado.mismaInstancia(frontera.catalogo) ||
		frontera.ruta != httpinterno.RutaSeleccionLlamamiento || frontera.metodo != http.MethodPost || capacidad.ruta != frontera.ruta || capacidad.metodo != frontera.metodo {
		return nil, "", ct.ErrAutorizacionDenegada
	}
	descriptor, ok := m.fronteras.resolver(capacidad.metodo, capacidad.ruta)
	if !ok || descriptor.Clave != fronteraPersonalDesdeCT {
		return nil, "", ct.ErrAutorizacionDenegada
	}
	hijo := context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
		metodo: capacidad.metodo, ruta: capacidad.ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: m.fronteras, descriptor: descriptor})
	hijo = context.WithValue(hijo, claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
	hijo, err = m.fuente.vincular(hijo)
	if err != nil {
		return nil, "", err
	}
	actor, organismo, _, err := m.fuente.ContextoVinculadoOrganizacionHistorica(hijo)
	if err != nil {
		return nil, "", err
	}
	a, b := actor.Resultado.Contexto, captura.ct.Resultado.Contexto
	if a.Principal.ID != b.Principal.ID || a.PersonaRef != b.PersonaRef || a.Instantanea.CuentaRef != b.Instantanea.CuentaRef {
		return nil, "", ct.ErrAutorizacionDenegada
	}
	return consultaPersonalLigadaCT{origen: ctx, contexto: hijo, actor: actor, consulta: m.consulta, barrera: m.barrera}, organismo, nil
}
