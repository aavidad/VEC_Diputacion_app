package bootstrap

import (
	"context"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Las hojas nominales provienen del inventario mTLS del servidor. El material
// OH sólo restringe cuenta, perfil y ámbito; no crea identidades ni permisos.
func sesionesActoresOrganizacionHistorica(ctx context.Context, a *autoridadConsultasContratacionTemporalDesarrollo, sesion *proveedorSesionConsultaRRHHDesarrollo,
	contextos map[string]internagobierno.AmbitoOrganizacionHistorica,
) (map[string]*seguridadComunDesarrollo, func(), error) {
	sesiones := []*proveedorSesionConsultaRRHHDesarrollo{sesion}
	cerrar := func() {
		for _, s := range sesiones {
			clear(s.clave[:])
		}
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesion, sesion.reloj)
	if err != nil {
		return nil, cerrar, err
	}
	porHoja := map[string]*seguridadComunDesarrollo{sesion.soporte.certificadoSHA256: seguridad}
	if a.resolvedor == nil || a.resolvedor.segundoTecnicoRRHH == nil {
		return porHoja, cerrar, nil
	}
	principal := a.resolvedor.segundoTecnicoRRHH.principal
	// Se busca por persona verificada y se exige una sola cuenta en la fuente
	// registrada. Ningún identificador enviado por HTTP interviene aquí.
	var adicional *proveedorSesionConsultaRRHHDesarrollo
	for cuenta, ambito := range contextos {
		if cuenta == sesion.base.Contexto.Instantanea.CuentaRef {
			continue
		}
		registrado, e := sesion.resolutor.ResolverContextoActorRegistradoV2(ctx, core.SolicitudContextoActor{
			Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}, PerfilActivoRef: ambito.PerfilActivoRef})
		if e != nil || registrado.Validar() != nil {
			return nil, cerrar, ct.ErrConsultaRRHHNoDisponible
		}
		actor := registrado.Contexto
		if actor.Principal.ID != principal.ID || actor.PersonaRef != principal.ID || actor.Instantanea.CuentaRef != cuenta || actor.PerfilActivoRef != ambito.PerfilActivoRef || actor.Instantanea.PerfilVersion != ambito.PerfilVersion || adicional != nil {
			return nil, cerrar, ct.ErrAutorizacionDenegada
		}
		soporte := &soporteAltaContratacionTemporalDesarrollo{sello: sesion.soporte.sello, principalID: principal.ID,
			certificadoSHA256: principal.Attributes["certificate_sha256"], contextoEsperadoRegistrado: registrado, reloj: sesion.soporte.reloj}
		base, e := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, sesion.registro, sesion.revalidador, sesion.reloj, sesion.resolutor, sesion.fronteras)
		if e != nil {
			return nil, cerrar, e
		}
		sesiones = append(sesiones, base)
		base.fijarVinculoCorporativo(sesion.vinculoCorporativo, nil)
		base.organizacionHistoricaPorCuenta = map[string]seleccionPerfilOrganizacionHistorica{cuenta: {perfil: ambito.PerfilActivoRef, version: ambito.PerfilVersion}}
		adicional = base
	}
	if adicional != nil {
		s, e := nuevaSeguridadComunDesarrollo(adicional, adicional.reloj)
		if e != nil {
			return nil, cerrar, e
		}
		if _, duplicada := porHoja[adicional.soporte.certificadoSHA256]; duplicada {
			return nil, cerrar, ct.ErrAutorizacionDenegada
		}
		porHoja[adicional.soporte.certificadoSHA256] = s
	}
	return porHoja, cerrar, nil
}
