#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
[[ $imagen =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] || {
  echo 'la imagen PostgreSQL debe estar fijada por digest sha256' >&2
  exit 1
}
contenedor="vec-identidad-c21b-${USER:-usuario}-$$"
base=vec_identidad_c21b
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
salidas=$(mktemp -d)
up=deploy/postgresql/identidad_sesiones_v1/migraciones/000004_revalidacion_contexto_corporativo_rrhh_v1.up.sql
down=deploy/postgresql/identidad_sesiones_v1/migraciones/000004_revalidacion_contexto_corporativo_rrhh_v1.down.sql
rol_up=deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql
rol_down=deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_down.sql
login=vec_c21b_login
selector=vec_contexto_actor_corporativo_rrhh_selector
consumidor=vec_contexto_actor_v1_propietario
fachada=vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1
proxy=vec_contexto_actor_v1.c21b_proxy_identidad
marca_rol=vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1
marca_fachada=vec_identidad_sesiones_v1:fachada-contexto-corporativo-rrhh:v1

limpiar() {
  local estado=$?
  if [[ $estado -ne 0 ]]; then
    docker logs --tail 160 "$contenedor" 2>&1 |
      grep -E ' (LOG|ERROR|FATAL): ' | tail -60 >&2 || true
  fi
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf "$salidas"
}
trap limpiar EXIT INT TERM

paso() {
  printf '[CT-000047C2.1b:PG18.4] %s\n' "$1"
}
psql_admin() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -U postgres -d "$base"
}
psql_valor() {
  docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}
