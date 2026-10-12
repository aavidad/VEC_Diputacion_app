package administracion

import (
	"context"

	gobierno "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/application"
)

// NuevoServicioVersionarRolBolsaB103V3 reutiliza la confianza y el emisor
// nominales B1 con materiales privados B103; AUT75 v2 acredita su LOGIN propio.
func NuevoServicioVersionarRolBolsaB103V3(ctx context.Context,
	d DependenciasVersionarRolBolsaV3) (*application.ServicioAdministracionPerfiles, error) {
	if ctx == nil || d.PoolGobierno == nil || dependenciaConfianzaPerfilesNula(d.FuenteCatalogo) ||
		dependenciaConfianzaPerfilesNula(d.Reloj) || d.PoolGobierno == d.DependenciasPDP.PoolFuente ||
		d.PoolGobierno == d.DependenciasPDP.PoolRegistro || d.PoolGobierno == d.DependenciasPDP.PoolMotivos {
		return nil, ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	confianza, err := NuevaConfianzaVersionarRolBolsaV3(d.Confianza, d.DependenciasPDP)
	if err != nil {
		return nil, ErrConfiguracion
	}
	emisor, err := NuevoEmisorVersionarRolBolsaV3(confianza.Emisores, d.Motivos, d.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	autoridad, err := gobierno.NuevaAutoridadVersionarRolBolsaB103(ctx, d.PoolGobierno, d.FuenteCatalogo, emisor, d.Reloj)
	if err != nil {
		return nil, err
	}
	return application.NuevoServicioAdministracionPerfiles(autoridad, autoridad, d.Reloj)
}
