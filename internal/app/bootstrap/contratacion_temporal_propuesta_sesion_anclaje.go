package bootstrap

import (
	"context"

	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type claveSeleccionSesionNominalIncorporacion struct{}

// La ligadura nace en resolver, después de comprobar la captura nominal.
// No se transporta por HTTP ni modifica el descriptor público de propuesta.
type seleccionSesionNominalIncorporacion struct {
	propietario *perfilesNominalesIncorporacion
	perfil      *perfilFijoCTDesarrollo
	captura     *capturaContextosNominalesIncorporacion
}

func (p *proveedorSesionConsultaRRHHDesarrollo) perfilAnclajePropuestaLigado(ctx context.Context, perfil string) bool {
	if p == nil || ctx == nil || p.soporte == nil || p.anclajePropuesta == nil ||
		p.anclajePropuesta.soporte != p.soporte {
		return false
	}
	ligadura, ligada := ctx.Value(claveSeleccionSesionNominalIncorporacion{}).(seleccionSesionNominalIncorporacion)
	captura, capturada := ctx.Value(claveContextosNominalesIncorporacion{}).(*capturaContextosNominalesIncorporacion)
	bolsa := p.anclajePropuesta.b2[bp.AccionConsultaAnclajeAceptacionNativaCT]
	if !ligada || !capturada || captura == nil || bolsa == nil ||
		ligadura.propietario != p.anclajePropuesta || ligadura.perfil != bolsa ||
		ligadura.captura != captura || perfil != bolsa.perfilRef() {
		return false
	}
	// Reutiliza la guarda de la autoridad B113: ruta real CT, sello,
	// expediente fiscalizado y coordenadas de la selección de esta petición.
	autoridad := autoridadIncorporacionPersonalB2{perfiles: p.anclajePropuesta}
	return autoridad.anclajeNativoPropuestaPermitido(ctx, bp.AccionConsultaAnclajeAceptacionNativaCT)
}
