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
	identidades := map[string]core.Principal{}
	if a.resolvedor != nil {
		for _, principal := range a.resolvedor.porHuella {
			if !principalContratacionTemporalDesarrolloValido(principal) {
				continue
			}
			semilla, e := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, sesion.reloj.Ahora())
			if e != nil {
				return nil, cerrar, e
			}
			cuenta := semilla.Resultado.Contexto.Instantanea.CuentaRef
			if _, existe := identidades[cuenta]; existe {
				return nil, cerrar, ct.ErrAutorizacionDenegada
			}
			identidades[cuenta] = principal
		}
	}
	// La cuenta configurada debe corresponder a una hoja nominal inventariada.
	for cuenta, ambito := range contextos {
		if cuenta == sesion.base.Contexto.Instantanea.CuentaRef {
			continue
		}
		principal, conocido := identidades[cuenta]
		if !conocido {
			return nil, cerrar, ct.ErrAutorizacionDenegada
		}
		semilla, e := nuevoContextoSinteticoContratacionTemporalDesarrollo(principal, sesion.reloj.Ahora())
		if e != nil {
			return nil, cerrar, e
		}
		esperado := semilla.Resultado.Contexto
		registrado, e := sesion.resolutor.ResolverContextoActorRegistradoV2(ctx, core.SolicitudContextoActor{
			Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}, PerfilActivoRef: ambito.PerfilActivoRef})
		if e != nil || registrado.Validar() != nil {
			return nil, cerrar, ct.ErrConsultaRRHHNoDisponible
		}
		actor := registrado.Contexto
		if actor.Principal.ID != esperado.Principal.ID || actor.PersonaRef != esperado.PersonaRef || actor.Instantanea.CuentaRef != esperado.Instantanea.CuentaRef || actor.Instantanea.CuentaRef != cuenta || actor.PerfilActivoRef != ambito.PerfilActivoRef || actor.Instantanea.PerfilVersion != ambito.PerfilVersion {
			return nil, cerrar, ct.ErrAutorizacionDenegada
		}
		soporte := &soporteAltaContratacionTemporalDesarrollo{referenciaCT: sesion.soporte.referenciaCT, sello: sesion.soporte.sello, principalID: principal.ID,
			certificadoSHA256: principal.Attributes["certificate_sha256"], contextoEsperadoRegistrado: registrado, reloj: sesion.soporte.reloj}
		base, e := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, sesion.registro, sesion.revalidador, sesion.reloj, sesion.resolutor, sesion.fronteras)
		if e != nil {
			return nil, cerrar, e
		}
		sesiones = append(sesiones, base)
		base.fijarVinculoCorporativo(sesion.vinculoCorporativo, nil)
		base.organizacionHistoricaPorCuenta = map[string]seleccionPerfilOrganizacionHistorica{cuenta: {perfil: ambito.PerfilActivoRef, version: ambito.PerfilVersion}}
		s, e := nuevaSeguridadComunDesarrollo(base, base.reloj)
		if e != nil {
			return nil, cerrar, e
		}
		if _, duplicada := porHoja[base.soporte.certificadoSHA256]; duplicada {
			return nil, cerrar, ct.ErrAutorizacionDenegada
		}
		porHoja[base.soporte.certificadoSHA256] = s
	}
	return porHoja, cerrar, nil
}

// El despachador selecciona únicamente una sesión nominal de la hoja que la
// frontera mTLS ha sellado; no hay retorno a la técnica original.
type sesionesOrganizacionHistoricaPorHoja map[string]*seguridadComunDesarrollo

func (s sesionesOrganizacionHistoricaPorHoja) ResolverContexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	if ctx == nil {
		return contextoSeguridadComunDesarrollo{}, ct.ErrAutorizacionDenegada
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok {
		return contextoSeguridadComunDesarrollo{}, ct.ErrAutorizacionDenegada
	}
	sesion := s[capacidad.principal.Attributes["certificate_sha256"]]
	if sesion == nil {
		return contextoSeguridadComunDesarrollo{}, ct.ErrAutorizacionDenegada
	}
	return sesion.ResolverContexto(ctx)
}
