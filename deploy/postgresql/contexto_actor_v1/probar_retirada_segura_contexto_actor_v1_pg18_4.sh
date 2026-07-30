#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-contexto-actor-safe-down-pg-${USER:-usuario}-$$"
base_control=ct95_control
clave_admin=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
down=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.down.sql
up=deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
down_000002=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.down.sql
up_000002=deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql

limpiar() {
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

esperar_postgres() {
  local base=$1 consecutivas=0 respuesta
  for _ in $(seq 1 200); do
    if respuesta=$(docker exec "$contenedor" psql -XAt \
      --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
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
  echo "PostgreSQL 18.4 primario no quedó disponible para $base" >&2
  return 1
}

docker run --detach --rm --name "$contenedor" \
  --env POSTGRES_DB="$base_control" \
  --env POSTGRES_PASSWORD="$clave_admin" \
  "$imagen" >/dev/null
esperar_postgres "$base_control"

version=$(docker exec "$contenedor" psql -XAt \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base_control" \
  --command "SELECT current_setting('server_version_num')")
[[ $version == 180004 ]] || {
  echo "se requiere PostgreSQL 18.4 exacto; se obtuvo $version" >&2
  exit 1
}
recuperacion=$(docker exec "$contenedor" psql -XAt \
  --set ON_ERROR_STOP=1 --username postgres --dbname "$base_control" \
  --command 'SELECT pg_catalog.pg_is_in_recovery()')
[[ $recuperacion == f ]] || {
  echo 'la retirada focal requiere un primario real' >&2
  exit 1
}

psql_archivo() {
  local base=$1 archivo=$2
  shift 2
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" "$@" \
    < "$raiz/$archivo"
}

psql_admin() {
  local base=$1
  docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base"
}

consulta() {
  local base=$1 sql=$2
  docker exec "$contenedor" psql -XAt --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "$sql"
}

huella_esquema() {
  local base=$1
  docker exec "$contenedor" pg_dump -s --username postgres --dbname "$base" |
    sed -E '/^\\(un)?restrict /d' |
    sha256sum | cut -d' ' -f1
}

crear_base() {
  local base=$1
  docker exec "$contenedor" createdb -U postgres "$base"
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
  psql_archivo "$base" "$up"
}

esperar_fallo_sin_cambio() {
  local base=$1 descripcion=$2 modo=${3:-correcta}
  local antes despues salida estado
  antes=$(huella_esquema "$base")
  set +e
  case "$modo" in
    ausente)
      salida=$(psql_archivo "$base" "$down" 2>&1)
      estado=$?
      ;;
    incorrecta)
      salida=$(psql_archivo "$base" "$down" \
        --set confirmar_destruccion_contexto_actor_v1=NO_AUTORIZADA 2>&1)
      estado=$?
      ;;
    correcta)
      salida=$(psql_archivo "$base" "$down" \
        --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 2>&1)
      estado=$?
      ;;
    *)
      echo "modo de prueba desconocido: $modo" >&2
      exit 1
      ;;
  esac
  set -e
  if [[ $estado -eq 0 ]]; then
    echo "retirada aceptada indebidamente: $descripcion" >&2
    exit 1
  fi
  despues=$(huella_esquema "$base")
  if [[ $antes != "$despues" ]]; then
    echo "rollback incompleto tras rechazo: $descripcion" >&2
    echo "$salida" >&2
    exit 1
  fi
}

# Los roles se crean una sola vez en el clúster efímero. Las demás bases
# reciben la misma ACL explícita que el bootstrap acredita.
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
psql_archivo "$base_control" deploy/postgresql/contexto_actor_v1/roles_up.sql

base_exacta=ct95_exacta
crear_base "$base_exacta"
esperar_fallo_sin_cambio "$base_exacta" \
  'confirmación ausente' ausente
esperar_fallo_sin_cambio "$base_exacta" \
  'confirmación incorrecta' incorrecta

