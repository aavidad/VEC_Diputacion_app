// Package postgrespublico implementa la proyeccion autoritativa de solo
// lectura que puede conocer el proceso exterior. No importa adaptadores ni
// esquemas internos de RRHH.
package postgrespublico

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

const (
	rolConsultaPostgreSQLPublica = "vec_bolsa_publica_consulta"
	aplicacionPostgreSQLPublica  = "vec-bolsa-publica"

	duracionConexionPostgreSQLPublica    = 5 * time.Second
	duracionSondaPostgreSQLPublica       = 5 * time.Second
	duracionVidaPostgreSQLPublica        = 30 * time.Minute
	duracionInactividadPostgreSQLPublica = 5 * time.Minute
	periodoSaludPostgreSQLPublica        = 30 * time.Second
)

var (
	ErrConfiguracionPostgreSQLPublicaInvalida = errors.New("bolsa publica: configuracion PostgreSQL invalida")
	ErrTLSPostgreSQLPublicaInseguro           = errors.New("bolsa publica: TLS PostgreSQL no verifica la identidad del servidor")
	ErrIdentidadPostgreSQLPublicaInvalida     = errors.New("bolsa publica: identidad PostgreSQL no autorizada")
	ErrPostgreSQLPublicoNoDisponible          = errors.New("bolsa publica: proyeccion PostgreSQL no disponible")
	ErrDatosPostgreSQLPublicosNoConfiables    = errors.New("bolsa publica: proyeccion PostgreSQL no confiable")

	patronIDCatalogoPublicoPostgreSQL = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,127}$`)
	patronHuellaPublicaPostgreSQL     = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronIdentificadorPostgreSQL     = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
)

var (
	_ puertosbolsa.ConsultaConvocatoriasPublicas       = (*Fuente)(nil)
	_ puertosbolsa.ConsultaCategoriasPublicas          = (*Fuente)(nil)
	_ puertosbolsa.ConsultaSnapshotsCategoriasPublicas = (*Fuente)(nil)
	_ puertosbolsa.ConsultaPublicaConsistente          = (*Fuente)(nil)
)

// Fuente comparte una unica instantanea PostgreSQL de lectura entre los dos
// puertos publicos. El catalogo profesional queda fijado al construirla.
type Fuente struct {
	pool                          *pgxpool.Pool
	manifiestoSHA256              string
	catalogoCategorias            string
	versionCategorias             int
	huellaCategoriasGobernadaHex  string
	huellaProyeccionCategoriasHex string
	cacheManifiesto               atomic.Pointer[cacheManifiestoPublico]
	disponibilidadMu              sync.Mutex
	disponibilidadHasta           time.Time
	disponibilidadErr             error
	disponibilidadCancelada       bool
	disponibilidadEnCurso         bool
	disponibilidadEstadoConocido  bool
	disponibilidadDisponible      bool
	observadorDisponibilidad      func(bool)
	sondaDisponibilidadPrueba     func(context.Context) error
	integridadMu                  sync.RWMutex
	integridadHasta               time.Time
	integridadErr                 error
	integridadCancelar            context.CancelFunc
	integridadTerminada           chan struct{}
	sondaIntegridadPrueba         func(context.Context) error
	cerrarUnaVez                  sync.Once
}

// Abrir analiza el DSN efectivo, exige verify-full en todos los fallbacks,
// abre el pool y reidentifica la cuenta LOGIN antes de entregar la fuente.
func Abrir(
	ctx context.Context,
	dsn string,
	catalogoCategorias string,
	versionCategorias int,
	huellaCategoriasGobernadaHex string,
	huellaProyeccionCategoriasHex string,
	manifiestoSHA256 string,
) (*Fuente, error) {
	if ctx == nil {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(ErrPostgreSQLPublicoNoDisponible, err)
	}
	configuracion, err := prepararConfiguracionPool(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		return nil, ErrPostgreSQLPublicoNoDisponible
	}
	fuente, err := NuevaFuente(
		pool, catalogoCategorias, versionCategorias,
		huellaCategoriasGobernadaHex, huellaProyeccionCategoriasHex, manifiestoSHA256,
	)
	if err != nil {
		pool.Close()
		return nil, err
	}
	ctxSonda, cancelar := context.WithTimeout(ctx, duracionSondaPostgreSQLPublica)
	defer cancelar()
	if err := pool.Ping(ctxSonda); err != nil {
		pool.Close()
		return nil, errorPostgreSQLPublico(ctxSonda, err)
	}
	if err := comprobarIdentidad(ctxSonda, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return fuente, nil
}

func NuevaFuente(
	pool *pgxpool.Pool,
	catalogoCategorias string,
	versionCategorias int,
	huellaCategoriasGobernadaHex string,
	huellaProyeccionCategoriasHex string,
	manifiestoSHA256 string,
) (*Fuente, error) {
	catalogoCategorias = strings.TrimSpace(catalogoCategorias)
	huellaCategoriasGobernadaHex = strings.TrimSpace(huellaCategoriasGobernadaHex)
	huellaProyeccionCategoriasHex = strings.TrimSpace(huellaProyeccionCategoriasHex)
	manifiestoSHA256 = strings.TrimSpace(manifiestoSHA256)
	if pool == nil || !patronIDCatalogoPublicoPostgreSQL.MatchString(catalogoCategorias) ||
		versionCategorias < 1 ||
		!patronHuellaPublicaPostgreSQL.MatchString(huellaCategoriasGobernadaHex) ||
		!patronHuellaPublicaPostgreSQL.MatchString(huellaProyeccionCategoriasHex) ||
		!patronHuellaPublicaPostgreSQL.MatchString(manifiestoSHA256) ||
		manifiestoSHA256 == strings.Repeat("0", 64) {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	return &Fuente{
		pool: pool, catalogoCategorias: catalogoCategorias,
		manifiestoSHA256:              manifiestoSHA256,
		versionCategorias:             versionCategorias,
		huellaCategoriasGobernadaHex:  huellaCategoriasGobernadaHex,
		huellaProyeccionCategoriasHex: huellaProyeccionCategoriasHex,
	}, nil
}

func (f *Fuente) Cerrar() {
	if f == nil {
		return
	}
	f.cerrarUnaVez.Do(func() {
		f.integridadMu.Lock()
		cancelar, terminada := f.integridadCancelar, f.integridadTerminada
		f.integridadMu.Unlock()
		if cancelar != nil {
			cancelar()
		}
		if terminada != nil {
			<-terminada
		}
		if f.pool != nil {
			f.pool.Close()
		}
	})
}

func prepararConfiguracionPool(dsn string) (*pgxpool.Config, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	configuracion, err := pgxpool.ParseConfig(strings.TrimSpace(dsn))
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	// No se permite que options, PGOPTIONS ni parámetros de sesión del DSN
	// compitan con la política fijada por este adaptador. Incluso si luego los
	// sobrescribiésemos, aceptar claves desconocidas haría frágil el cierre.
	if len(configuracion.ConnConfig.RuntimeParams) != 0 {
		return nil, ErrConfiguracionPostgreSQLPublicaInvalida
	}
	if err := validarTLSPostgreSQLPublico(&configuracion.ConnConfig.Config); err != nil {
		return nil, err
	}
	configuracion.MaxConns = 6
	configuracion.MinConns = 0
	configuracion.MinIdleConns = 0
	configuracion.MaxConnLifetime = duracionVidaPostgreSQLPublica
	configuracion.MaxConnIdleTime = duracionInactividadPostgreSQLPublica
	configuracion.HealthCheckPeriod = periodoSaludPostgreSQLPublica
	configuracion.PingTimeout = duracionSondaPostgreSQLPublica
	configuracion.ConnConfig.ConnectTimeout = duracionConexionPostgreSQLPublica
	if configuracion.ConnConfig.RuntimeParams == nil {
		configuracion.ConnConfig.RuntimeParams = make(map[string]string)
	}
	parametros := configuracion.ConnConfig.RuntimeParams
	parametros["application_name"] = aplicacionPostgreSQLPublica
	parametros["timezone"] = "UTC"
	parametros["search_path"] = "pg_catalog,pg_temp"
	parametros["default_transaction_read_only"] = "on"
	parametros["default_transaction_isolation"] = "repeatable read"
	parametros["statement_timeout"] = "10s"
	parametros["lock_timeout"] = "2s"
	parametros["idle_in_transaction_session_timeout"] = "10s"
	configuracion.AfterConnect = func(ctx context.Context, conexion *pgx.Conn) error {
		return comprobarIdentidad(ctx, conexion)
	}
	configuracion.PrepareConn = func(ctx context.Context, conexion *pgx.Conn) (bool, error) {
		if err := comprobarIdentidad(ctx, conexion); err != nil {
			return false, err
		}
		return true, nil
	}
	return configuracion, nil
}

func validarTLSPostgreSQLPublico(configuracion *pgconn.Config) error {
	if configuracion == nil || !tlsPostgreSQLPublicoVerificaIdentidad(configuracion.TLSConfig, configuracion.Host) {
		return ErrTLSPostgreSQLPublicaInseguro
	}
	for _, alternativa := range configuracion.Fallbacks {
		if alternativa == nil || !tlsPostgreSQLPublicoVerificaIdentidad(alternativa.TLSConfig, alternativa.Host) {
			return ErrTLSPostgreSQLPublicaInseguro
		}
	}
	return nil
}

func tlsPostgreSQLPublicoVerificaIdentidad(configuracion *tls.Config, host string) bool {
	if configuracion == nil || configuracion.InsecureSkipVerify ||
		strings.TrimSpace(configuracion.ServerName) == "" ||
		!strings.EqualFold(strings.TrimSpace(configuracion.ServerName), strings.TrimSpace(host)) ||
		versionesTLSPostgreSQLPublicoInseguras(configuracion) {
		return false
	}
	return configuracion.RootCAs == nil || poolCertificadosPostgreSQLPublicoNoVacio(configuracion.RootCAs)
}

func versionesTLSPostgreSQLPublicoInseguras(configuracion *tls.Config) bool {
	if configuracion == nil ||
		(configuracion.MinVersion != 0 && configuracion.MinVersion < tls.VersionTLS12) ||
		(configuracion.MaxVersion != 0 && configuracion.MaxVersion < tls.VersionTLS12) {
		return true
	}
	return configuracion.MinVersion != 0 && configuracion.MaxVersion != 0 &&
		configuracion.MinVersion > configuracion.MaxVersion
}

func poolCertificadosPostgreSQLPublicoNoVacio(pool *x509.CertPool) bool {
	return pool != nil && len(pool.Subjects()) > 0
}

type consultadorFila interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const consultaIdentidadPostgreSQLPublica = `
WITH vistas_esperadas(nombre) AS (VALUES
    ('fuente_publica_v2'),
    ('entradas_catalogos_publicos_v2'),
    ('catalogos_categorias_publicos_v2'),
    ('categorias_publicas_v2'),
    ('convocatorias_publicadas_v2'),
    ('categorias_convocatorias_publicas_v2'),
    ('plazos_convocatorias_publicas_v2'),
    ('requisitos_convocatorias_publicas_v2'),
    ('documentos_convocatorias_publicas_v2'),
    ('ayuda_convocatorias_publicas_v2')
), privilegios_relaciones AS (
    SELECT COALESCE(bool_and(
        CASE
          WHEN espacio.nspname = 'vec_bolsa_publica_lectura'
           AND relacion.relkind = 'v'
           AND esperada.nombre IS NOT NULL
          THEN pg_catalog.has_table_privilege(
                   session_user, relacion.oid, 'SELECT'
               ) AND NOT pg_catalog.has_table_privilege(
                   session_user, relacion.oid,
                   'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               )
          ELSE NOT pg_catalog.has_table_privilege(
                   session_user, relacion.oid,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               )
        END
    ), false)
    AND count(*) FILTER (
        WHERE espacio.nspname = 'vec_bolsa_publica_lectura'
          AND relacion.relkind = 'v'
          AND esperada.nombre IS NOT NULL
          AND pg_catalog.has_table_privilege(session_user, relacion.oid, 'SELECT')
    ) = 10 AS validos
      FROM pg_catalog.pg_class AS relacion
      JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = relacion.relnamespace
 LEFT JOIN vistas_esperadas AS esperada
        ON espacio.nspname = 'vec_bolsa_publica_lectura'
       AND esperada.nombre = relacion.relname
     WHERE espacio.nspname !~ '^pg_'
       AND espacio.nspname <> 'information_schema'
       AND relacion.relkind IN ('r','p','v','m','f')
), privilegios_funciones AS (
    SELECT NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND pg_catalog.has_function_privilege(session_user, funcion.oid, 'EXECUTE')
    ) AS validos
), privilegios_secuencias AS (
    SELECT NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS secuencia
          JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = secuencia.relnamespace
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND secuencia.relkind = 'S'
           AND pg_catalog.has_sequence_privilege(
               session_user, secuencia.oid, 'USAGE,SELECT,UPDATE'
           )
    ) AS validos
), roles_control AS (
    SELECT identidad.oid AS identidad_oid, objetivo.oid AS objetivo_oid
      FROM pg_catalog.pg_roles AS identidad
      CROSS JOIN pg_catalog.pg_roles AS objetivo
     WHERE identidad.rolname = session_user AND objetivo.rolname = $1
), acl_relaciones AS (
    SELECT NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS relacion
          JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = relacion.relnamespace
          CROSS JOIN roles_control AS roles
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(relacion.relacl, pg_catalog.acldefault('r', relacion.relowner))
          ) AS permiso
          LEFT JOIN vistas_esperadas AS esperada
            ON espacio.nspname = 'vec_bolsa_publica_lectura'
           AND esperada.nombre = relacion.relname
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND relacion.relkind IN ('r','p','v','m','f')
           AND permiso.grantee IN (0, roles.identidad_oid, roles.objetivo_oid)
           AND NOT (
               permiso.grantee = roles.objetivo_oid
               AND espacio.nspname = 'vec_bolsa_publica_lectura'
               AND relacion.relkind = 'v'
               AND esperada.nombre IS NOT NULL
               AND permiso.privilege_type = 'SELECT'
           )
    ) AND (
        SELECT count(*) = 10
          FROM pg_catalog.pg_class AS relacion
          JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = relacion.relnamespace
          CROSS JOIN roles_control AS roles
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(relacion.relacl, pg_catalog.acldefault('r', relacion.relowner))
          ) AS permiso
          JOIN vistas_esperadas AS esperada ON esperada.nombre = relacion.relname
         WHERE espacio.nspname = 'vec_bolsa_publica_lectura'
           AND relacion.relkind = 'v'
           AND permiso.grantee = roles.objetivo_oid
           AND permiso.privilege_type = 'SELECT'
    ) AS validos
), acl_esquemas AS (
    SELECT NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN roles_control AS roles
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(espacio.nspacl, pg_catalog.acldefault('n', espacio.nspowner))
          ) AS permiso
         WHERE espacio.nspname !~ '^pg_'
           AND espacio.nspname <> 'information_schema'
           AND permiso.grantee IN (0, roles.identidad_oid, roles.objetivo_oid)
           AND NOT (
               permiso.grantee = roles.objetivo_oid
               AND espacio.nspname = 'vec_bolsa_publica_lectura'
               AND permiso.privilege_type = 'USAGE'
           )
           AND NOT (
               permiso.grantee = 0
               AND espacio.nspname = 'public'
               AND permiso.privilege_type = 'USAGE'
           )
    ) AND (
        SELECT count(*) = 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN roles_control AS roles
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(espacio.nspacl, pg_catalog.acldefault('n', espacio.nspowner))
          ) AS permiso
         WHERE espacio.nspname = 'vec_bolsa_publica_lectura'
           AND permiso.grantee = roles.objetivo_oid
           AND permiso.privilege_type = 'USAGE'
    ) AS validos
), acl_bases AS (
    SELECT NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN roles_control AS roles
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(base.datacl, pg_catalog.acldefault('d', base.datdba))
          ) AS permiso
         WHERE base.datallowconn
           AND permiso.grantee IN (0, roles.identidad_oid, roles.objetivo_oid)
           AND NOT (
               permiso.grantee = roles.objetivo_oid
               AND base.datname = current_database()
               AND permiso.privilege_type = 'CONNECT'
           )
    ) AND (
        SELECT count(*) = 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN roles_control AS roles
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(base.datacl, pg_catalog.acldefault('d', base.datdba))
          ) AS permiso
         WHERE base.datname = current_database()
           AND permiso.grantee = roles.objetivo_oid
           AND permiso.privilege_type = 'CONNECT'
    ) AS validos
)
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
					  AND membresia.set_option IS FALSE
                  ), false)
             FROM pg_catalog.pg_auth_members AS membresia
            WHERE membresia.member = identidad.oid
       )
       AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS anidada
            WHERE anidada.member = objetivo.oid
       )
       AND pg_catalog.has_database_privilege(session_user, current_database(), 'CONNECT')
       AND NOT pg_catalog.has_database_privilege(
           session_user, current_database(), 'CREATE,TEMPORARY'
       )
       AND NOT pg_catalog.has_schema_privilege(session_user, 'public', 'CREATE')
       AND pg_catalog.has_schema_privilege(
           session_user, 'vec_bolsa_publica_lectura', 'USAGE'
       )
       AND NOT pg_catalog.has_schema_privilege(
           session_user, 'vec_bolsa_publica_lectura', 'CREATE'
       )
       AND NOT pg_catalog.has_schema_privilege(
           session_user, 'vec_bolsa_publica_datos', 'USAGE,CREATE'
       )
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_database AS base
            WHERE base.datallowconn
              AND base.datname <> current_database()
              AND pg_catalog.has_database_privilege(session_user, base.oid, 'CONNECT')
       )
       AND NOT EXISTS (
           SELECT 1
             FROM pg_catalog.pg_namespace AS espacio
            WHERE espacio.nspname !~ '^pg_'
              AND espacio.nspname <> 'information_schema'
              AND espacio.nspname NOT IN (
                  'public', 'vec_bolsa_publica_datos', 'vec_bolsa_publica_lectura'
              )
              AND pg_catalog.has_schema_privilege(
                  session_user, espacio.oid, 'USAGE,CREATE'
              )
       )
       AND (SELECT validos FROM privilegios_relaciones)
       AND (SELECT validos FROM privilegios_funciones)
       AND (SELECT validos FROM privilegios_secuencias)
       AND (SELECT validos FROM acl_relaciones)
       AND (SELECT validos FROM acl_esquemas)
       AND (SELECT validos FROM acl_bases)
       AND current_setting('application_name') = 'vec-bolsa-publica'
       AND current_setting('TimeZone') = 'UTC'
       AND replace(current_setting('search_path'), ' ', '') = 'pg_catalog,pg_temp'
       AND current_setting('default_transaction_read_only') = 'on'
       AND current_setting('transaction_read_only') = 'on'
       AND current_setting('default_transaction_isolation') = 'repeatable read'
       AND current_setting('statement_timeout') = '10s'
       AND current_setting('lock_timeout') = '2s'
       AND current_setting('idle_in_transaction_session_timeout') = '10s'
      FROM pg_catalog.pg_roles AS identidad
      CROSS JOIN pg_catalog.pg_roles AS objetivo
     WHERE identidad.rolname = session_user
       AND objetivo.rolname = $1
), false)`

func comprobarIdentidad(ctx context.Context, consulta consultadorFila) error {
	if ctx == nil || consulta == nil {
		return ErrIdentidadPostgreSQLPublicaInvalida
	}
	var usuarioSesion, usuarioEfectivo string
	var valida bool
	err := consulta.QueryRow(ctx, consultaIdentidadPostgreSQLPublica, rolConsultaPostgreSQLPublica).Scan(
		&usuarioSesion, &usuarioEfectivo, &valida,
	)
	if err != nil || usuarioSesion == "" || usuarioSesion != usuarioEfectivo || !valida {
		return errorPostgreSQLPublico(ctx, ErrIdentidadPostgreSQLPublicaInvalida)
	}
	return nil
}

func huellasIguales(obtenida, esperada string) bool {
	return len(obtenida) == 64 && len(esperada) == 64 &&
		subtle.ConstantTimeCompare([]byte(obtenida), []byte(esperada)) == 1
}

func errorPostgreSQLPublico(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return errors.Join(ErrPostgreSQLPublicoNoDisponible, ctx.Err())
	}
	if errors.Is(err, ErrIdentidadPostgreSQLPublicaInvalida) {
		return ErrIdentidadPostgreSQLPublicaInvalida
	}
	if errors.Is(err, ErrDatosPostgreSQLPublicosNoConfiables) {
		return ErrDatosPostgreSQLPublicosNoConfiables
	}
	return ErrPostgreSQLPublicoNoDisponible
}
