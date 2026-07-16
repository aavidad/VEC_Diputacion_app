#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
base=vec_bolsa_baremacion_v3_prueba
contenedor="vec-bolsa-v3-${USER:-usuario}-$$"
contenedor_vacio="${contenedor}-vacio"
salida_uno=$(mktemp)
salida_dos=$(mktemp)
salida_bloqueador=$(mktemp)
estado_e2e=$(mktemp)
pid_uno=
pid_dos=
pid_bloqueador=

generar_clave_prueba() {
    local destino=$1
    local valor
    if [[ ! -r /dev/urandom ]] \
        || ! valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]') \
        || [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "no se pudo generar una clave de prueba segura" >&2
        return 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_ejecutor=
generar_clave_prueba clave_admin
generar_clave_prueba clave_ejecutor
if [[ "$clave_admin" == "$clave_ejecutor" ]]; then
    echo "la fuente de entropia produjo claves repetidas" >&2
    exit 1
fi

limpiar() {
    [[ -z "$pid_uno" ]] || kill "$pid_uno" >/dev/null 2>&1 || true
    [[ -z "$pid_dos" ]] || kill "$pid_dos" >/dev/null 2>&1 || true
    [[ -z "$pid_bloqueador" ]] \
        || kill "$pid_bloqueador" >/dev/null 2>&1 || true
    docker rm -f "$contenedor" "$contenedor_vacio" >/dev/null 2>&1 || true
    rm -f "$salida_uno" "$salida_dos" "$salida_bloqueador" "$estado_e2e"
}
trap limpiar EXIT INT TERM

arrancar_postgres() {
    local nombre=$1
    docker run --detach --rm --name "$nombre" \
        --volume "$raiz:/workspace:ro" \
        --publish 127.0.0.1::5432 \
        --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
        "$imagen" >/dev/null
    for _ in $(seq 1 60); do
        if docker exec "$nombre" pg_isready --username postgres \
            --dbname "$base" >/dev/null 2>&1; then
            break
        fi
        sleep 1
    done
    docker exec "$nombre" pg_isready --username postgres \
        --dbname "$base" >/dev/null
}

reiniciar_postgres() {
    local nombre=$1
    docker restart "$nombre" >/dev/null
    for _ in $(seq 1 60); do
        if docker exec "$nombre" pg_isready --username postgres \
            --dbname "$base" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    docker exec "$nombre" pg_isready --username postgres \
        --dbname "$base" >/dev/null
}

