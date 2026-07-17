#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-autorizacion-atestada-v2-${USER:-usuario}-$$"
base=vec_autorizacion_atestada_v2_prueba

generar_clave() {
    local destino=$1
    local valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "no se pudo obtener entropia local" >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_consumidor=
clave_emisor=
clave_ajeno=
generar_clave clave_admin
generar_clave clave_consumidor
generar_clave clave_emisor
generar_clave clave_ajeno

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -f /tmp/vec_ad2_holder_"$$".out /tmp/vec_ad2_error_"$$".out \
        /tmp/vec_ad2_retro_"$$".out
}
trap limpiar EXIT INT TERM

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

consulta_login() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    docker exec --env PGPASSWORD="$clave" "$contenedor" psql -X --quiet \
        --tuples-only --no-align \
        --set ON_ERROR_STOP=1 --host 127.0.0.1 --username "$usuario" \
        --dbname "$base" --command "$consulta"
}

exigir_rechazo_login() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    local descripcion=$4
    if consulta_login "$usuario" "$clave" "$consulta" \
        >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

exigir_snapshot_obsoleto() {
    local preparacion=$1
    local escritor_sql=$2
    local lector_sql=$3
    local aplicacion=$4
    local escritor
    local activo=0
    local estado=0
    psql_archivo "$preparacion"
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "SET application_name='$aplicacion'" --file=- \
        < "$raiz/$escritor_sql" \
        > /tmp/vec_ad2_holder_"$$".out 2>&1 &
    escritor=$!
    for _ in $(seq 1 50); do
        numero=$(docker exec "$contenedor" psql -X -Atq \
            --username postgres --dbname "$base" --command \
            "SELECT count(*) FROM pg_stat_activity WHERE application_name='$aplicacion' AND state='active' AND wait_event='PgSleep'" || true)
        if [[ "$numero" == "1" ]]; then
            activo=1
            break
        fi
        sleep 0.1
    done
    if [[ "$activo" != 1 ]]; then
        echo "el escritor $aplicacion no alcanzo el punto concurrente" >&2
        cat /tmp/vec_ad2_holder_"$$".out >&2
        wait "$escritor" || true
        exit 1
    fi
    if psql_archivo "$lector_sql" \
        > /tmp/vec_ad2_error_"$$".out 2>&1; then
        estado=0
    else
        estado=$?
    fi
    wait "$escritor"
    if [[ "$estado" == 0 ]] || ! grep -Eq \
        '40001|could not serialize|no se pudo serializar' \
        /tmp/vec_ad2_error_"$$".out; then
        echo "el checkpoint $aplicacion no rechazo el snapshot obsoleto" >&2
        cat /tmp/vec_ad2_error_"$$".out >&2
        exit 1
    fi
}

instalar_dependencias() {
    psql_archivo deploy/postgresql/autorizacion/roles_up.sql
    psql_archivo \
        deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
    psql_archivo \
        deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
    psql_archivo deploy/postgresql/autorizacion/roles_v2_up.sql
    psql_archivo \
        deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
    psql_archivo \
        deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
    psql_archivo deploy/postgresql/confianza_atestacion_v2/roles_up.sql
    psql_archivo \
        deploy/postgresql/confianza_atestacion_v2/migraciones/000001_catalogo_confianza_v2.up.sql
    psql_archivo \
        deploy/postgresql/confianza_atestacion_v2/pruebas_sql/integracion_catalogo.sql
}

instalar_modulo() {
    psql_archivo deploy/postgresql/autorizacion_atestada_v2/roles_up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones_autorizacion/000001_vinculo_decision_atestada_v2.up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones_confianza/000001_cotejo_consumo_atestado_v2.up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones/000001_registro_consumo_atestado_v2.up.sql
}

retirar_modulo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "SET vec.confirmar_destruccion_autorizacion_atestada_v2='DESTRUIR_AUTORIZACION_ATESTADA_V2_IRREVERSIBLE'" \
        --file=- \
        < "$raiz/deploy/postgresql/autorizacion_atestada_v2/migraciones/000001_registro_consumo_atestado_v2.down.sql"
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones_confianza/000001_cotejo_consumo_atestado_v2.down.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones_autorizacion/000001_vinculo_decision_atestada_v2.down.sql
}

command -v docker >/dev/null 2>&1 || {
    echo "docker es obligatorio para esta integracion" >&2
    exit 1
}

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres \
        --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres \
    --dbname "$base" >/dev/null

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
SQL
instalar_dependencias

