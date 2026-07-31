#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"

imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-catalogos-pg-${USER:-usuario}-$$"
base_control=ct135_control
base=ct135_catalogos
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
roles_up=deploy/postgresql/contexto_actor_v1/roles_up.sql
roles_down=deploy/postgresql/contexto_actor_v1/roles_down.sql
up_000001=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
down_000001=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql
up_000002=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
down_000002=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql
firma_acreditar='vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
temporales=()

limpiar() {
  local proceso archivo
  while IFS= read -r proceso; do
    kill "$proceso" >/dev/null 2>&1 || true
    wait "$proceso" >/dev/null 2>&1 || true
  done < <(jobs -pr)
  for archivo in "${temporales[@]:-}"; do
    rm -f "$archivo"
  done
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fallar() {
  local archivo
  echo "$1" >&2
  for archivo in "${temporales[@]:-}"; do
    [[ -s $archivo ]] && {
      echo "diagnóstico ${archivo##*/}:" >&2
      tail -n 40 "$archivo" >&2
    }
  done
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec "$contenedor" psql -XAt \
      --set ON_ERROR_STOP=1 --username postgres --dbname "$base_control" \
      --command \
      "SELECT current_setting('server_version_num') || '|' ||
              pg_catalog.pg_is_in_recovery()" 2>/dev/null) &&
      [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.05
  done
  fallar 'PostgreSQL 18.4 primario no quedó disponible'
}

psql_archivo() {
  local nombre_base=$1 archivo=$2
  shift 2
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$nombre_base" "$@" \
    < "$raiz/$archivo"
}

psql_admin() {
  local nombre_base=$1
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$nombre_base"
}

consulta() {
  local sql=$1
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "$sql"
}

retirar_000001() {
  psql_archivo "$base" "$down_000001" \
    --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1
}

retirar_000002() {
  docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/$down_000002"
}

huella_catalogal() {
  consulta "
WITH modulo AS (
  SELECT n.oid AS esquema FROM pg_catalog.pg_namespace AS n
   WHERE n.nspname='vec_contexto_actor_v1'
), lineas AS (
  SELECT pg_catalog.format(
           'c|%s|%s|%s|%s|%s|%s', c.oid, c.relname, c.relkind,
           c.relowner, c.relacl, c.reloptions
         ) AS linea
    FROM pg_catalog.pg_class AS c, modulo AS m
   WHERE c.relnamespace=m.esquema
  UNION ALL
  SELECT pg_catalog.format(
           'p|%s|%s|%s|%s|%s', p.oid, p.proname, p.proowner,
           p.proacl, p.prolang
         )
    FROM pg_catalog.pg_proc AS p, modulo AS m
   WHERE p.pronamespace=m.esquema
  UNION ALL
  SELECT pg_catalog.format(
           't|%s|%s|%s|%s', t.oid, t.tgname, t.tgrelid, t.tgfoid
         )
    FROM pg_catalog.pg_trigger AS t
   WHERE t.tgrelid IN (
     SELECT c.oid FROM pg_catalog.pg_class AS c, modulo AS m
      WHERE c.relnamespace=m.esquema
   )
  UNION ALL
  SELECT pg_catalog.format(
           'r|%s|%s|%s|%s|%s|%s|%s|%s|%s', r.oid, r.rolname,
           r.rolsuper, r.rolinherit, r.rolcreaterole, r.rolcreatedb,
           r.rolcanlogin, r.rolreplication, r.rolbypassrls
         )
    FROM pg_catalog.pg_authid AS r
   WHERE r.rolname LIKE 'vec_contexto_actor_v1_%'
  UNION ALL
  SELECT pg_catalog.format(
           'm|%s|%s|%s|%s|%s|%s', m.roleid, m.member, m.grantor,
           m.admin_option, m.inherit_option, m.set_option
         )
    FROM pg_catalog.pg_auth_members AS m
   WHERE m.roleid IN (
           SELECT oid FROM pg_catalog.pg_authid
            WHERE rolname LIKE 'vec_contexto_actor_v1_%'
         )
      OR m.member IN (
           SELECT oid FROM pg_catalog.pg_authid
            WHERE rolname LIKE 'vec_contexto_actor_v1_%'
         )
  UNION ALL
  SELECT pg_catalog.format('s|%s|%s|%s', s.setdatabase, s.setrole, s.setconfig)
    FROM pg_catalog.pg_db_role_setting AS s
   WHERE s.setrole=0 OR s.setrole IN (
     SELECT oid FROM pg_catalog.pg_authid
      WHERE rolname LIKE 'vec_contexto_actor_v1_%'
        OR rolname='ct135_login_runtime'
   )
  UNION ALL
  SELECT pg_catalog.format('d|%s|%s|%s|%s', d.classoid, d.objoid, d.objsubid, d.description)
    FROM pg_catalog.pg_description AS d
   WHERE d.objoid IN (
     SELECT c.oid FROM pg_catalog.pg_class AS c, modulo AS m
      WHERE c.relnamespace=m.esquema
     UNION ALL
     SELECT p.oid FROM pg_catalog.pg_proc AS p, modulo AS m
      WHERE p.pronamespace=m.esquema
   )
)
SELECT pg_catalog.md5(coalesce(
         pg_catalog.string_agg(linea, E'\n' ORDER BY linea), ''
       ))
  FROM lineas"
}

esperar_sesion() {
  local aplicacion=$1 predicado=$2 descripcion=$3
  for _ in $(seq 1 240); do
    if [[ $(consulta "
SELECT pg_catalog.count(*) FROM pg_catalog.pg_stat_activity
 WHERE application_name='$aplicacion' AND $predicado") == 1 ]]; then
      return
    fi
    sleep 0.02
  done
  fallar "no se observó $descripcion"
}

terminar_backend() {
  local aplicacion=$1 pid
  pid=$(consulta "
SELECT pid FROM pg_catalog.pg_stat_activity
 WHERE application_name='$aplicacion'")
  [[ -n $pid ]] || fallar "no existe backend $aplicacion"
  [[ $(consulta "SELECT pg_catalog.pg_terminate_backend($pid)") == t ]] ||
    fallar "no se pudo terminar $aplicacion"
}

iniciar_barrera() {
  local aplicacion=$1 clave=$2
  docker exec --env PGAPPNAME="$aplicacion" "$contenedor" \
    psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command \
    "SELECT pg_catalog.pg_advisory_lock(
       pg_catalog.hashtextextended('$clave',0));
     SELECT pg_catalog.pg_sleep(300)" >/dev/null 2>&1 &
  proceso_iniciado=$!
  esperar_sesion "$aplicacion" \
    "state='active' AND query LIKE '%pg_sleep(300)%'" \
    "la barrera $aplicacion"
}

iniciar_retirada_000001() {
  local salida=$1
  docker exec --env PGAPPNAME=ct135_retirada_000001 \
    --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
    --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
    --username postgres --dbname "$base" \
    < "$raiz/$down_000001" >"$salida" 2>&1 &
  proceso_iniciado=$!
}

iniciar_retirada_000002() {
  local salida=$1
  docker exec --env PGAPPNAME=ct135_retirada_000002 \
    --env PGOPTIONS="-c vec.confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2" \
    --interactive "$contenedor" psql -Xq --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/$down_000002" >"$salida" 2>&1 &
  proceso_iniciado=$!
}

probar_mutacion_bloqueada() {
  local aplicacion=$1 sql=$2 salida proceso estado
  salida=$(mktemp "$raiz/.ct135-${aplicacion}.XXXXXX")
  temporales+=("$salida")
  docker exec --env PGAPPNAME="$aplicacion" "$contenedor" \
    psql -Xq --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SET lock_timeout='500ms'; $sql" >"$salida" 2>&1 &
  proceso=$!
  esperar_sesion "$aplicacion" "wait_event_type='Lock' AND EXISTS (
    SELECT 1 FROM pg_catalog.pg_locks AS l
     WHERE l.pid=pg_stat_activity.pid AND NOT l.granted
  )" "el bloqueo observable de $aplicacion"
  set +e
  wait "$proceso"
  estado=$?
  set -e
  [[ $estado -ne 0 ]] || fallar "$aplicacion confirmó antes del COMMIT"
  grep -Fq 'canceling statement due to lock timeout' "$salida" ||
    fallar "$aplicacion no terminó por lock_timeout"
}

exigir_intacta_000001() {
  [[ $(consulta "
SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NOT NULL
   AND pg_catalog.to_regclass('vec_contexto_actor_v1.registros_contexto') IS NOT NULL
   AND pg_catalog.to_regprocedure(
         'vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()'
       ) IS NOT NULL") == t ]] ||
    fallar '000001 quedó parcialmente retirada'
}

exigir_intacta_000002() {
  [[ $(consulta "
SELECT pg_catalog.to_regclass(
         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
       ) IS NOT NULL
   AND pg_catalog.to_regprocedure('$firma_acreditar') IS NOT NULL
   AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_trigger
         WHERE NOT tgisinternal AND tgname IN (
           'puntero_actual_no_truncable_v2',
           'serializar_mutacion_punteros_actuales_v2',
           'avanzar_generacion_punteros_actuales_v2'
         ))=15") == t ]] ||
    fallar '000002 quedó parcialmente retirada'
}

rechazar_000002_sin_cambio() {
  local descripcion=$1 antes despues estado
  antes=$(huella_catalogal)
  set +e
  retirar_000002 >/dev/null 2>&1
  estado=$?
  set -e
  [[ $estado -ne 0 ]] || fallar "000002 aceptó $descripcion"
  despues=$(huella_catalogal)
  [[ $antes == "$despues" ]] || fallar "000002 alteró estado al rechazar $descripcion"
  exigir_intacta_000002
}

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base_control" \
  --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
esperar_postgres

psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$base_control" "$roles_up"
psql_admin "$base_control" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
SQL
docker exec "$contenedor" createdb --username postgres "$base"
psql_admin "$base" <<'SQL'
DO $base$
BEGIN
  EXECUTE pg_catalog.format(
    'REVOKE ALL PRIVILEGES ON DATABASE %I FROM PUBLIC',
    pg_catalog.current_database()
  );
  EXECUTE pg_catalog.format(
    'GRANT CONNECT ON DATABASE %I TO vec_contexto_actor_v1_propietario, vec_contexto_actor_v1_migrador, vec_contexto_actor_v1_runtime',
    pg_catalog.current_database()
  );
END
$base$;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$base" "$up_000001"
psql_admin "$base" <<'SQL'
CREATE ROLE ct135_receptor NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  NOINHERIT NOREPLICATION NOBYPASSRLS;
SQL

# La barrera se dispara solo en el primer DDL destructivo posterior al
# inventario. No muta catálogos ni toca objetos del módulo.
psql_admin "$base" <<'SQL'
CREATE FUNCTION public.ct135_estacionar_retirada()
RETURNS event_trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $barrera$
BEGIN
  IF pg_catalog.current_setting('application_name')='ct135_retirada_000001'
     AND tg_tag='DROP TABLE'
  THEN
    PERFORM pg_catalog.pg_advisory_xact_lock(
      pg_catalog.hashtextextended('ct135:postinventario:000001',0)
    );
  ELSIF pg_catalog.current_setting('application_name')='ct135_retirada_000002'
        AND tg_tag='DROP TRIGGER'
  THEN
    PERFORM pg_catalog.pg_advisory_xact_lock(
      pg_catalog.hashtextextended('ct135:postinventario:000002',0)
    );
  END IF;
END
$barrera$;
CREATE EVENT TRIGGER ct135_estacionar_retirada
  ON ddl_command_start
  WHEN TAG IN ('DROP TABLE', 'DROP TRIGGER')
  EXECUTE FUNCTION public.ct135_estacionar_retirada();
SQL

# 000001: tres DDL independientes quedan detrás de la fotografía catalogal.
huella_antes=$(huella_catalogal)
salida_down_1=$(mktemp "$raiz/.ct135-down-000001.XXXXXX")
temporales+=("$salida_down_1")
iniciar_barrera ct135_barrera_000001 ct135:postinventario:000001
proceso_barrera=$proceso_iniciado
iniciar_retirada_000001 "$salida_down_1"
proceso_down=$proceso_iniciado
esperar_sesion ct135_retirada_000001 \
  "wait_event_type='Lock' AND wait_event='advisory'" \
  '000001 estacionada después del inventario'
probar_mutacion_bloqueada ct135_alter_role_1 \
  'ALTER ROLE vec_contexto_actor_v1_runtime RENAME TO ct135_runtime_mutado'
probar_mutacion_bloqueada ct135_comment_1 \
  "COMMENT ON TABLE vec_contexto_actor_v1.procedencias IS 'ct135'"
probar_mutacion_bloqueada ct135_grant_1 \
  'GRANT SELECT ON vec_contexto_actor_v1.procedencias TO ct135_receptor'
[[ $(huella_catalogal) == "$huella_antes" ]] ||
  fallar 'las mutaciones concurrentes alteraron 000001'
exigir_intacta_000001
terminar_backend ct135_barrera_000001
wait "$proceso_barrera" >/dev/null 2>&1 || true
wait "$proceso_down"
[[ $(consulta "
SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NULL") == t ]] ||
  fallar '000001 no terminó su retirada íntegra'
psql_archivo "$base" "$up_000001"
exigir_intacta_000001

psql_admin "$base" <<'SQL'
CREATE ROLE ct135_login_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
  INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_runtime TO ct135_login_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
psql_archivo "$base" "$up_000002"
oid_000002_antes=$(consulta "
SELECT 'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass::oid")

# Un default de otra base del clúster no contamina la base dedicada actual.
docker exec "$contenedor" createdb --username postgres ct135_ajena
psql_admin "$base" <<'SQL'
ALTER DATABASE ct135_ajena SET application_name='ct135_ajena';
SQL
retirar_000002
[[ $(consulta "
SELECT pg_catalog.count(*) FROM pg_catalog.pg_db_role_setting AS s
 JOIN pg_catalog.pg_database AS d ON d.oid=s.setdatabase
 WHERE d.datname='ct135_ajena' AND s.setrole=0
   AND s.setconfig @> ARRAY['application_name=ct135_ajena']") == 1 ]] ||
  fallar 'la retirada borró o ignoró el default de otra base'
psql_archivo "$base" "$up_000002"
psql_admin "$base" <<'SQL'
ALTER DATABASE ct135_catalogos SET application_name='ct135_actual';
SQL
rechazar_000002_sin_cambio 'default común de la base actual'
psql_admin "$base" <<'SQL'
ALTER DATABASE ct135_catalogos RESET application_name;
SQL

# El runtime puede tener LOGIN directos, pero su forma y topología son exactas.
psql_admin "$base" <<'SQL'
GRANT vec_contexto_actor_v1_runtime TO ct135_login_runtime
  WITH ADMIN TRUE, INHERIT TRUE, SET FALSE;
SQL
rechazar_000002_sin_cambio 'ADMIN OPTION en LOGIN runtime'
psql_admin "$base" <<'SQL'
GRANT vec_contexto_actor_v1_runtime TO ct135_login_runtime
  WITH ADMIN FALSE, INHERIT FALSE, SET FALSE;
SQL
rechazar_000002_sin_cambio 'INHERIT OPTION desactivado'
psql_admin "$base" <<'SQL'
GRANT vec_contexto_actor_v1_runtime TO ct135_login_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
SQL
rechazar_000002_sin_cambio 'SET OPTION activado'
psql_admin "$base" <<'SQL'
GRANT vec_contexto_actor_v1_runtime TO ct135_login_runtime
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE ROLE ct135_grupo_extra NOLOGIN;
GRANT ct135_grupo_extra TO ct135_login_runtime WITH SET FALSE;
SQL
rechazar_000002_sin_cambio 'membresía adicional del LOGIN runtime'
psql_admin "$base" <<'SQL'
REVOKE ct135_grupo_extra FROM ct135_login_runtime;
DROP ROLE ct135_grupo_extra;
ALTER ROLE ct135_login_runtime SET application_name='ct135_hostil';
SQL
rechazar_000002_sin_cambio 'setting del LOGIN runtime'
psql_admin "$base" <<'SQL'
ALTER ROLE ct135_login_runtime RESET ALL;
SQL

# 000002: además de rol/ACL/comentario se serializan los nombres globales que
# forman parte de pg_get_functiondef y del canon de colaciones.
huella_antes=$(huella_catalogal)
salida_down_2=$(mktemp "$raiz/.ct135-down-000002.XXXXXX")
temporales+=("$salida_down_2")
iniciar_barrera ct135_barrera_000002 ct135:postinventario:000002
proceso_barrera=$proceso_iniciado
iniciar_retirada_000002 "$salida_down_2"
proceso_down=$proceso_iniciado
esperar_sesion ct135_retirada_000002 \
  "wait_event_type='Lock' AND wait_event='advisory'" \
  '000002 estacionada después del inventario'
probar_mutacion_bloqueada ct135_alter_role_2 \
  'ALTER ROLE vec_contexto_actor_v1_runtime RENAME TO ct135_runtime_mutado'
probar_mutacion_bloqueada ct135_comment_2 \
  "COMMENT ON FUNCTION $firma_acreditar IS 'ct135'"
probar_mutacion_bloqueada ct135_grant_2 \
  "GRANT EXECUTE ON FUNCTION $firma_acreditar TO ct135_receptor"
probar_mutacion_bloqueada ct135_language_2 \
  'ALTER LANGUAGE plpgsql RENAME TO ct135_plpgsql'
probar_mutacion_bloqueada ct135_collation_2 \
  'ALTER COLLATION pg_catalog."default" RENAME TO ct135_default'
[[ $(consulta "
SELECT EXISTS (
         SELECT 1 FROM pg_catalog.pg_language WHERE lanname='plpgsql'
       ) AND EXISTS (
         SELECT 1 FROM pg_catalog.pg_collation AS c
          JOIN pg_catalog.pg_namespace AS n ON n.oid=c.collnamespace
         WHERE n.nspname='pg_catalog' AND c.collname='default'
       )") == t ]] ||
  fallar 'lenguaje o colación cambiaron antes del COMMIT'
[[ $(huella_catalogal) == "$huella_antes" ]] ||
  fallar 'las mutaciones concurrentes alteraron 000002'
exigir_intacta_000002
terminar_backend ct135_barrera_000002
wait "$proceso_barrera" >/dev/null 2>&1 || true
wait "$proceso_down"
[[ $(consulta "
SELECT pg_catalog.to_regclass(
         'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
       ) IS NULL
   AND pg_catalog.to_regprocedure('$firma_acreditar') IS NULL
   AND pg_catalog.to_regclass(
         'vec_contexto_actor_v1.registros_contexto'
       ) IS NOT NULL") == t ]] ||
  fallar '000002 no terminó su retirada íntegra'
[[ $(consulta "
SELECT pg_catalog.current_setting(
  'vec.confirmar_retirada_acreditacion_contexto_actor_v2', true
) IS NULL") == t ]] ||
  fallar 'una sesión nueva heredó la confirmación destructiva'

psql_archivo "$base" "$up_000002"
exigir_intacta_000002
oid_000002_despues=$(consulta "
SELECT 'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'::regclass::oid")
[[ $oid_000002_antes != "$oid_000002_despues" ]] ||
  fallar 'la reinstalación 000002 conservó la identidad física'

# Limpieza funcional completa antes de destruir el contenedor.
retirar_000002
psql_admin "$base" <<'SQL'
DROP EVENT TRIGGER ct135_estacionar_retirada;
DROP FUNCTION public.ct135_estacionar_retirada();
REVOKE vec_contexto_actor_v1_runtime FROM ct135_login_runtime;
DROP ROLE ct135_login_runtime;
DROP ROLE ct135_receptor;
SQL
docker exec "$contenedor" dropdb --force --username postgres ct135_ajena
retirar_000001
psql_archivo "$base" "$roles_down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1
[[ $(consulta "
SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NULL
   AND NOT EXISTS (
     SELECT 1 FROM pg_catalog.pg_roles
      WHERE rolname LIKE 'vec_contexto_actor_v1_%'
   )") == t ]] ||
  fallar 'la sesión de prueba no quedó limpia'

echo 'ContextoActor: catálogos concurrentes de 000001/000002 superados en PostgreSQL 18.4'
