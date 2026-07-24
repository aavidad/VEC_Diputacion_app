#!/usr/bin/env bash
set -euo pipefail
umask 077

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
base=vec_bolsa_baremacion_v3_prueba
origen="vec-bolsa-copia-origen-${USER:-usuario}-$$"
destino="vec-bolsa-copia-destino-${USER:-usuario}-$$"
volumen="vec-bolsa-copia-${USER:-usuario}-$$"
clave_admin=

generar_clave_prueba() {
    if [[ ! -r /dev/urandom ]] \
        || ! clave_admin=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]') \
        || [[ ${#clave_admin} -ne 64 || $clave_admin == *[!0-9a-f]* ]]; then
        echo "no se pudo generar una clave de prueba segura" >&2
        return 1
    fi
}

limpiar() {
    docker rm -f "$origen" "$destino" >/dev/null 2>&1 || true
    docker volume rm -f "$volumen" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

arrancar_postgres() {
    local nombre=$1
    local base_inicial=$2
    docker run --detach --rm --name "$nombre" \
        --volume "$raiz:/workspace:ro" \
        --volume "$volumen:/copias" \
        --env POSTGRES_DB="$base_inicial" \
        --env POSTGRES_PASSWORD="$clave_admin" \
        "$imagen" >/dev/null
    for _ in $(seq 1 60); do
        if docker exec "$nombre" pg_isready --username postgres \
            --dbname "$base_inicial" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    docker exec "$nombre" pg_isready --username postgres \
        --dbname "$base_inicial" >/dev/null
}

psql_archivo() {
    local nombre=$1
    local archivo=$2
    shift 2
    docker exec --interactive "$nombre" psql -X --quiet \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose "$@" \
        --username postgres --dbname "$base" \
        --file "/workspace/$archivo"
}

psql_archivo_con_mantenimiento_v3() {
    local nombre=$1
    local archivo=$2
    docker exec --interactive \
        --env PGOPTIONS="-c vec.confirmar_mantenimiento_bolsa_baremacion_v3=INSTALAR_MIGRACION_BOLSA_BAREMACION_V3_SIN_TRAFICO" \
        "$nombre" psql -X --quiet --set ON_ERROR_STOP=1 \
        --set VERBOSITY=verbose \
        --username postgres --dbname "$base" \
        --file "/workspace/$archivo"
}

instalar_base() {
    local nombre=$1
    local archivo
    local archivos=(
        deploy/postgresql/autorizacion/roles_up.sql
        deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
        deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
        deploy/postgresql/bolsa_baremacion/roles_up.sql
        deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.up.sql
        deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.up.sql
        deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.up.sql
        deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.up.sql
        deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.up.sql
    )
    for archivo in "${archivos[@]}"; do
        psql_archivo "$nombre" "$archivo"
    done
}

huella_estado() {
    local nombre=$1
    docker exec --interactive "$nombre" psql -X --quiet \
        --tuples-only --no-align --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" \
        --file /workspace/deploy/postgresql/bolsa_baremacion/pruebas_sql/huella_estado_restaurable_v3.sql \
        | sha256sum | cut -d ' ' -f 1
}

verificar_roles_sin_credenciales() {
    local nombre=$1
    local roles
    roles=$(docker exec "$nombre" psql -X --quiet --tuples-only --no-align \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "SELECT string_agg(rolname, ',' ORDER BY rolname)
                     FROM pg_catalog.pg_roles
                    WHERE rolname='postgres' OR rolname LIKE 'vec\\_%' ESCAPE '\\'")
    if [[ "$roles" != "postgres,vec_autorizacion_fuente,vec_autorizacion_migrador,vec_autorizacion_propietario,vec_autorizacion_registro,vec_bolsa_baremacion_ejecutor,vec_bolsa_baremacion_lector_outbox,vec_bolsa_baremacion_migrador,vec_bolsa_baremacion_propietario,vec_bolsa_baremacion_registrador_atestacion" ]]; then
        echo "el inventario de roles previo a la copia no es el esperado" >&2
        return 1
    fi
    if docker exec "$nombre" sh -c \
        "grep -Eiq 'PASSWORD|SCRAM-SHA-256|md5[0-9a-f]{32}' /copias/roles.sql"; then
        echo "la copia de roles contiene material de autenticacion" >&2
        return 1
    fi
}

generar_clave_prueba
docker volume create "$volumen" >/dev/null
arrancar_postgres "$origen" "$base"
docker exec --user root "$origen" \
    install -d -o postgres -g postgres -m 0700 /copias

instalar_base "$origen"
psql_archivo "$origen" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/integracion_v1.sql \
    --set CONFIRMAR_FIXTURE=1
psql_archivo_con_mantenimiento_v3 "$origen" \
    deploy/postgresql/bolsa_baremacion/migraciones/000005_manifiesto_probatorio_v3.up.sql
psql_archivo "$origen" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/integracion_v3.sql \
    --set RUTA_MANIFIESTO_DORADO_V3=/workspace/internal/modules/bolsa/testdata/manifiesto_probatorio_v3_dorado.json \
    --set CONFIRMAR_FIXTURE_V3=1

huella_origen=$(huella_estado "$origen")
docker exec "$origen" pg_dumpall --username postgres --roles-only \
    --no-role-passwords --file /copias/roles.sql
docker exec "$origen" sed -i '/^CREATE ROLE postgres;$/d' /copias/roles.sql
verificar_roles_sin_credenciales "$origen"
docker exec "$origen" pg_dump --username postgres --dbname "$base" \
    --format=custom --create --no-password --file /copias/bolsa.dump
docker exec "$origen" sh -c \
    'cd /copias && sha256sum roles.sql bolsa.dump > SHA256SUMS'
docker exec "$origen" chmod 0600 \
    /copias/roles.sql /copias/bolsa.dump /copias/SHA256SUMS

# Se elimina la instancia completa: el destino no comparte su volumen de datos.
docker rm -f "$origen" >/dev/null
arrancar_postgres "$destino" postgres
docker exec "$destino" sh -c 'cd /copias && sha256sum --check SHA256SUMS'
docker exec --interactive "$destino" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname postgres \
    --file /copias/roles.sql
docker exec "$destino" pg_restore --username postgres --dbname postgres \
    --create --exit-on-error --no-password /copias/bolsa.dump

docker restart "$destino" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$destino" pg_isready --username postgres \
        --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$destino" pg_isready --username postgres \
    --dbname "$base" >/dev/null

huella_destino=$(huella_estado "$destino")
if [[ "$huella_origen" != "$huella_destino" ]]; then
    echo "la restauracion no conserva exactamente el estado funcional" >&2
    exit 1
fi

psql_archivo "$destino" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/inventario_global_restaurado_v3.sql
psql_archivo "$destino" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/acl_inventario_v3.sql
psql_archivo "$destino" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/recuperacion_reinicio_v3.sql
psql_archivo "$destino" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/lectura_exacta_replay_v3.sql
psql_archivo "$destino" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/lectura_evidencia_replay_v3.sql

echo "copia y restauracion logica PostgreSQL 18.4 de Bolsa V3: correcta"
