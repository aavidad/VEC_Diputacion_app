#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
[[ $imagen =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] || {
  echo 'la imagen PostgreSQL debe estar fijada por digest sha256' >&2
  exit 1
}

sufijo="vec-c21b-rec-${UID:-0}-$$"
primaria="${sufijo}-primaria"
replica="${sufijo}-replica"
red="${sufijo}-red"
volumen_primaria="${sufijo}-datos-primaria"
volumen_replica="${sufijo}-datos-replica"
base=vec_identidad_c21b_recuperacion
replicador=vec_c21b_replicador
login=vec_c21b_login
selector=vec_contexto_actor_corporativo_rrhh_selector
consumidor=vec_contexto_actor_v1_propietario
fachada=vec_identidad_sesiones_v1.revalidar_contexto_corporativo_rrhh_v1
base_fachada=vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1
proxy=vec_contexto_actor_v1.c21b_proxy_identidad
ranura=vec_c21b_recuperacion
pgdata=/var/lib/postgresql/data
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
clave_actor=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
clave_replica=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
salidas=$(mktemp -d)
up=deploy/postgresql/identidad_sesiones_v1/migraciones/000004_revalidacion_contexto_corporativo_rrhh_v1.up.sql
rol_up=deploy/postgresql/contexto_actor_v1/roles_contexto_corporativo_rrhh_selector_v1_up.sql

limpiar_recursos() {
  docker rm --force "$replica" "$primaria" >/dev/null 2>&1 || true
  docker network rm "$red" >/dev/null 2>&1 || true
  docker volume rm --force "$volumen_replica" "$volumen_primaria" \
    >/dev/null 2>&1 || true
}

finalizar() {
  local estado=$?
  if [[ $estado -ne 0 ]]; then
    for contenedor in "$primaria" "$replica"; do
      docker logs --tail 120 "$contenedor" 2>&1 |
        grep -E ' (LOG|ERROR|FATAL): ' | tail -40 >&2 || true
    done
  fi
  limpiar_recursos
  rm -rf "$salidas"
}
trap finalizar EXIT INT TERM

paso() {
  printf '[CT-000047C2.1b:RECUPERACION:PG18.4] %s\n' "$1"
}

psql_primaria() {
  docker exec --interactive "$primaria" psql -Xq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -U postgres -d "$base"
}

valor_primaria() {
  docker exec "$primaria" psql -XAtq --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose \
    -U postgres -d "$base" -c "$1"
}

archivo_primaria() {
  psql_primaria <"$raiz/$1"
}

psql_replica() {
  docker exec --interactive "$replica" psql -Xq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -U postgres -d "$base"
}

valor_replica() {
  docker exec "$replica" psql -XAtq --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose \
    -U postgres -d "$base" -c "$1"
}

actor_en() {
  local contenedor=$1 sql=$2
  docker exec --env PGPASSWORD="$clave_actor" "$contenedor" psql -XAtq \
    --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
    -h 127.0.0.1 -U "$login" -d "$base" -c "$sql"
}

esperar_valor() {
  local lado=$1 consulta=$2 esperado=$3 caso=$4 observado=
  for _ in $(seq 1 300); do
    if [[ $lado == primaria ]]; then
      observado=$(valor_primaria "$consulta" 2>/dev/null || true)
    else
      observado=$(valor_replica "$consulta" 2>/dev/null || true)
    fi
    [[ $observado == "$esperado" ]] && return 0
    sleep 0.1
  done
  echo "no se alcanzó el estado: $caso ($observado)" >&2
  return 1
}

esperar_lista() {
  local lado=$1 consulta=$2 caso=$3 observado=
  shift 3
  for _ in $(seq 1 300); do
    if [[ $lado == primaria ]]; then
      observado=$(valor_primaria "$consulta" 2>/dev/null || true)
    else
      observado=$(valor_replica "$consulta" 2>/dev/null || true)
    fi
    for esperado in "$@"; do
      [[ $observado == "$esperado" ]] && return 0
    done
    sleep 0.1
  done
  echo "no se alcanzó el estado: $caso ($observado)" >&2
  return 1
}

esperar_servidor_final() {
  local lado=$1 contenedor=$2 caso=$3 version=
  for _ in $(seq 1 300); do
    if docker exec "$contenedor" sh -euc \
        'test "$(cat /proc/1/comm)" = postgres' >/dev/null 2>&1; then
      if [[ $lado == primaria ]]; then
        version=$(valor_primaria \
          "SELECT current_setting('server_version_num')" 2>/dev/null || true)
      else
        version=$(valor_replica \
          "SELECT current_setting('server_version_num')" 2>/dev/null || true)
      fi
      [[ $version == 180004 ]] && return 0
    fi
    sleep 0.1
  done
  echo "no terminó el arranque: $caso ($version)" >&2
  return 1
}

esperar_replay() {
  local lsn=$1 caso=$2
  esperar_valor replica \
    "SELECT (pg_wal_lsn_diff(pg_last_wal_replay_lsn(),'$lsn')>=0)::text" \
    true "$caso"
}

exigir_cierre_replica() {
  local caso=$1
  local salida_ro="$salidas/cierre_replica_ro"
  local salida_serializable="$salidas/cierre_replica_serializable"
  local estado estado_sql modo sufijo_modo
  actor_en "$replica" \
    "BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT count(*) FROM $proxy('$autenticacion','$sesion'); COMMIT" \
    >"$salida_ro" 2>&1
  [[ $(grep -E '^[0-9]+$' "$salida_ro" | tail -1) == 0 ]] || {
    echo "la réplica concedió identidad de solo lectura durante $caso" >&2
    return 1
  }
  for modo in 'READ ONLY' 'READ WRITE'; do
    sufijo_modo=${modo// /_}
    set +e
    actor_en "$replica" \
      "BEGIN ISOLATION LEVEL SERIALIZABLE $modo;
SELECT count(*) FROM $proxy('$autenticacion','$sesion'); COMMIT" \
      >"${salida_serializable}_${sufijo_modo}" 2>&1
    estado=$?
    set -e
    if [[ $estado -eq 0 ]]; then
      echo "la réplica admitió SERIALIZABLE $modo durante $caso" >&2
      return 1
    fi
    estado_sql=$(sed -n \
      's/^ERROR:  \([0-9A-Z]\{5\}\):.*/\1/p' \
      "${salida_serializable}_${sufijo_modo}" | tail -1)
    [[ $estado_sql == 0A000 ]] || {
      echo "SQLSTATE inestable en SERIALIZABLE $modo: $estado_sql" >&2
      return 1
    }
  done
  [[ $(valor_replica "SELECT pg_is_in_recovery()::text") == true ]]
}

paso "arranque aislado con $imagen"
docker network create --internal "$red" >/dev/null
docker volume create "$volumen_primaria" >/dev/null
docker volume create "$volumen_replica" >/dev/null
docker run --detach --name "$primaria" --network "$red" \
  --env PGDATA="$pgdata" --env POSTGRES_DB="$base" \
  --env POSTGRES_PASSWORD="$clave_admin" \
  --mount "source=$volumen_primaria,target=$pgdata" \
  "$imagen" >/dev/null
[[ $(docker network inspect --format '{{.Internal}}' "$red") == true ]]
puertos_primaria=$(docker inspect \
  --format '{{json .HostConfig.PortBindings}}' "$primaria")
[[ $puertos_primaria == null || $puertos_primaria == '{}' ]]
esperar_servidor_final primaria "$primaria" \
  'PostgreSQL 18.4 primario preparado'
[[ $(valor_primaria "SELECT pg_is_in_recovery()::text") == false ]]

paso 'pila C2.1b y datos exclusivamente sintéticos en la primaria'
psql_primaria <<SQL
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
  archivo_primaria "$archivo" >/dev/null
done
psql_primaria <<'SQL'
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
  deploy/postgresql/identidad_sesiones_v1/migraciones/000003_revalidacion_autenticacion_actor_v1.up.sql \
  "$up"
do
  archivo_primaria "$archivo" >/dev/null
done
psql_primaria <<SQL
CREATE ROLE $login LOGIN PASSWORD '$clave_actor' NOSUPERUSER NOCREATEDB
  NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT $selector TO $login WITH ADMIN FALSE,INHERIT TRUE,SET FALSE;
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
cuenta=$(valor_primaria "SELECT cuenta_ref FROM
vec_identidad_sesiones_v1.provisionar_cuenta_v1(
'opr_'||repeat('a',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('2',64),'hex'),decode(repeat('1',64),'hex'),false,NULL)")
IFS='|' read -r autenticacion sesion _control _revision cuenta_sesion \
  <<<"$(valor_primaria "SELECT autenticacion_ref||'|'||sesion_ref||'|'||
control_sesion_ref||'|'||control_sesion_revision_texto||'|'||cuenta_ref
FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
'opr_'||repeat('c',24),'vec.identidad.hmac-sha256.v1',
'idh_aaaaaaaaaaaaaaaaaaaaaaaa','clave-hsm-prueba',1,
decode(repeat('3',64),'hex'),decode(repeat('4',64),'hex'),
decode(repeat('1',64),'hex'),decode(repeat('2',64),'hex'),NULL,false,
'interna_corporativa','kerberos_ad','alto',repeat('c',64),
clock_timestamp()-interval '2 seconds',clock_timestamp()-interval '1 second',
clock_timestamp()+interval '4 minutes','pga_aaaaaaaaaaaaaaaaaaaaaaaa',
repeat('b',64))")"
[[ $cuenta == "$cuenta_sesion" ]]
[[ $(actor_en "$primaria" \
  "BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SELECT count(*) FROM $proxy('$autenticacion','$sesion'); COMMIT" |
  grep -E '^[0-9]+$' | tail -1) == 1 ]]

paso 'base backup física y hot standby sin puertos publicados'
psql_primaria <<SQL
CREATE ROLE $replicador LOGIN REPLICATION PASSWORD '$clave_replica'
  NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOBYPASSRLS;
SQL
docker exec "$primaria" sh -euc \
  "printf '%s\n' 'host replication $replicador all scram-sha-256' >> '$pgdata/pg_hba.conf'"
valor_primaria "SELECT pg_reload_conf()" >/dev/null
docker run --rm --user root \
  --mount "source=$volumen_replica,target=$pgdata" \
  --entrypoint sh "$imagen" -euc \
  "chown -R postgres:postgres '$pgdata'"
docker run --rm --user postgres --network "$red" \
  --env PGPASSWORD="$clave_replica" \
  --mount "source=$volumen_replica,target=$pgdata" \
  --entrypoint pg_basebackup "$imagen" \
  --host "$primaria" --username "$replicador" \
  --pgdata "$pgdata" --write-recovery-conf --wal-method stream \
  --create-slot --slot "$ranura" --checkpoint fast
docker run --detach --name "$replica" --network "$red" \
  --env PGDATA="$pgdata" \
  --mount "source=$volumen_replica,target=$pgdata" \
  "$imagen" >/dev/null
puertos_replica=$(docker inspect \
  --format '{{json .HostConfig.PortBindings}}' "$replica")
[[ $puertos_replica == null || $puertos_replica == '{}' ]]
esperar_servidor_final replica "$replica" \
  'PostgreSQL 18.4 de réplica preparado'
esperar_valor replica "SELECT pg_is_in_recovery()::text" true \
  'réplica física en recuperación'
esperar_valor replica \
  "SELECT COALESCE((SELECT status FROM pg_stat_wal_receiver LIMIT 1),'')" \
  streaming 'receptor WAL conectado'
objetivo_inicial=$(valor_primaria "SELECT pg_current_wal_flush_lsn()")
esperar_replay "$objetivo_inicial" 'base backup reproducida'

paso 'cierre serializable y guard previo al revalidador con bloqueos'
exigir_cierre_replica 'réplica sincronizada'
if psql_replica >"$salidas/base_en_replica" 2>&1 <<SQL
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT count(*) FROM $base_fachada('$autenticacion','$sesion');
COMMIT;
SQL
then
  echo 'el revalidador base con FOR UPDATE se ejecutó en hot standby' >&2
  exit 1
fi
grep -Fq '25006' "$salidas/base_en_replica"
[[ $(valor_replica "SELECT pg_is_in_recovery()::text") == true ]]

paso 'retardo físico inducido y ausencia de escritura local'
valor_replica "SELECT pg_wal_replay_pause()" >/dev/null
esperar_valor replica "SELECT pg_get_wal_replay_pause_state()" paused \
  'replay WAL pausado'
lsn_antes=$(valor_replica "SELECT pg_last_wal_replay_lsn()")
valor_primaria "CREATE TABLE public.c21b_marca_recuperacion(
id integer PRIMARY KEY); REVOKE ALL ON public.c21b_marca_recuperacion
FROM PUBLIC; INSERT INTO public.c21b_marca_recuperacion VALUES(1)" \
  >/dev/null
objetivo_retardo=$(valor_primaria "SELECT pg_current_wal_flush_lsn()")
[[ $objetivo_retardo =~ ^[0-9A-F]+/[0-9A-F]+$ ]]
esperar_valor replica \
  "SELECT (pg_wal_lsn_diff('$objetivo_retardo',
COALESCE(pg_last_wal_replay_lsn(),'0/0'))>0)::text" true \
  'retardo físico observable'
