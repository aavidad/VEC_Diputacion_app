#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-reglas-baremo-pg-${USER:-usuario}-$$"
base=vec_reglas_baremo_prueba

generar_clave() {
    local destino=$1 valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo 'no se pudo generar una clave efimera de prueba' >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_gobierno=
clave_consulta=
clave_publicador=
clave_ajeno=
clave_consumidor_atestado=
generar_clave clave_admin
generar_clave clave_gobierno
generar_clave clave_consulta
generar_clave clave_publicador
generar_clave clave_ajeno
generar_clave clave_consumidor_atestado

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

psql_archivo_destruccion_confirmada() {
    docker exec --interactive \
        --env PGOPTIONS='-c vec.confirmar_destruccion_bolsa_reglas_baremo=DESTRUIR_HISTORIA_BOLSA_REGLAS_BAREMO_IRREVERSIBLE' \
        "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" < "$raiz/$1"
}

consulta_runtime() {
    local usuario=$1 clave=$2 consulta=$3
    docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" \
        --command "$consulta" >/dev/null
}

exigir_rechazo_runtime() {
    local usuario=$1 clave=$2 consulta=$3 descripcion=$4
    if consulta_runtime "$usuario" "$clave" "$consulta" 2>/dev/null; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

instalar_autorizacion_base() {
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
    psql_archivo deploy/postgresql/autorizacion_atestada_v2/roles_up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones_autorizacion/000001_vinculo_decision_atestada_v2.up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones_confianza/000001_cotejo_consumo_atestado_v2.up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/migraciones/000001_registro_consumo_atestado_v2.up.sql
    psql_archivo \
        deploy/postgresql/autorizacion_atestada_v2/pruebas_sql/sembrar_clave_prueba.sql
}

instalar_reglas() {
    psql_archivo deploy/postgresql/bolsa_reglas_baremo/roles_up.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones_autorizacion/000001_revalidacion_reglas_baremo.up.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones/000001_almacen_reglas_baremo.up.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones/000002_operaciones_reglas_baremo.up.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones_atestacion/000001_composicion_vec_ad2_reglas_baremo.up.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones/000003_gobierno_atestado_v2.up.sql
}

retirar_reglas_vacias() {
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones/000003_gobierno_atestado_v2.down.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones_atestacion/000001_composicion_vec_ad2_reglas_baremo.down.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones/000002_operaciones_reglas_baremo.down.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones/000001_almacen_reglas_baremo.down.sql
    psql_archivo \
        deploy/postgresql/bolsa_reglas_baremo/migraciones_autorizacion/000001_revalidacion_reglas_baremo.down.sql
    psql_archivo deploy/postgresql/bolsa_reglas_baremo/roles_down.sql
}

command -v docker >/dev/null 2>&1 || {
    echo 'docker es obligatorio para esta integracion' >&2
    exit 1
}

(cd "$raiz" && go test ./internal/modules/bolsa/domain/reglasbaremo)

docker run --detach --rm --name "$contenedor" \
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

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE EXTENSION pgcrypto WITH SCHEMA public;
REVOKE ALL ON FUNCTION public.hmac(bytea, bytea, text) FROM PUBLIC;
SQL

instalar_autorizacion_base
instalar_reglas
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/acl_y_modelo.sql
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/acl_atestado_v2.sql
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/integracion_mecanica.sql

# Prueba ACL con identidades LOGIN reales y claves aleatorias no impresas.
docker exec --interactive \
    --env CLAVE_GOBIERNO="$clave_gobierno" \
    --env CLAVE_CONSULTA="$clave_consulta" \
    --env CLAVE_PUBLICADOR="$clave_publicador" \
    --env CLAVE_AJENO="$clave_ajeno" \
    --env CLAVE_CONSUMIDOR_ATESTADO="$clave_consumidor_atestado" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_gobierno CLAVE_GOBIERNO
\getenv clave_consulta CLAVE_CONSULTA
\getenv clave_publicador CLAVE_PUBLICADOR
\getenv clave_ajeno CLAVE_AJENO
\getenv clave_consumidor_atestado CLAVE_CONSUMIDOR_ATESTADO
CREATE ROLE vec_reglas_gobierno_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_gobierno';
CREATE ROLE vec_reglas_consulta_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_consulta';
CREATE ROLE vec_reglas_publicador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_publicador';
CREATE ROLE vec_reglas_ajeno_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ajeno';
CREATE ROLE vec_reglas_consumidor_atestado_prueba LOGIN INHERIT NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS
    PASSWORD :'clave_consumidor_atestado';
GRANT vec_bolsa_reglas_baremo_ejecutor_gobierno
    TO vec_reglas_gobierno_prueba;
GRANT vec_bolsa_reglas_baremo_ejecutor_consulta
    TO vec_reglas_consulta_prueba;
GRANT vec_bolsa_reglas_baremo_publicador_outbox
    TO vec_reglas_publicador_prueba;
GRANT vec_autorizacion_atestada_v2_consumidor
    TO vec_reglas_consumidor_atestado_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
-- Apertura exclusivamente de prueba y directamente al LOGIN. Nunca se
-- concede el wrapper al grupo consumidor generico ni un segundo grupo al
-- LOGIN, pues identidad_runtime_valida exige una unica membresia directa.
GRANT USAGE ON SCHEMA vec_bolsa_reglas_baremo
    TO vec_reglas_consumidor_atestado_prueba;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) TO vec_reglas_consumidor_atestado_prueba;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
        text, text, text, text, text, text
    ) TO vec_reglas_consumidor_atestado_prueba;