# Un objeto posterior, incluso vacío, no pertenece al manifiesto de 000001.
psql_admin "$base_exacta" <<'SQL'
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.consumidor_futuro_v1()
RETURNS integer LANGUAGE sql BEGIN ATOMIC SELECT 1; END;
COMMIT;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'función futura'
psql_admin "$base_exacta" <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
DROP FUNCTION vec_contexto_actor_v1.consumidor_futuro_v1();
RESET ROLE;
SQL

# Toda concesión externa y toda ACL predeterminada añadida fallan cerradas.
psql_admin "$base_exacta" <<'SQL'
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO PUBLIC;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'ACL de esquema para PUBLIC'
psql_admin "$base_exacta" <<'SQL'
REVOKE USAGE ON SCHEMA vec_contexto_actor_v1 FROM PUBLIC;
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
  GRANT SELECT ON TABLES TO PUBLIC;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'ACL predeterminada hostil'
psql_admin "$base_exacta" <<'SQL'
ALTER DEFAULT PRIVILEGES FOR ROLE vec_contexto_actor_v1_propietario
  REVOKE SELECT ON TABLES FROM PUBLIC;
SQL

# Una deriva del trigger forma parte del manifiesto, aunque no añada objetos.
psql_admin "$base_exacta" <<'SQL'
ALTER TABLE vec_contexto_actor_v1.procedencias
  DISABLE TRIGGER procedencia_monotona;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'trigger deshabilitado'
psql_admin "$base_exacta" <<'SQL'
ALTER TABLE vec_contexto_actor_v1.procedencias
  ENABLE TRIGGER procedencia_monotona;
SQL

# Un consumidor exterior no aparece en el esquema, pero RESTRICT conserva la
# dependencia y PostgreSQL revierte también los DROP anteriores.
psql_admin "$base_exacta" <<'SQL'
CREATE SCHEMA ct95_consumidor AUTHORIZATION postgres;
CREATE VIEW ct95_consumidor.procedencias AS
  SELECT procedencia_ref
    FROM vec_contexto_actor_v1.procedencias;
SQL
esperar_fallo_sin_cambio "$base_exacta" 'dependencia exterior real'
psql_admin "$base_exacta" <<'SQL'
DROP VIEW ct95_consumidor.procedencias;
DROP SCHEMA ct95_consumidor RESTRICT;
SQL

# Una instalación 000001 exacta y vacía sí puede retirarse. Tras reconectar y
# reiniciar, el ciclo up -> down -> up produce un OID nuevo.
oid_antes=$(consulta "$base_exacta" \
  "SELECT 'vec_contexto_actor_v1'::regnamespace::oid")
docker restart "$contenedor" >/dev/null
esperar_postgres "$base_exacta"
psql_archivo "$base_exacta" "$down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1
[[ $(consulta "$base_exacta" \
  "SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NULL") == t ]]
set +e
psql_archivo "$base_exacta" "$down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  >/dev/null 2>&1
reentrada=$?
set -e
[[ $reentrada -ne 0 ]] || {
  echo 'la retirada ausente se presentó como éxito' >&2
  exit 1
}
psql_archivo "$base_exacta" "$up"
oid_despues=$(consulta "$base_exacta" \
  "SELECT 'vec_contexto_actor_v1'::regnamespace::oid")
[[ $oid_antes != "$oid_despues" ]] || {
  echo 'up -> down -> up conservó el OID del esquema' >&2
  exit 1
}

# Cualquier evidencia bloquea la retirada aun con el literal correcto.
base_datos=ct95_datos
crear_base "$base_datos"
psql_admin "$base_datos" <<'SQL'
SET ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias VALUES (
  'prc_evidencia_sintetica_ct95_000001',
  1, repeat('a', 64), 'no_autoritativa'
);
RESET ROLE;
SQL
esperar_fallo_sin_cambio "$base_datos" 'evidencia base'
[[ $(consulta "$base_datos" \
  'SELECT count(*) FROM vec_contexto_actor_v1.procedencias') == 1 ]]

