#!/usr/bin/env bash
set -Eeuo pipefail

DIRECTORIO_SCRIPT="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
readonly DIRECTORIO_SCRIPT
readonly IMAGEN_PREDETERMINADA='postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296'
readonly IMAGEN_POSTGRES="${IMAGEN_POSTGRES:-${IMAGEN_PREDETERMINADA}}"
readonly CONTENEDOR="vec-ct-pg-${PPID}-${RANDOM}"
readonly CLAVE_EFIMERA='solo-prueba-efimera-no-reutilizar'

limpiar() {
    docker rm --force --volumes "${CONTENEDOR}" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

paso() {
    printf '[contratacion-temporal:postgres] %s\n' "$1"
}

esperar_postgresql() {
    local _intento
    for _intento in {1..60}; do
        if docker exec "${CONTENEDOR}" \
            pg_isready --quiet --username postgres --dbname postgres; then
            return 0
        fi
        if [[ "$(docker inspect --format '{{.State.Running}}' \
            "${CONTENEDOR}" 2>/dev/null || true)" != 'true' ]]; then
            docker logs "${CONTENEDOR}" >&2 || true
            return 1
        fi
        sleep 1
    done
    docker logs "${CONTENEDOR}" >&2 || true
    return 1
}

ejecutar_como() {
    local usuario="$1"
    local fichero="$2"
    docker exec "${CONTENEDOR}" \
        psql -X --set ON_ERROR_STOP=1 \
        --username "${usuario}" \
        --dbname postgres \
        --file "/pruebas/${fichero}"
}

esperar_fallo() {
    local descripcion="$1"
    local salida
    shift
    if salida="$("$@" 2>&1)"; then
        printf 'Se esperaba rechazo: %s\n' "${descripcion}" >&2
        printf '%s\n' "${salida}" >&2
        return 1
    fi
    paso "rechazo verificado: ${descripcion}"
}

esperar_fallo_con_patron() {
    local descripcion="$1"
    local patron="$2"
    local salida
    shift 2
    if salida="$("$@" 2>&1)"; then
        printf 'Se esperaba rechazo: %s\n' "${descripcion}" >&2
        printf '%s\n' "${salida}" >&2
        return 1
    fi
    if ! grep -Fq "${patron}" <<<"${salida}"; then
        printf 'Rechazo distinto del esperado: %s\n' "${descripcion}" >&2
        printf '%s\n' "${salida}" >&2
        return 1
    fi
    paso "rechazo verificado: ${descripcion}"
}

ejecutar_pgbench_verificado() {
    local descripcion="$1"
    local fichero="$2"
    local salida
    if ! salida="$(
        docker exec "${CONTENEDOR}" \
            pgbench --no-vacuum \
            --client=8 --jobs=4 --transactions=2 \
            --max-tries=3 \
            --username vec_ct_runtime_prueba \
            --file "/pruebas/pruebas_sql/${fichero}" \
            postgres 2>&1
    )"; then
        printf '%s\n' "${salida}" >&2
        return 1
    fi
    printf '%s\n' "${salida}"
    if ! grep -Fq \
        'number of transactions actually processed: 16/16' \
        <<<"${salida}" ||
        ! grep -Eq \
            '^number of failed transactions: 0([[:space:]]|$)' \
            <<<"${salida}"; then
        printf 'pgbench incompleto: %s\n' "${descripcion}" >&2
        return 1
    fi
}

paso "imagen fijada: ${IMAGEN_POSTGRES}"
docker run --detach \
    --name "${CONTENEDOR}" \
    --network none \
    --env POSTGRES_PASSWORD="${CLAVE_EFIMERA}" \
    --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
    --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=512m \
    "${IMAGEN_POSTGRES}" >/dev/null
esperar_postgresql
docker cp "${DIRECTORIO_SCRIPT}/." "${CONTENEDOR}:/pruebas"

