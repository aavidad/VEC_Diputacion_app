package administracion

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	gobierno "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasGobiernoRolNuevoV3 llegan de la composición ADMIN protegida.
// La fuente del catálogo tiene su propio LOGIN de lectura AUT58; el pool de
// gobierno sólo ejecuta las dos fachadas AUT60. El actor se resuelve en la
// sesión ADMIN de cada petición y jamás se configura aquí.
type DependenciasGobiernoRolNuevoV3 struct {
	PoolGobierno    *pgxpool.Pool
	FuenteCatalogo  ports.FuenteCatalogoAccionesAdministracionV1
	CatalogoRoles   ports.CatalogoRolesAdministrables
	ActosExistentes ports.AutoridadActosAdministracionPerfiles
	Confianza       ConfiguracionConfianzaPerfilesV3
	DependenciasPDP DependenciasConfianzaPerfilesV3
	Motivos         map[string]domain.ReferenciaEntradaCatalogo
	Reloj           ports.Reloj
}

// NuevoServicioGobiernoRolNuevoV3 es el punto de composición nominal para el
// handler ADMIN. No instala SQL, no publica roles y no sustituye la autoridad
// AUT24: la envuelve con un puerto Gov que usa otro LOGIN y otra ACL.
func NuevoServicioGobiernoRolNuevoV3(ctx context.Context,
	d DependenciasGobiernoRolNuevoV3) (*application.ServicioAdministracionPerfiles, error) {
	if ctx == nil || d.PoolGobierno == nil || dependenciaConfianzaPerfilesNula(d.FuenteCatalogo) ||
		dependenciaConfianzaPerfilesNula(d.CatalogoRoles) || dependenciaConfianzaPerfilesNula(d.ActosExistentes) ||
		dependenciaConfianzaPerfilesNula(d.Reloj) || d.PoolGobierno == d.DependenciasPDP.PoolFuente ||
		d.PoolGobierno == d.DependenciasPDP.PoolRegistro || d.PoolGobierno == d.DependenciasPDP.PoolMotivos {
		return nil, ErrConfiguracion
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	confianza, err := NuevaConfianzaGobiernoRolNuevoV3(d.Confianza, d.DependenciasPDP)
	if err != nil {
		return nil, ErrConfiguracion
	}
	emisor, err := NuevoEmisorGobiernoRolNuevoV3(confianza.Emisores, d.Motivos, d.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	autoridad, err := gobierno.NuevaAutoridadGobiernoRolNuevo(ctx, d.PoolGobierno, d.FuenteCatalogo, emisor, d.Reloj)
	if err != nil {
		return nil, err
	}
	actos, err := gobierno.NuevaAutoridadPerfilesConGobierno(d.ActosExistentes, autoridad)
	if err != nil {
		return nil, err
	}
	return application.NuevoServicioAdministracionPerfiles(d.CatalogoRoles, actos, d.Reloj)
}