# El paquete no cambia ACL globales. Debe negarse antes de crear estado si el
# bootstrap de la base no ha endurecido pgcrypto.
if psql_archivo deploy/postgresql/autorizacion_atestada_v2/roles_up.sql \
    >/dev/null 2>&1; then
    echo "roles_up acepto hmac expuesto a PUBLIC" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $sin_estado$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_autorizacion_atestada_v2_%'
    ) OR to_regnamespace('vec_autorizacion_atestada_v2_guardia') IS NOT NULL THEN
        RAISE EXCEPTION 'el preflight de pgcrypto dejo estado';
    END IF;
END
$sin_estado$;
REVOKE ALL ON FUNCTION public.hmac(bytea, bytea, text) FROM PUBLIC;
SQL

instalar_modulo
psql_archivo \
    deploy/postgresql/confianza_atestacion_v2/pruebas_sql/acl_y_contrato.sql
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/acl_y_contrato.sql
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/sembrar_clave_prueba.sql

docker exec --interactive \
    --env CLAVE_CONSUMIDOR="$clave_consumidor" \
    --env CLAVE_EMISOR="$clave_emisor" --env CLAVE_AJENO="$clave_ajeno" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_consumidor CLAVE_CONSUMIDOR
\getenv clave_emisor CLAVE_EMISOR
\getenv clave_ajeno CLAVE_AJENO
CREATE ROLE vec_ad2_consumidor_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_consumidor';
CREATE ROLE vec_ad2_emisor_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_emisor';
CREATE ROLE vec_ad2_ajeno_prueba LOGIN INHERIT NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ajeno';
GRANT vec_autorizacion_atestada_v2_consumidor
    TO vec_ad2_consumidor_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
GRANT vec_autorizacion_atestada_v2_emisor_capacidad
    TO vec_ad2_emisor_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
DO $conectar_ajeno$
BEGIN
    EXECUTE format(
        'GRANT CONNECT ON DATABASE %I TO vec_ad2_ajeno_prueba',
        current_database()
    );
END
$conectar_ajeno$;
SQL

exigir_snapshot_obsoleto \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/preparar_revocacion_snapshot_clave.sql \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/revocar_clave_snapshot_concurrente.sql \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/leer_clave_snapshot_obsoleto.sql \
    vec_ad2_snapshot_clave
exigir_snapshot_obsoleto \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/preparar_revocacion_snapshot_confianza.sql \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/revocar_raiz_snapshot_concurrente.sql \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/leer_confianza_snapshot_obsoleto.sql \
    vec_ad2_snapshot_confianza
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/puntero_clave_futuro.sql

# Una activación cuyo reloj se fijó antes de esperar conserva dos ejes:
# efectiva_en y conocida/registrada_en sellada dentro del advisory lock.
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/preparar_clave_activacion_concurrente.sql
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SET application_name='vec_ad2_material_clave'" --file=- \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/mantener_material_clave.sql" \
    > /tmp/vec_ad2_holder_"$$".out 2>&1 &
holder=$!
activo=0
for _ in $(seq 1 50); do
    numero=$(docker exec "$contenedor" psql -X -Atq --username postgres \
        --dbname "$base" --command \
        "SELECT count(*) FROM pg_stat_activity WHERE application_name='vec_ad2_material_clave' AND state='active' AND wait_event='PgSleep'" || true)
    if [[ "$numero" == "1" ]]; then
        activo=1
        break
    fi
    sleep 0.1
done
if [[ "$activo" != 1 ]]; then
    echo "el lector de clave no retuvo su lock" >&2
    cat /tmp/vec_ad2_holder_"$$".out >&2
    wait "$holder" || true
    exit 1
fi
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/activar_clave_tras_espera.sql
wait "$holder"

material=$(consulta_login vec_ad2_emisor_prueba "$clave_emisor" \
    'SELECT count(*) FROM vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()' \
    | tr -d '[:space:]')
[[ "$material" == "1" ]] || {
    echo "el emisor aislado no obtuvo una clave exacta" >&2
    exit 1
}
consulta_login vec_ad2_consumidor_prueba "$clave_consumidor" \
    "SELECT count(*) FROM vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2('decision:ausente',repeat('1',64),'efecto:ausente',repeat('2',64),repeat('3',64))" \
    >/dev/null
