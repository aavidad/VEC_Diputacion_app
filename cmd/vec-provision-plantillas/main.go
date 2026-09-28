// vec-provision-plantillas instala una sola preimagen publicada de plantillas
// por el canal migrador, antes de arrancar la API RRHH. No forma parte del
// servidor ni acepta credenciales en argumentos.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/app/bootstrap"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const sqlProvision = `SELECT resultado,recibo_ref,version,revision,catalogo_huella_sha256,contenido_json_sha256,procedencia_ref,registrada_en
 FROM vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1($1::jsonb,$2,$3,$4)`

// La CLI usa una credencial de instalación externa. La sesión debe tener una
// sola membresía nominal, heredando únicamente la fachada de provisión.
// La consulta se ejecuta dentro de la misma transacción que el efecto.
const sqlPreflight = `WITH RECURSIVE miembros(rol_id,admin_option) AS (
 SELECT m.roleid,m.admin_option FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::regrole
 UNION
 SELECT m.roleid,prev.admin_option OR m.admin_option
 FROM pg_catalog.pg_auth_members m JOIN miembros prev ON prev.rol_id=m.member
), objetivo AS (
 SELECT pg_catalog.to_regprocedure('vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text)') AS oid
)
SELECT coalesce(pg_catalog.bool_and(
 session_user=current_user
 AND l.rolcanlogin AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
 AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND NOT m.rolcanlogin AND NOT m.rolsuper AND NOT m.rolcreatedb AND NOT m.rolcreaterole
 AND NOT m.rolreplication AND NOT m.rolbypassrls
 AND NOT p.rolcanlogin AND NOT p.rolsuper AND NOT p.rolcreatedb AND NOT p.rolcreaterole
 AND NOT p.rolreplication AND NOT p.rolbypassrls
 AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members d
      WHERE d.member=session_user::regrole)=1
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members d
      WHERE d.member=session_user::regrole AND d.roleid=m.oid
        AND NOT d.admin_option AND d.inherit_option AND NOT d.set_option)
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members d
      WHERE d.member=m.oid AND d.roleid=p.oid
        AND NOT d.admin_option AND NOT d.inherit_option)
 AND NOT EXISTS (SELECT 1 FROM miembros WHERE rol_id NOT IN (m.oid,p.oid) OR admin_option)
 AND pg_catalog.pg_has_role(session_user,m.oid,'MEMBER')
 AND NOT pg_catalog.pg_has_role(session_user,p.oid,'USAGE')
 AND s.nspowner=p.oid
 AND pg_catalog.has_schema_privilege(session_user,s.oid,'USAGE')
 AND NOT pg_catalog.has_schema_privilege(session_user,s.oid,'CREATE')
 AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(s.nspacl) a
      WHERE a.grantee=m.oid AND a.privilege_type='USAGE' AND NOT a.is_grantable)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(s.nspacl) a
      WHERE a.grantee IN (0,l.oid)
         OR (a.grantee=m.oid AND (a.privilege_type<>'USAGE' OR a.is_grantable)))
 AND f.proowner=p.oid AND f.prosecdef AND f.prokind='f' AND f.provolatile='v'
 AND pg_catalog.has_function_privilege(session_user,f.oid,'EXECUTE')
 AND EXISTS (SELECT 1 FROM pg_catalog.aclexplode(f.proacl) a
      WHERE a.grantee=m.oid AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.aclexplode(f.proacl) a
      WHERE a.grantee NOT IN (p.oid,m.oid) OR a.privilege_type<>'EXECUTE'
         OR (a.grantee=m.oid AND a.is_grantable))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc otra
      WHERE otra.pronamespace=s.oid AND otra.oid<>f.oid
        AND pg_catalog.has_function_privilege(session_user,otra.oid,'EXECUTE'))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class tabla
      WHERE tabla.relnamespace=s.oid AND tabla.relkind IN ('r','p','v','m','f','S')
        AND CASE WHEN tabla.relkind='S'
             THEN pg_catalog.has_sequence_privilege(session_user,tabla.oid,'USAGE,SELECT,UPDATE')
             ELSE pg_catalog.has_table_privilege(session_user,tabla.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
               OR pg_catalog.has_any_column_privilege(session_user,tabla.oid,'SELECT,INSERT,UPDATE,REFERENCES') END)
 AND pg_catalog.current_setting('transaction_isolation')='serializable'
 AND pg_catalog.current_setting('transaction_read_only')='off'
),false)
FROM pg_catalog.pg_roles l,pg_catalog.pg_roles m,pg_catalog.pg_roles p,
 pg_catalog.pg_namespace s,pg_catalog.pg_proc f,objetivo o
