#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-llamamientos-pg-${USER:-usuario}-$$"
base=vec_bolsa_llamamientos_prueba

clave_admin=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
clave_runtime=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
if [[ ${#clave_admin} -ne 64 || ${#clave_runtime} -ne 64 ]]; then
    echo 'no se pudieron generar credenciales efimeras' >&2
    exit 1
fi

limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 --env POSTGRES_DB="$base" \
    --env POSTGRES_PASSWORD="$clave_admin" "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres \
        --dbname "$base" >/dev/null 2>&1; then break; fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres \
    --dbname "$base" >/dev/null

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/roles_up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones/000001_almacen_llamamientos.up.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones/000002_guardado_cerrado.up.sql

docker exec --interactive --env CLAVE_RUNTIME="$clave_runtime" "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --username postgres \
    --dbname "$base" <<'SQL'
\getenv clave_runtime CLAVE_RUNTIME
CREATE ROLE vec_llamamientos_runtime_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_runtime';
GRANT vec_bolsa_llamamientos_ejecutor
    TO vec_llamamientos_runtime_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_llamamientos/pruebas_sql/acl_cierre.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/pruebas_sql/idempotencia_y_unicidad.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/pruebas_sql/preparar_concurrencia.sql

if docker exec --env PGPASSWORD="$clave_runtime" "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
    --username vec_llamamientos_runtime_prueba --dbname "$base" \
    --command "SELECT * FROM vec_bolsa_llamamientos.guardar_propuesta_v1('{}','{}',''::bytea,''::bytea)" \
    >/dev/null 2>&1; then
    echo 'la cuenta runtime pudo invocar una funcion cerrada' >&2
    exit 1
fi

# Dos transacciones compiten por la misma version inmutable de necesidad. La
# primera confirma; la segunda debe recibir unique_violation, nunca dos filas.
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL' &
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.propuesta VALUES (
 'propuesta-concurrencia-1', 'necesidad-concurrencia', 1,
 encode(sha256(convert_to('{"n":1}', 'UTF8')), 'hex'),
 'instantanea-concurrencia-1', 1,
 encode(sha256(convert_to('{"i":1}', 'UTF8')), 'hex'),
 convert_to('{"p":1}', 'UTF8'), repeat('1',64),
 encode(sha256(convert_to('{"p":1}', 'UTF8')), 'hex'),
 'decision-concurrencia-1', clock_timestamp()
);
SELECT pg_sleep(1);
COMMIT;
SQL
pid_primero=$!
sleep 0.2
set +e
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL' >/dev/null 2>&1
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
INSERT INTO vec_bolsa_llamamientos.propuesta VALUES (
 'propuesta-concurrencia-2', 'necesidad-concurrencia', 1,
 encode(sha256(convert_to('{"n":1}', 'UTF8')), 'hex'),
 'instantanea-concurrencia-2', 1,
 encode(sha256(convert_to('{"i":2}', 'UTF8')), 'hex'),
 convert_to('{"p":2}', 'UTF8'), repeat('2',64),
 encode(sha256(convert_to('{"p":2}', 'UTF8')), 'hex'),
 'decision-concurrencia-2', clock_timestamp()
);
COMMIT;
SQL
estado_segundo=$?
set -e
wait "$pid_primero"
if [[ $estado_segundo -eq 0 ]]; then
    echo 'la concurrencia creo dos propuestas para una necesidad' >&2
    exit 1
fi

filas=$(docker exec "$contenedor" psql -X --quiet --tuples-only \
    --username postgres --dbname "$base" --command \
    "SELECT count(*) FROM vec_bolsa_llamamientos.propuesta WHERE necesidad_ref='necesidad-concurrencia'" \
    | tr -d '[:space:]')
if [[ $filas != 1 ]]; then
    echo "unicidad concurrente invalida: $filas" >&2
    exit 1
fi

docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
REVOKE vec_bolsa_llamamientos_ejecutor
    FROM vec_llamamientos_runtime_prueba;
DROP ROLE vec_llamamientos_runtime_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones/000002_guardado_cerrado.down.sql
docker exec --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "SET vec.confirmar_destruccion_bolsa_llamamientos='DESTRUIR_HISTORIA_BOLSA_LLAMAMIENTOS_IRREVERSIBLE';" \
    --file /dev/stdin < "$raiz/deploy/postgresql/bolsa_llamamientos/migraciones/000001_almacen_llamamientos.down.sql"
psql_archivo deploy/postgresql/bolsa_llamamientos/migraciones_autorizacion/000001_revalidacion_llamamientos.down.sql
psql_archivo deploy/postgresql/bolsa_llamamientos/roles_down.sql

echo 'integracion PostgreSQL de llamamientos: OK (frontera COSE cerrada)'