DO $membresia_exacta$
DECLARE
    login_oid oid := 'vec_reglas_consumidor_atestado_prueba'::regrole;
    consumidor_oid oid := 'vec_autorizacion_atestada_v2_consumidor'::regrole;
BEGIN
    IF (SELECT count(*) FROM pg_catalog.pg_auth_members
         WHERE member = login_oid) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = login_oid AND roleid = consumidor_oid
              AND NOT admin_option AND inherit_option AND set_option
       ) THEN
        RAISE EXCEPTION 'LOGIN modular no tiene membresia VEC exclusiva';
    END IF;
END
$membresia_exacta$;
SQL

exigir_rechazo_runtime vec_reglas_gobierno_prueba "$clave_gobierno" \
    "SELECT * FROM vec_bolsa_reglas_baremo.confirmar_cambio_v1('{}','{}',''::bytea,''::bytea,''::bytea)" \
    'el ejecutor de gobierno uso el almacen sin composicion VEC-AD-2'
exigir_rechazo_runtime vec_reglas_consulta_prueba "$clave_consulta" \
    "SELECT * FROM vec_bolsa_reglas_baremo.obtener_version_exacta_v1('{}','{}',''::bytea,''::bytea)" \
    'el ejecutor de consulta uso el almacen sin composicion VEC-AD-2'
exigir_rechazo_runtime vec_reglas_gobierno_prueba "$clave_gobierno" \
    'INSERT INTO vec_bolsa_reglas_baremo.contenido_reglas_baremo DEFAULT VALUES' \
    'un rol runtime obtuvo DML'
exigir_rechazo_runtime vec_reglas_publicador_prueba "$clave_publicador" \
    'SELECT * FROM vec_bolsa_reglas_baremo.outbox' \
    'el publicador reservado leyo la outbox directamente'
exigir_rechazo_runtime vec_reglas_ajeno_prueba "$clave_ajeno" \
    'SELECT * FROM vec_bolsa_reglas_baremo.version_reglas_baremo' \
    'una identidad ajena uso el almacen'
exigir_rechazo_runtime vec_reglas_consumidor_atestado_prueba \
    "$clave_consumidor_atestado" \
    "SELECT * FROM vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada('x'::bytea,'x'::bytea,'x'::bytea,repeat('x',16)::bytea,'x'::bytea,repeat('x',44)::bytea,'{}'::jsonb)" \
    'el LOGIN modular alcanzo la puerta VEC-AD-2 cruda'

psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/integracion_atestada_v2.sql

