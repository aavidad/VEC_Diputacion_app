#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-rol-selector-contexto-rrhh-${USER:-usuario}-$$"
base=vec_rol_selector_contexto_rrhh
rol=vec_contexto_actor_corporativo_rrhh_selector
marca=vec_contexto_actor_v1:rol-contexto-corporativo-rrhh-selector:v1
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
salidas=$(mktemp -d)
up=deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql
down=deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_down.sql
numero_carrera=0

limpiar() {
  local estado=$?
  if [[ $estado -ne 0 ]]; then
    docker logs --tail 200 "$contenedor" 2>&1 |
      grep -E ' (LOG|ERROR|FATAL): ' | tail -80 >&2 || true
  fi
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  rm -rf "$salidas"
}
trap limpiar EXIT INT TERM

docker run --detach --name "$contenedor" \
  --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave" \
  "$imagen" >/dev/null
disponible=false
for _ in $(seq 1 120); do
  if docker exec "$contenedor" pg_isready -U postgres -d "$base" \
      >/dev/null 2>&1; then
    disponible=true
    break
  fi
  sleep 1
done
[[ $disponible == true ]] || {
  docker logs "$contenedor" >&2 || true
  echo 'PostgreSQL 18.4 no quedó disponible' >&2
  exit 1
}
[[ $(docker exec "$contenedor" psql -XAtq -U postgres -d "$base" \
  -c "SELECT current_setting('server_version_num')") == 180004 ]]

psql_archivo() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base" <"$raiz/$1"
}
psql_valor() {
  docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
    -U postgres -d "$base" -c "$1"
}
psql_valor_base() {
  local base_destino=$1 sql=$2
  docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
    -U postgres -d "$base_destino" -c "$sql"
}
psql_sql() {
  docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base"
}
psql_archivo_como() {
  local principal=$1 archivo=$2
  {
    printf 'SET ROLE %s;\n' "$principal"
    sed -n '1,$p' "$raiz/$archivo"
  } | docker exec --interactive "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -U postgres -d "$base"
}
exigir_fallo_up() {
  local caso=$1
  if psql_archivo "$up" >"$salidas/fallo_up" 2>&1; then
    echo "el alta aceptó un estado hostil: $caso" >&2
    exit 1
  fi
  grep -Fq 'alta del selector corporativo RRHH rechazada' \
    "$salidas/fallo_up"
}
exigir_fallo_down() {
  local caso=$1
  if psql_archivo "$down" >"$salidas/fallo_down" 2>&1; then
    echo "la retirada aceptó un estado hostil: $caso" >&2
    exit 1
  fi
}
exigir_rol() {
  [[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_authid WHERE rolname='$rol'") == true ]]
}
exigir_ausencia() {
  [[ $(psql_valor "SELECT (count(*)=0)::text FROM pg_authid WHERE rolname='$rol'") == true ]]
}
exigir_ausencia_oid() {
  local oid_anterior=$1
  exigir_ausencia
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting WHERE setrole=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE roleid=$oid_anterior OR member=$oid_anterior OR grantor=$oid_anterior) AND NOT EXISTS(SELECT 1 FROM pg_shseclabel WHERE classoid='pg_authid'::regclass AND objoid=$oid_anterior))::text") == true ]]
}
huella_objetivo() {
  {
    psql_valor "SELECT rolname,rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolinherit,rolreplication,rolbypassrls,rolconnlimit,COALESCE(rolpassword,'<NULL>'),COALESCE(rolvaliduntil::text,'<NULL>'),COALESCE(shobj_description(oid,'pg_authid'),'<NULL>') FROM pg_authid WHERE rolname='$rol'"
    psql_valor "SELECT setdatabase,setrole,setconfig FROM pg_db_role_setting WHERE setrole='$rol'::regrole ORDER BY 1,2,3"
    psql_valor "SELECT roleid,member,grantor,admin_option,inherit_option,set_option FROM pg_auth_members WHERE roleid='$rol'::regrole OR member='$rol'::regrole OR grantor='$rol'::regrole ORDER BY 1,2,3"
    psql_valor "SELECT dbid,classid,objid,objsubid,refclassid,deptype FROM pg_shdepend WHERE refclassid='pg_authid'::regclass AND refobjid='$rol'::regrole ORDER BY 1,2,3,4,6"
    psql_valor "SELECT classoid,objoid,provider,label FROM pg_shseclabel WHERE classoid='pg_authid'::regclass AND objoid='$rol'::regrole ORDER BY 1,3,4"
  } | sha256sum | cut -d' ' -f1
}
huella_contexto() {
  local filtro="ARRAY['vec_contexto_actor_v1_propietario'::regrole::oid,'vec_contexto_actor_v1_migrador'::regrole::oid,'vec_contexto_actor_v1_runtime'::regrole::oid]"
  {
    psql_valor "SELECT rolname,rolcanlogin,rolsuper,rolcreatedb,rolcreaterole,rolinherit,rolreplication,rolbypassrls,rolconnlimit,COALESCE(shobj_description(oid,'pg_authid'),'<NULL>') FROM pg_authid WHERE oid=ANY($filtro) ORDER BY rolname"
    psql_valor "SELECT setdatabase,setrole,setconfig FROM pg_db_role_setting WHERE setrole=ANY($filtro) ORDER BY 1,2,3"
    psql_valor "SELECT roleid,member,grantor,admin_option,inherit_option,set_option FROM pg_auth_members WHERE roleid=ANY($filtro) OR member=ANY($filtro) OR grantor=ANY($filtro) ORDER BY 1,2,3"
    psql_valor "SELECT n.oid,n.nspowner,n.nspacl FROM pg_namespace n WHERE n.nspname='vec_contexto_actor_v1'"
    psql_valor "SELECT c.oid,c.relowner,c.relrowsecurity,c.relforcerowsecurity,c.relacl FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='vec_contexto_actor_v1' ORDER BY c.oid"
    psql_valor "SELECT p.oid,p.proowner,p.prosecdef,p.proconfig,p.proacl FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contexto_actor_v1' ORDER BY p.oid"
    psql_valor "SELECT defaclrole,defaclnamespace,defaclobjtype,defaclacl FROM pg_default_acl WHERE defaclrole=ANY($filtro) ORDER BY 1,2,3"
  } | sha256sum | cut -d' ' -f1
}
exigir_aislamiento() {
  [[ $(psql_valor "SELECT (count(*)=1 AND bool_and(d.datname=current_database() AND a.grantor=d.datdba AND a.privilege_type='CONNECT' AND NOT a.is_grantable))::text FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) a WHERE a.grantee='$rol'::regrole OR a.grantor='$rol'::regrole") == true ]]
  [[ $(psql_valor "SELECT vec_contexto_actor_v1.privilegios_efectivos_runtime_minimos('$rol'::regrole,(SELECT oid FROM pg_database WHERE datname=current_database()),0::oid,ARRAY[]::oid[])::text") == true ]]
  [[ $(psql_valor "SELECT (NOT EXISTS(SELECT 1 FROM pg_database d WHERE NOT d.datistemplate AND d.datname<>current_database() AND has_database_privilege('$rol',d.oid,'CONNECT')) AND NOT EXISTS(SELECT 1 FROM pg_default_acl a LEFT JOIN LATERAL aclexplode(a.defaclacl) x ON true WHERE a.defaclrole='$rol'::regrole OR x.grantee='$rol'::regrole OR x.grantor='$rol'::regrole) AND NOT EXISTS(SELECT 1 FROM pg_policy WHERE '$rol'::regrole=ANY(polroles)) AND NOT EXISTS(SELECT 1 FROM pg_shseclabel WHERE classoid='pg_authid'::regclass AND objoid='$rol'::regrole))::text") == true ]]
}
exigir_fallo_selector() {
  local caso=$1 sql=$2
  if {
    printf 'SET SESSION AUTHORIZATION %s;\n' "$rol"
    printf '%s\n' "$sql"
  } | docker exec --interactive "$contenedor" psql -Xq \
      --set ON_ERROR_STOP=1 -U postgres -d "$base" \
      >"$salidas/fallo_selector" 2>&1; then
    echo "el selector ejecutó una operación prohibida: $caso" >&2
    exit 1
  fi
}

