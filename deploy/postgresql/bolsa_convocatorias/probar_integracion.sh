#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-convocatorias-pg-${USER:-usuario}-$$"
base=vec_bolsa_convocatorias_prueba

generar_clave() {
    local destino=$1 valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "no se pudo generar una clave de prueba" >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_ejecutor=
clave_proyector=
clave_registrador=
generar_clave clave_admin
generar_clave clave_ejecutor
generar_clave clave_proyector
generar_clave clave_registrador

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
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
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

rechazar_runtime() {
    local usuario=$1 clave=$2 consulta=$3 descripcion=$4
    if docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" \
        --command "$consulta" >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/roles_up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000001_revalidacion_convocatorias.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000001_almacen_convocatorias.up.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000002_consulta_exacta_cerrada.up.sql

docker exec --interactive \
    --env CLAVE_EJECUTOR="$clave_ejecutor" \
    --env CLAVE_PROYECTOR="$clave_proyector" \
    --env CLAVE_REGISTRADOR="$clave_registrador" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
\getenv clave_proyector CLAVE_PROYECTOR
\getenv clave_registrador CLAVE_REGISTRADOR
CREATE ROLE vec_convocatorias_ejecutor_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecutor';
CREATE ROLE vec_convocatorias_proyector_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_proyector';
CREATE ROLE vec_convocatorias_registrador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registrador';
GRANT vec_bolsa_convocatorias_ejecutor_consulta
    TO vec_convocatorias_ejecutor_prueba;
GRANT vec_bolsa_convocatorias_proyector_gobierno
    TO vec_convocatorias_proyector_prueba;
GRANT vec_bolsa_convocatorias_registrador_atestacion
    TO vec_convocatorias_registrador_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_convocatorias/pruebas_sql/acl_cierre.sql

rechazar_runtime vec_convocatorias_ejecutor_prueba "$clave_ejecutor" \
    'SELECT * FROM vec_bolsa_convocatorias.version_convocatoria' \
    'el ejecutor pudo leer tablas'
rechazar_runtime vec_convocatorias_ejecutor_prueba "$clave_ejecutor" \
    "SELECT * FROM vec_bolsa_convocatorias.obtener_version_exacta_v1('{}','{}',''::bytea,''::bytea)" \
    'el ejecutor pudo invocar la consulta antes del registrador COSE'
rechazar_runtime vec_convocatorias_proyector_prueba "$clave_proyector" \
    'INSERT INTO vec_bolsa_convocatorias.version_convocatoria DEFAULT VALUES' \
    'el proyector reservado pudo escribir tablas'
rechazar_runtime vec_convocatorias_registrador_prueba "$clave_registrador" \
    'INSERT INTO vec_bolsa_convocatorias.atestacion_autorizacion_version DEFAULT VALUES' \
    'el registrador reservado pudo fabricar atestaciones'

# Invariantes de huella e inmutabilidad: todo se revierte y no deja fixtures.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
DO $huella_rechazada$
BEGIN
    BEGIN
        INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
            convocatoria_id, secuencia, referencia, estado,
            version_canonica, huella_version_sha256, registrada_en
        ) VALUES (
            'conv-prueba', 1, 'conv-prueba#1', 'borrador',
            convert_to('{}', 'UTF8'), repeat('0', 64), clock_timestamp()
        );
        RAISE EXCEPTION 'se acepto una huella falsa';
    EXCEPTION WHEN check_violation THEN
        NULL;
    END;
END
$huella_rechazada$;
INSERT INTO vec_bolsa_convocatorias.version_convocatoria(
    convocatoria_id, secuencia, referencia, estado,
    version_canonica, huella_version_sha256, registrada_en
) VALUES (
    'conv-prueba', 1, 'conv-prueba#1', 'borrador',
    convert_to('{}', 'UTF8'),
    encode(sha256(convert_to('{}', 'UTF8')), 'hex'), clock_timestamp()
);
DO $inmutable$
BEGIN
    BEGIN
        UPDATE vec_bolsa_convocatorias.version_convocatoria
           SET estado = 'publicada'
         WHERE convocatoria_id = 'conv-prueba' AND secuencia = 1;
        RAISE EXCEPTION 'se permitio mutar historia';
    EXCEPTION WHEN object_not_in_prerequisite_state THEN
        NULL;
    END;
END
$inmutable$;
ROLLBACK;
SQL

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_bolsa_convocatorias_ejecutor_consulta
    FROM vec_convocatorias_ejecutor_prueba;
REVOKE vec_bolsa_convocatorias_proyector_gobierno
    FROM vec_convocatorias_proyector_prueba;
REVOKE vec_bolsa_convocatorias_registrador_atestacion
    FROM vec_convocatorias_registrador_prueba;
DROP ROLE vec_convocatorias_ejecutor_prueba;
DROP ROLE vec_convocatorias_proyector_prueba;
DROP ROLE vec_convocatorias_registrador_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000002_consulta_exacta_cerrada.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones/000001_almacen_convocatorias.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/migraciones_autorizacion/000001_revalidacion_convocatorias.down.sql
psql_archivo deploy/postgresql/bolsa_convocatorias/roles_down.sql

echo 'integracion PostgreSQL de convocatorias: OK'