exigir_cierre_replica 'retardo de replay'
[[ $(valor_replica "SELECT pg_last_wal_replay_lsn()") == "$lsn_antes" ]]
if valor_replica \
  "INSERT INTO vec_identidad_sesiones_v1.estado_cuenta_actual
VALUES('cta_aaaaaaaaaaaaaaaaaaaaaa',1)" \
  >"$salidas/escritura_replica" 2>&1; then
  echo 'la réplica admitió una escritura accidental' >&2
  exit 1
fi
grep -Fq '25006' "$salidas/escritura_replica"

paso 'pérdida de primaria, recuperación cerrada y reenganche'
docker stop --time 5 "$primaria" >/dev/null
esperar_valor replica "SELECT pg_is_in_recovery()::text" true \
  'hot standby disponible sin primaria'
exigir_cierre_replica 'pérdida de primaria'
[[ $(valor_replica "SELECT pg_last_wal_replay_lsn()") == "$lsn_antes" ]]
docker start "$primaria" >/dev/null
esperar_servidor_final primaria "$primaria" 'reinicio de primaria'
valor_replica "SELECT pg_wal_replay_resume()" >/dev/null
esperar_lista replica "SELECT pg_get_wal_replay_pause_state()" \
  'replay WAL reanudado' 'not paused' replaying
