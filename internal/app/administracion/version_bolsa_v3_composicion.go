package administracion

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	gobierno "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasVersionarRolBolsaV3 llegan de la composición ADMIN protegida.
// La fuente del catálogo tiene su propio LOGIN de lectura AUT58; el pool de
// gobierno sólo ejecuta las dos fachadas AUT63. El actor se resuelve en la
// sesión ADMIN de cada petición y jamás se configura aquí.
type DependenciasVersionarRolBolsaV3 struct {
	PoolGobierno    *pgxpool.Pool
	FuenteCatalogo  ports.FuenteCatalogoAccionesAdministracionV1
	Confianza       ConfiguracionConfianzaPerfilesV3
	DependenciasPDP DependenciasConfianzaPerfilesV3
	Motivos         map[string]domain.ReferenciaEntradaCatalogo
	Reloj           ports.Reloj
}

// NuevoServicioVersionarRolBolsaV3 es el punto de composición nominal para el
// handler ADMIN. No instala SQL, no publica roles y no sustituye la autoridad
// AUT24: la envuelve con un puerto Gov que usa otro LOGIN y otra ACL.
func NuevoServicioVersionarRolBolsaV3(ctx context.Context,
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
	autoridad, err := gobierno.NuevaAutoridadVersionarRolBolsa(ctx, d.PoolGobierno, d.FuenteCatalogo, emisor, d.Reloj)
	if err != nil {
		return nil, err
	}
	return application.NuevoServicioAdministracionPerfiles(autoridad, autoridad, d.Reloj)
}
