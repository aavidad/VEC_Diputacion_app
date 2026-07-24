package postgrespublico

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	rolLoginPublicadorPostgreSQLPublica = "vec_bolsa_publica_publicador_login"
	rolPublicadorPostgreSQLPublica      = "vec_bolsa_publica_publicador"
	aplicacionPublicadorPostgreSQL      = "vec-bolsa-publicador"
)

var ErrPublicacionPostgreSQLPublicaRechazada = errors.New("bolsa publica: publicacion PostgreSQL rechazada")

// PublicarProyeccion abre una conexion de un solo uso e invoca la frontera de
// escritura como una sentencia autocommit. No expone una transaccion al
// llamante: al retornar no puede quedar un advisory xact lock en esa sesion.
func PublicarProyeccion(
	ctx context.Context,
	dsn string,
	proyeccion []byte,
	anclaManifiestoSHA256 string,
) error {
	anclaManifiestoSHA256 = strings.TrimSpace(anclaManifiestoSHA256)
	if ctx == nil || ctx.Err() != nil || len(proyeccion) == 0 ||
		len(proyeccion) > 256*1024*1024 || !json.Valid(proyeccion) ||
		!patronHuellaPublicaPostgreSQL.MatchString(anclaManifiestoSHA256) ||
		anclaManifiestoSHA256 == strings.Repeat("0", 64) {
		return ErrPublicacionPostgreSQLPublicaRechazada
	}
	configuracion, err := prepararConfiguracionPublicador(dsn)
	if err != nil {
		return err
	}
	conexion, err := pgx.ConnectConfig(ctx, configuracion)
	if err != nil {
		return ErrPublicacionPostgreSQLPublicaRechazada
	}
	defer conexion.Close(context.Background())
	if err := comprobarIdentidadPublicador(ctx, conexion); err != nil {
		return err
	}
	if _, err := conexion.Exec(ctx, `
		SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v2(
			$1::jsonb, $2
		)`, string(proyeccion), anclaManifiestoSHA256); err != nil {
		return ErrPublicacionPostgreSQLPublicaRechazada
	}
	return nil
}

func prepararConfiguracionPublicador(dsn string) (*pgx.ConnConfig, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	configuracion, err := pgx.ParseConfig(strings.TrimSpace(dsn))
	if err != nil || configuracion == nil {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	if len(configuracion.RuntimeParams) != 0 {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	if err := validarTLSPostgreSQLPublico(&configuracion.Config); err != nil {
		return nil, err
	}
	configuracion.ConnectTimeout = duracionConexionPostgreSQLPublica
	if configuracion.RuntimeParams == nil {
		configuracion.RuntimeParams = make(map[string]string)
	}
	configuracion.RuntimeParams["application_name"] = aplicacionPublicadorPostgreSQL
	configuracion.RuntimeParams["search_path"] = "pg_catalog,pg_temp"
	configuracion.RuntimeParams["statement_timeout"] = "60s"
	configuracion.RuntimeParams["lock_timeout"] = "5s"
	configuracion.RuntimeParams["idle_in_transaction_session_timeout"] = "5s"
	configuracion.RuntimeParams["transaction_timeout"] = "2min"
	return configuracion, nil
}

const consultaIdentidadPublicadorPostgreSQL = `
SELECT session_user,
       current_user,
       session_user = $1
       AND current_user = session_user
       AND pg_catalog.pg_has_role(session_user, $2, 'MEMBER')
       AND pg_catalog.has_function_privilege(
           session_user,
           'vec_bolsa_publica_publicacion.publicar_proyeccion_v2(jsonb,text)',
           'EXECUTE'
       )
       AND pg_catalog.current_setting('application_name') = $3
       AND pg_catalog.replace(
           pg_catalog.current_setting('search_path'), ' ', ''
       ) = 'pg_catalog,pg_temp'
       AND pg_catalog.current_setting('statement_timeout')::interval
           = interval '60 seconds'
       AND pg_catalog.current_setting('lock_timeout')::interval
           = interval '5 seconds'
       AND pg_catalog.current_setting('idle_in_transaction_session_timeout')::interval
           = interval '5 seconds'
       AND pg_catalog.current_setting('transaction_timeout')::interval
           = interval '2 minutes'
       AND pg_catalog.current_setting('log_parameter_max_length_on_error') = '0'
       AND (
           SELECT COALESCE(identidad.rolconfig, ARRAY[]::text[]) @> ARRAY[
               'application_name=vec-bolsa-publicador',
               'search_path="pg_catalog,pg_temp"',
               'statement_timeout=60s',
               'lock_timeout=5s',
               'idle_in_transaction_session_timeout=5s',
               'transaction_timeout=2min',
               'log_parameter_max_length_on_error=0'
           ]
             FROM pg_catalog.pg_roles AS identidad
            WHERE identidad.rolname = session_user
       )`

func comprobarIdentidadPublicador(ctx context.Context, conexion *pgx.Conn) error {
	if ctx == nil || conexion == nil {
		return ErrIdentidadPostgreSQLPublicaInvalida
	}
	var usuarioSesion, usuarioEfectivo string
	var valida bool
	if err := conexion.QueryRow(
		ctx, consultaIdentidadPublicadorPostgreSQL,
		rolLoginPublicadorPostgreSQLPublica,
		rolPublicadorPostgreSQLPublica,
		aplicacionPublicadorPostgreSQL,
	).Scan(&usuarioSesion, &usuarioEfectivo, &valida); err != nil ||
		usuarioSesion != rolLoginPublicadorPostgreSQLPublica ||
		usuarioEfectivo != usuarioSesion || !valida {
		return ErrIdentidadPostgreSQLPublicaInvalida
	}
	return nil
}
