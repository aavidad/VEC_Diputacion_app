#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
base=vec_bolsa_firma_prueba
contenedor="vec-bolsa-firma-${USER:-usuario}-$$"

generar_clave() {
    local destino=$1
    local valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "no se pudo generar una clave de prueba segura" >&2
        return 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_ejecutor=
generar_clave clave_admin
generar_clave clave_ejecutor

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
    --volume "$raiz:/workspace:ro" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" \
    --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready \
        --username postgres --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready \
    --username postgres --dbname "$base" >/dev/null

psql_archivo() {
    local archivo=$1
    docker exec --interactive "$contenedor" \
        psql -X --quiet --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname "$base" \
        --file "/workspace/$archivo"
}

psql_consulta() {
    local consulta=$1
    docker exec "$contenedor" psql -X --quiet --tuples-only --no-align \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "$consulta"
}

psql_archivo deploy/postgresql/bolsa_firma/roles_up.sql
psql_archivo \
    deploy/postgresql/bolsa_firma/migraciones/000001_almacen_flujo_firma.up.sql
psql_archivo \
    deploy/postgresql/bolsa_firma/migraciones/000002_operaciones_flujo_firma.up.sql

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --set clave="$clave_ejecutor" \
    --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_bolsa_firma_login_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave';
GRANT vec_bolsa_firma_ejecutor TO vec_bolsa_firma_login_prueba;
SQL

puerto=
dsn=
actualizar_conexion() {
    puerto=$(docker port "$contenedor" 5432/tcp |
        sed -n 's/.*://p' | head -1)
    if [[ -z "$puerto" || "$puerto" == *[!0-9]* ]]; then
        echo "Docker no publicó un puerto PostgreSQL válido" >&2
        return 1
    fi
    dsn="postgres://vec_bolsa_firma_login_prueba:${clave_ejecutor}@127.0.0.1:${puerto}/${base}?sslmode=disable"
}
actualizar_conexion

esperar_postgres_publicado() {
    for _ in $(seq 1 100); do
        if docker exec "$contenedor" pg_isready \
            --username postgres --dbname "$base" >/dev/null 2>&1 &&
            (exec 3<>"/dev/tcp/127.0.0.1/${puerto}") 2>/dev/null; then
            return 0
        fi
        sleep 0.1
    done
    echo "PostgreSQL reiniciado no quedó accesible por el puerto publicado" >&2
    return 1
}

ejecutar_fase() {
    local fase=$1
    (
        cd "$raiz"
        VEC_BOLSA_FIRMA_POSTGRES_E2E_DSN="$dsn" \
        VEC_BOLSA_FIRMA_POSTGRES_E2E_ADMIN_DSN="postgres://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable" \
        VEC_BOLSA_FIRMA_POSTGRES_E2E_FASE="$fase" \
            go test ./internal/modules/bolsa/adapters/postgres \
                -run '^TestFlujoFirmaPostgreSQLE2E$' -count=1
    )
}

ejecutar_fase crear
docker restart "$contenedor" >/dev/null
actualizar_conexion
esperar_postgres_publicado
ejecutar_fase continuar
docker restart "$contenedor" >/dev/null
actualizar_conexion
esperar_postgres_publicado
ejecutar_fase verificar

if [[ $(psql_consulta "
    SELECT count(*) FROM information_schema.role_table_grants
     WHERE grantee = 'vec_bolsa_firma_ejecutor'
") != 0 ]]; then
    echo "el ejecutor recibió privilegios directos sobre tablas" >&2
    exit 1
fi

if [[ $(psql_consulta "
    SELECT count(*)
      FROM information_schema.routine_privileges
     WHERE grantee = 'vec_bolsa_firma_ejecutor'
       AND routine_schema = 'vec_bolsa_firma'
") != 5 ]]; then
    echo "la superficie ejecutable no coincide con las cinco fachadas" >&2
    exit 1
fi

estado=$(psql_consulta "
    SELECT
      (SELECT version_actual FROM vec_bolsa_firma.flujo) || '|' ||
      (SELECT count(*) FROM vec_bolsa_firma.version_flujo) || '|' ||
      (SELECT count(*) FROM vec_bolsa_firma.auditoria) || '|' ||
      (SELECT count(*) FROM vec_bolsa_firma.outbox) || '|' ||
      (SELECT count(*) FROM (
          SELECT huella_anterior_sha256,
                 lag(huella_evento_sha256) OVER (ORDER BY secuencia) AS anterior
            FROM vec_bolsa_firma.auditoria
      ) AS cadena
       WHERE huella_anterior_sha256 IS DISTINCT FROM anterior) || '|' ||
      (SELECT count(*) FROM vec_bolsa_firma.auditoria
        WHERE huella_evento_sha256 IS DISTINCT FROM encode(sha256(convert_to(
          concat_ws(chr(31), 'vec.bolsa.firma.auditoria.v1',
            coalesce(huella_anterior_sha256, ''), flujo_ref,
            coalesce(version::text, ''), tipo_evento, detalle::text,
            to_char(ocurrido_en AT TIME ZONE 'UTC',
                    'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')),
          'UTF8'
        )), 'hex'))
")
if [[ "$estado" != "3|3|7|3|0|0" ]]; then
    echo "estado durable o cadena de auditoría inesperados: $estado" >&2
    exit 1
fi

salida=$(mktemp)
if docker exec "$contenedor" env PGPASSWORD="$clave_ejecutor" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_bolsa_firma_login_prueba --dbname "$base" \
    --command "SELECT count(*) FROM vec_bolsa_firma.flujo" \
    >"$salida" 2>&1; then
    echo "el ejecutor pudo leer una tabla directamente" >&2
    rm -f "$salida"
    exit 1
fi
rm -f "$salida"

salida=$(mktemp)
if docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command "
      SET ROLE vec_bolsa_firma_propietario;
      UPDATE vec_bolsa_firma.version_flujo SET version = version + 10;
    " >"$salida" 2>&1; then
    echo "la historia aceptó una reescritura" >&2
    rm -f "$salida"
    exit 1
fi
if ! rg -q 'historia de firma es inmutable' "$salida"; then
    sed -n '1,100p' "$salida" >&2
    rm -f "$salida"
    exit 1
fi
rm -f "$salida"

docker exec --env \
    PGOPTIONS="-c vec.confirmar_reversion_bolsa_firma=REVERTIR_OPERACIONES_FIRMA_BOLSA" \
    --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --file /workspace/deploy/postgresql/bolsa_firma/migraciones/000002_operaciones_flujo_firma.down.sql
docker exec --env \
    PGOPTIONS="-c vec.confirmar_reversion_bolsa_firma=REVERTIR_ALMACEN_FIRMA_BOLSA" \
    --interactive "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --file /workspace/deploy/postgresql/bolsa_firma/migraciones/000001_almacen_flujo_firma.down.sql
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command "DROP ROLE vec_bolsa_firma_login_prueba" >/dev/null
psql_archivo deploy/postgresql/bolsa_firma/roles_down.sql

echo "integración PostgreSQL 18 de firma de Bolsa: OK"
