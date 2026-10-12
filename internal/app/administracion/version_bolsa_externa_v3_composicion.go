package administracion

import (
	"context"

	gobierno "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/application"
)

// El servicio externo conserva el PDP, el catálogo y la sesión ADMIN; su
// LOGIN sólo ejecuta las fachadas AUT77 y se acredita al construirlo.
func NuevoServicioVersionarRolBolsaExternaV3(ctx context.Context,
	d DependenciasVersionarRolBolsaV3) (*application.ServicioAdministracionPerfiles, error) {
	if ctx == nil || d.PoolGobierno == nil || dependenciaConfianzaPerfilesNula(d.FuenteCatalogo) ||
		dependenciaConfianzaPerfilesNula(d.Reloj) || d.PoolGobierno == d.DependenciasPDP.PoolFuente ||
		d.PoolGobierno == d.DependenciasPDP.PoolRegistro || d.PoolGobierno == d.DependenciasPDP.PoolMotivos {
		return nil, ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	confianza, err := NuevaConfianzaVersionarRolBolsaExternaV3(d.Confianza, d.DependenciasPDP)
	if err != nil {
		return nil, ErrConfiguracion
	}
	emisor, err := NuevoEmisorVersionarRolBolsaExternaV3(confianza.Emisores, d.Motivos, d.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	autoridad, err := gobierno.NuevaAutoridadVersionarRolBolsaExterna(ctx, d.PoolGobierno, d.FuenteCatalogo, emisor, d.Reloj)
	if err != nil {
		return nil, err
	}
	return application.NuevoServicioVersionarRolBolsaExterna(autoridad, d.Reloj)
}
