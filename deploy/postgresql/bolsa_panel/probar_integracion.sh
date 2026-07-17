#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-panel-pg-${USER:-usuario}-$$"
base=vec_bolsa_panel_prueba

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
psql_archivo deploy/postgresql/autorizacion/roles_v2_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
psql_archivo deploy/postgresql/bolsa_panel/roles_up.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones_autorizacion/000001_revalidacion_panel_v2.up.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones/000001_proyeccion_panel.up.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones/000002_publicador_proyeccion.up.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones/000003_consulta_panel_cerrada.up.sql

docker exec --interactive \
    --env CLAVE_EJECUTOR="$clave_ejecutor" \
    --env CLAVE_PROYECTOR="$clave_proyector" \
    --env CLAVE_REGISTRADOR="$clave_registrador" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
\getenv clave_proyector CLAVE_PROYECTOR
\getenv clave_registrador CLAVE_REGISTRADOR
CREATE ROLE vec_panel_ejecutor_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecutor';
CREATE ROLE vec_panel_proyector_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_proyector';
CREATE ROLE vec_panel_registrador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registrador';
GRANT vec_bolsa_panel_ejecutor_consulta TO vec_panel_ejecutor_prueba;
GRANT vec_bolsa_panel_proyector TO vec_panel_proyector_prueba;
GRANT vec_bolsa_panel_registrador_atestacion
    TO vec_panel_registrador_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_panel/pruebas_sql/acl_cierre.sql

rechazar_runtime vec_panel_ejecutor_prueba "$clave_ejecutor" \
    "SELECT * FROM vec_bolsa_panel.consultar_panel_interno_v1('{}','{}',''::bytea,''::bytea,'correlacion_0123456789abcdef0123456789abcdef')" \
    'el ejecutor invoco la consulta sin autoridad COSE productiva'
rechazar_runtime vec_panel_ejecutor_prueba "$clave_ejecutor" \
    'SELECT * FROM vec_bolsa_panel.consulta_confirmada' \
    'el ejecutor leyo consumos'
rechazar_runtime vec_panel_proyector_prueba "$clave_proyector" \
    'SELECT * FROM vec_bolsa_panel.proyeccion_panel' \
    'el proyector leyo tablas directamente'
rechazar_runtime vec_panel_registrador_prueba "$clave_registrador" \
    'INSERT INTO vec_bolsa_panel.atestacion_autorizacion_version DEFAULT VALUES' \
    'el registrador reservado fabrico una atestacion'

psql_archivo deploy/postgresql/bolsa_panel/pruebas_sql/integracion_mecanica_cerrada.sql

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_bolsa_panel_ejecutor_consulta FROM vec_panel_ejecutor_prueba;
REVOKE vec_bolsa_panel_proyector FROM vec_panel_proyector_prueba;
REVOKE vec_bolsa_panel_registrador_atestacion
    FROM vec_panel_registrador_prueba;
DROP ROLE vec_panel_ejecutor_prueba;
DROP ROLE vec_panel_proyector_prueba;
DROP ROLE vec_panel_registrador_prueba;
SQL

psql_archivo deploy/postgresql/bolsa_panel/migraciones/000003_consulta_panel_cerrada.down.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones/000002_publicador_proyeccion.down.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones/000001_proyeccion_panel.down.sql
psql_archivo deploy/postgresql/bolsa_panel/migraciones_autorizacion/000001_revalidacion_panel_v2.down.sql
psql_archivo deploy/postgresql/bolsa_panel/roles_down.sql

echo 'integracion PostgreSQL del panel interno de bolsas: OK'