WHERE l.rolname=session_user AND m.rolname='vec_contratacion_temporal_migrador'
 AND p.rolname='vec_contratacion_temporal_propietario'
 AND s.nspname='vec_contratacion_temporal' AND f.oid=o.oid`

const sqlContenidoSHA256 = `SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to($1::jsonb::text,'UTF8')),'hex')`

var (
	errEntrada   = errors.New("provisión CT: catálogo o aprobación inválidos")
	errConexion  = errors.New("provisión CT: conexión migradora no disponible")
	errSalida    = errors.New("provisión CT: recibo no entregado; repita la misma solicitud para recuperar el recibo")
	huellaValida = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reciboValido = regexp.MustCompile(`^recibo:[0-9a-f-]{36}$`)
)

type reciboProvision struct {
	Resultado            string    `json:"resultado"`
	ReciboRef            string    `json:"recibo_ref"`
	Version              int64     `json:"version"`
	Revision             int64     `json:"revision"`
	CatalogoHuellaSHA256 string    `json:"catalogo_huella_sha256"`
	ContenidoJSONSHA256  string    `json:"contenido_json_sha256"`
	ProcedenciaRef       string    `json:"procedencia_ref"`
	RegistradaEn         time.Time `json:"registrada_en"`
}

func main() {
	var ruta, aprobacion string
	flag.StringVar(&ruta, "catalogo", "", "ruta absoluta del catálogo JSON publicado")
	flag.StringVar(&aprobacion, "aprobacion-ref", "", "referencia de la aprobación de la fuente")
	flag.Parse()
	if ruta == "" {
		ruta = os.Getenv("VEC_CT_PLANTILLAS_SOURCE_PATH")
	}
	if flag.NArg() != 0 || ruta == "" || aprobacion == "" || len(aprobacion) > 512 || aprobacion != strings.TrimSpace(aprobacion) {
		salir(errEntrada, 2)
	}
	dsn := os.Getenv("VEC_CT_PLANTILLAS_MIGRADOR_DATABASE_URL")
	if dsn == "" {
		salir(errConexion, 2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := bootstrap.CargarCatalogoPlantillasCT(ruta)
	if err != nil {
		salir(errEntrada, 2)
	}
	if aprobacion != c.AprobacionRef {
		salir(errEntrada, 2)
	}
	huella, err := c.HuellaSHA256()
	if err != nil {
		salir(errEntrada, 2)
	}
	b, err := json.Marshal(c)
	if err != nil {
		salir(errEntrada, 2)
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		salir(errConexion, 1)
	}
	if cfg.ConnectTimeout == 0 || cfg.ConnectTimeout > 10*time.Second {
		cfg.ConnectTimeout = 10 * time.Second
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = map[string]string{}
	}
	cfg.RuntimeParams["application_name"] = "vec-provision-plantillas"
	con, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		salir(errConexion, 1)
	}
	defer con.Close(context.Background())
	tx, err := con.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		salir(errConexion, 1)
	}
	defer tx.Rollback(context.Background())
	if err := comprobarPreflight(ctx, tx); err != nil {
		salir(errConexion, 1)
	}
	var contenidoSHA256 string
	if err := tx.QueryRow(ctx, sqlContenidoSHA256, string(b)).Scan(&contenidoSHA256); err != nil || !huellaValida.MatchString(contenidoSHA256) {
		salir(errConexion, 1)
	}
	var recibo reciboProvision
	err = tx.QueryRow(ctx, sqlProvision, string(b), huella, c.FuenteRef, aprobacion).Scan(&recibo.Resultado, &recibo.ReciboRef, &recibo.Version, &recibo.Revision, &recibo.CatalogoHuellaSHA256, &recibo.ContenidoJSONSHA256, &recibo.ProcedenciaRef, &recibo.RegistradaEn)
	if err != nil || !reciboCompatible(recibo, c, huella, contenidoSHA256) {
		salir(errConexion, 1)
	}
	if err = tx.Commit(ctx); err != nil {
		salir(errConexion, 1)
	}
	if err := emitirRecibo(os.Stdout, recibo); err != nil {
		salir(errSalida, 1)
	}
}

func emitirRecibo(w io.Writer, recibo reciboProvision) error {
	return json.NewEncoder(w).Encode(recibo)
}

func comprobarPreflight(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || q == nil {
		return errConexion
	}
	var valido bool
	if err := q.QueryRow(ctx, sqlPreflight).Scan(&valido); err != nil || !valido {
		return errConexion
	}
	return nil
}

func reciboCompatible(r reciboProvision, c vecdomain.CatalogoConfigurable, huella, contenidoSHA256 string) bool {
	return (r.Resultado == "registrado" || r.Resultado == "replay") && reciboValido.MatchString(r.ReciboRef) &&
		r.Version == int64(c.Version) && r.Revision == int64(c.Revision) && r.CatalogoHuellaSHA256 == huella &&
		huellaValida.MatchString(contenidoSHA256) && r.ContenidoJSONSHA256 == contenidoSHA256 &&
		r.ProcedenciaRef == c.FuenteRef && !r.RegistradaEn.IsZero()
}

func salir(err error, codigo int) { fmt.Fprintln(os.Stderr, err); os.Exit(codigo) }