exigir_rechazo_login vec_ad2_consumidor_prueba "$clave_consumidor" \
    'SELECT count(*) FROM vec_autorizacion_atestada_v2.atestacion_decision_v2' \
    'el consumidor pudo leer tablas'
exigir_rechazo_login vec_ad2_consumidor_prueba "$clave_consumidor" \
    "SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente('x'::bytea,'y'::bytea)" \
    'el consumidor alcanzo el registrador nominal'
exigir_rechazo_login vec_ad2_consumidor_prueba "$clave_consumidor" \
    "SELECT vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1('x',repeat('0',64),now(),now(), 'x',1,repeat('0',64),now(),now(),'x','x',now())" \
    'el consumidor alcanzo el cotejo de confianza'
exigir_rechazo_login vec_ad2_emisor_prueba "$clave_emisor" \
    "SELECT * FROM vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada('x'::bytea,'x'::bytea,'x'::bytea,repeat('x',16)::bytea,'x'::bytea,repeat('x',44)::bytea,'{}'::jsonb)" \
    'el emisor alcanzo la puerta de consumo'
exigir_rechazo_login vec_ad2_ajeno_prueba "$clave_ajeno" \
    'SELECT * FROM vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()' \
    'un rol ajeno obtuvo material HMAC'

# La cuenta emisora deja de ser valida si acumula otra membresia directa.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_ad2_grupo_extra NOLOGIN;
GRANT vec_ad2_grupo_extra TO vec_ad2_emisor_prueba;
SQL
exigir_rechazo_login vec_ad2_emisor_prueba "$clave_emisor" \
    'SELECT * FROM vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()' \
    'un emisor con membresia adicional obtuvo el secreto'
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_ad2_grupo_extra FROM vec_ad2_emisor_prueba;
DROP ROLE vec_ad2_grupo_extra;
SQL

# Una cadena transitiva no sustituye la única membresía directa exacta.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_ad2_puente_prueba NOLOGIN INHERIT;
GRANT vec_autorizacion_atestada_v2_emisor_capacidad,
      vec_autorizacion_atestada_v2_consumidor
    TO vec_ad2_puente_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
GRANT vec_ad2_puente_prueba TO vec_ad2_ajeno_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
SQL
exigir_rechazo_login vec_ad2_ajeno_prueba "$clave_ajeno" \
    'SELECT * FROM vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()' \
    'un puente transitivo alcanzo el secreto HMAC'
exigir_rechazo_login vec_ad2_ajeno_prueba "$clave_ajeno" \
    "SELECT * FROM vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada('x'::bytea,'x'::bytea,'x'::bytea,repeat('x',16)::bytea,'x'::bytea,repeat('x',44)::bytea,'{}'::jsonb)" \
    'un puente transitivo alcanzo la puerta consumidora'
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_ad2_puente_prueba FROM vec_ad2_ajeno_prueba;
REVOKE vec_autorizacion_atestada_v2_emisor_capacidad,
       vec_autorizacion_atestada_v2_consumidor FROM vec_ad2_puente_prueba;
DROP ROLE vec_ad2_puente_prueba;
SQL

psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/integracion_consumo.sql

# Una revocación que se hace efectiva mientras la puerta espera la cabeza
# global de auditoría debe fallar después del registro nominal y revertirlo.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SET application_name='vec_ad2_bloqueo_auditoria'" --file=- \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/bloquear_cadena_auditoria.sql" \
    > /tmp/vec_ad2_holder_"$$".out 2>&1 &
holder=$!
activo=0
for _ in $(seq 1 50); do
    numero=$(docker exec "$contenedor" psql -X -Atq --username postgres \
        --dbname "$base" --command \
        "SELECT count(*) FROM pg_stat_activity WHERE application_name='vec_ad2_bloqueo_auditoria' AND state='active' AND wait_event='PgSleep'" || true)
    if [[ "$numero" == "1" ]]; then
        activo=1
        break
    fi
    sleep 0.1
done
if [[ "$activo" != 1 ]]; then
    echo "no se bloqueo la cabeza de auditoria" >&2
    cat /tmp/vec_ad2_holder_"$$".out >&2
    wait "$holder" || true
    exit 1
fi
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/revocar_clave_actual_en_futuro.sql
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --set VEC_AD2_REVOCACION_FUTURA=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/integracion_consumo.sql"
wait "$holder"