# Ejecuta una mutacion mientras el down conserva authid/auth_members y espera
# pg_database. Solo una transaccion puede ganar; el estado final se normaliza.
carrera_catalogo() {
  local caso=$1 mutacion=$2 limpieza=$3
  local fifo fifo_mutacion app_barrera app_down app_mutacion
  local pid_barrera pid_down pid_mut descriptor descriptor_mutacion
  local estado_down estado_mut intento ruta_estado
  local estado_observado=false mutacion_conectada=false barrera_lista=false
  numero_carrera=$((numero_carrera + 1))
  fifo="$salidas/fifo_catalogo_$numero_carrera"
  fifo_mutacion="$salidas/fifo_mutacion_$numero_carrera"
  app_barrera="c21a_barrera_$numero_carrera"
  app_down="c21a_down_$numero_carrera"
  app_mutacion="c21a_mutacion_$numero_carrera"
  mkfifo "$fifo" "$fifo_mutacion"
  docker exec --interactive --env PGAPPNAME="$app_barrera" "$contenedor" \
    psql -XAtq --set ON_ERROR_STOP=1 -U postgres -d "$base" \
    <"$fifo" >"$salidas/barrera_$numero_carrera" 2>&1 &
  pid_barrera=$!
  exec {descriptor}>"$fifo"
  printf 'BEGIN;\nLOCK TABLE pg_catalog.pg_database IN ACCESS SHARE MODE;\n' \
    >&"$descriptor"
  for _ in $(seq 1 200); do
    if [[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid WHERE a.application_name='$app_barrera' AND l.relation='pg_database'::regclass AND l.mode='AccessShareLock' AND l.granted") == true ]]; then
      barrera_lista=true
      break
    fi
    sleep 0.02
  done
  [[ $barrera_lista == true ]]
  docker exec --interactive --env PGAPPNAME="$app_mutacion" "$contenedor" \
    psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" \
    <"$fifo_mutacion" >"$salidas/mutacion_$numero_carrera" 2>&1 &
  pid_mut=$!
  exec {descriptor_mutacion}>"$fifo_mutacion"
  printf 'SELECT 1;\n' >&"$descriptor_mutacion"
  for _ in $(seq 1 200); do
    if [[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_stat_activity WHERE application_name='$app_mutacion'") == true ]]; then
      mutacion_conectada=true
      break
    fi
    sleep 0.02
  done
  [[ $mutacion_conectada == true ]]
  (docker exec --interactive --env PGAPPNAME="$app_down" "$contenedor" \
    psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" \
    <"$raiz/$down" >"$salidas/down_$numero_carrera" 2>&1) &
  pid_down=$!
  for intento in $(seq 1 200); do
    ruta_estado="/tmp/c21a_estado_${numero_carrera}_$intento"
    printf 'SELECT pg_stat_clear_snapshot();\n\\o %s\n' \
      "$ruta_estado" >&"$descriptor"
    cat >&"$descriptor" <<SQL
WITH a AS MATERIALIZED (
  SELECT * FROM pg_stat_get_activity(NULL::integer)
), l AS MATERIALIZED (
  SELECT * FROM pg_lock_status()
)
SELECT concat(
  (SELECT count(*) FROM a WHERE application_name='$app_down'),
  '|',
  COALESCE((SELECT wait_event_type FROM a
    WHERE application_name='$app_down'),'<nulo>'),
  '|',
  COALESCE((SELECT cardinality(pg_blocking_pids(pid)) FROM a
    WHERE application_name='$app_down'),0),
  '|',
  COALESCE((SELECT count(*) FROM l
    WHERE pid=(SELECT pid FROM a WHERE application_name='$app_down')
      AND granted AND mode='AccessExclusiveLock'
      AND relation=ANY(ARRAY[
        'pg_authid'::regclass::oid,
        'pg_auth_members'::regclass::oid
      ])),0),
  '|',
  COALESCE((SELECT count(*) FROM l
    WHERE pid=(SELECT pid FROM a WHERE application_name='$app_down')
      AND NOT granted AND mode='AccessExclusiveLock'
      AND relation='pg_database'::regclass::oid),0)
);
\o
SQL
    for _ in $(seq 1 20); do
      docker exec "$contenedor" test -s "$ruta_estado" && break
      sleep 0.01
    done
    estado_observado=$(docker exec "$contenedor" sh -c \
      "cat '$ruta_estado' 2>/dev/null || true")
    [[ $estado_observado == '1|Lock|1|2|1' ]] && break
    sleep 0.02
  done
  [[ $estado_observado == '1|Lock|1|2|1' ]] || {
    echo "down no alcanzó la barrera causal: $caso (${estado_observado:-vacío})" >&2
    sed -n '1,120p' "$salidas/down_$numero_carrera" >&2
    return 1
  }
  printf '%s;\n' "$mutacion" >&"$descriptor_mutacion"
  eval "exec ${descriptor_mutacion}>&-"
  printf 'COMMIT;\n' >&"$descriptor"
  eval "exec ${descriptor}>&-"
  wait "$pid_barrera"
  set +e
  wait "$pid_down"; estado_down=$?
  wait "$pid_mut"; estado_mut=$?
  set -e
  if [[ $estado_down -eq 0 && $estado_mut -ne 0 ]]; then
    exigir_ausencia
    psql_archivo "$up" >/dev/null
  elif [[ $estado_down -ne 0 && $estado_mut -eq 0 ]]; then
    exigir_rol
    psql_valor "$limpieza" >/dev/null
  else
    echo "resultado no linealizable en carrera $caso: down=$estado_down mutacion=$estado_mut" >&2
    exit 1
  fi
  exigir_aislamiento
}

carrera_fachada() {
  local caso=$1 esquema=$2 funcion=$3
  local fifo app pid_futura pid_down descriptor espera=false
  numero_carrera=$((numero_carrera + 1))
  fifo="$salidas/fifo_fachada_$numero_carrera"
  app="c21a_fachada_$numero_carrera"
  mkfifo "$fifo"
  docker exec --interactive --env PGAPPNAME="$app" "$contenedor" \
    psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" \
    <"$fifo" >"$salidas/fachada_$numero_carrera" 2>&1 &
  pid_futura=$!
  exec {descriptor}>"$fifo"
  cat >&"$descriptor" <<SQL
BEGIN;
SELECT pg_advisory_xact_lock_shared(hashtextextended('$marca',0));
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION $esquema.$funcion(timestamptz)
RETURNS text LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog
AS 'SELECT NULL::text';
SQL
  for _ in $(seq 1 200); do
    if [[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_stat_activity WHERE application_name='$app' AND state='idle in transaction'") == true ]]; then
      break
    fi
    sleep 0.02
  done
  (docker exec --interactive --env PGAPPNAME="c21a_down_fachada_$numero_carrera" \
    "$contenedor" psql -Xq --set ON_ERROR_STOP=1 -U postgres -d "$base" \
    <"$raiz/$down" >"$salidas/down_fachada_$numero_carrera" 2>&1) &
  pid_down=$!
  for _ in $(seq 1 200); do
    if [[ $(psql_valor "SELECT (count(*)=1)::text FROM pg_stat_activity WHERE application_name='c21a_down_fachada_$numero_carrera' AND wait_event='advisory'") == true ]]; then
      espera=true
      break
    fi
    sleep 0.02
  done
  [[ $espera == true ]]
  printf 'COMMIT;\n' >&"$descriptor"
  eval "exec ${descriptor}>&-"
  wait "$pid_futura"
  if wait "$pid_down"; then
    echo "el down atravesó la barrera compartida: $caso" >&2
    exit 1
  fi
  exigir_rol
  [[ $(psql_valor "SELECT (to_regprocedure('$esquema.$funcion(timestamptz)') IS NOT NULL)::text") == true ]]
  psql_valor "SET ROLE vec_contexto_actor_v1_propietario; DROP FUNCTION $esquema.$funcion(timestamptz)" >/dev/null
}

# La base sin endurecer se rechaza antes de crear cualquier rol.
exigir_fallo_up 'base con privilegios de PUBLIC'
exigir_ausencia

psql_sql <<SQL
CREATE ROLE vec_c21a_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS;
ALTER DATABASE $base OWNER TO vec_c21a_dueno_base;
REVOKE ALL PRIVILEGES ON DATABASE $base FROM PUBLIC;
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
for archivo in \
  deploy/postgresql/contexto_actor_v1/roles_up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql \
  deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
do
  psql_archivo "$archivo" >/dev/null
done
psql_valor "CREATE ROLE vec_c21a_operador NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS" >/dev/null
huella_contexto_inicial=$(huella_contexto)

# Ejecutor no superusuario, topologia base degradada y PUBLIC hostil.
if psql_archivo_como vec_c21a_operador "$up" >/dev/null 2>&1; then
  echo 'un no-superusuario instaló el selector' >&2
  exit 1
fi
exigir_ausencia
psql_valor "ALTER ROLE vec_contexto_actor_v1_runtime LOGIN" >/dev/null
exigir_fallo_up 'atributo de rol base'
psql_valor "ALTER ROLE vec_contexto_actor_v1_runtime NOLOGIN" >/dev/null
psql_valor "ALTER FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1() SECURITY INVOKER" >/dev/null
exigir_fallo_up 'SECURITY INVOKER base'
psql_valor "ALTER FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1() SECURITY DEFINER" >/dev/null
psql_valor "ALTER FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1() RESET ALL" >/dev/null
exigir_fallo_up 'search_path base'
psql_valor "ALTER FUNCTION vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1() SET search_path=pg_catalog" >/dev/null
psql_valor "CREATE TABLE public.c21a_publica(id integer); GRANT SELECT ON public.c21a_publica TO PUBLIC" >/dev/null
exigir_fallo_up 'privilegio efectivo de PUBLIC'
psql_valor "REVOKE SELECT ON public.c21a_publica FROM PUBLIC; DROP TABLE public.c21a_publica" >/dev/null
[[ $(huella_contexto) == "$huella_contexto_inicial" ]]

# Homonimos exacto y hostil nunca se adoptan.
psql_sql <<SQL
CREATE ROLE $rol NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
  NOREPLICATION NOBYPASSRLS;
COMMENT ON ROLE $rol IS '$marca';
GRANT CONNECT ON DATABASE $base TO $rol;
SQL
huella_homonimo=$(huella_objetivo)
exigir_fallo_up 'homónimo exacto'
[[ $(huella_objetivo) == "$huella_homonimo" ]]
psql_valor "REVOKE CONNECT ON DATABASE $base FROM $rol; DROP ROLE $rol" >/dev/null
psql_valor "CREATE ROLE $rol LOGIN SUPERUSER CREATEDB CREATEROLE INHERIT REPLICATION BYPASSRLS" >/dev/null
exigir_fallo_up 'homónimo hostil'
[[ $(psql_valor "SELECT (rolcanlogin AND rolsuper AND rolcreatedb AND rolcreaterole AND rolreplication AND rolbypassrls)::text FROM pg_authid WHERE rolname='$rol'") == true ]]
psql_valor "DROP ROLE $rol" >/dev/null

# Dos altas concurrentes: un exito, un rechazo y un unico rol.
set +e
(psql_archivo "$up" >"$salidas/up_a" 2>&1) & pid_a=$!
(psql_archivo "$up" >"$salidas/up_b" 2>&1) & pid_b=$!
wait "$pid_a"; estado_a=$?
wait "$pid_b"; estado_b=$?
set -e
[[ $((estado_a + estado_b)) -ne 0 ]]
rechazos=$(grep -lF 'alta del selector corporativo RRHH rechazada' \
  "$salidas/up_a" "$salidas/up_b" | wc -l || true)
if [[ $rechazos != 1 ]]; then
  sed -n '1,160p' "$salidas/up_a" "$salidas/up_b" >&2
  exit 1
fi
exigir_rol
exigir_aislamiento
huella_instalada=$(huella_objetivo)
exigir_fallo_up 'reentrada'
[[ $(huella_objetivo) == "$huella_instalada" ]]
if psql_archivo_como vec_c21a_operador "$down" >/dev/null 2>&1; then
  echo 'un no-superusuario retiró el selector' >&2
  exit 1
fi
[[ $(huella_objetivo) == "$huella_instalada" ]]

# Manifiesto completo: atributos, comentario, contraseña, vigencia y ajustes.
while IFS='|' read -r mutacion restauracion caso; do
  psql_valor "ALTER ROLE $rol $mutacion" >/dev/null
  exigir_fallo_down "$caso"
  psql_valor "ALTER ROLE $rol $restauracion" >/dev/null
done <<'ATRIBUTOS'
LOGIN|NOLOGIN|LOGIN
SUPERUSER|NOSUPERUSER|SUPERUSER
CREATEDB|NOCREATEDB|CREATEDB
CREATEROLE|NOCREATEROLE|CREATEROLE
INHERIT|NOINHERIT|INHERIT
REPLICATION|NOREPLICATION|REPLICATION
BYPASSRLS|NOBYPASSRLS|BYPASSRLS
CONNECTION LIMIT 2|CONNECTION LIMIT -1|límite de conexiones
ATRIBUTOS
psql_valor "COMMENT ON ROLE $rol IS 'marca hostil'" >/dev/null
exigir_fallo_down 'comentario'
psql_valor "COMMENT ON ROLE $rol IS '$marca'" >/dev/null
docker exec --interactive --env CLAVE="$clave" "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<SQL
\getenv clave CLAVE
ALTER ROLE $rol PASSWORD :'clave';
SQL
exigir_fallo_down 'contraseña'
psql_valor "ALTER ROLE $rol PASSWORD NULL" >/dev/null
psql_valor "ALTER ROLE $rol VALID UNTIL 'infinity'" >/dev/null
exigir_fallo_down 'caducidad'
psql_valor "REVOKE CONNECT ON DATABASE $base FROM $rol; DROP ROLE $rol" >/dev/null
psql_archivo "$up" >/dev/null
psql_valor "ALTER ROLE $rol SET application_name='hostil'" >/dev/null
exigir_fallo_down 'ajuste global'
psql_valor "ALTER ROLE $rol RESET application_name" >/dev/null
psql_valor "ALTER ROLE $rol IN DATABASE $base SET application_name='hostil'" >/dev/null
exigir_fallo_down 'ajuste por base'
psql_valor "ALTER ROLE $rol IN DATABASE $base RESET application_name" >/dev/null

# ACL de todas las clases en la base actual.
psql_sql <<'SQL'
CREATE TABLE public.c21a_objeto(id integer, dato text);
CREATE SEQUENCE public.c21a_secuencia;
CREATE FUNCTION public.c21a_funcion() RETURNS integer LANGUAGE sql
  AS 'SELECT 1';
CREATE DOMAIN public.c21a_tipo AS text;
REVOKE ALL ON FUNCTION public.c21a_funcion() FROM PUBLIC;
REVOKE ALL ON TYPE public.c21a_objeto, public.c21a_tipo FROM PUBLIC;
SQL
psql_valor "GRANT SELECT ON public.c21a_objeto TO PUBLIC" >/dev/null
exigir_fallo_down 'privilegio efectivo de PUBLIC tras el alta'
psql_valor "REVOKE SELECT ON public.c21a_objeto FROM PUBLIC" >/dev/null
while IFS='|' read -r concesion retirada caso; do
  psql_valor "$concesion" >/dev/null
  exigir_fallo_down "$caso"
  psql_valor "$retirada" >/dev/null
done <<ACL
GRANT CREATE ON DATABASE $base TO $rol|REVOKE CREATE ON DATABASE $base FROM $rol|base
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO $rol|REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM $rol|esquema
GRANT SELECT ON TABLE public.c21a_objeto TO $rol|REVOKE SELECT ON TABLE public.c21a_objeto FROM $rol|tabla
GRANT SELECT(dato) ON TABLE public.c21a_objeto TO $rol|REVOKE SELECT(dato) ON TABLE public.c21a_objeto FROM $rol|columna
GRANT USAGE ON SEQUENCE public.c21a_secuencia TO $rol|REVOKE USAGE ON SEQUENCE public.c21a_secuencia FROM $rol|secuencia
GRANT EXECUTE ON FUNCTION public.c21a_funcion() TO $rol|REVOKE EXECUTE ON FUNCTION public.c21a_funcion() FROM $rol|función
GRANT USAGE ON TYPE public.c21a_tipo TO $rol|REVOKE USAGE ON TYPE public.c21a_tipo FROM $rol|tipo
ACL

# Otra base aporta ACL de base y de objeto, ambas visibles en pg_shdepend.
psql_valor "CREATE DATABASE vec_c21a_ajena" >/dev/null
psql_valor "REVOKE ALL PRIVILEGES ON DATABASE vec_c21a_ajena FROM PUBLIC" >/dev/null
psql_valor_base vec_c21a_ajena "REVOKE ALL ON SCHEMA public FROM PUBLIC; CREATE TABLE public.objeto_ajeno(id integer); GRANT SELECT ON public.objeto_ajeno TO $rol" >/dev/null
exigir_fallo_down 'ACL de objeto en otra base'
psql_valor_base vec_c21a_ajena "REVOKE SELECT ON public.objeto_ajeno FROM $rol" >/dev/null
psql_valor "GRANT CONNECT ON DATABASE vec_c21a_ajena TO $rol" >/dev/null
exigir_fallo_down 'ACL de otra base'
psql_valor "REVOKE CONNECT ON DATABASE vec_c21a_ajena FROM $rol" >/dev/null

# Default ACL, policy, LO, FDW, servidor, lenguaje, tablespace y parametro.
psql_valor "CREATE ROLE vec_c21a_grupo NOLOGIN; CREATE ROLE vec_c21a_miembro NOLOGIN" >/dev/null
psql_valor "ALTER DEFAULT PRIVILEGES FOR ROLE $rol GRANT SELECT ON TABLES TO vec_c21a_grupo" >/dev/null
exigir_fallo_down 'default ACL como propietario y otorgante'
psql_valor "ALTER DEFAULT PRIVILEGES FOR ROLE $rol REVOKE SELECT ON TABLES FROM vec_c21a_grupo" >/dev/null
psql_valor "ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario GRANT SELECT ON TABLES TO $rol" >/dev/null
exigir_fallo_down 'default ACL como beneficiario'
psql_valor "ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario REVOKE SELECT ON TABLES FROM $rol" >/dev/null
psql_valor "SET ROLE vec_contexto_actor_v1_propietario; CREATE POLICY c21a_policy ON vec_contexto_actor_v1.procedencias FOR SELECT TO $rol USING (false)" >/dev/null
exigir_fallo_down 'política RLS'
psql_valor "SET ROLE vec_contexto_actor_v1_propietario; DROP POLICY c21a_policy ON vec_contexto_actor_v1.procedencias" >/dev/null
oid_lo=$(psql_valor "SELECT lo_create(0)")
psql_valor "GRANT SELECT ON LARGE OBJECT $oid_lo TO $rol" >/dev/null
exigir_fallo_down 'objeto grande'
psql_valor "REVOKE SELECT ON LARGE OBJECT $oid_lo FROM $rol; SELECT lo_unlink($oid_lo)" >/dev/null
psql_valor "CREATE FOREIGN DATA WRAPPER c21a_fdw NO HANDLER; CREATE SERVER c21a_server FOREIGN DATA WRAPPER c21a_fdw" >/dev/null
psql_valor "GRANT USAGE ON FOREIGN DATA WRAPPER c21a_fdw TO $rol" >/dev/null
exigir_fallo_down 'FDW'
psql_valor "REVOKE USAGE ON FOREIGN DATA WRAPPER c21a_fdw FROM $rol" >/dev/null
psql_valor "GRANT USAGE ON FOREIGN SERVER c21a_server TO $rol" >/dev/null
exigir_fallo_down 'servidor'
psql_valor "REVOKE USAGE ON FOREIGN SERVER c21a_server FROM $rol" >/dev/null
psql_valor "GRANT USAGE ON LANGUAGE plpgsql TO $rol" >/dev/null
exigir_fallo_down 'lenguaje'
psql_valor "REVOKE USAGE ON LANGUAGE plpgsql FROM $rol" >/dev/null
psql_valor "GRANT CREATE ON TABLESPACE pg_default TO $rol" >/dev/null
exigir_fallo_down 'tablespace'
psql_valor "REVOKE CREATE ON TABLESPACE pg_default FROM $rol" >/dev/null
psql_valor "GRANT SET ON PARAMETER log_statement TO $rol" >/dev/null
exigir_fallo_down 'parámetro'
psql_valor "REVOKE SET ON PARAMETER log_statement FROM $rol" >/dev/null

# Si la imagen expone un proveedor ya materializado, reutiliza una etiqueta
# valida del proveedor para demostrar la denegacion de pg_shseclabel.
proveedor=$(psql_valor "SELECT provider FROM pg_shseclabel ORDER BY provider LIMIT 1")
if [[ -n $proveedor ]]; then
  psql_sql <<SQL
SELECT format(
  'SECURITY LABEL FOR %I ON ROLE $rol IS %L',
  provider,label
) FROM pg_shseclabel WHERE provider='$proveedor' LIMIT 1
\gexec
SQL
  exigir_fallo_down 'etiqueta de seguridad'
  psql_valor "SECURITY LABEL FOR $proveedor ON ROLE $rol IS NULL" >/dev/null
fi

# Membresias en las tres coordenadas y opciones PostgreSQL 18.
psql_valor "GRANT $rol TO vec_c21a_miembro WITH ADMIN TRUE, INHERIT FALSE, SET FALSE" >/dev/null
exigir_fallo_down 'rol como grupo'
psql_valor "REVOKE $rol FROM vec_c21a_miembro" >/dev/null
psql_valor "GRANT vec_c21a_grupo TO $rol WITH ADMIN FALSE, INHERIT TRUE, SET TRUE" >/dev/null
exigir_fallo_down 'rol como miembro'
psql_valor "REVOKE vec_c21a_grupo FROM $rol" >/dev/null
psql_sql <<SQL
GRANT vec_c21a_grupo TO $rol WITH ADMIN TRUE, INHERIT FALSE, SET TRUE;
SET ROLE $rol;
GRANT vec_c21a_grupo TO vec_c21a_miembro;
RESET ROLE;
SQL
exigir_fallo_down 'rol como otorgante'
psql_sql <<SQL
SET ROLE $rol;
REVOKE vec_c21a_grupo FROM vec_c21a_miembro;
RESET ROLE;
REVOKE vec_c21a_grupo FROM $rol;
SQL

# Propiedad SQL y las cuatro fachadas, con cualquier sobrecarga.
psql_valor "CREATE TABLE public.c21a_propiedad(id integer); ALTER TABLE public.c21a_propiedad OWNER TO $rol" >/dev/null
exigir_fallo_down 'propiedad SQL'
psql_valor "ALTER TABLE public.c21a_propiedad OWNER TO postgres; DROP TABLE public.c21a_propiedad" >/dev/null
psql_valor "CREATE SCHEMA vec_identidad_sesiones_v1 AUTHORIZATION vec_contexto_actor_v1_propietario; REVOKE ALL ON SCHEMA vec_identidad_sesiones_v1 FROM PUBLIC" >/dev/null
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(timestamptz) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_corporativo_rrhh_v1(timestamptz) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_corporativo_rrhh_v1(timestamptz) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_corporativo_rrhh_v1(timestamptz) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
SQL
exigir_fallo_down 'fachadas C2 nominales'
psql_valor "SET ROLE vec_contexto_actor_v1_propietario; DROP FUNCTION vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(timestamptz),vec_contexto_actor_v1.resolver_y_registrar_contexto_corporativo_rrhh_v1(timestamptz),vec_contexto_actor_v1.reconciliar_contexto_corporativo_rrhh_v1(timestamptz),vec_contexto_actor_v1.acreditar_uso_registro_contexto_corporativo_rrhh_v1(timestamptz)" >/dev/null
psql_sql <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(integer) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_contexto_actor_v1.resolver_y_registrar_contexto_corporativo_rrhh_v1(integer) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_contexto_actor_v1.reconciliar_contexto_corporativo_rrhh_v1(integer) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
CREATE FUNCTION vec_contexto_actor_v1.acreditar_uso_registro_contexto_corporativo_rrhh_v1(integer) RETURNS text LANGUAGE sql AS 'SELECT NULL::text';
SQL
exigir_fallo_down 'sobrecargas C2'
psql_valor "SET ROLE vec_contexto_actor_v1_propietario; DROP FUNCTION vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1(integer),vec_contexto_actor_v1.resolver_y_registrar_contexto_corporativo_rrhh_v1(integer),vec_contexto_actor_v1.reconciliar_contexto_corporativo_rrhh_v1(integer),vec_contexto_actor_v1.acreditar_uso_registro_contexto_corporativo_rrhh_v1(integer)" >/dev/null

# Aislamiento funcional, incluido MAINTAIN, parametros y SET ROLE real.
exigir_fallo_selector 'lectura' 'SELECT * FROM vec_contexto_actor_v1.procedencias'
exigir_fallo_selector 'escritura' "INSERT INTO public.c21a_objeto VALUES (1,'x')"
exigir_fallo_selector 'TRUNCATE' 'TRUNCATE TABLE public.c21a_objeto'
exigir_fallo_selector 'MAINTAIN' 'VACUUM public.c21a_objeto'
exigir_fallo_selector 'función' 'SELECT public.c21a_funcion()'
exigir_fallo_selector 'CREATE' 'CREATE TABLE public.c21a_denegada(id integer)'
exigir_fallo_selector 'TEMP' 'CREATE TEMP TABLE c21a_temporal(id integer)'
exigir_fallo_selector 'parámetro' "ALTER SYSTEM SET log_statement='all'"
exigir_fallo_selector 'SET ROLE' 'SET ROLE vec_c21a_grupo'
docker exec --env PGPASSWORD="$clave" "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -h 127.0.0.1 -U "$rol" -d "$base" \
  -c 'SELECT 1' >"$salidas/nologin" 2>&1 && {
    echo 'el grupo NOLOGIN abrió una conexión' >&2
    exit 1
  }
docker exec --interactive --env CLAVE="$clave" "$contenedor" psql -Xq \
  --set ON_ERROR_STOP=1 -U postgres -d "$base" <<SQL
\getenv clave CLAVE
CREATE ROLE vec_c21a_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
GRANT CONNECT ON DATABASE $base TO vec_c21a_login;
SQL
if docker exec --env PGPASSWORD="$clave" "$contenedor" psql -Xq \
    --set ON_ERROR_STOP=1 -h 127.0.0.1 -U vec_c21a_login -d "$base" \
    -c "SET ROLE $rol" >"$salidas/set_role_login" 2>&1; then
  echo 'un LOGIN ajeno pudo asumir el selector' >&2
  exit 1
fi

# Carreras catalogales: no hay resultado donde mutacion y down sobrevivan.
carrera_catalogo GRANT \
  "GRANT $rol TO vec_c21a_miembro" \
  "REVOKE $rol FROM vec_c21a_miembro"
carrera_catalogo COMMENT \
  "COMMENT ON ROLE $rol IS 'carrera'" \
  "COMMENT ON ROLE $rol IS '$marca'"
carrera_catalogo ajuste_global \
  "ALTER ROLE $rol SET application_name='carrera'" \
  "ALTER ROLE $rol RESET application_name"
carrera_catalogo ajuste_base \
  "ALTER ROLE $rol IN DATABASE $base SET application_name='carrera'" \
  "ALTER ROLE $rol IN DATABASE $base RESET application_name"
carrera_catalogo parametro \
  "GRANT SET ON PARAMETER log_statement TO $rol" \
  "REVOKE SET ON PARAMETER log_statement FROM $rol"
carrera_catalogo acl_interbase \
  "GRANT CONNECT ON DATABASE vec_c21a_ajena TO $rol" \
  "REVOKE CONNECT ON DATABASE vec_c21a_ajena FROM $rol"
carrera_catalogo default_acl \
  "ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario GRANT SELECT ON TABLES TO $rol" \
  "ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario REVOKE SELECT ON TABLES FROM $rol"
carrera_catalogo policy \
  "SET ROLE vec_contexto_actor_v1_propietario; CREATE POLICY c21a_policy_carrera ON vec_contexto_actor_v1.procedencias FOR SELECT TO $rol USING (false)" \
  "SET ROLE vec_contexto_actor_v1_propietario; DROP POLICY c21a_policy_carrera ON vec_contexto_actor_v1.procedencias"

# Futuras C2.1b y C2.5 comparten el advisory y ganan antes del down.
carrera_fachada C2.1b vec_identidad_sesiones_v1 \
  revalidar_contexto_corporativo_rrhh_v1
carrera_fachada C2.5 vec_contexto_actor_v1 \
  resolver_y_registrar_contexto_corporativo_rrhh_v1

# Ciclo limpio, OID nuevo, huellas exactas y ausencia de residuos.
[[ $(huella_objetivo) == "$huella_instalada" ]]
[[ $(huella_contexto) == "$huella_contexto_inicial" ]]
oid_ciclo=$(psql_valor "SELECT '$rol'::regrole::oid")
psql_archivo "$down" >/dev/null
exigir_ausencia_oid "$oid_ciclo"
if psql_archivo "$down" >/dev/null 2>&1; then
  echo 'la segunda retirada fue aceptada' >&2
  exit 1
fi
psql_archivo "$up" >/dev/null
oid_final=$(psql_valor "SELECT '$rol'::regrole::oid")
[[ $oid_final != "$oid_ciclo" ]]
[[ $(huella_objetivo) == "$huella_instalada" ]]
exigir_aislamiento
[[ $(huella_contexto) == "$huella_contexto_inicial" ]]
psql_archivo "$down" >/dev/null
exigir_ausencia_oid "$oid_final"

psql_valor_base vec_c21a_ajena "DROP TABLE public.objeto_ajeno" >/dev/null
psql_valor "DROP DATABASE vec_c21a_ajena" >/dev/null
psql_valor "DROP SERVER c21a_server; DROP FOREIGN DATA WRAPPER c21a_fdw; DROP FUNCTION public.c21a_funcion(); DROP TYPE public.c21a_tipo; DROP SEQUENCE public.c21a_secuencia; DROP TABLE public.c21a_objeto" >/dev/null
psql_valor "DROP SCHEMA vec_identidad_sesiones_v1" >/dev/null
psql_valor "REVOKE CONNECT ON DATABASE $base FROM vec_c21a_login; DROP ROLE vec_c21a_login,vec_c21a_miembro,vec_c21a_grupo,vec_c21a_operador" >/dev/null
psql_valor "ALTER DATABASE $base OWNER TO postgres; DROP ROLE vec_c21a_dueno_base" >/dev/null

echo 'OK: rol selector corporativo RRHH aislado en PostgreSQL 18.4'