# 000002 aporta objetos y su fila de generación: debe retirarse primero por su
# propio down seguro, nunca arrastrarse desde la base.
base_000002=ct95_000002
crear_base "$base_000002"
psql_archivo "$base_000002" "$up_000002"
esperar_fallo_sin_cambio "$base_000002" 'migración 000002 instalada'
psql_archivo "$base_000002" "$down_000002" \
  --set confirmar_retirada_acreditacion_contexto_actor_v2=RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2
psql_archivo "$base_000002" "$down" \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1

# Un propietario alterado se rechaza y no se repara dentro del down.
base_propietario=ct95_propietario
crear_base "$base_propietario"
psql_admin "$base_propietario" <<'SQL'
CREATE ROLE ct95_propietario_hostil NOLOGIN;
ALTER FUNCTION vec_contexto_actor_v1.instante_valido(timestamptz)
  OWNER TO ct95_propietario_hostil;
SQL
esperar_fallo_sin_cambio "$base_propietario" 'propietario hostil'

# Carrera determinista: se observa el advisory no concedido antes de liberar
# al bloqueador. pg_sleep solo estaciona la sesión; no acredita el resultado.
base_carrera=ct95_carrera
crear_base "$base_carrera"
docker exec --env PGAPPNAME=ct95_bloqueador "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 --username postgres \
  --dbname "$base_carrera" --command \
  "BEGIN;
   SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
     'vec_contexto_actor_v1:migracion:base:v1',0));
   SELECT pg_catalog.pg_sleep(300);
   COMMIT" >/dev/null 2>&1 &
proceso_bloqueador=$!
for _ in $(seq 1 200); do
  pid_bloqueador=$(consulta "$base_carrera" \
    "SELECT pid FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct95_bloqueador'
        AND state='active'
        AND query LIKE '%pg_sleep(300)%'
      LIMIT 1")
  [[ -n $pid_bloqueador ]] && break
  sleep 0.02
done
[[ -n ${pid_bloqueador:-} ]] || {
  echo 'no se observó el bloqueador de la carrera' >&2
  exit 1
}

docker exec --env PGAPPNAME=ct95_retirada --interactive "$contenedor" \
  psql -Xq --set ON_ERROR_STOP=1 \
  --set confirmar_destruccion_contexto_actor_v1=DESTRUIR_CONTEXTO_ACTOR_V1 \
  --username postgres --dbname "$base_carrera" \
  < "$raiz/$down" >/dev/null 2>&1 &
proceso_retirada=$!
for _ in $(seq 1 200); do
  espera=$(consulta "$base_carrera" \
    "SELECT count(*) FROM pg_catalog.pg_stat_activity
      WHERE application_name='ct95_retirada'
        AND wait_event_type='Lock' AND wait_event='advisory'")
  [[ $espera == 1 ]] && break
  sleep 0.02
done
[[ ${espera:-0} == 1 ]] || {
  echo 'la retirada concurrente no esperó la guarda base' >&2
  exit 1
}
[[ $(consulta "$base_carrera" \
  "SELECT pg_catalog.pg_terminate_backend($pid_bloqueador)") == t ]]
wait "$proceso_bloqueador" 2>/dev/null || true
wait "$proceso_retirada"
[[ $(consulta "$base_carrera" \
  "SELECT pg_catalog.to_regnamespace('vec_contexto_actor_v1') IS NULL") == t ]]

if rg -n --ignore-case \
  'drop[[:space:]]+owned|drop[[:space:]]+schema[^;]*cascade' "$down"; then
  echo 'la retirada contiene una primitiva destructiva prohibida' >&2
  exit 1
fi

echo 'ContextoActor V1: retirada base segura PostgreSQL 18.4 superada'
