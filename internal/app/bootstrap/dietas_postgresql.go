package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

const (
	rolDietasBorradoresPostgreSQLDesarrollo = "vec_dietas_ejecutor"
	rolPersonalDietasPostgreSQLDesarrollo   = "vec_personal_ejecutor"
)

var (
	errPoolsPostgreSQLDietasDesarrolloInvalidos       = errors.New("bootstrap: pools PostgreSQL nominales de dietas invalidos")
	errConexionPostgreSQLDietasDesarrolloNoDisponible = errors.New("bootstrap: conexion PostgreSQL nominal de dietas no disponible")
	errIdentidadPostgreSQLDietasDesarrolloInvalida    = errors.New("bootstrap: identidad PostgreSQL nominal de dietas invalida")
)

type poolOperativoPostgreSQLDietasDesarrollo interface {
	Ping(context.Context) error
	QueryRow(context.Context, string, ...any) pgx.Row
	Close()
}

type fabricaPoolPostgreSQLDietasDesarrollo func(context.Context, *pgxpool.Config) (poolOperativoPostgreSQLDietasDesarrollo, error)

type perfilPoolPostgreSQLDietasDesarrollo struct {
	rol, aplicacion string
	maxConexiones   int32
	soloLectura     bool
}

var perfilesPoolPostgreSQLDietasDesarrollo = [2]perfilPoolPostgreSQLDietasDesarrollo{
	{rolDietasBorradoresPostgreSQLDesarrollo, "vec-dietas-borradores", 4, false},
	{rolPersonalDietasPostgreSQLDesarrollo, "vec-dietas-personal-relaciones", 2, true},
}

// poolsPostgreSQLDietasDesarrollo no retiene DSN. Cada getter entrega sólo el
// pool que necesita el adaptador propietario de esa capacidad.
type poolsPostgreSQLDietasDesarrollo struct {
	pools        [2]poolOperativoPostgreSQLDietasDesarrollo
	cerrarUnaVez sync.Once
}

func nuevosPoolsPostgreSQLDietasDesarrollo(ctx context.Context, cfg config.Config) (*poolsPostgreSQLDietasDesarrollo, error) {
	return nuevosPoolsPostgreSQLDietasDesarrolloConFabrica(ctx, cfg, crearPoolPostgreSQLDietasDesarrollo)
}

func nuevosPoolsPostgreSQLDietasDesarrolloConFabrica(ctx context.Context, cfg config.Config, crear fabricaPoolPostgreSQLDietasDesarrollo) (*poolsPostgreSQLDietasDesarrollo, error) {
	if ctx == nil || crear == nil {
		return nil, errPoolsPostgreSQLDietasDesarrolloInvalidos
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(errConexionPostgreSQLDietasDesarrolloNoDisponible, err)
	}
	dietas, personal, err := cfg.Normalize().DietasBorradoresPostgreSQL.DSNSeparados()
	if err != nil {
		return nil, err
	}
	dsns := [2]string{dietas, personal}
	resultado := &poolsPostgreSQLDietasDesarrollo{}
	usuarios := make(map[string]struct{}, len(dsns))
	var topologia topologiaPostgreSQLDietasDesarrollo
	for indice, perfil := range perfilesPoolPostgreSQLDietasDesarrollo {
		configuracion, err := prepararConfiguracionPoolPostgreSQLDietasDesarrollo(dsns[indice], perfil)
		if err != nil {
			resultado.Cerrar()
			return nil, err
		}
		pool, err := crear(ctx, configuracion)
		resultado.pools[indice] = pool
		if err != nil || poolPostgreSQLDietasDesarrolloNulo(pool) {
			resultado.Cerrar()
			return nil, errConexionPostgreSQLDietasDesarrolloNoDisponible
		}
		sonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
		err = pool.Ping(sonda)
		cancelar()
		if err != nil {
			resultado.Cerrar()
			return nil, errConexionPostgreSQLDietasDesarrolloNoDisponible
		}
		usuario, actual, err := acreditarPoolPostgreSQLDietasDesarrollo(ctx, pool, perfil.rol)
		if err != nil || usuario == "" || !actual.igual(actual) {
			resultado.Cerrar()
			return nil, errIdentidadPostgreSQLDietasDesarrolloInvalida
		}
		if _, repetido := usuarios[usuario]; repetido {
			resultado.Cerrar()
			return nil, errIdentidadPostgreSQLDietasDesarrolloInvalida
		}
		usuarios[usuario] = struct{}{}
		if indice == 0 {
			topologia = actual
		} else if !topologia.igual(actual) {
			resultado.Cerrar()
			return nil, errIdentidadPostgreSQLDietasDesarrolloInvalida
		}
	}
	return resultado, nil
}

func crearPoolPostgreSQLDietasDesarrollo(ctx context.Context, cfg *pgxpool.Config) (poolOperativoPostgreSQLDietasDesarrollo, error) {
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errConexionPostgreSQLDietasDesarrolloNoDisponible
	}
	return pool, nil
}

