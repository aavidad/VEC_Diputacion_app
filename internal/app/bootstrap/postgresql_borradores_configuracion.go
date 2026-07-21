package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	rolEjecutorConsultaPostgreSQLBorradores  = "vec_bolsa_convocatorias_ejecutor_consulta"
	rolProyectorGobiernoPostgreSQLBorradores = "vec_bolsa_convocatorias_proyector_gobierno"
	rolVerificadorReciboPostgreSQLBorradores = "vec_bolsa_convocatorias_verificador_recibo"

	maxConexionesEjecutorPostgreSQLBorradores    = int32(4)
	maxConexionesProyectorPostgreSQLBorradores   = int32(2)
	maxConexionesVerificadorPostgreSQLBorradores = int32(2)
	duracionConexionPostgreSQLBorradores         = 5 * time.Second
	duracionSondaPostgreSQLBorradores            = 5 * time.Second
	duracionVidaPostgreSQLBorradores             = 30 * time.Minute
	duracionJitterPostgreSQLBorradores           = 3 * time.Minute
	duracionInactividadPostgreSQLBorradores      = 5 * time.Minute
	periodoSaludPostgreSQLBorradores             = 30 * time.Second
)

var (
	ErrConfiguracionPoolPostgreSQLBorradoresInvalida = errors.New(
		"bootstrap: configuracion de pools PostgreSQL de borradores invalida",
	)
	ErrConexionPostgreSQLBorradoresNoDisponible = errors.New(
		"bootstrap: conexion PostgreSQL de borradores no disponible",
	)
	ErrIdentidadPostgreSQLBorradoresInvalida = errors.New(
		"bootstrap: identidad PostgreSQL de borradores invalida",
	)
)

type perfilPoolPostgreSQLBorradores struct {
	rolEsperado   string
	aplicacion    string
	maxConexiones int32
	soloLectura   bool
}

var perfilesPoolPostgreSQLBorradores = [3]perfilPoolPostgreSQLBorradores{
	{
		rolEsperado:   rolEjecutorConsultaPostgreSQLBorradores,
		aplicacion:    "vec-bolsa-borradores-ejecutor-consulta",
		maxConexiones: maxConexionesEjecutorPostgreSQLBorradores,
		// listar_borradores_v1 y obtener_borrador_v1 son consultas gobernadas:
		// registran consumo, auditoria y cursor dentro de la misma transaccion.
		// El rol nominal limita el efecto; la sesion no puede ser read-only.
		soloLectura: false,
	},
	{
		rolEsperado:   rolProyectorGobiernoPostgreSQLBorradores,
		aplicacion:    "vec-bolsa-borradores-proyector-gobierno",
		maxConexiones: maxConexionesProyectorPostgreSQLBorradores,
	},
	{
		rolEsperado:   rolVerificadorReciboPostgreSQLBorradores,
		aplicacion:    "vec-bolsa-borradores-verificador-recibo",
		maxConexiones: maxConexionesVerificadorPostgreSQLBorradores,
		soloLectura:   true,
	},
}

// La consulta replica el cierre nominal de las funciones SQL existentes: la
// cuenta LOGIN debe ser austera, pertenecer directamente a un unico rol
// runtime y ese rol no puede heredar a su vez otra membresia. Se ejecuta sin
// SECURITY DEFINER, por lo que session_user y current_user deben coincidir.
const consultaIdentidadPoolPostgreSQLBorradores = `
SELECT session_user::text, current_user::text, COALESCE((
    SELECT identidad.rolcanlogin
       AND identidad.rolinherit
       AND NOT identidad.rolsuper
       AND NOT identidad.rolcreatedb
       AND NOT identidad.rolcreaterole
       AND NOT identidad.rolreplication
       AND NOT identidad.rolbypassrls
       AND NOT objetivo.rolcanlogin
       AND objetivo.rolinherit
       AND NOT objetivo.rolsuper
       AND NOT objetivo.rolcreatedb
       AND NOT objetivo.rolcreaterole
       AND NOT objetivo.rolreplication
       AND NOT objetivo.rolbypassrls
       AND (
           SELECT count(*) = 1 AND COALESCE(bool_and(
                      membresia.roleid = objetivo.oid
                      AND membresia.admin_option IS FALSE
                      AND membresia.inherit_option IS TRUE
                      AND membresia.set_option IS TRUE
                  ), false)
             FROM pg_catalog.pg_auth_members AS membresia
            WHERE membresia.member = identidad.oid
       )
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_auth_members AS membresia_objetivo
            WHERE membresia_objetivo.member = objetivo.oid
       )
      FROM pg_catalog.pg_roles AS identidad
      CROSS JOIN pg_catalog.pg_roles AS objetivo
     WHERE identidad.rolname = session_user
       AND objetivo.rolname = $1
), false)`