paso 'bootstrap de roles técnicos'
ejecutar_como postgres roles_up.sql >/dev/null
docker exec --interactive "${CONTENEDOR}" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres <<'SQL' >/dev/null
CREATE ROLE vec_ct_migrador_prueba
    LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT
    NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_ct_runtime_prueba
    LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_migrador
    TO vec_ct_migrador_prueba
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
GRANT vec_contratacion_temporal_ejecutor
    TO vec_ct_runtime_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL

paso 'instalación transaccional con identidad de migración'
ejecutar_como \
    vec_ct_migrador_prueba \
    migraciones/000001_preparacion_altas.up.sql >/dev/null

paso 'privilegios mínimos y separación de funciones'
ejecutar_como postgres pruebas_sql/verificar_privilegios.sql >/dev/null
esperar_fallo \
    'invocación de runtime desde una identidad de migración' \
    docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_migrador_prueba --dbname postgres \
    --command \
    "SELECT * FROM vec_contratacion_temporal.preparar_alta_v1('{}'::jsonb)"
esperar_fallo \
    'lectura directa por runtime' \
    docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_runtime_prueba --dbname postgres \
    --command 'TABLE vec_contratacion_temporal.identidad_reserva_alta'
esperar_fallo \
    'escalada de runtime a propietario' \
    docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_runtime_prueba --dbname postgres \
    --command 'SET ROLE vec_contratacion_temporal_propietario'

paso 'alta, reintento estable y conflicto semántico'
ejecutar_como \
    vec_ct_runtime_prueba \
    pruebas_sql/integracion_preparacion.sql >/dev/null
ejecutar_como \
    vec_ct_migrador_prueba \
    pruebas_sql/confirmar_reserva_v1_prueba.sql >/dev/null

paso 'concurrencia real: ocho sesiones y propuestas distintas'
ejecutar_pgbench_verificado \
    'preparación v1' \
    concurrencia_preparacion.sql
ejecutar_como \
    vec_ct_migrador_prueba \
    pruebas_sql/verificar_concurrencia.sql >/dev/null

paso 'migración aditiva v2 y retención obligatoria de v1'
ejecutar_como \
    vec_ct_migrador_prueba \
    migraciones/000002_rotacion_hmac.up.sql >/dev/null
ejecutar_como \
    vec_ct_migrador_prueba \
    pruebas_sql/verificar_privilegios_rotacion.sql >/dev/null
ejecutar_como \
    vec_ct_runtime_prueba \
    pruebas_sql/rechazar_limites_ambientales_rotacion.sql >/dev/null
esperar_fallo \
    'runtime conserva acceso a la función v1 revocada' \
    docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_runtime_prueba --dbname postgres \
    --command \
    "SELECT * FROM vec_contratacion_temporal.preparar_alta_v1('{}'::jsonb)"

paso 'rotación v1 a v2, alta nativa, conflicto y fallo cerrado'
ejecutar_como \
    vec_ct_runtime_prueba \
    pruebas_sql/integracion_rotacion_hmac.sql >/dev/null

paso 'límites propios de la función ante bloqueo directo'
docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_migrador_prueba --dbname postgres \
    --command \
    "BEGIN; SET LOCAL ROLE vec_contratacion_temporal_propietario; SELECT pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_ct:ambito:hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:$(printf 'd%.0s' {1..64})', 0)); SELECT pg_sleep(5); COMMIT" \
    >/dev/null 2>&1 &