esperar_valor replica \
  "SELECT COALESCE((SELECT status FROM pg_stat_wal_receiver LIMIT 1),'')" \
  streaming 'reconexión del receptor WAL'
esperar_replay "$objetivo_retardo" 'recuperación del retardo'
[[ $(valor_replica \
  "SELECT count(*) FROM public.c21b_marca_recuperacion") == 1 ]]
exigir_cierre_replica 'réplica reenganchada'

paso 'reinicio de réplica y nueva reproducción física'
docker restart --time 5 "$replica" >/dev/null
esperar_valor replica "SELECT pg_is_in_recovery()::text" true \
  'réplica reiniciada en recuperación'
esperar_valor replica \
  "SELECT COALESCE((SELECT status FROM pg_stat_wal_receiver LIMIT 1),'')" \
  streaming 'reconexión tras reinicio de réplica'
valor_primaria \
  "INSERT INTO public.c21b_marca_recuperacion VALUES(2)" >/dev/null
objetivo_reinicio=$(valor_primaria "SELECT pg_current_wal_flush_lsn()")
[[ $objetivo_reinicio =~ ^[0-9A-F]+/[0-9A-F]+$ ]]
esperar_replay "$objetivo_reinicio" 'WAL posterior al reinicio'
[[ $(valor_replica \
  "SELECT array_agg(id ORDER BY id)::text
FROM public.c21b_marca_recuperacion") == '{1,2}' ]]
exigir_cierre_replica 'reinicio y reconexión'

paso 'limpieza comprobada de contenedores, red y volúmenes'
limpiar_recursos
for contenedor in "$primaria" "$replica"; do
  if docker container inspect "$contenedor" >/dev/null 2>&1; then
    echo "quedó un contenedor efímero: $contenedor" >&2
    exit 1
  fi
done
if docker network inspect "$red" >/dev/null 2>&1; then
  echo "quedó una red efímera: $red" >&2
  exit 1
fi
for volumen in "$volumen_primaria" "$volumen_replica"; do
  if docker volume inspect "$volumen" >/dev/null 2>&1; then
    echo "quedó un volumen efímero: $volumen" >&2
    exit 1
  fi
done
echo 'OK: recuperación física C2.1b en PostgreSQL 18.4'
