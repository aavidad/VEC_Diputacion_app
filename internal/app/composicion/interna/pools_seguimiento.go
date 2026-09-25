package interna

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
)

// MaterialPoolSeguimiento se entrega desde el inventario privado del arranque.
// Ni los DSN ni los LOGIN se publican, registran o incorporan a errores.
type MaterialPoolSeguimiento struct {
	DSN   string
	Login string
}

const materialPoolSeguimientoRedactado = "[material PostgreSQL privado]"

var ErrMaterialPoolSeguimientoNoSerializable = errors.New("composicion interna: material PostgreSQL no reconstruible desde serializacion")

func (MaterialPoolSeguimiento) String() string               { return materialPoolSeguimientoRedactado }
func (m MaterialPoolSeguimiento) GoString() string           { return m.String() }
func (m MaterialPoolSeguimiento) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(m.String())) }
func (MaterialPoolSeguimiento) LogValue() slog.Value {
	return slog.StringValue(materialPoolSeguimientoRedactado)
}
func (MaterialPoolSeguimiento) MarshalJSON() ([]byte, error) {
	return []byte(`"` + materialPoolSeguimientoRedactado + `"`), nil
}
func (*MaterialPoolSeguimiento) UnmarshalJSON([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}
func (MaterialPoolSeguimiento) MarshalText() ([]byte, error) {
	return []byte(materialPoolSeguimientoRedactado), nil
}
func (*MaterialPoolSeguimiento) UnmarshalText([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}
func (MaterialPoolSeguimiento) MarshalBinary() ([]byte, error) {
	return []byte(materialPoolSeguimientoRedactado), nil
}
func (*MaterialPoolSeguimiento) UnmarshalBinary([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}
func (MaterialPoolSeguimiento) GobEncode() ([]byte, error) {
	return []byte(materialPoolSeguimientoRedactado), nil
}
func (*MaterialPoolSeguimiento) GobDecode([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}

// Los once LOGIN son distintos aunque dos capacidades compartan un grupo SQL.
// Este material no contiene autoridades F1/V3, políticas ni claves de identidad.
type MaterialPoolsSeguimiento struct {
	AltaPersonal, RegistroCT, InicialCT, LocalizadorCT      MaterialPoolSeguimiento
	LocalizadorPersonal, LecturaPersonal                    MaterialPoolSeguimiento
	HistoriaRegistroCT, HistoriaAutenticacion               MaterialPoolSeguimiento
	HistoriaContexto, HistoriaEvaluacion, HistoriaConcesion MaterialPoolSeguimiento
}

func (MaterialPoolsSeguimiento) String() string               { return materialPoolSeguimientoRedactado }
func (m MaterialPoolsSeguimiento) GoString() string           { return m.String() }
func (m MaterialPoolsSeguimiento) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(m.String())) }
func (MaterialPoolsSeguimiento) LogValue() slog.Value {
	return slog.StringValue(materialPoolSeguimientoRedactado)
}
func (MaterialPoolsSeguimiento) MarshalJSON() ([]byte, error) {
	return []byte(`"` + materialPoolSeguimientoRedactado + `"`), nil
}
func (*MaterialPoolsSeguimiento) UnmarshalJSON([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}
func (MaterialPoolsSeguimiento) MarshalText() ([]byte, error) {
	return []byte(materialPoolSeguimientoRedactado), nil
}
func (*MaterialPoolsSeguimiento) UnmarshalText([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}
func (MaterialPoolsSeguimiento) MarshalBinary() ([]byte, error) {
	return []byte(materialPoolSeguimientoRedactado), nil
}
func (*MaterialPoolsSeguimiento) UnmarshalBinary([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}
func (MaterialPoolsSeguimiento) GobEncode() ([]byte, error) {
	return []byte(materialPoolSeguimientoRedactado), nil
}
func (*MaterialPoolsSeguimiento) GobDecode([]byte) error {
	return ErrMaterialPoolSeguimientoNoSerializable
}

var ErrPoolsSeguimientoNoDisponibles = errors.New("composicion interna: pools de seguimiento no disponibles")

const (
	plazoConexionPoolSeguimiento = 5 * time.Second
	plazoSondaPoolSeguimiento    = 5 * time.Second
)

type perfilPoolSeguimiento struct {
	rol, funcion, aplicacion string
	material                 MaterialPoolSeguimiento
}

// Las firmas proceden de las fachadas SQL existentes. La sonda comprueba el
// permiso nominal de la funcion que consume cada adaptador, sin ejecutarla.
func perfilesPoolsSeguimiento(m MaterialPoolsSeguimiento) [11]perfilPoolSeguimiento {
	return [11]perfilPoolSeguimiento{
		{"vec_personal_ejecutor", "vec_personal.registrar_alta_ejercicio_v1(jsonb,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)", "vec-interno-alta-personal", m.AltaPersonal},
		{"vec_contratacion_temporal_ejecutor", "vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)", "vec-interno-registro-ct", m.RegistroCT},
		{"vec_contratacion_temporal_lector_raices_historicas", "vec_contratacion_temporal.leer_raices_incorporacion_ejercicio_v2(text,text,text)", "vec-interno-raices-ct", m.InicialCT},
		{"vec_contratacion_temporal_localizador_incorporacion", "vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)", "vec-interno-localizador-ct", m.LocalizadorCT},
		{"vec_personal_localizador_solicitud_alta", "vec_personal.localizar_solicitud_alta_ejercicio_v1(text,text,text)", "vec-interno-localizador-personal", m.LocalizadorPersonal},
		{"vec_personal_ejecutor", "vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)", "vec-interno-lectura-personal", m.LecturaPersonal},
		{"vec_contratacion_temporal_lector_historia_incorporacion", "vec_contratacion_temporal.leer_historia_incorporacion_original_v2(text,text,text)", "vec-interno-historia-ct", m.HistoriaRegistroCT},
		{"vec_identidad_sesiones_v1_lector_historico", "vec_identidad_sesiones_v1.leer_autenticacion_original_v1(text,text,text)", "vec-interno-historia-autenticacion", m.HistoriaAutenticacion},
		{"vec_contexto_actor_v1_lector_historico", "vec_contexto_actor_v1.leer_contexto_original_v2(text,text,text)", "vec-interno-historia-contexto", m.HistoriaContexto},
		{"vec_autorizacion_evaluacion_historica_lector", "vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)", "vec-interno-historia-evaluacion", m.HistoriaEvaluacion},
		{"vec_autorizacion_registro", "vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)", "vec-interno-historia-concesion", m.HistoriaConcesion},
	}
}

func rolHeredaPoolSeguimiento(rol string) bool {
	switch rol {
	case "vec_personal_ejecutor", "vec_contratacion_temporal_ejecutor", "vec_autorizacion_registro":
		return true
	default:
		return false
	}
}

// AbrirPoolsSeguimiento solo monta la parte PostgreSQL de ConfiguracionV2.
// El llamador conserva propiedad y debe cerrar sus once pools al fallar un
// montaje posterior o al apagar la aplicacion. No instala SQL ni crea gobierno.
func AbrirPoolsSeguimiento(ctx context.Context, material MaterialPoolsSeguimiento) (inc.ConfiguracionServidorV2PostgreSQL, error) {
	var salida inc.ConfiguracionServidorV2PostgreSQL
	if ctx == nil || ctx.Err() != nil {
		return salida, ErrPoolsSeguimientoNoDisponibles
	}
	perfiles := perfilesPoolsSeguimiento(material)
	configuraciones := make([]*pgxpool.Config, len(perfiles))
	loginVistos := make(map[string]bool, len(perfiles))
	for i, perfil := range perfiles {
		configuracion, err := configurarPoolSeguimiento(perfil)
		if err != nil || loginVistos[perfil.material.Login] {
			return salida, ErrPoolsSeguimientoNoDisponibles
		}
		loginVistos[perfil.material.Login] = true
		configuraciones[i] = configuracion
	}
	pools := make([]*pgxpool.Pool, 0, len(configuraciones))
	for _, configuracion := range configuraciones {
		pool, err := pgxpool.NewWithConfig(ctx, configuracion)
		if err != nil {
			cerrarPoolsSeguimiento(pools)
			return salida, ErrPoolsSeguimientoNoDisponibles
		}
		pools = append(pools, pool)
		ctxSonda, cancelar := context.WithTimeout(ctx, plazoSondaPoolSeguimiento)
		err = pool.Ping(ctxSonda) // AfterConnect acredita LOGIN y ACL antes de aceptar la conexion.
		cancelar()
		if err != nil {
			cerrarPoolsSeguimiento(pools)
			return salida, ErrPoolsSeguimientoNoDisponibles
		}
	}
	salida.AltaPersonal, salida.RegistroCT = pools[0], pools[1]
	salida.Preparacion.Pools = inc.PoolsPreparacionDurableV2{
		InicialCT: pools[2], LocalizadorCT: pools[3], LocalizadorPersonal: pools[4], LecturaPersonal: pools[5],
		Historia: pgct.PoolsHistoriaIncorporacionV2{
			RegistroCT: pools[6], Autenticacion: pools[7], Contexto: pools[8], Evaluacion: pools[9], Concesion: pools[10],
		},
	}
	return salida, nil
}

func configurarPoolSeguimiento(perfil perfilPoolSeguimiento) (*pgxpool.Config, error) {
	if perfil.material.Login == "" || strings.TrimSpace(perfil.material.Login) != perfil.material.Login ||
		strings.TrimSpace(perfil.material.DSN) == "" {
		return nil, ErrPoolsSeguimientoNoDisponibles
	}
	configuracion, err := pgxpool.ParseConfig(perfil.material.DSN)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.ConnConfig.User != perfil.material.Login ||
		!tlsPoolSeguimientoVerificado(&configuracion.ConnConfig.Config) {
		return nil, ErrPoolsSeguimientoNoDisponibles
	}
	configuracion.MaxConns = 2
	configuracion.MinConns = 0
	configuracion.MinIdleConns = 0
	configuracion.ConnConfig.ConnectTimeout = plazoConexionPoolSeguimiento
	configuracion.PingTimeout = plazoSondaPoolSeguimiento
	configuracion.MaxConnLifetime = 30 * time.Minute
	configuracion.MaxConnIdleTime = 5 * time.Minute
	if configuracion.ConnConfig.RuntimeParams == nil {
		configuracion.ConnConfig.RuntimeParams = make(map[string]string)
	}
	for clave, valor := range map[string]string{
		"application_name": perfil.aplicacion, "timezone": "UTC", "search_path": "pg_catalog,pg_temp",
		"default_transaction_isolation": "serializable", "default_transaction_read_only": "off", "statement_timeout": "15s",
		"lock_timeout": "3s", "idle_in_transaction_session_timeout": "15s",
	} {
		configuracion.ConnConfig.RuntimeParams[clave] = valor
	}
	configuracion.AfterConnect = func(ctx context.Context, conexion *pgx.Conn) error {
		conexion.TypeMap().RegisterType(&pgtype.Type{Name: "timestamptz", OID: pgtype.TimestamptzOID, Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC}})
		conexion.TypeMap().RegisterType(&pgtype.Type{Name: "timestamp", OID: pgtype.TimestampOID, Codec: &pgtype.TimestampCodec{ScanLocation: time.UTC}})
		return acreditarPoolSeguimiento(ctx, conexion, perfil)
	}
	return configuracion, nil
}

// Equivale a verify-full tambien para las rutas de fallback de pgx. No hay
// excepcion de loopback: esta raiz no depende del perfil de desarrollo legado.
func tlsPoolSeguimientoVerificado(configuracion *pgconn.Config) bool {
	if configuracion == nil || !tlsDestinoPoolSeguimiento(configuracion.TLSConfig, configuracion.Host) {
		return false
	}
	for _, alternativa := range configuracion.Fallbacks {
		if alternativa == nil || !tlsDestinoPoolSeguimiento(alternativa.TLSConfig, alternativa.Host) {
			return false
		}
	}
	return true
}

func tlsDestinoPoolSeguimiento(c *tls.Config, host string) bool {
	if c == nil || c.InsecureSkipVerify || strings.TrimSpace(host) == "" ||
		!strings.EqualFold(strings.TrimSpace(c.ServerName), strings.TrimSpace(host)) ||
		(c.MinVersion != 0 && c.MinVersion < tls.VersionTLS12) ||
		(c.MaxVersion != 0 && c.MaxVersion < tls.VersionTLS12) ||
		(c.MinVersion != 0 && c.MaxVersion != 0 && c.MinVersion > c.MaxVersion) {
		return false
	}
	return c.RootCAs == nil || certificadosPoolSeguimiento(c.RootCAs)
}

func certificadosPoolSeguimiento(p *x509.CertPool) bool { return p != nil && len(p.Subjects()) > 0 }

// La consulta se ejecuta como invocador, sin SECURITY DEFINER ni SET ROLE.
// Exige un unico grupo directo sin herencia ulterior, ningun privilegio
// administrativo/directo del LOGIN y ACL explicita sobre la funcion propietaria.
const consultaACLPoolSeguimiento = `
SELECT session_user::text, current_user::text,
 COALESCE((SELECT l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
   AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
   AND l.rolconfig IS NULL AND r.rolconfig IS NULL
   AND pg_catalog.current_setting('role')='none'
   AND NOT r.rolcanlogin AND r.rolinherit=$3 AND NOT r.rolsuper AND NOT r.rolcreatedb
   AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls
   AND (SELECT count(*)=1 AND bool_and(m.roleid=r.oid AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
          FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_shdepend d
      WHERE d.refclassid='pg_catalog.pg_authid'::regclass AND d.refobjid=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_db_role_setting s WHERE s.setrole IN (l.oid,r.oid))
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl da WHERE da.defaclrole IN (l.oid,r.oid))
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_policy pol WHERE l.oid=ANY(pol.polroles) OR r.oid=ANY(pol.polroles))
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p WHERE p.proowner=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c WHERE c.relowner=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspowner=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_type t WHERE t.typowner=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_database d WHERE d.datdba=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p, LATERAL pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class c, LATERAL pg_catalog.aclexplode(c.relacl) a WHERE a.grantee=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n, LATERAL pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_type t, LATERAL pg_catalog.aclexplode(t.typacl) a WHERE a.grantee=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_database d, LATERAL pg_catalog.aclexplode(d.datacl) a WHERE a.grantee=l.oid)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_default_acl da, LATERAL pg_catalog.aclexplode(da.defaclacl) a WHERE a.grantee=l.oid OR a.grantor=l.oid)
  FROM pg_catalog.pg_roles l CROSS JOIN pg_catalog.pg_roles r
  WHERE l.rolname=session_user AND r.rolname=$1), false),
 COALESCE((SELECT pg_catalog.has_schema_privilege(session_user,n.oid,'USAGE')
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
      WHERE a.grantee=r.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a WHERE a.grantee=0)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(n.nspacl) a WHERE a.grantee=0 AND a.privilege_type='USAGE')
  FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
  JOIN pg_catalog.pg_roles r ON r.rolname=$1
  WHERE p.oid=pg_catalog.to_regprocedure($2)), false)`

type consultadorACLPoolSeguimiento interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func acreditarPoolSeguimiento(ctx context.Context, q consultadorACLPoolSeguimiento, p perfilPoolSeguimiento) error {
	if ctx == nil || q == nil || ctx.Err() != nil {
		return ErrPoolsSeguimientoNoDisponibles
	}
	ctxSonda, cancelar := context.WithTimeout(ctx, plazoSondaPoolSeguimiento)
	defer cancelar()
	var sesion, efectivo string
	var loginValido, aclValida bool
	if err := q.QueryRow(ctxSonda, consultaACLPoolSeguimiento, p.rol, p.funcion, rolHeredaPoolSeguimiento(p.rol)).Scan(&sesion, &efectivo, &loginValido, &aclValida); err != nil ||
		sesion == "" || sesion != efectivo || sesion != p.material.Login || !loginValido || !aclValida {
		return ErrPoolsSeguimientoNoDisponibles
	}
	return nil
}
