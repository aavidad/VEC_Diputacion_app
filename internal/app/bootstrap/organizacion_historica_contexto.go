package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type claveContextoOrganizacionHistoricaCombinada struct{}

type contextoOrganizacionHistoricaCombinada struct {
	fuente *fuenteOrganizacionHistoricaCombinada
	actor  ct.ContextoAutorizacionAltaV3
	ambito internagobierno.AmbitoOrganizacionHistorica
}

// fuenteOrganizacionHistoricaCombinada conserva una sola resolución registrada
// por petición. Los ámbitos privados restringen el perfil; no son permisos.
type fuenteOrganizacionHistoricaCombinada struct {
	sesion      *seguridadComunDesarrollo
	corporativo vecports.RevalidadorVinculoCorporativoRRHHV1
	reloj       ct.Reloj
	contextos   map[string]internagobierno.AmbitoOrganizacionHistorica
	fronteras   catalogoFronterasComunDesarrollo
}

func (f *fuenteOrganizacionHistoricaCombinada) vincular(ctx context.Context) (context.Context, error) {
	if f == nil || f.sesion == nil || f.corporativo == nil || f.reloj == nil || ctx == nil || ctx.Err() != nil || ctx.Value(claveContextoOrganizacionHistoricaCombinada{}) != nil {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	frontera, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || frontera.ruta != httpapi.RutaOrganizacionHistoricaPersonal || frontera.metodo != "GET" || frontera.superficie != superficieInternaSeguridadComunDesarrollo || !f.fronteras.mismaInstancia(frontera.catalogo) {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	c, err := f.sesion.ResolverContexto(ctx)
	if err != nil || c.Resultado.Validar() != nil || c.Vinculo.ValidarPara(c.Resultado) != nil {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	a := c.Resultado.Contexto
	ambito, ok := f.contextos[a.Instantanea.CuentaRef]
	if !ok || ambito.Validar() != nil || ambito.PerfilActivoRef != a.PerfilActivoRef || ambito.PerfilVersion != a.Instantanea.PerfilVersion || !frontera.descriptor.admitePerfil(a.PerfilActivoRef) {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	datos, err := c.Vinculo.Datos()
	if err != nil || datos.CuentaRef != a.Instantanea.CuentaRef || datos.PerfilActivoRef != a.PerfilActivoRef || datos.PrincipalID != a.Principal.ID ||
		f.corporativo.RevalidarVinculoCorporativoRRHHV1(ctx, vecports.SolicitudRevalidacionVinculoCorporativoRRHHV1{
			CuentaRef: datos.CuentaRef, PerfilRef: datos.PerfilActivoRef, PersonaRef: datos.PrincipalID,
			VinculoContextoRef: datos.ContextoActorRef, VinculoContextoVersion: datos.ContextoActorVersion,
		}) != nil || ctx.Err() != nil {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	clon, err := c.Resultado.Clonar()
	if err != nil {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	return context.WithValue(ctx, claveContextoOrganizacionHistoricaCombinada{}, contextoOrganizacionHistoricaCombinada{
		fuente: f, actor: ct.ContextoAutorizacionAltaV3{Vinculo: c.Vinculo, Resultado: clon}, ambito: ambito,
	}), nil
}

func (f *fuenteOrganizacionHistoricaCombinada) ContextoVinculadoOrganizacionHistorica(ctx context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	r, org, unidad, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx)
	if err != nil || ctx.Err() != nil || !r.Vinculo.VigenteEn(f.reloj.Ahora(), r.Resultado) {
		return ct.ContextoAutorizacionAltaV3{}, "", "", httpapi.ErrAccesoRutaExactaDenegado
	}
	return r, org, unidad, nil
}

func (f *fuenteOrganizacionHistoricaCombinada) ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	var vacio ct.ContextoAutorizacionAltaV3
	if f == nil || ctx == nil {
		return vacio, "", "", httpapi.ErrAccesoRutaExactaDenegado
	}
	c, ok := ctx.Value(claveContextoOrganizacionHistoricaCombinada{}).(contextoOrganizacionHistoricaCombinada)
	if !ok || c.fuente != f || c.actor.Resultado.Validar() != nil || c.actor.Vinculo.ValidarPara(c.actor.Resultado) != nil ||
		c.ambito.Validar() != nil || c.actor.Resultado.Contexto.PerfilActivoRef != c.ambito.PerfilActivoRef || c.actor.Resultado.Contexto.Instantanea.PerfilVersion != c.ambito.PerfilVersion {
		return vacio, "", "", httpapi.ErrAccesoRutaExactaDenegado
	}
	r, err := c.actor.Resultado.Clonar()
	if err != nil {
		return vacio, "", "", httpapi.ErrAccesoRutaExactaDenegado
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: c.actor.Vinculo, Resultado: r}, c.ambito.OrganismoRef, c.ambito.UnidadClave, nil
}

var _ interface {
	ContextoVinculadoOrganizacionHistorica(context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error)
} = (*fuenteOrganizacionHistoricaCombinada)(nil)