# El down ordinario jamas puede retirar la capa V2 si existe un recibo. Se
# conserva la transaccion mecanica anterior y se concatena el down en la misma
# sesion para comprobar historia real sin dejar fixtures persistentes.
log_down_v2="/tmp/vec-reglas-down-v2-$$.log"
set +e
{
    sed -n '1,$p' \
        "$raiz/deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/integracion_atestada_v2.sql"
    sed -n '1,$p' \
        "$raiz/deploy/postgresql/bolsa_reglas_baremo/migraciones/000003_gobierno_atestado_v2.down.sql"
} | docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 \
    --set vec_reglas_probar_down_con_historia=1 \
    --username postgres --dbname "$base" \
    >"$log_down_v2" 2>&1
estado_down_v2=$?
set -e
if [[ $estado_down_v2 -eq 0 ]] || ! rg --quiet \
    'retirada V2 rechazada: conservar recibos atestados' "$log_down_v2"; then
    sed -n '1,200p' "$log_down_v2" >&2
    rm -f "$log_down_v2"
    echo 'el down V2 no protegió su historia atestada' >&2
    exit 1
fi
rm -f "$log_down_v2"

psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/preparar_concurrencia.sql
set +e
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/concurrencia_a.sql \
    >/tmp/vec-reglas-concurrencia-a-$$.log 2>&1 &
pid_a=$!
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/concurrencia_b.sql \
    >/tmp/vec-reglas-concurrencia-b-$$.log 2>&1 &
pid_b=$!
wait "$pid_a"
estado_a=$?
wait "$pid_b"
estado_b=$?
set -e
rm -f /tmp/vec-reglas-concurrencia-a-$$.log \
    /tmp/vec-reglas-concurrencia-b-$$.log
if [[ $estado_a -eq 0 && $estado_b -eq 0 ]] \
    || [[ $estado_a -ne 0 && $estado_b -ne 0 ]]; then
    echo 'la carrera CAS no produjo exactamente un ganador' >&2
    exit 1
fi
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/verificar_concurrencia.sql

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DROP ROLE vec_reglas_gobierno_prueba;
DROP ROLE vec_reglas_consulta_prueba;
DROP ROLE vec_reglas_publicador_prueba;
DROP ROLE vec_reglas_ajeno_prueba;
REVOKE USAGE ON SCHEMA vec_bolsa_reglas_baremo
    FROM vec_reglas_consumidor_atestado_prueba;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_reglas_baremo.confirmar_cambio_atestado_v2(
        bytea, bytea, bytea, bytea, bytea, bytea, jsonb
    ) FROM vec_reglas_consumidor_atestado_prueba;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_reglas_baremo.reconciliar_cambio_atestado_v2(
        text, text, text, text, text, text
    ) FROM vec_reglas_consumidor_atestado_prueba;
REVOKE vec_autorizacion_atestada_v2_consumidor
    FROM vec_reglas_consumidor_atestado_prueba;
DROP ROLE vec_reglas_consumidor_atestado_prueba;
SQL

psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/migraciones/000003_gobierno_atestado_v2.down.sql
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/migraciones_atestacion/000001_composicion_vec_ad2_reglas_baremo.down.sql
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/migraciones/000002_operaciones_reglas_baremo.down.sql
if psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/migraciones/000001_almacen_reglas_baremo.down.sql \
    >/dev/null 2>&1; then
    echo 'el down destruyo historia sin confirmacion explicita' >&2
    exit 1
fi
docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    'SELECT 1 FROM vec_bolsa_reglas_baremo.version_reglas_baremo LIMIT 1' \
    >/dev/null
psql_archivo_destruccion_confirmada \
    deploy/postgresql/bolsa_reglas_baremo/migraciones/000001_almacen_reglas_baremo.down.sql
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/migraciones_autorizacion/000001_revalidacion_reglas_baremo.down.sql
psql_archivo deploy/postgresql/bolsa_reglas_baremo/roles_down.sql

# Segunda instalacion limpia: prueba retirada y reinstalacion reproducibles.
instalar_reglas
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/acl_y_modelo.sql
psql_archivo \
    deploy/postgresql/bolsa_reglas_baremo/pruebas_sql/acl_atestado_v2.sql
retirar_reglas_vacias

echo 'integracion PostgreSQL 18 de reglas gobernadas de baremo: OK'