puerto_postgres() {
    local nombre=$1
    docker port "$nombre" 5432/tcp | sed -n 's/.*://p' | head -1
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

psql_consulta() {
    local nombre=$1
    local consulta=$2
    docker exec "$nombre" psql -X --quiet --tuples-only --no-align \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "$consulta"
}

esperar_consulta_verdadera() {
    local nombre=$1
    local consulta=$2
    local descripcion=$3
    for _ in $(seq 1 100); do
        if [[ $(psql_consulta "$nombre" "$consulta" 2>/dev/null || true) == t ]]; then
            return 0
        fi
        sleep 0.1
    done
    echo "$descripcion" >&2
    return 1
}

psql_archivo_con_reversion_v3() {
    local nombre=$1
    local archivo=$2
    docker exec --interactive \
        --env PGOPTIONS="-c vec.confirmar_reversion_bolsa_baremacion_v3=REVERTIR_MIGRACION_BOLSA_BAREMACION_V3" \
        "$nombre" psql -X --quiet --set ON_ERROR_STOP=1 \
        --set VERBOSITY=verbose \
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

psql_archivo_serializable() {
    local nombre=$1
    local archivo=$2
    local aplicacion=$3
    shift 3
    docker exec --interactive \
        --env PGOPTIONS='-c default_transaction_isolation=serializable' \
        --env PGAPPNAME="$aplicacion" \
        "$nombre" psql -X --quiet --set ON_ERROR_STOP=1 \
        --set VERBOSITY=verbose "$@" \
        --username postgres --dbname "$base" \
        --file "/workspace/$archivo"
}

exigir_fallo_sqlstate_archivo() {
    local nombre=$1
    local archivo=$2
    local sqlstate=$3
    local descripcion=$4
    local salida
    salida=$(mktemp)
    if psql_archivo "$nombre" "$archivo" >"$salida" 2>&1; then
        echo "$descripcion" >&2
        rm -f "$salida"
        exit 1
    fi
    if ! rg -q "ERROR:[[:space:]]+${sqlstate}:" "$salida"; then
        sed -n '1,120p' "$salida" >&2
        echo "fallo sin SQLSTATE ${sqlstate}: ${descripcion}" >&2
        rm -f "$salida"
        exit 1
    fi
    rm -f "$salida"
}

exigir_fallo_sqlstate_mantenimiento_v3() {
    local nombre=$1
    local archivo=$2
    local sqlstate=$3
    local descripcion=$4
    local salida
    salida=$(mktemp)
    if psql_archivo_con_mantenimiento_v3 "$nombre" "$archivo" \
        >"$salida" 2>&1; then
        echo "$descripcion" >&2
        rm -f "$salida"
        exit 1
    fi
    if ! rg -q "ERROR:[[:space:]]+${sqlstate}:" "$salida"; then
        sed -n '1,120p' "$salida" >&2
        echo "fallo sin SQLSTATE ${sqlstate}: ${descripcion}" >&2
        rm -f "$salida"
        exit 1
    fi
    rm -f "$salida"
}

exigir_fallo_sqlstate_reversion_v3() {
    local nombre=$1
    local archivo=$2
    local sqlstate=$3
    local descripcion=$4
    local salida
    salida=$(mktemp)
    if psql_archivo_con_reversion_v3 "$nombre" "$archivo" \
        >"$salida" 2>&1; then
        echo "$descripcion" >&2
        rm -f "$salida"
        exit 1
    fi
    if ! rg -q "ERROR:[[:space:]]+${sqlstate}:" "$salida"; then
        sed -n '1,120p' "$salida" >&2
        echo "fallo sin SQLSTATE ${sqlstate}: ${descripcion}" >&2
        rm -f "$salida"
        exit 1
    fi
    rm -f "$salida"
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

crear_login_ejecutor() {
    local nombre=$1
    docker exec --interactive --env CLAVE_EJECUTOR="$clave_ejecutor" \
        "$nombre" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
CREATE ROLE vec_bolsa_v3_login_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecutor';
GRANT vec_bolsa_baremacion_ejecutor TO vec_bolsa_v3_login_prueba;
SQL
}

exigir_fallo_sqlstate_login_ejecutor() {
    local nombre=$1
    local consulta=$2
    local sqlstate=$3
    local descripcion=$4
    local salida
    salida=$(mktemp)
    if docker exec --env PGPASSWORD="$clave_ejecutor" "$nombre" \
        psql -X --quiet --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --host 127.0.0.1 --username vec_bolsa_v3_login_prueba \
        --dbname "$base" --command "$consulta" >"$salida" 2>&1; then
        echo "$descripcion" >&2
        rm -f "$salida"
        exit 1
    fi
    if ! rg -q "ERROR:[[:space:]]+${sqlstate}:" "$salida"; then
        sed -n '1,120p' "$salida" >&2
        echo "rechazo ACL sin SQLSTATE ${sqlstate}: ${descripcion}" >&2
        rm -f "$salida"
        exit 1
    fi
    rm -f "$salida"
}

probar_login_real() {
    local nombre=$1
    local resultado
    exigir_fallo_sqlstate_login_ejecutor "$nombre" \
        'SELECT * FROM vec_bolsa_baremacion.manifiesto_probatorio_v3' \
        42501 "el LOGIN ejecutor obtuvo acceso directo a la tabla V3"
    exigir_fallo_sqlstate_login_ejecutor "$nombre" \
        "SELECT vec_bolsa_baremacion.construir_archivo_probatorio_v3('baremacion:001',2)" \
        42501 "el LOGIN ejecutor pudo invocar un helper V3"
    exigir_fallo_sqlstate_login_ejecutor "$nombre" \
        "SELECT * FROM vec_bolsa_baremacion.confirmar_cambio('{}','{}',''::bytea,''::bytea,''::bytea)" \
        42501 "el LOGIN ejecutor conserva la confirmacion V1"
    resultado=$(docker exec --env PGPASSWORD="$clave_ejecutor" "$nombre" \
        psql -X --quiet --tuples-only --no-align --set ON_ERROR_STOP=1 \
        --host 127.0.0.1 --username vec_bolsa_v3_login_prueba \
        --dbname "$base" --command \
        "SELECT resultado FROM vec_bolsa_baremacion.reservar_cambio_con_archivo_probatorio_v3('{}','{}',''::bytea,''::bytea)")
    if [[ "$resultado" != "rechazada" ]]; then
        echo "la fachada V3 no fallo cerrada para el LOGIN real" >&2
        exit 1
    fi
    psql_consulta "$nombre" \
        'GRANT SELECT ON vec_bolsa_baremacion.manifiesto_probatorio_v3 TO vec_bolsa_v3_login_prueba' \
        >/dev/null
    resultado=$(docker exec --env PGPASSWORD="$clave_ejecutor" "$nombre" \
        psql -X --quiet --tuples-only --no-align --set ON_ERROR_STOP=1 \
        --host 127.0.0.1 --username vec_bolsa_v3_login_prueba \
        --dbname "$base" --command \
        'SELECT count(*) FROM vec_bolsa_baremacion.manifiesto_probatorio_v3')
    if [[ "$resultado" != "0" ]]; then
        echo "RLS no oculto la historia al LOGIN con GRANT accidental" >&2
        exit 1
    fi
    psql_consulta "$nombre" \
        'REVOKE SELECT ON vec_bolsa_baremacion.manifiesto_probatorio_v3 FROM vec_bolsa_v3_login_prueba' \
        >/dev/null
}

ejecutar_e2e_go() {
    local nombre=$1
    local fase=$2
    local puerto
    puerto=$(puerto_postgres "$nombre")
    if [[ -z "$puerto" ]]; then
        echo "no se pudo resolver el puerto PostgreSQL publicado" >&2
        return 1
    fi
    env \
        VEC_BOLSA_POSTGRES_E2E_DSN="postgres://vec_bolsa_v3_login_prueba:${clave_ejecutor}@127.0.0.1:${puerto}/${base}?sslmode=disable" \
        VEC_BOLSA_POSTGRES_E2E_ADMIN_DSN="postgres://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable" \
        VEC_BOLSA_POSTGRES_E2E_FASE="$fase" \
        VEC_BOLSA_POSTGRES_E2E_ESTADO="$estado_e2e" \
        go test -count=1 \
            -run '^TestIntegracionBaremacionPostgreSQLV3Real$' \
            ./internal/modules/bolsa/adapters/postgres
}

migracion_v3=deploy/postgresql/bolsa_baremacion/migraciones/000005_manifiesto_probatorio_v3.up.sql
reversion_v3=deploy/postgresql/bolsa_baremacion/migraciones/000005_manifiesto_probatorio_v3.down.sql
integracion_v1=deploy/postgresql/bolsa_baremacion/pruebas_sql/integracion_v1.sql
integracion_v3=deploy/postgresql/bolsa_baremacion/pruebas_sql/integracion_v3.sql
inventario_acl_v3=deploy/postgresql/bolsa_baremacion/pruebas_sql/acl_inventario_v3.sql
historia_no_reconstruible_v3=deploy/postgresql/bolsa_baremacion/pruebas_sql/preparar_historia_no_reconstruible_v3.sql
ruta_dorada=/workspace/internal/modules/bolsa/testdata/manifiesto_probatorio_v3_dorado.json

arrancar_postgres "$contenedor"
instalar_base "$contenedor"
psql_archivo "$contenedor" "$integracion_v1" --set CONFIRMAR_FIXTURE=1

# 000005 no es una migracion en caliente: sin confirmacion explicita debe
# fallar sin dejar ningun objeto ni privilegio V3.
exigir_fallo_sqlstate_archivo "$contenedor" "$migracion_v3" 55000 \
    "la migracion V3 se instalo sin ventana de mantenimiento"
if [[ $(psql_consulta "$contenedor" \
    "SELECT to_regclass('vec_bolsa_baremacion.manifiesto_probatorio_v3') IS NULL AND to_regprocedure('vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea,bytea,jsonb,bytea,bytea,bytea,text)') IS NULL") != "t" ]]; then
    echo "el up V3 sin opt-in dejo cambios parciales" >&2
    exit 1
fi

# Incluso con el literal correcto, una sesion LOGIN miembro del ejecutor
# demuestra trafico y obliga a abortar antes de tomar los locks de tablas.
docker exec --interactive --env CLAVE_EJECUTOR="$clave_ejecutor" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
CREATE ROLE vec_bolsa_v3_login_activo_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecutor';
GRANT vec_bolsa_baremacion_ejecutor TO vec_bolsa_v3_login_activo_prueba;
SQL
docker exec --env PGPASSWORD="$clave_ejecutor" \
    --env PGAPPNAME='vec_migracion_v3_trafico_prueba' \
    "$contenedor" psql -X --quiet --host 127.0.0.1 \
    --username vec_bolsa_v3_login_activo_prueba --dbname "$base" \
    --command 'SELECT pg_sleep(120)' >"$salida_bloqueador" 2>&1 &
pid_bloqueador=$!
if ! esperar_consulta_verdadera "$contenedor" \
    "SELECT count(*)=1 FROM pg_stat_activity WHERE usename='vec_bolsa_v3_login_activo_prueba'" \
    "no se observo la sesion ejecutora de prueba de mantenimiento"; then
    sed -n '1,120p' "$salida_bloqueador" >&2
    exit 1
fi
exigir_fallo_sqlstate_mantenimiento_v3 \
    "$contenedor" "$migracion_v3" 55000 \
    "la migracion V3 acepto trafico ejecutor activo"
if [[ $(psql_consulta "$contenedor" \
    "SELECT to_regclass('vec_bolsa_baremacion.manifiesto_probatorio_v3') IS NULL") != "t" ]]; then
    echo "el up V3 con trafico dejo cambios parciales" >&2
    exit 1
fi
psql_consulta "$contenedor" \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename='vec_bolsa_v3_login_activo_prueba'" \
    >/dev/null
wait "$pid_bloqueador" || true
pid_bloqueador=
psql_consulta "$contenedor" \
    'REVOKE vec_bolsa_baremacion_ejecutor FROM vec_bolsa_v3_login_activo_prueba; DROP ROLE vec_bolsa_v3_login_activo_prueba' \
    >/dev/null

psql_archivo_con_mantenimiento_v3 "$contenedor" "$migracion_v3"
crear_login_ejecutor "$contenedor"
psql_archivo "$contenedor" "$inventario_acl_v3"
psql_archivo "$contenedor" "$integracion_v3" \
    --set RUTA_MANIFIESTO_DORADO_V3="$ruta_dorada"
psql_archivo "$contenedor" "$integracion_v3" \
    --set RUTA_MANIFIESTO_DORADO_V3="$ruta_dorada" \
    --set CONFIRMAR_FIXTURE_V3=1

reiniciar_postgres "$contenedor"
psql_archivo "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/recuperacion_reinicio_v3.sql
psql_archivo "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/lectura_exacta_replay_v3.sql
psql_archivo "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/lectura_evidencia_replay_v3.sql
psql_archivo "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/integridad_escalabilidad_v3.sql
exigir_fallo_sqlstate_archivo "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/manifiesto_incompleto_v3.sql \
    23514 "el COMMIT acepto una cabecera V3 sin hijos"
if [[ $(psql_consulta "$contenedor" \
    "SELECT (SELECT count(*)=1 FROM vec_bolsa_baremacion.manifiesto_probatorio_v3 WHERE baremacion_merito_ref='baremacion:001' AND numero_version=2) AND (SELECT count(*)=1 FROM vec_bolsa_baremacion.manifiesto_autorizacion_v3 WHERE manifiesto_ref='manifiesto:bolsa:v3:001') AND (SELECT count(*)=1 FROM vec_bolsa_baremacion.manifiesto_evidencia_v3 WHERE manifiesto_ref='manifiesto:bolsa:v3:001')") != "t" ]]; then
    echo "las pruebas adversarias no revirtieron limpiamente" >&2
    exit 1
fi

# Esta prueba usa el adaptador Go y un manifiesto completo. La primera llamada
# mantiene la capacidad solo en memoria durante el fallo y reintento KMS; las
# dos llamadas posteriores, separadas por reinicios, recuperan el resultado ya
# confirmado sin persistir ni reconstruir esa capacidad temporal.
if ! rg -q 'func TestIntegracionBaremacionPostgreSQLV3Real' \
    internal/modules/bolsa/adapters/postgres; then
    echo "falta la prueba E2E Go/PostgreSQL V3 obligatoria" >&2
    exit 1
fi
chmod 600 "$estado_e2e"
ejecutar_e2e_go "$contenedor" prevalidar_fallo
reiniciar_postgres "$contenedor"
ejecutar_e2e_go "$contenedor" confirmar
reiniciar_postgres "$contenedor"
ejecutar_e2e_go "$contenedor" recuperar

psql_archivo "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/preparar_concurrencia_v3.sql

# Una tercera sesion posee dos compuertas exclusivamente de prueba. A y B fijan
# antes snapshots SERIALIZABLE y esperan en claves distintas; al terminar la
# tercera sesion ambas recorren la revalidacion productiva desde el mismo punto.
docker exec \
    --env PGAPPNAME='vec_bloqueador_carrera_v3' \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "SELECT pg_advisory_lock(hashtextextended('vec_prueba_bolsa_baremacion_v3:barrera_carrera:a',0)); SELECT pg_advisory_lock(hashtextextended('vec_prueba_bolsa_baremacion_v3:barrera_carrera:b',0)); SELECT pg_sleep(120)" \
    >"$salida_bloqueador" 2>&1 &
pid_bloqueador=$!
esperar_consulta_verdadera "$contenedor" \
    "SELECT count(*)=2 AND bool_and(bloqueo.granted) FROM pg_stat_activity AS actividad JOIN pg_locks AS bloqueo ON bloqueo.pid=actividad.pid AND bloqueo.locktype='advisory' WHERE actividad.application_name='vec_bloqueador_carrera_v3'" \
    "la tercera sesion no adquirio las dos compuertas advisory"
psql_archivo_serializable "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/concurrencia_reserva_v3.sql \
    vec_carrera_v3_a --set SUFIJO=a >"$salida_uno" 2>&1 &
pid_uno=$!
psql_archivo_serializable "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/concurrencia_reserva_v3.sql \
    vec_carrera_v3_b --set SUFIJO=b >"$salida_dos" 2>&1 &
pid_dos=$!
esperar_consulta_verdadera "$contenedor" \
    "SELECT count(*)=2 AND bool_and(wait_event_type='Lock' AND wait_event='advisory') FROM pg_stat_activity WHERE application_name IN ('vec_carrera_v3_a','vec_carrera_v3_b')" \
    "las dos reservas no alcanzaron juntas sus compuertas advisory"
psql_consulta "$contenedor" \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name='vec_bloqueador_carrera_v3'" \
    >/dev/null
wait "$pid_bloqueador" || true
pid_bloqueador=
estado_uno=0
estado_dos=0
wait "$pid_uno" || estado_uno=$?
wait "$pid_dos" || estado_dos=$?
pid_uno=
pid_dos=
if [[ $estado_uno -eq 0 && $estado_dos -eq 0 ]] \
    || [[ $estado_uno -ne 0 && $estado_dos -ne 0 ]]; then
    sed -n '1,120p' "$salida_uno" >&2
    sed -n '1,120p' "$salida_dos" >&2
    echo "la carrera SERIALIZABLE no produjo exactamente un ganador" >&2
    exit 1
fi
if [[ $estado_uno -ne 0 ]]; then
    perdedor=a
    salida_perdedor=$salida_uno
else
    perdedor=b
    salida_perdedor=$salida_dos
fi
if ! rg -q 'ERROR:[[:space:]]+40001:' "$salida_perdedor"; then
    sed -n '1,120p' "$salida_perdedor" >&2
    echo "el perdedor no fallo con SQLSTATE 40001" >&2
    exit 1
fi
psql_archivo_serializable "$contenedor" \
    deploy/postgresql/bolsa_baremacion/pruebas_sql/concurrencia_reserva_v3.sql \
    "vec_carrera_v3_reintento_${perdedor}" --set SUFIJO="$perdedor"
if [[ $(psql_consulta "$contenedor" \
    "SELECT count(*)=2 AND count(*) FILTER (WHERE resultado='reservada')=1 AND count(*) FILTER (WHERE resultado='en_curso')=1 FROM vec_prueba_bolsa_baremacion_v3.resultado_concurrencia") != "t" ]]; then
    echo "la carrera V3 no produjo un ganador y un en_curso" >&2
    exit 1
fi
if [[ $(psql_consulta "$contenedor" \
    "SELECT (SELECT count(*)=1 FROM vec_bolsa_baremacion.reserva_actual AS actual JOIN vec_bolsa_baremacion.reserva_version AS version ON version.ambito_idempotencia_sha256=actual.ambito_idempotencia_sha256 AND version.reserva_ref=actual.reserva_ref AND version.version=actual.version WHERE version.baremacion_merito_ref='baremacion:001' AND actual.estado='activa') AND (SELECT count(*)=1 FROM vec_bolsa_baremacion.uso_decision WHERE decision_ref LIKE 'decision:concurrencia:v3:%' AND tipo_efecto='reserva')") != "t" ]]; then
    echo "la carrera V3 no conservo una unica reserva y un unico consumo" >&2
    exit 1
fi

probar_login_real "$contenedor"
exigir_fallo_sqlstate_archivo "$contenedor" "$reversion_v3" 55000 \
    "el down V3 sin opt-in acepto historia"
exigir_fallo_sqlstate_reversion_v3 \
    "$contenedor" "$reversion_v3" 55000 \
    "el down V3 con opt-in borro historia"
if [[ $(psql_consulta "$contenedor" \
    "SELECT to_regclass('vec_bolsa_baremacion.manifiesto_probatorio_v3') IS NOT NULL AND to_regprocedure('vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea,bytea,jsonb,bytea,bytea,bytea,text)') IS NOT NULL AND has_function_privilege('vec_bolsa_baremacion_ejecutor','vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea,bytea,jsonb,bytea,bytea,bytea,text)','EXECUTE')") != "t" ]]; then
    echo "el down fallido dejo cambios parciales" >&2
    exit 1
fi

arrancar_postgres "$contenedor_vacio"
instalar_base "$contenedor_vacio"
psql_archivo "$contenedor_vacio" "$integracion_v1" --set CONFIRMAR_FIXTURE=1
psql_archivo "$contenedor_vacio" "$historia_no_reconstruible_v3"
exigir_fallo_sqlstate_mantenimiento_v3 \
    "$contenedor_vacio" "$migracion_v3" 55000 \
    "el up V3 acepto historia heredada no reconstruible"
if [[ $(psql_consulta "$contenedor_vacio" \
    "SELECT count(*)=1 AND to_regclass('vec_bolsa_baremacion.manifiesto_probatorio_v3') IS NULL FROM vec_bolsa_baremacion.version_baremacion WHERE numero=2") != "t" ]]; then
    echo "el rechazo de historia heredada dejo cambios parciales" >&2
    exit 1
fi
docker rm -f "$contenedor_vacio" >/dev/null

# Segunda instalacion realmente vacia: prueba el down conservador sin alterar
# la instancia principal, que mantiene toda su historia.
arrancar_postgres "$contenedor_vacio"
instalar_base "$contenedor_vacio"
exigir_fallo_sqlstate_archivo \
    "$contenedor_vacio" "$migracion_v3" 55000 \
    "el up V3 vacio funciono sin opt-in de mantenimiento"
psql_archivo_con_mantenimiento_v3 "$contenedor_vacio" "$migracion_v3"
exigir_fallo_sqlstate_archivo \
    "$contenedor_vacio" "$reversion_v3" 55000 \
    "el down V3 vacio funciono sin opt-in"
psql_archivo_con_reversion_v3 "$contenedor_vacio" "$reversion_v3"
if [[ $(psql_consulta "$contenedor_vacio" \
    "SELECT to_regnamespace('vec_bolsa_baremacion') IS NOT NULL AND to_regclass('vec_bolsa_baremacion.manifiesto_probatorio_v3') IS NULL AND to_regprocedure('vec_bolsa_baremacion.confirmar_cambio_con_archivo_probatorio_v3(jsonb,jsonb,bytea,bytea,bytea,jsonb,bytea,bytea,bytea,text)') IS NULL AND has_function_privilege('vec_bolsa_baremacion_ejecutor','vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)','EXECUTE')") != "t" ]]; then
    echo "el down V3 vacio no restauro el inventario/superficie V1" >&2
    exit 1
fi

echo "integracion PostgreSQL 18.4 del archivo probatorio V3: correcta"