# El cotejo vivo conserva el advisory lock compartido hasta fin de transaccion.
psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/preparar_revocaciones_concurrentes.sql
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SET application_name='vec_ad2_holder'" --file=- \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/mantener_cotejo_confianza.sql" \
    > /tmp/vec_ad2_holder_"$$".out 2>&1 &
holder=$!
activo=0
for _ in $(seq 1 50); do
    numero=$(docker exec "$contenedor" psql -X -Atq --username postgres \
        --dbname "$base" --command \
        "SELECT count(*) FROM pg_stat_activity WHERE application_name='vec_ad2_holder' AND state='active' AND wait_event='PgSleep'" || true)
    if [[ "$numero" == "1" ]]; then
        activo=1
        break
    fi
    sleep 0.1
done
if [[ "$activo" != 1 ]]; then
    echo "el cotejo no alcanzo el punto de concurrencia" >&2
    cat /tmp/vec_ad2_holder_"$$".out >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SET application_name='vec_ad2_revocacion_retroactiva'" \
    --file=- \
    < "$raiz/deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/revocar_raiz_con_reloj_previo_a_espera.sql" \
    > /tmp/vec_ad2_retro_"$$".out 2>&1 &
retro=$!
esperando=0
for _ in $(seq 1 50); do
    numero=$(docker exec "$contenedor" psql -X -Atq --username postgres \
        --dbname "$base" --command \
        "SELECT count(*) FROM pg_stat_activity WHERE application_name='vec_ad2_revocacion_retroactiva' AND state='active' AND wait_event_type='Lock'" || true)
    if [[ "$numero" == "1" ]]; then
        esperando=1
        break
    fi
    sleep 0.1
done
if [[ "$esperando" != 1 ]]; then
    echo "la revocacion retroactiva no espero el lock" >&2
    cat /tmp/vec_ad2_retro_"$$".out >&2
    wait "$retro" || true
    exit 1
fi
for prueba in \
    revocar_raiz_bajo_cotejo.sql \
    revocar_configuracion_bajo_cotejo.sql \
    rotar_configuracion_bajo_cotejo.sql; do
    if psql_archivo \
        "deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/$prueba" \
        > /tmp/vec_ad2_error_"$$".out 2>&1; then
        echo "$prueba no quedo serializada" >&2
        exit 1
    fi
    if ! grep -Eq 'lock timeout|tiempo de espera' \
        /tmp/vec_ad2_error_"$$".out; then
        cat /tmp/vec_ad2_error_"$$".out >&2
        exit 1
    fi
done
wait "$holder"
if wait "$retro"; then
    echo "se acepto una revocacion fechada antes de esperar" >&2
    exit 1
fi
if ! grep -Eq '55000|revocacion de confianza retroactiva' \
    /tmp/vec_ad2_retro_"$$".out; then
    cat /tmp/vec_ad2_retro_"$$".out >&2
    exit 1
fi
for prueba in \
    revocar_raiz_bajo_cotejo.sql \
    revocar_configuracion_bajo_cotejo.sql \
    rotar_configuracion_bajo_cotejo.sql; do
    psql_archivo \
        "deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/$prueba"
done

# Existe historia de clave. El down sin token debe abortar y conservarla.
if psql_archivo \
    deploy/postgresql/autorizacion_atestada_v2/migraciones/000001_registro_consumo_atestado_v2.down.sql \
    >/dev/null 2>&1; then
    echo "el down destruyo historia sin confirmacion" >&2
    exit 1
fi
docker exec "$contenedor" psql -X -Atq --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "SELECT count(*) > 0 FROM vec_autorizacion_atestada_v2.clave_capacidad_version" \
    | grep -qx 't'

retirar_modulo
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $retirar_conexion_ajena$
BEGIN
    EXECUTE format(
        'REVOKE CONNECT ON DATABASE %I FROM vec_ad2_ajeno_prueba',
        current_database()
    );
END
$retirar_conexion_ajena$;
DROP ROLE vec_ad2_consumidor_prueba;
DROP ROLE vec_ad2_emisor_prueba;
DROP ROLE vec_ad2_ajeno_prueba;
SQL
psql_archivo deploy/postgresql/autorizacion_atestada_v2/roles_down.sql

# Reinstalación limpia y segunda retirada prueban que no quedan ACL/default ACL.
instalar_modulo
retirar_modulo
psql_archivo deploy/postgresql/autorizacion_atestada_v2/roles_down.sql

echo "integracion autorizacion atestada VEC-AD-2: OK"