func prepararConfiguracionPoolPostgreSQLDietasDesarrollo(dsn string, perfil perfilPoolPostgreSQLDietasDesarrollo) (*pgxpool.Config, error) {
	if perfil.rol == "" || perfil.aplicacion == "" || perfil.maxConexiones < 1 {
		return nil, errPoolsPostgreSQLDietasDesarrolloInvalidos
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil || cfg == nil || cfg.ConnConfig == nil || validarTLSPostgreSQLBorradores(&cfg.ConnConfig.Config, true) != nil {
		return nil, errPoolsPostgreSQLDietasDesarrolloInvalidos
	}
	cfg.MaxConns, cfg.MinConns, cfg.MinIdleConns = perfil.maxConexiones, 0, 0
	cfg.MaxConnLifetime, cfg.MaxConnLifetimeJitter, cfg.MaxConnIdleTime = 30*time.Minute, 3*time.Minute, 5*time.Minute
	cfg.HealthCheckPeriod, cfg.PingTimeout, cfg.ConnConfig.ConnectTimeout = 30*time.Second, 5*time.Second, 5*time.Second
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	p := cfg.ConnConfig.RuntimeParams
	p["application_name"], p["timezone"], p["search_path"] = perfil.aplicacion, "UTC", "pg_catalog,pg_temp"
	p["default_transaction_isolation"], p["default_transaction_read_only"] = "serializable", "off"
	if perfil.soloLectura {
		p["default_transaction_read_only"] = "on"
	}
	p["statement_timeout"], p["lock_timeout"], p["idle_in_transaction_session_timeout"] = "15s", "3s", "15s"
	cfg.AfterConnect = func(ctx context.Context, conexion *pgx.Conn) error {
		_, _, err := acreditarPoolPostgreSQLDietasDesarrollo(ctx, conexion, perfil.rol)
		return err
	}
	return cfg, nil
}

type topologiaPostgreSQLDietasDesarrollo struct{ base, direccion, puerto, inicio string }

func (t topologiaPostgreSQLDietasDesarrollo) igual(otra topologiaPostgreSQLDietasDesarrollo) bool {
	return t.base != "" && t == otra
}

// La sonda acredita la conexión observada, no lo declarado por el DSN. Cierra
// SET ROLE, privilegios de servidor y cualquier membresía adicional directa o
// transitiva; el login sólo puede heredar el rol nominal sin poder asumirlo.
const consultaAcreditacionPoolPostgreSQLDietasDesarrollo = `
WITH RECURSIVE membresias(rol_id) AS (
 SELECT m.roleid FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
 UNION
 SELECT siguiente.roleid FROM pg_catalog.pg_auth_members siguiente JOIN membresias previa ON siguiente.member=previa.rol_id
)
SELECT session_user::text, current_user::text, current_database()::text,
       COALESCE(inet_server_addr()::text,'local')::text, COALESCE(inet_server_port()::text,'local')::text,
       pg_postmaster_start_time()::text,
       COALESCE((SELECT identidad.rolcanlogin AND identidad.rolinherit
          AND NOT identidad.rolsuper AND NOT identidad.rolbypassrls AND NOT identidad.rolcreaterole
          AND NOT identidad.rolcreatedb AND NOT identidad.rolreplication
          AND session_user=current_user
          AND (SELECT count(*)=1 AND bool_and(m.roleid=objetivo.oid AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option) FROM pg_catalog.pg_auth_members m WHERE m.member=identidad.oid)
          AND NOT EXISTS (SELECT 1 FROM membresias WHERE rol_id<>objetivo.oid)
          AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=objetivo.oid)
          AND NOT objetivo.rolcanlogin AND NOT objetivo.rolsuper AND NOT objetivo.rolbypassrls
          AND NOT objetivo.rolcreaterole AND NOT objetivo.rolcreatedb AND NOT objetivo.rolreplication
        FROM pg_catalog.pg_roles identidad CROSS JOIN pg_catalog.pg_roles objetivo
       WHERE identidad.rolname=session_user AND objetivo.rolname=$1),false)`

func acreditarPoolPostgreSQLDietasDesarrollo(ctx context.Context, consulta interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rol string) (string, topologiaPostgreSQLDietasDesarrollo, error) {
	if ctx == nil || consulta == nil || !rolPoolPostgreSQLDietasDesarrolloValido(rol) {
		return "", topologiaPostgreSQLDietasDesarrollo{}, errIdentidadPostgreSQLDietasDesarrolloInvalida
	}
	sonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	var usuario, efectivo string
	var topologia topologiaPostgreSQLDietasDesarrollo
	var valido bool
	err := consulta.QueryRow(sonda, consultaAcreditacionPoolPostgreSQLDietasDesarrollo, rol).Scan(&usuario, &efectivo, &topologia.base, &topologia.direccion, &topologia.puerto, &topologia.inicio, &valido)
	if err != nil || usuario == "" || efectivo == "" || usuario != efectivo || !valido || !topologia.igual(topologia) {
		return "", topologiaPostgreSQLDietasDesarrollo{}, errIdentidadPostgreSQLDietasDesarrolloInvalida
	}
	return usuario, topologia, nil
}

func rolPoolPostgreSQLDietasDesarrolloValido(rol string) bool {
	for _, p := range perfilesPoolPostgreSQLDietasDesarrollo {
		if p.rol == rol {
			return true
		}
	}
	return false
}
func poolPostgreSQLDietasDesarrolloNulo(pool any) bool {
	if pool == nil {
		return true
	}
	v := reflect.ValueOf(pool)
	return v.Kind() == reflect.Pointer && v.IsNil()
}

func (p *poolsPostgreSQLDietasDesarrollo) pool(indice int) *pgxpool.Pool {
	if p == nil {
		return nil
	}
	pool, _ := p.pools[indice].(*pgxpool.Pool)
	return pool
}
func (p *poolsPostgreSQLDietasDesarrollo) Dietas() *pgxpool.Pool   { return p.pool(0) }
func (p *poolsPostgreSQLDietasDesarrollo) Personal() *pgxpool.Pool { return p.pool(1) }
func (p *poolsPostgreSQLDietasDesarrollo) Cerrar() {
	if p != nil {
		p.cerrarUnaVez.Do(func() {
			for i := len(p.pools) - 1; i >= 0; i-- {
				if !poolPostgreSQLDietasDesarrolloNulo(p.pools[i]) {
					p.pools[i].Close()
				}
			}
		})
	}
}