psql_archivo() {
  psql_admin <"$raiz/$1"
}
actor_valor() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" psql -XAtq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -h 127.0.0.1 -U "$login" -d "$base" -c "$1"
}
esperar_fallo_archivo() {
  local archivo=$1 caso=$2
  if psql_archivo "$archivo" >"$salidas/fallo_archivo" 2>&1; then
    echo "se aceptó un estado hostil: $caso" >&2
    exit 1
  fi
}
esperar_fallo_actor() {
  local sql=$1 caso=$2
  if actor_valor "$sql" >"$salidas/fallo_actor" 2>&1; then
    echo "el LOGIN ejecutó una operación prohibida: $caso" >&2
    exit 1
  fi
  grep -Eq '42501|permission denied|not allowed' "$salidas/fallo_actor"
}
proxy_contar() {
  local autenticacion=$1 sesion=$2
  actor_valor "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT count(*) FROM $proxy('$autenticacion','$sesion'); COMMIT" |
    grep -E '^[0-9]+$' | tail -1
}
exigir_cero() {
  [[ $(proxy_contar "$1" "$2") == 0 ]] || {
    echo "se esperaba ausencia de fila: $3" >&2
    exit 1
  }
}
exigir_uno() {
  [[ $(proxy_contar "$1" "$2") == 1 ]] || {
    echo "se esperaba una fila nominal: $3" >&2
    exit 1
  }
}
esperar_estado() {
  local consulta=$1 esperado=$2 caso=$3 observado=
  for _ in $(seq 1 300); do
    observado=$(psql_valor "$consulta")
    [[ $observado == "$esperado" ]] && return 0
    sleep 0.01
  done
  echo "no se observó la barrera causal: $caso ($observado)" >&2
  return 1
}
crear_sesion() {
  local operacion=$1 asercion=$2 sesion_id=$3 superficie=$4 garantia=$5
  psql_valor "SELECT autenticacion_ref||'|'||sesion_ref||'|'||
control_sesion_ref||'|'||control_sesion_revision_texto||'|'||cuenta_ref
FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
'opr_'||repeat('$operacion',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('$asercion',64),'hex'),decode(repeat('$sesion_id',64),'hex'),
decode(repeat('1',64),'hex'),decode(repeat('2',64),'hex'),NULL,false,
'$superficie','kerberos_ad','$garantia',repeat('$operacion',64),
date_trunc('microseconds',clock_timestamp()-interval '2 seconds'),
date_trunc('microseconds',clock_timestamp()-interval '1 second'),
date_trunc('microseconds',clock_timestamp()+interval '4 minutes'),
'pga_aaaaaaaaaaaaaaaaaaaaaaaa',repeat('b',64))"
}
crear_proxy() {
  psql_admin <<SQL
SET ROLE $consumidor;
CREATE FUNCTION $proxy(p_autenticacion_ref text,p_sesion_ref text)
RETURNS TABLE(cuenta_ref text,metodo_observado text,
  garantia_observada text,identidad_valida_hasta timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
BEGIN ATOMIC SELECT * FROM $fachada(p_autenticacion_ref,p_sesion_ref); END;
REVOKE ALL ON FUNCTION $proxy(text,text) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO $selector;
GRANT EXECUTE ON FUNCTION $proxy(text,text) TO $selector;
RESET ROLE;
SQL
}
eliminar_proxy() {
  psql_valor "SET ROLE $consumidor; DROP FUNCTION $proxy(text,text)" >/dev/null
}
huella_base() {
  psql_valor "SELECT encode(sha256(convert_to(
pg_get_functiondef('vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure)||
COALESCE((SELECT proacl::text FROM pg_proc WHERE oid=
'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure),''),
'UTF8')),'hex')"
}

source deploy/postgresql/identidad_sesiones_v1/pruebas_c21b/readiness_pg18_4.sh
source deploy/postgresql/identidad_sesiones_v1/pruebas_c21b/estados_carreras_cardinalidad.sh
source deploy/postgresql/identidad_sesiones_v1/pruebas_c21b/venenos_acl_topologia.sh
source deploy/postgresql/identidad_sesiones_v1/pruebas_c21b/venenos_fachada_viva.sh
source deploy/postgresql/identidad_sesiones_v1/pruebas_c21b/safe_down_consumidores_efectivos.sh

paso "arranque aislado con $imagen"
docker run --detach --name "$contenedor" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave" \
  "$imagen" >/dev/null
esperar_postgresql_definitivo "$contenedor" "$base" "$clave"
[[ $(psql_valor "SELECT current_setting('server_version_num')") == 180004 ]]

paso 'autoridades reales y cierre previo de PUBLIC'
psql_admin <<SQL
CREATE ROLE vec_c21b_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
ALTER DATABASE $base OWNER TO vec_c21b_dueno_base;
REVOKE ALL ON DATABASE $base FROM PUBLIC;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
for archivo in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql \
  "$rol_up"
do
  psql_archivo "$archivo" >/dev/null
done
psql_admin <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC;
SQL
for archivo in \
  deploy/postgresql/autorizacion/roles_up.sql \
  deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql \
  deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql \
  deploy/postgresql/identidad_sesiones_v1/roles_up.sql \
  deploy/postgresql/identidad_sesiones_v1/migraciones_autorizacion/000001_capacidad_tablas_v1.up.sql \
  deploy/postgresql/identidad_sesiones_v1/migraciones/000001_registro_base_v1.up.sql \
  deploy/postgresql/identidad_sesiones_v1/migraciones/000002_operaciones_v1.up.sql \
  deploy/postgresql/identidad_sesiones_v1/migraciones/000003_revalidacion_autenticacion_actor_v1.up.sql
do
  psql_archivo "$archivo" >/dev/null
done
[[ $(psql_valor "SELECT (NOT EXISTS(
SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
JOIN LATERAL aclexplode(COALESCE(t.typacl,acldefault('T',t.typowner))) a
ON true
WHERE n.nspname=ANY(ARRAY['vec_autorizacion','vec_identidad_sesiones_v1'])
AND NOT EXISTS(SELECT 1 FROM pg_type e
 WHERE e.oid=t.typelem AND e.typarray=t.oid)
AND a.grantee=0))::text") == true ]]

paso 'instalación sin LOGIN, manifiesto, dependencia y reentrada'
huella_base_inicial=$(huella_base)
base_oid=$(psql_valor "SELECT 'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure::oid")
psql_archivo "$up" >/dev/null
oid_inicial=$(psql_valor "SELECT '$fachada(text,text)'::regprocedure::oid")
[[ $(psql_valor "SELECT (p.proowner='vec_identidad_sesiones_v1_propietario'::regrole
AND l.lanname='sql' AND p.provolatile='v' AND p.prosecdef
AND p.proparallel='u' AND NOT p.proisstrict AND p.proretset
AND p.prosqlbody IS NOT NULL
AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=1s']
AND pg_get_function_result(p.oid)=
'TABLE(cuenta_ref text, metodo_observado text, garantia_observada text, identidad_valida_hasta timestamp with time zone)'
)::text FROM pg_proc p JOIN pg_language l ON l.oid=p.prolang
WHERE p.oid='$fachada(text,text)'::regprocedure") == true ]]
[[ $(psql_valor "SELECT count(*) FROM pg_depend WHERE classid='pg_proc'::regclass
AND objid='$fachada(text,text)'::regprocedure AND objsubid=0
AND refclassid='pg_proc'::regclass AND refobjid=$base_oid
AND refobjsubid=0 AND deptype='n'") == 1 ]]
[[ $(psql_valor "WITH d AS (SELECT pg_get_functiondef(
'$fachada(text,text)'::regprocedure) AS cuerpo)
SELECT (least(regexp_instr(cuerpo,'current_setting.*role'),
regexp_instr(cuerpo,'transaction_isolation'),
regexp_instr(cuerpo,'transaction_read_only'),
regexp_instr(cuerpo,'pg_is_in_recovery'))>0
AND greatest(regexp_instr(cuerpo,'current_setting.*role'),
regexp_instr(cuerpo,'transaction_isolation'),
regexp_instr(cuerpo,'transaction_read_only'),
regexp_instr(cuerpo,'pg_is_in_recovery'))<regexp_instr(cuerpo,
'CROSS JOIN LATERAL[[:space:]]+vec_identidad_sesiones_v1[.]revalidar_autenticacion_actor_v1')
)::text FROM d") == true ]]
[[ $(psql_valor "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT count(*) FROM $fachada('aut_inexistente_aaaaaaaaaaaa',
'ses_inexistente_aaaaaaaaaaaa'); COMMIT" | grep -E '^[0-9]+$' | tail -1) == 0 ]]
esperar_fallo_archivo "$up" 'reentrada de 000004'
[[ $(psql_valor "SELECT '$fachada(text,text)'::regprocedure::oid") == "$oid_inicial" ]]
esperar_fallo_archivo \
  deploy/postgresql/identidad_sesiones_v1/migraciones/000003_revalidacion_autenticacion_actor_v1.down.sql \
  '000003 down atravesó la dependencia de 000004'
grep -Fq '2BP01' "$salidas/fallo_archivo"

paso 'LOGIN único y proxy temporal de ContextoActor'
psql_admin <<SQL
CREATE ROLE $login LOGIN PASSWORD '$clave' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT $selector TO $login WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
crear_proxy
esperar_fallo_actor "SELECT * FROM $fachada('x','y')" \
  'llamada directa a Identidad'
esperar_fallo_actor \
  'SELECT * FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(NULL,NULL)' \
  'fachada rica'
esperar_fallo_actor \
  'SELECT * FROM vec_identidad_sesiones_v1.cuenta' 'lectura de tabla'
esperar_fallo_actor "SET ROLE $selector" 'SET ROLE selector'

paso 'fixtures sintéticos y resultado exacto de cuatro columnas'
cuenta=$(psql_valor "SELECT cuenta_ref FROM
vec_identidad_sesiones_v1.provisionar_cuenta_v1(
'opr_'||repeat('a',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('2',64),'hex'),decode(repeat('1',64),'hex'),false,NULL)")
IFS='|' read -r autenticacion sesion _control _revision cuenta_sesion \
  <<<"$(crear_sesion c 3 4 interna_corporativa alto)"
[[ $cuenta == "$cuenta_sesion" ]]
salida=$(actor_valor "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT cuenta_ref||'|'||metodo_observado||'|'||garantia_observada||'|'||
to_char(identidad_valida_hasta AT TIME ZONE 'UTC',
'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')
FROM $proxy('$autenticacion','$sesion'); COMMIT" |
  grep -E "^cta_" | tail -1)
esperada=$(psql_valor "SELECT s.cuenta_ref||'|'||s.metodo_observado||'|'||
s.garantia_observada||'|'||to_char(LEAST(c.sesion_valida_hasta,
s.autenticacion_verificada_en+interval '15 minutes') AT TIME ZONE 'UTC',
'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')
FROM vec_autorizacion.sesion_autenticacion_v1 s
JOIN vec_autorizacion.control_sesion_actual_v1 a USING(sesion_ref)
JOIN vec_autorizacion.control_sesion_v1 c USING(sesion_ref,control_sesion_ref)
WHERE s.autenticacion_ref='$autenticacion' AND s.sesion_ref='$sesion'")
[[ $salida == "$esperada" ]]
exigir_cero NULL NULL 'nulos'
exigir_cero mala mala 'malformados'
exigir_cero aut_aaaaaaaaaaaaaaaaaaaaaa ses_aaaaaaaaaaaaaaaaaaaaaa \
  'inexistentes'
IFS='|' read -r autenticacion_2 sesion_2 control_2 revision_2 _ \
  <<<"$(crear_sesion d 5 6 interna_corporativa alto)"
exigir_cero "$autenticacion" "$sesion_2" 'referencias cruzadas'
exigir_cero "$autenticacion_2" "$sesion" 'referencias cruzadas inversas'
[[ $(actor_valor "SELECT count(*) FROM $proxy('$autenticacion','$sesion')") == 0 ]]
[[ $(actor_valor "BEGIN ISOLATION LEVEL SERIALIZABLE READ ONLY;
SELECT count(*) FROM $proxy('$autenticacion','$sesion'); COMMIT" |
  grep -E '^[0-9]+$' | tail -1) == 0 ]]

paso 'superficie, garantía, cuenta privilegiada, caducidad y revocación'
psql_valor "SET session_replication_role=replica;
UPDATE vec_autorizacion.sesion_autenticacion_v1
SET superficie='externa_personal',garantia_observada='sustancial'
WHERE sesion_ref='$sesion_2'; RESET session_replication_role" >/dev/null
exigir_cero "$autenticacion_2" "$sesion_2" 'superficie/garantía no nominal'
psql_valor "SET session_replication_role=replica;
UPDATE vec_autorizacion.sesion_autenticacion_v1
SET superficie='interna_corporativa',garantia_observada='alto'
WHERE sesion_ref='$sesion_2'; RESET session_replication_role" >/dev/null
[[ $(psql_valor "SELECT vec_identidad_sesiones_v1.revocar_sesion_v1(
'$sesion_2','$control_2','$revision_2','opr_'||repeat('r',24))") == 2 ]]
exigir_cero "$autenticacion_2" "$sesion_2" 'sesión revocada'
IFS='|' read -r autenticacion_3 sesion_3 _control_3 _revision_3 _ \
  <<<"$(crear_sesion e 7 8 interna_corporativa alto)"
psql_valor "SET session_replication_role=replica;
UPDATE vec_autorizacion.sesion_autenticacion_v1
SET autenticacion_verificada_en=date_trunc('microseconds',
 clock_timestamp()-interval '15 minutes')
WHERE sesion_ref='$sesion_3'; RESET session_replication_role" >/dev/null
exigir_cero "$autenticacion_3" "$sesion_3" \
  'frontera exacta de quince minutos'
cuenta_priv=$(psql_valor "SELECT cuenta_ref FROM
vec_identidad_sesiones_v1.provisionar_cuenta_v1(
'opr_'||repeat('p',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('d',64),'hex'),decode(repeat('1',64),'hex'),true,
decode(repeat('2',64),'hex'))")
priv=$(psql_valor "SELECT autenticacion_ref||'|'||sesion_ref FROM
vec_identidad_sesiones_v1.registrar_sesion_v1(
'opr_'||repeat('q',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('e',64),'hex'),decode(repeat('f',64),'hex'),
decode(repeat('1',64),'hex'),decode(repeat('d',64),'hex'),
decode(repeat('2',64),'hex'),true,'administracion_privilegiada',
'kerberos_ad','alto',repeat('9',64),clock_timestamp()-interval '2 seconds',
clock_timestamp()-interval '1 second',clock_timestamp()+interval '4 minutes',
'pga_aaaaaaaaaaaaaaaaaaaaaaaa',repeat('b',64))")
[[ -n $cuenta_priv && $priv == *'|'* ]]
exigir_cero "${priv%%|*}" "${priv#*|}" 'cuenta privilegiada/distinta'

probar_estados_carreras_cardinalidad \
  "$contenedor" "$base" "$clave" "$login" "$proxy" "$salidas"

paso 'ACL efectiva hostil: PUBLIC, herencia, tipos, secuencias y defaults'
psql_valor "GRANT EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)
TO PUBLIC" >/dev/null
exigir_cero "$autenticacion" "$sesion" 'EXECUTE de PUBLIC'
psql_valor "REVOKE EXECUTE ON FUNCTION
vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)
FROM PUBLIC" >/dev/null
psql_valor "GRANT SELECT ON vec_identidad_sesiones_v1.cuenta TO PUBLIC" >/dev/null
exigir_cero "$autenticacion" "$sesion" 'tabla por PUBLIC'
psql_valor "REVOKE SELECT ON vec_identidad_sesiones_v1.cuenta FROM PUBLIC" >/dev/null
psql_valor "GRANT USAGE ON TYPE vec_identidad_sesiones_v1.cuenta TO PUBLIC" >/dev/null
exigir_cero "$autenticacion" "$sesion" 'tipo por PUBLIC'
psql_valor "REVOKE USAGE ON TYPE vec_identidad_sesiones_v1.cuenta FROM PUBLIC" >/dev/null
psql_admin <<'SQL'
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE SEQUENCE vec_identidad_sesiones_v1.c21b_secuencia;
REVOKE ALL ON SEQUENCE vec_identidad_sesiones_v1.c21b_secuencia FROM PUBLIC;
GRANT USAGE ON SEQUENCE vec_identidad_sesiones_v1.c21b_secuencia TO PUBLIC;
SQL
exigir_cero "$autenticacion" "$sesion" 'secuencia por PUBLIC'
psql_valor "SET ROLE vec_identidad_sesiones_v1_propietario;
DROP SEQUENCE vec_identidad_sesiones_v1.c21b_secuencia" >/dev/null
psql_valor "ALTER DEFAULT PRIVILEGES FOR ROLE
vec_identidad_sesiones_v1_propietario IN SCHEMA vec_identidad_sesiones_v1
GRANT EXECUTE ON FUNCTIONS TO PUBLIC" >/dev/null
exigir_cero "$autenticacion" "$sesion" 'ACL predeterminada de PUBLIC'
psql_valor "ALTER DEFAULT PRIVILEGES FOR ROLE
vec_identidad_sesiones_v1_propietario IN SCHEMA vec_identidad_sesiones_v1
REVOKE EXECUTE ON FUNCTIONS FROM PUBLIC" >/dev/null
psql_valor "SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE POLICY c21b_policy ON vec_identidad_sesiones_v1.cuenta
FOR SELECT TO $consumidor USING(false)" >/dev/null
exigir_cero "$autenticacion" "$sesion" 'policy del consumidor'
psql_valor "SET ROLE vec_identidad_sesiones_v1_propietario;
DROP POLICY c21b_policy ON vec_identidad_sesiones_v1.cuenta" >/dev/null
psql_admin <<SQL
CREATE ROLE vec_c21b_puente NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT SELECT ON vec_identidad_sesiones_v1.cuenta TO vec_c21b_puente;
ALTER ROLE $consumidor INHERIT;
GRANT vec_c21b_puente TO $consumidor
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
exigir_cero "$autenticacion" "$sesion" 'lectura por herencia'
psql_admin <<SQL
REVOKE vec_c21b_puente FROM $consumidor;
ALTER ROLE $consumidor NOINHERIT;
REVOKE SELECT ON vec_identidad_sesiones_v1.cuenta FROM vec_c21b_puente;
DROP ROLE vec_c21b_puente;
ALTER ROLE $consumidor SET application_name='hostil';
SQL
exigir_cero "$autenticacion" "$sesion" 'ajuste global del consumidor'
psql_valor "ALTER ROLE $consumidor RESET application_name" >/dev/null
exigir_uno "$autenticacion" "$sesion" 'estado restaurado tras ACL hostiles'

paso 'topología exacta de LOGIN y selector'
psql_admin <<SQL
CREATE ROLE vec_c21b_segundo LOGIN PASSWORD '$clave' NOSUPERUSER
  NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT $selector TO vec_c21b_segundo
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
exigir_cero "$autenticacion" "$sesion" 'segundo LOGIN'
psql_admin <<SQL
REVOKE $selector FROM vec_c21b_segundo;
DROP ROLE vec_c21b_segundo;
CREATE ROLE vec_c21b_transitivo NOLOGIN NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_c21b_transitivo TO $login
  WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
SQL
exigir_cero "$autenticacion" "$sesion" 'membresía adicional'
psql_valor "REVOKE vec_c21b_transitivo FROM $login;
DROP ROLE vec_c21b_transitivo" >/dev/null
psql_valor "ALTER ROLE $login SET application_name='hostil'" >/dev/null
exigir_cero "$autenticacion" "$sesion" 'ajuste del LOGIN'
psql_valor "ALTER ROLE $login RESET application_name" >/dev/null
exigir_uno "$autenticacion" "$sesion" 'topología restaurada'

probar_venenos_acl_topologia \
  "$fachada" "$login" "$selector" "$consumidor" \
  "$autenticacion" "$sesion" "$contenedor" "$base" "$clave" "$salidas"
probar_venenos_fachada_viva \
  "$down" "$fachada" "$consumidor" "$autenticacion" "$sesion" "$salidas"

paso 'timeout, cancelación e interbloqueo no capturados'
fifo_lock="$salidas/fifo_lock"
mkfifo "$fifo_lock"
docker exec -i --env PGAPPNAME=c21b_bloqueo "$contenedor" psql -Xq \
  -v ON_ERROR_STOP=1 -U postgres -d "$base" <"$fifo_lock" \
  >"$salidas/bloqueo" 2>&1 &
pid_lock=$!
exec {fd_lock}>"$fifo_lock"
printf "BEGIN; SELECT pg_advisory_xact_lock(hashtextextended('%s',0));\n" \
  "$marca_rol" >&"$fd_lock"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_bloqueo' AND state='idle in transaction'" 1 \
  'lock exclusivo'
if actor_valor "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT * FROM $proxy('$autenticacion','$sesion'); COMMIT" \
    >"$salidas/timeout" 2>&1; then
  echo 'lock_timeout no se propagó' >&2
  exit 1
fi
grep -Fq '55P03' "$salidas/timeout"
(actor_valor "SET application_name='c21b_cancelada';
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT * FROM $proxy('$autenticacion','$sesion'); COMMIT" \
  >"$salidas/cancelada" 2>&1) &
pid_cancel=$!
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_cancelada' AND wait_event='advisory'" 1 \
  'cancelación sobre advisory'
psql_valor "SELECT pg_cancel_backend(pid) FROM pg_stat_activity
WHERE application_name='c21b_cancelada'" >/dev/null
if wait "$pid_cancel"; then
  echo 'la cancelación fue capturada' >&2
  exit 1
fi
grep -Fq '57014' "$salidas/cancelada"
printf 'COMMIT;\n' >&"$fd_lock"
exec {fd_lock}>&-
wait "$pid_lock"

fifo_actor="$salidas/fifo_dead_actor"
fifo_admin="$salidas/fifo_dead_admin"
mkfifo "$fifo_actor" "$fifo_admin"
psql_valor "ALTER SYSTEM SET deadlock_timeout='100ms'" >/dev/null
psql_valor "SELECT pg_reload_conf()" >/dev/null
docker exec -i --env PGPASSWORD="$clave" "$contenedor" psql -Xq \
  -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -h 127.0.0.1 \
  -U "$login" -d "$base" <"$fifo_actor" >"$salidas/dead_actor" 2>&1 &
pid_dead_actor=$!
docker exec -i --env PGAPPNAME=c21b_dead_admin "$contenedor" psql -Xq \
  -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U postgres -d "$base" \
  <"$fifo_admin" >"$salidas/dead_admin" 2>&1 &
pid_dead_admin=$!
exec {fd_actor}>"$fifo_actor"
exec {fd_admin}>"$fifo_admin"
printf 'BEGIN;\n' >&"$fd_admin"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_dead_admin' AND state='idle in transaction'" 1 \
  'transacción administrativa anterior'
printf "SELECT pg_advisory_xact_lock(hashtextextended('%s',0));\n" \
  "$marca_rol" >&"$fd_admin"
esperar_estado "SELECT count(*) FROM pg_stat_activity
AS a JOIN pg_locks l ON l.pid=a.pid
WHERE a.application_name='c21b_dead_admin'
AND l.locktype='advisory' AND l.granted" 1 \
  'lock administrativo del interbloqueo'
printf "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET application_name='c21b_dead_actor';
SELECT pg_advisory_xact_lock(47121);\n" >&"$fd_actor"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_dead_actor' AND state='idle in transaction'" 1 \
  'lock del actor'
printf "SELECT * FROM %s('%s','%s');\n" \
  "$proxy" "$autenticacion" "$sesion" >&"$fd_actor"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_dead_actor' AND wait_event='advisory'" 1 \
  'actor espera antes de cerrar ciclo'
printf 'SELECT pg_advisory_xact_lock(47121);\n' >&"$fd_admin"
exec {fd_actor}>&-
printf 'COMMIT;\n' >&"$fd_admin"
exec {fd_admin}>&-
set +e
wait "$pid_dead_actor"; estado_dead_actor=$?
wait "$pid_dead_admin"; estado_dead_admin=$?
set -e
[[ $estado_dead_actor -ne 0 || $estado_dead_admin -ne 0 ]]
grep -Fq '40P01' "$salidas/dead_actor"
psql_valor "ALTER SYSTEM RESET deadlock_timeout" >/dev/null
psql_valor "SELECT pg_reload_conf()" >/dev/null

paso '40001, carreras linealizables y barreras C2.1a/C2.5'
psql_admin <<SQL
SET ROLE $consumidor;
CREATE TABLE vec_contexto_actor_v1.c21b_serializable(id integer PRIMARY KEY);
GRANT SELECT,INSERT ON vec_contexto_actor_v1.c21b_serializable TO $selector;
RESET ROLE;
SQL
IFS='|' read -r autenticacion_40001_a sesion_40001_a _ _ _ \
  <<<"$(crear_sesion 1 b c interna_corporativa alto)"
cuenta_40001_b=$(psql_valor "SELECT cuenta_ref FROM
vec_identidad_sesiones_v1.provisionar_cuenta_v1(
'opr_'||repeat('z',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('c',64),'hex'),decode(repeat('a',64),'hex'),false,NULL)")
IFS='|' read -r autenticacion_40001_b sesion_40001_b _ _ _ \
  <<<"$(psql_valor "SELECT autenticacion_ref||'|'||sesion_ref||'|'||
control_sesion_ref||'|'||control_sesion_revision_texto||'|'||cuenta_ref
FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
'opr_'||repeat('2',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('d',64),'hex'),decode(repeat('e',64),'hex'),
decode(repeat('a',64),'hex'),decode(repeat('c',64),'hex'),NULL,false,
'interna_corporativa','kerberos_ad','alto',repeat('2',64),
clock_timestamp()-interval '2 seconds',clock_timestamp()-interval '1 second',
clock_timestamp()+interval '4 minutes','pga_aaaaaaaaaaaaaaaaaaaaaaaa',
repeat('b',64))")"
[[ -n $cuenta_40001_b ]]
fifo_40001_a="$salidas/fifo_40001_a"
fifo_40001_b="$salidas/fifo_40001_b"
mkfifo "$fifo_40001_a" "$fifo_40001_b"
pids_40001=()
for lado in a b; do
  docker exec -i --env PGPASSWORD="$clave" "$contenedor" psql -Xq \
    -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -h 127.0.0.1 \
    -U "$login" -d "$base" <"$salidas/fifo_40001_$lado" \
    >"$salidas/40001_$lado" 2>&1 &
  pids_40001+=("$!")
done
exec {fd_40001_a}>"$fifo_40001_a"
exec {fd_40001_b}>"$fifo_40001_b"
for lado in a b; do
  if [[ $lado == a ]]; then
    autenticacion_40001=$autenticacion_40001_a
    sesion_40001=$sesion_40001_a
    fd_40001=$fd_40001_a
  else
    autenticacion_40001=$autenticacion_40001_b
    sesion_40001=$sesion_40001_b
    fd_40001=$fd_40001_b
  fi
  printf "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET application_name='c21b_40001_%s';
SELECT count(*) FROM %s('%s','%s');
SELECT count(*) FROM vec_contexto_actor_v1.c21b_serializable;\n" \
    "$lado" "$proxy" "$autenticacion_40001" "$sesion_40001" >&"$fd_40001"
done
for lado in a b; do
  esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_40001_$lado' AND state='idle in transaction'" 1 \
    "lectura serializable $lado"
done
printf 'INSERT INTO vec_contexto_actor_v1.c21b_serializable VALUES(1);
COMMIT;\n' >&"$fd_40001_a"
exec {fd_40001_a}>&-
wait "${pids_40001[0]}"
printf 'INSERT INTO vec_contexto_actor_v1.c21b_serializable VALUES(2);
COMMIT;\n' >&"$fd_40001_b"
exec {fd_40001_b}>&-
set +e
wait "${pids_40001[1]}"; estado_40001=$?
set -e
[[ $estado_40001 -ne 0 ]]
grep -Fq '40001' "$salidas/40001_b"
psql_valor "SET ROLE $consumidor;
DROP TABLE vec_contexto_actor_v1.c21b_serializable" >/dev/null

IFS='|' read -r autenticacion_r sesion_r control_r revision_r _ \
  <<<"$(crear_sesion f 9 a interna_corporativa alto)"
fifo_race="$salidas/fifo_race"
mkfifo "$fifo_race"
docker exec -i --env PGPASSWORD="$clave" "$contenedor" psql -XAtq \
  -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$login" -d "$base" \
  <"$fifo_race" >"$salidas/race_actor" 2>&1 &
pid_race=$!
exec {fd_race}>"$fifo_race"
printf "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET application_name='c21b_race_actor';
SELECT count(*) FROM %s('%s','%s');\n" \
  "$proxy" "$autenticacion_r" "$sesion_r" >&"$fd_race"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_race_actor' AND state='idle in transaction'" 1 \
  'revalidación antes de revocación'
(psql_valor "SET application_name='c21b_race_revoca';
SELECT vec_identidad_sesiones_v1.revocar_sesion_v1(
'$sesion_r','$control_r','$revision_r','opr_'||repeat('s',24))" \
  >"$salidas/race_revoca" 2>&1) &
pid_revoca=$!
esperar_estado "SELECT count(*) FROM pg_stat_activity x
WHERE x.application_name='c21b_race_revoca'
AND cardinality(pg_blocking_pids(x.pid))=1" 1 'revocación espera locks'
printf 'COMMIT;\n' >&"$fd_race"
exec {fd_race}>&-
wait "$pid_race"
wait "$pid_revoca"
grep -Fxq 1 "$salidas/race_actor"
exigir_cero "$autenticacion_r" "$sesion_r" 'revocación ganadora'

fifo_c21a="$salidas/fifo_c21a"
mkfifo "$fifo_c21a"
docker exec -i --env PGPASSWORD="$clave" "$contenedor" psql -XAtq \
  -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$login" -d "$base" \
  <"$fifo_c21a" >"$salidas/c21a_actor" 2>&1 &
pid_c21a=$!
exec {fd_c21a}>"$fifo_c21a"
printf "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET application_name='c21b_c21a_actor';
SELECT count(*) FROM %s('%s','%s');\n" \
  "$proxy" "$autenticacion" "$sesion" >&"$fd_c21a"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_c21a_actor'
AND state='idle in transaction'" 1 'C2.1b conserva shared'
(docker exec -i --env PGAPPNAME=c21b_down_rol "$contenedor" psql -Xq \
  -v ON_ERROR_STOP=1 -v VERBOSITY=verbose -U postgres -d "$base" \
  <"$raiz/$rol_down" >"$salidas/down_rol" 2>&1) &
pid_down_rol=$!
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_down_rol' AND wait_event='advisory'
AND cardinality(pg_blocking_pids(pid))=1" 1 'C2.1a espera C2.1b'
printf 'COMMIT;\n' >&"$fd_c21a"
exec {fd_c21a}>&-
if wait "$pid_down_rol"; then
  echo 'C2.1a down atravesó la fachada viva' >&2
  exit 1
fi
wait "$pid_c21a"
grep -Fxq 1 "$salidas/c21a_actor"
[[ $(psql_valor "SELECT (to_regrole('$selector') IS NOT NULL)::text") == true ]]

fifo_c25="$salidas/fifo_c25"
mkfifo "$fifo_c25"
docker exec -i --env PGAPPNAME=c21b_c25 "$contenedor" psql -Xq \
  -v ON_ERROR_STOP=1 -U postgres -d "$base" <"$fifo_c25" \
  >"$salidas/c25" 2>&1 &
pid_c25=$!
exec {fd_c25}>"$fifo_c25"
printf "BEGIN; SELECT pg_advisory_xact_lock_shared(hashtextextended(
'%s',0)); SET ROLE %s;
CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_corporativo_rrhh_v1()
RETURNS boolean LANGUAGE sql AS 'SELECT false';\n" \
  "$marca_fachada" "$consumidor" >&"$fd_c25"
esperar_estado "SELECT count(*) FROM pg_stat_activity
WHERE application_name='c21b_c25' AND state='idle in transaction'" 1 \
  'consumidor C2.5'
(psql_archivo "$down" >"$salidas/down_c25" 2>&1) &
pid_down_c25=$!
esperar_estado "SELECT count(*) FROM pg_stat_activity a
WHERE a.pid<>(SELECT pid FROM pg_stat_activity WHERE
 application_name='c21b_c25') AND a.wait_event='advisory'
AND cardinality(pg_blocking_pids(a.pid))=1" 1 'down espera C2.5'
printf 'COMMIT;\n' >&"$fd_c25"
exec {fd_c25}>&-
wait "$pid_c25"
if wait "$pid_down_c25"; then
  echo 'down atravesó el consumidor C2.5' >&2
  exit 1
fi
psql_valor "SET ROLE $consumidor; DROP FUNCTION
vec_contexto_actor_v1.resolver_y_registrar_contexto_corporativo_rrhh_v1()" \
  >/dev/null

paso 'safe-down: cuerpo, configuración, propietario y dependencia'
definicion=$(psql_valor "SELECT pg_get_functiondef('$fachada(text,text)'::regprocedure)")
psql_valor "ALTER FUNCTION $fachada(text,text) SET lock_timeout='2s'" >/dev/null
esperar_fallo_archivo "$down" 'configuración alterada'
psql_valor "ALTER FUNCTION $fachada(text,text) SET lock_timeout='1s'" >/dev/null
psql_valor "ALTER FUNCTION $fachada(text,text) OWNER TO postgres" >/dev/null
esperar_fallo_archivo "$down" 'propietario alterado'
psql_valor "ALTER FUNCTION $fachada(text,text)
OWNER TO vec_identidad_sesiones_v1_propietario" >/dev/null
psql_admin <<SQL
SET ROLE vec_identidad_sesiones_v1_propietario;
CREATE OR REPLACE FUNCTION $fachada(
 p_autenticacion_ref text,p_sesion_ref text)
RETURNS TABLE(cuenta_ref text,metodo_observado text,
 garantia_observada text,identidad_valida_hasta timestamptz)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='1s'
BEGIN ATOMIC SELECT NULL::text,NULL::text,NULL::text,NULL::timestamptz; END;
SQL
esperar_fallo_archivo "$down" 'cuerpo alterado'
printf '%s\n' "$definicion" | psql_admin >/dev/null
esperar_fallo_archivo "$down" 'dependencia proxy RESTRICT'
eliminar_proxy

probar_safe_down_consumidores_efectivos \
  "$up" "$down" "$fachada" "$consumidor" "$oid_inicial"
[[ $(huella_base) == "$huella_base_inicial" ]]
[[ $(psql_valor "SELECT 'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure::oid") == "$base_oid" ]]
crear_proxy
exigir_uno "$autenticacion" "$sesion" 'up-down-up'

paso 'reconexión, reinicio y estado durable'
exigir_uno "$autenticacion" "$sesion" 'reconexión nueva'
docker restart "$contenedor" >/dev/null
esperar_postgresql_definitivo "$contenedor" "$base" "$clave"
[[ $(psql_valor "SELECT current_setting('server_version_num')") == 180004 ]]
exigir_uno "$autenticacion" "$sesion" 'reinicio'
[[ $(huella_base) == "$huella_base_inicial" ]]

paso 'retirada final protegida y base intacta'
eliminar_proxy
psql_archivo "$down" >/dev/null
[[ $(psql_valor "SELECT (NOT has_schema_privilege('$consumidor',
'vec_identidad_sesiones_v1','USAGE'))::text") == true ]]
esperar_fallo_archivo "$down" 'reentrada del down'
[[ $(huella_base) == "$huella_base_inicial" ]]
[[ $(psql_valor "SELECT 'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure::oid") == "$base_oid" ]]

echo 'OK: fachada de revalidación corporativa RRHH 000004 en PostgreSQL 18.4'
