#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-calculo-experiencia-pg-${USER:-usuario}-$$"
base=vec_calculo_experiencia_prueba

generar_clave_prueba() {
    local destino=$1
    local valor

    if [[ ! -c /dev/urandom || ! -r /dev/urandom ]] \
        || ! command -v od >/dev/null 2>&1 \
        || ! command -v tr >/dev/null 2>&1; then
        echo "no hay una fuente local de entropia utilizable" >&2
        return 1
    fi
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "la fuente local de entropia devolvio una clave invalida" >&2
        return 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_aplicacion=
clave_lector=
clave_publicador=
clave_ajeno=
generar_clave_prueba clave_admin
generar_clave_prueba clave_aplicacion
generar_clave_prueba clave_lector
generar_clave_prueba clave_publicador
generar_clave_prueba clave_ajeno
if [[ "$clave_admin" == "$clave_aplicacion" \
    || "$clave_admin" == "$clave_lector" \
    || "$clave_admin" == "$clave_publicador" \
    || "$clave_admin" == "$clave_ajeno" \
    || "$clave_aplicacion" == "$clave_lector" \
    || "$clave_aplicacion" == "$clave_publicador" \
    || "$clave_aplicacion" == "$clave_ajeno" \
    || "$clave_lector" == "$clave_publicador" \
    || "$clave_lector" == "$clave_ajeno" \
    || "$clave_publicador" == "$clave_ajeno" ]]; then
    echo "la fuente local de entropia produjo claves repetidas" >&2
    exit 1
fi

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

consulta_runtime() {
    local usuario=$1
    local clave=$2
    local consulta=$3

    docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" \
        --command "$consulta" >/dev/null
}

exigir_rechazo_runtime() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    local descripcion=$4

    if consulta_runtime "$usuario" "$clave" "$consulta" 2>/dev/null; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

instalar_base_autorizacion() {
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
}

instalar_calculo() {
    psql_archivo deploy/postgresql/bolsa_calculo_experiencia/roles_up.sql
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/migraciones_autorizacion/000001_frontera_v2.up.sql
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/migraciones/000001_resultados_oficiales.up.sql
}

probar_sql() {
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/pruebas_sql/acl_y_modelo.sql
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/pruebas_sql/frontera_v2.sql
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/pruebas_sql/integracion_mecanica.sql
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/pruebas_sql/rectificaciones.sql
}

retirar_calculo_vacio() {
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/migraciones/000001_resultados_oficiales.down.sql
    psql_archivo \
        deploy/postgresql/bolsa_calculo_experiencia/migraciones_autorizacion/000001_frontera_v2.down.sql
    psql_archivo deploy/postgresql/bolsa_calculo_experiencia/roles_down.sql
}

command -v docker >/dev/null 2>&1 || {
    echo "docker es obligatorio para esta integracion" >&2
    exit 1
}

(cd "$raiz" && go test \
    ./internal/modules/bolsa/domain/calculoexperienciaoficial \
    ./internal/modules/bolsa/ports)

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

# El orden es deliberado: 000002 aporta el vinculo de autenticacion que
# consumen las migraciones 000003/000004 y la frontera estrecha del calculo.
instalar_base_autorizacion
instalar_calculo
probar_sql

# La retirada sin confirmación debe fallar cerrada incluso si la historia fue
# dañada fuera de las fronteras normales. Se siembra una fila sintética como
# superusuario con los disparadores suspendidos y después se limpia igual.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
SET session_replication_role = replica;
INSERT INTO vec_bolsa_calculo_experiencia.outbox(
    tenant_id, outbox_ref, resultado_ref, ruta, esquema_evento,
    evento_canonico, huella_evento_sha256, creada_en
) VALUES (
    'diputacion_granada', 'outbox:barrera-down', 'resultado:ausente',
    'bolsa.calculo_experiencia.resultado_oficial.v1',
    'vec.bolsa.calculo-experiencia.resultado-confirmado.v1',
    convert_to('xx', 'UTF8'),
    encode(sha256(convert_to('xx', 'UTF8')), 'hex'),
    '2026-07-17T00:00:00.000000Z'
);
RESET session_replication_role;
SQL
if psql_archivo \
    deploy/postgresql/bolsa_calculo_experiencia/migraciones/000001_resultados_oficiales.down.sql \
    >/dev/null 2>&1; then
    echo "el down destruyo historia sin confirmacion explicita" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $historia_conservada$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM vec_bolsa_calculo_experiencia.outbox
         WHERE outbox_ref = 'outbox:barrera-down'
    ) THEN
        RAISE EXCEPTION 'el down fallido no conservo la historia';
    END IF;