pid_bloqueo=$!
for _intento in {1..40}; do
    if [[ "$(
        docker exec "${CONTENEDOR}" \
            psql -X --tuples-only --no-align \
            --username postgres --dbname postgres \
            --command \
            "SELECT EXISTS (SELECT 1 FROM pg_catalog.pg_locks WHERE locktype = 'advisory' AND granted AND pid <> pg_backend_pid())"
    )" == 't' ]]; then
        break
    fi
    sleep 0.05
done
esperar_fallo_con_patron \
    'bloqueo directo limitado por preparar_alta_v2' \
    'canceling statement due to lock timeout' \
    ejecutar_como \
    vec_ct_runtime_prueba \
    pruebas_sql/bloqueo_directo_rotacion.sql
esperar_fallo_con_patron \
    'sentencia directa limitada antes de entrar en preparar_alta_v2' \
    'canceling statement due to statement timeout' \
    ejecutar_como \
    vec_ct_runtime_prueba \
    pruebas_sql/timeout_sentencia_directo_rotacion.sql
wait "${pid_bloqueo}"

paso 'inactividad transaccional limitada después del retorno'
esperar_fallo_con_patron \
    'conexión runtime inactiva terminada por el servidor' \
    'terminating connection due to idle-in-transaction timeout' \
    ejecutar_como \
    vec_ct_runtime_prueba \
    pruebas_sql/timeout_inactividad_directo_rotacion.sql
sesiones_runtime="$(
    docker exec "${CONTENEDOR}" \
        psql -X --tuples-only --no-align \
        --username postgres --dbname postgres \
        --command \
        "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE usename = 'vec_ct_runtime_prueba'"
)"
if [[ "${sesiones_runtime}" != '0' ]]; then
    printf 'Quedaron sesiones runtime tras el timeout: %s\n' \
        "${sesiones_runtime}" >&2
    exit 1
fi

paso 'concurrencia real: ocho sesiones con generaciones v2 y v1'
ejecutar_pgbench_verificado \
    'rotación v2 con v1 retenida' \
    concurrencia_rotacion_hmac.sql
ejecutar_como \
    vec_ct_migrador_prueba \
    pruebas_sql/verificar_concurrencia_rotacion.sql >/dev/null

paso 'inmutabilidad de identidad e historia'
esperar_fallo \
    'mutación de la identidad por el propietario' \
    docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_migrador_prueba --dbname postgres \
    --command \
    "BEGIN; SET LOCAL ROLE vec_contratacion_temporal_propietario; UPDATE vec_contratacion_temporal.identidad_reserva_alta SET actor_ref = 'actor:alterado'; COMMIT"
esperar_fallo \
    'mutación de un alias HMAC por el propietario' \
    docker exec "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_migrador_prueba --dbname postgres \
    --command \
    "BEGIN; SET LOCAL ROLE vec_contratacion_temporal_propietario; UPDATE vec_contratacion_temporal.alias_ambito_alta SET registrada_en = clock_timestamp(); COMMIT"

paso 'rollback destructivo cerrado por defecto'
esperar_fallo \
    'rollback v2 con historia sin autorización explícita' \
    ejecutar_como \
    vec_ct_migrador_prueba \
    migraciones/000002_rotacion_hmac.down.sql

paso 'rollback destructivo autorizado y limpieza total'
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_migrador_prueba \
    --dbname postgres \
    --file /pruebas/migraciones/000002_rotacion_hmac.down.sql \
    >/dev/null
docker exec \
    --env PGOPTIONS='-c vec.confirmar_destruccion_contratacion_temporal=DESTRUIR_HISTORIA_CONTRATACION_TEMPORAL_IRREVERSIBLE' \
    "${CONTENEDOR}" \
    psql -X --set ON_ERROR_STOP=1 \
    --username vec_ct_migrador_prueba \
    --dbname postgres \
    --file /pruebas/migraciones/000001_preparacion_altas.down.sql \
    >/dev/null
docker exec --interactive "${CONTENEDOR}" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres <<'SQL' >/dev/null
DROP ROLE vec_ct_runtime_prueba;
DROP ROLE vec_ct_migrador_prueba;
SQL
ejecutar_como postgres roles_down.sql >/dev/null

docker exec --interactive "${CONTENEDOR}" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres <<'SQL' >/dev/null
DO $prueba$
BEGIN
    IF to_regnamespace('vec_contratacion_temporal') IS NOT NULL THEN
        RAISE EXCEPTION 'el esquema no fue retirado';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_contratacion_temporal_%'
            OR rolname LIKE 'vec_ct_%_prueba'
    ) THEN
        RAISE EXCEPTION 'quedaron roles de contratación temporal';
    END IF;
END
$prueba$;
SQL

paso 'OK: rotación HMAC, mínimo privilegio, idempotencia y limpieza'