type consultadorFilaPostgreSQLBorradores interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func prepararConfiguracionPoolPostgreSQLBorradores(
	dsn string,
	perfil perfilPoolPostgreSQLBorradores,
) (*pgxpool.Config, error) {
	if !perfilPoolPostgreSQLBorradoresValido(perfil) {
		return nil, ErrConfiguracionPoolPostgreSQLBorradoresInvalida
	}
	configuracion, err := pgxpool.ParseConfig(dsn)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil {
		return nil, ErrConfiguracionPoolPostgreSQLBorradoresInvalida
	}
	configuracion.MaxConns = perfil.maxConexiones
	configuracion.MinConns = 0
	configuracion.MinIdleConns = 0
	configuracion.MaxConnLifetime = duracionVidaPostgreSQLBorradores
	configuracion.MaxConnLifetimeJitter = duracionJitterPostgreSQLBorradores
	configuracion.MaxConnIdleTime = duracionInactividadPostgreSQLBorradores
	configuracion.HealthCheckPeriod = periodoSaludPostgreSQLBorradores
	configuracion.PingTimeout = duracionSondaPostgreSQLBorradores
	configuracion.ConnConfig.ConnectTimeout = duracionConexionPostgreSQLBorradores
	if configuracion.ConnConfig.RuntimeParams == nil {
		configuracion.ConnConfig.RuntimeParams = make(map[string]string)
	}
	parametros := configuracion.ConnConfig.RuntimeParams
	parametros["application_name"] = perfil.aplicacion
	parametros["timezone"] = "UTC"
	parametros["search_path"] = "pg_catalog,pg_temp"
	parametros["default_transaction_isolation"] = "serializable"
	parametros["statement_timeout"] = "15s"
	parametros["lock_timeout"] = "3s"
	parametros["idle_in_transaction_session_timeout"] = "15s"
	if perfil.soloLectura {
		parametros["default_transaction_read_only"] = "on"
	} else {
		parametros["default_transaction_read_only"] = "off"
	}
	configuracion.AfterConnect = func(ctx context.Context, conexion *pgx.Conn) error {
		_, err := comprobarIdentidadPoolPostgreSQLBorradores(
			ctx, conexion, perfil.rolEsperado,
		)
		return err
	}
	return configuracion, nil
}

func perfilPoolPostgreSQLBorradoresValido(perfil perfilPoolPostgreSQLBorradores) bool {
	if perfil.aplicacion == "" || perfil.maxConexiones < 1 {
		return false
	}
	switch perfil.rolEsperado {
	case rolEjecutorConsultaPostgreSQLBorradores,
		rolProyectorGobiernoPostgreSQLBorradores,
		rolVerificadorReciboPostgreSQLBorradores:
		return true
	default:
		return false
	}
}

func comprobarIdentidadPoolPostgreSQLBorradores(
	ctx context.Context,
	consultador consultadorFilaPostgreSQLBorradores,
	rolEsperado string,
) (string, error) {
	if ctx == nil || consultador == nil || !perfilPoolPostgreSQLBorradoresValido(
		perfilPoolPostgreSQLBorradores{rolEsperado: rolEsperado, aplicacion: "sonda", maxConexiones: 1},
	) {
		return "", ErrIdentidadPostgreSQLBorradoresInvalida
	}
	ctxSonda, cancelar := context.WithTimeout(ctx, duracionSondaPostgreSQLBorradores)
	defer cancelar()
	var usuarioSesion, usuarioEfectivo string
	var identidadValida bool
	err := consultador.QueryRow(
		ctxSonda, consultaIdentidadPoolPostgreSQLBorradores, rolEsperado,
	).Scan(&usuarioSesion, &usuarioEfectivo, &identidadValida)
	if err != nil || usuarioSesion == "" || usuarioSesion != usuarioEfectivo || !identidadValida {
		return "", errorPostgreSQLBorradoresCerrado(ctxSonda, ErrIdentidadPostgreSQLBorradoresInvalida)
	}
	return usuarioSesion, nil
}

func errorPostgreSQLBorradoresCerrado(ctx context.Context, base error) error {
	if ctx != nil && ctx.Err() != nil {
		return errors.Join(base, ctx.Err())
	}
	return base
}