END
$historia_conservada$;
SET session_replication_role = replica;
DELETE FROM vec_bolsa_calculo_experiencia.outbox
 WHERE outbox_ref = 'outbox:barrera-down';
RESET session_replication_role;
SQL

# ACL ejercidas por conexiones LOGIN reales, no solo mediante SET ROLE desde
# una sesion superusuaria. Las claves aleatorias no se escriben ni imprimen.
docker exec --interactive \
    --env CLAVE_APLICACION="$clave_aplicacion" \
    --env CLAVE_LECTOR="$clave_lector" \
    --env CLAVE_PUBLICADOR="$clave_publicador" \
    --env CLAVE_AJENO="$clave_ajeno" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_aplicacion CLAVE_APLICACION
\getenv clave_lector CLAVE_LECTOR
\getenv clave_publicador CLAVE_PUBLICADOR
\getenv clave_ajeno CLAVE_AJENO
CREATE ROLE vec_calculo_aplicacion_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_aplicacion';
CREATE ROLE vec_calculo_lector_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_lector';
CREATE ROLE vec_calculo_publicador_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_publicador';
CREATE ROLE vec_calculo_ajeno_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ajeno';
GRANT vec_bolsa_calculo_experiencia_aplicacion
    TO vec_calculo_aplicacion_prueba;
GRANT vec_bolsa_calculo_experiencia_lector_operativo
    TO vec_calculo_lector_prueba;
GRANT vec_bolsa_calculo_experiencia_publicador
    TO vec_calculo_publicador_prueba;
SQL

consulta_runtime vec_calculo_aplicacion_prueba "$clave_aplicacion" \
    'SELECT count(*) FROM vec_bolsa_calculo_experiencia.resultado_oficial'
consulta_runtime vec_calculo_lector_prueba "$clave_lector" \
    'SELECT count(*) FROM vec_bolsa_calculo_experiencia.resultado_oficial'
consulta_runtime vec_calculo_publicador_prueba "$clave_publicador" \
    'SELECT count(*) FROM vec_bolsa_calculo_experiencia.outbox'

exigir_rechazo_runtime vec_calculo_aplicacion_prueba "$clave_aplicacion" \
    'TRUNCATE vec_bolsa_calculo_experiencia.resultado_oficial' \
    'la aplicacion pudo truncar historia'
exigir_rechazo_runtime vec_calculo_aplicacion_prueba "$clave_aplicacion" \
    "SELECT vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente('x'::bytea,'y'::bytea)" \
    'la aplicacion pudo usar el registrador V2 no atestado'
exigir_rechazo_runtime vec_calculo_aplicacion_prueba "$clave_aplicacion" \
    "INSERT INTO vec_bolsa_calculo_experiencia.configuracion_tenant VALUES (false,'otro','2026-07-17T00:00:00.000000Z')" \
    'la aplicacion pudo alterar el tenant fijo'
exigir_rechazo_runtime vec_calculo_lector_prueba "$clave_lector" \
    'SELECT selector_fuente_canonico FROM vec_bolsa_calculo_experiencia.resultado_oficial' \
    'el lector operativo pudo leer bytes canonicos'
exigir_rechazo_runtime vec_calculo_publicador_prueba "$clave_publicador" \
    'SELECT count(*) FROM vec_bolsa_calculo_experiencia.resultado_oficial' \
    'el publicador pudo leer resultados'
exigir_rechazo_runtime vec_calculo_ajeno_prueba "$clave_ajeno" \
    'SELECT count(*) FROM vec_bolsa_calculo_experiencia.outbox' \
    'una identidad sin grupo pudo usar el esquema'

# El down de roles es deliberadamente estricto con membresias externas; se
# retiran primero las identidades efimeras y luego se prueba down/up completo.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DROP ROLE vec_calculo_aplicacion_prueba;
DROP ROLE vec_calculo_lector_prueba;
DROP ROLE vec_calculo_publicador_prueba;
DROP ROLE vec_calculo_ajeno_prueba;
SQL

retirar_calculo_vacio
instalar_calculo
probar_sql

echo "integracion PostgreSQL del calculo oficial: OK"
