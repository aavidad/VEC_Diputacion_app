package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	calendarioshttp "vec-diputacion-granada/internal/modules/calendarios/adapters/httpinterno"
	calendariospg "vec-diputacion-granada/internal/modules/calendarios/adapters/postgres"
	calendariosapp "vec-diputacion-granada/internal/modules/calendarios/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

const rolLectorCalendariosDesarrollo = "vec_calendarios_lector"

var errCalendariosDesarrolloNoDisponible = errors.New("bootstrap: calendarios no disponible")

type relojCalendariosDesarrollo struct{}

func (relojCalendariosDesarrollo) Ahora() time.Time { return time.Now().UTC() }

// rutaCalendariosDesarrollo enumera las rutas de consulta de Calendarios que
// la frontera mTLS protege como lectura de RRHH.
func rutaCalendariosDesarrollo(ruta string) bool {
	for _, r := range calendarioshttp.Rutas() {
		if r == ruta {
			return true
		}
	}
	return false
}

// nuevasRutasCalendariosDesarrollo compone la lectura de Calendarios. Sin
// conexión configurada las rutas existen y responden 503; una conexión
// presente pero inválida o con otro rol impide arrancar.
func nuevasRutasCalendariosDesarrollo(cfg config.Config) ([]vechttp.RutaExacta, func(), error) {
	cerrar := func() {}
	dsn, err := cfg.DSNCalendarios()
	if err != nil {
		return nil, cerrar, err
	}
	var consulta *calendariosapp.Servicio
	if dsn != "" {
		if !cfg.DevelopmentEnabledByDoubleKey() {
			return nil, cerrar, errCalendariosDesarrolloNoDisponible
		}
		ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		pool, err := abrirPoolCalendariosDesarrollo(ctx, dsn)
		if err != nil {
			return nil, cerrar, err
		}
		repo, err := calendariospg.NuevoRepositorio(pool)
		if err == nil {
			consulta, err = calendariosapp.NuevoServicio(repo, relojCalendariosDesarrollo{})
		}
		if err != nil {
			pool.Close()
			return nil, cerrar, errCalendariosDesarrolloNoDisponible
		}
		cerrar = pool.Close
	}
	manejador := calendarioshttp.NuevoManejador(nil)
	if consulta != nil {
		manejador = calendarioshttp.NuevoManejador(consulta)
	}
	rutas := make([]vechttp.RutaExacta, 0, 3)
	for _, r := range calendarioshttp.Rutas() {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: r, Manejador: manejador})
	}
	return rutas, cerrar, nil
}

// abrirPoolCalendariosDesarrollo acredita en cada conexión que el login solo
// hereda vec_calendarios_lector, sin superusuario ni BYPASSRLS, y fija
// transacciones de solo lectura.
func abrirPoolCalendariosDesarrollo(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil || cfg == nil || cfg.ConnConfig == nil || validarTLSPostgreSQLBorradores(&cfg.ConnConfig.Config, true) != nil {
		return nil, errCalendariosDesarrolloNoDisponible
	}
	cfg.MaxConns, cfg.MinConns = 4, 0
	cfg.MaxConnLifetime, cfg.MaxConnIdleTime = 30*time.Minute, 5*time.Minute
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	p := cfg.ConnConfig.RuntimeParams
	p["application_name"], p["timezone"], p["search_path"] = "vec-calendarios-lectura", "UTC", "pg_catalog,pg_temp"
	p["default_transaction_read_only"], p["statement_timeout"], p["lock_timeout"] = "on", "10s", "2s"
	p["idle_in_transaction_session_timeout"] = "15s"
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error { return acreditarLectorCalendarios(ctx, c) }
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errCalendariosDesarrolloNoDisponible
	}
	conexion, err := pool.Acquire(ctx)
	if err != nil {
		pool.Close()
		return nil, errCalendariosDesarrolloNoDisponible
	}
	conexion.Release()
	return pool, nil
}

func acreditarLectorCalendarios(ctx context.Context, c *pgx.Conn) error {
	sonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	var usuario, efectivo, base, direccion, puerto, inicio string
	var valido bool
	err := c.QueryRow(sonda, consultaAcreditacionPoolPostgreSQLDietasDesarrollo, rolLectorCalendariosDesarrollo).
		Scan(&usuario, &efectivo, &base, &direccion, &puerto, &inicio, &valido)
	if err != nil || !valido || usuario == "" || usuario != efectivo {
		return errCalendariosDesarrolloNoDisponible
	}
	return nil
}
