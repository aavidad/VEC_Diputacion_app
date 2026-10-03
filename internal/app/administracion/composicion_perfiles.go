package administracion

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
	"net/http"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	identidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/ports"
)

// El único compositor recibe la fuente de lectura ya autorizada. No abre
// autoridad de actos ni requiere su pool. Fuentes ausentes conservan 503.
type DependenciasComposicionPerfiles struct {
	Confianza                                               ConfianzaPerfilesV3
	PoolCuentas, PoolRegistroSesion, PoolRevalidacionSesion *pgxpool.Pool
	Seudonimizador                                          identidad.SeudonimizadorAlta
	EspacioIdentidad, DominioHMACRef                        string
	Lecturas                                                api.FuenteLecturas
	FuenteSeleccion                                         FuenteSeleccionAuditadaADMIN
	Auditor                                                 api.AuditorFrontera
	Reloj                                                   ports.Reloj
	Activos                                                 fs.FS
}

func ComponerServidorPerfiles(ctx context.Context, cfg Configuracion, deps DependenciasComposicionPerfiles) (*http.Server, error) {
	if ctx == nil || ctx.Err() != nil || deps.Reloj == nil || deps.PoolCuentas == nil || deps.PoolRegistroSesion == nil || deps.PoolRevalidacionSesion == nil || deps.Auditor == nil || deps.Activos == nil || deps.Confianza.Fuente == nil {
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
	return NuevoServidorConLecturas(cfg, DependenciasPerfiles{ContextoConexion: contextoConexion, Sesiones: sesiones, Lecturas: deps.Lecturas, Auditor: deps.Auditor, Reloj: deps.Reloj, Activos: deps.Activos,
		ObservadorSelector: sesiones, FuenteSeleccion: deps.FuenteSeleccion, AudienciaSelector: cfg.Audiencia})
}
