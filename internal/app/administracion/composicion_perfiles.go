package administracion

import (
	"context"
	"io/fs"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasComposicionPerfiles entrega infraestructura concreta ya abierta
// por el proceso ADMIN. Ninguna petición puede reemplazar esas dependencias.
type DependenciasComposicionPerfiles struct {
	Confianza                                                          ConfianzaPerfilesV3
	PoolCuentas, PoolRegistroSesion, PoolRevalidacionSesion, PoolActos *pgxpool.Pool
	Seudonimizador                                                     identidad.SeudonimizadorAlta
	EspacioIdentidad, DominioHMACRef                                   string
	MotivosLectura                                                     map[string]domain.ReferenciaEntradaCatalogo
	Auditor                                                            api.AuditorFrontera
	Reloj                                                              ports.Reloj
	Activos                                                            fs.FS
}

func ComponerServidorPerfiles(ctx context.Context, cfg Configuracion, deps DependenciasComposicionPerfiles) (*http.Server, error) {
	if ctx == nil || ctx.Err() != nil || deps.Reloj == nil || deps.PoolCuentas == nil || deps.PoolRegistroSesion == nil || deps.PoolRevalidacionSesion == nil || deps.PoolActos == nil || deps.Auditor == nil || deps.Activos == nil || deps.Confianza.Fuente == nil {
		return nil, ErrConfiguracion
	}
	registro, err := identidad.NuevoRegistroSesionesPostgreSQL(ctx, deps.PoolRegistroSesion, deps.PoolRevalidacionSesion, deps.Seudonimizador, deps.EspacioIdentidad, deps.DominioHMACRef)
	if err != nil {
		return nil, ErrConfiguracion
	}
	revalidador, err := identidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, deps.PoolRevalidacionSesion)
	if err != nil {
		return nil, ErrConfiguracion
	}
	cuentas, err := adminperfiles.NuevoPostgreSQL(ctx, deps.PoolCuentas, deps.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	contextos, err := adminperfiles.NuevoContextoRegistradoPostgreSQL(ctx, deps.PoolCuentas, deps.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	sesiones, err := NuevoResolverSesionPerfiles(cfg, adminperfiles.Dependencias{Cuentas: cuentas, Registro: registro, Revalidador: revalidador, Contextos: contextos, Autorizacion: deps.Confianza.Fuente, Reloj: deps.Reloj})
	if err != nil {
		return nil, ErrConfiguracion
	}
	contextoConexion, err := NuevoContextoConexionPerfiles(deps.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	recursos, err := pg.NuevaFuenteRecursos(deps.PoolActos)
	if err != nil {
		return nil, ErrConfiguracion
	}
	emisor, err := NuevoEmisorPerfiles(deps.Confianza.Emisores, recursos.ResolverRecursoAdministracionPerfiles, deps.MotivosLectura, deps.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	actos, err := pg.Nueva(ctx, deps.PoolActos, emisor, deps.Reloj)
	if err != nil {
		return nil, ErrConfiguracion
	}
	lecturas, err := pg.NuevaFuenteLecturas(actos, deps.Confianza.Fuente)
	if err != nil {
		return nil, ErrConfiguracion
	}
	return NuevoServidorConPerfiles(cfg, DependenciasPerfiles{ContextoConexion: contextoConexion, Sesiones: sesiones, Lecturas: lecturas, Catalogo: actos, Actos: actos, Auditor: deps.Auditor, Reloj: deps.Reloj, Activos: deps.Activos})
}
