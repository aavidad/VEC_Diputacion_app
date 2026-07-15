#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-ejecucion-v4-pg-${USER:-usuario}-$$"
base=vec_ejecucion_v4_prueba
clave_admin="admin-v4-$$"
clave_fuente="fuente-v4-$$"
clave_registro="registro-v4-$$"
clave_emisor="emisor-v4-$$"
clave_ejecucion="ejecucion-v4-$$"

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker run --detach --rm --name "$contenedor" --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres --dbname "$base" >/dev/null

docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
SQL

docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/autorizacion/roles_up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql"
# Prueba el down completo antes de que existan decisiones V2 inmutables y
# reaplica la evolucion para la integracion. Con datos V2 el down falla cerrado.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.down.sql"
# La fila V1 se crea con el esquema historico; el segundo up debe conservarla.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/pruebas_sql/sembrar_decision_legacy_v1.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql"
# Incluso una instalacion sin confianza ni evidencia exige opt-in porque la
# retirada del esquema es irreversible. El fallo debe conservar todo el esquema.
if docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql" \
    >/dev/null 2>&1; then
    echo "el down V4 vacio acepto retirar el esquema sin opt-in" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $conservada_vacia$
BEGIN
    IF to_regnamespace('vec_ejecucion_documental_v4') IS NULL THEN
        RAISE EXCEPTION 'el down vacio fallido no conservo el esquema';
    END IF;
END
$conservada_vacia$;

-- Regresion ante evoluciones futuras: una relacion que la V4 original no
-- conocia tampoco puede eludir la barrera destructiva.
SET ROLE vec_ejecucion_documental_v4_propietario;
CREATE TABLE vec_ejecucion_documental_v4.evidencia_futura_prueba (
    evidencia_id bigint PRIMARY KEY
);
INSERT INTO vec_ejecucion_documental_v4.evidencia_futura_prueba
    (evidencia_id) VALUES (1);
RESET ROLE;
DO $acl_tipo_futuro$
BEGIN
    IF has_type_privilege(
        'vec_ejecucion_documental_v4_emisor_capacidad',
        'vec_ejecucion_documental_v4.evidencia_futura_prueba',
        'USAGE'
    ) OR has_type_privilege(
        'vec_ejecucion_documental_v4_ejecutor_atestado',
        'vec_ejecucion_documental_v4.evidencia_futura_prueba',
        'USAGE'
    ) THEN
        RAISE EXCEPTION 'la guarda DDL no cerro el tipo fila futuro';
    END IF;
END
$acl_tipo_futuro$;
SQL
if docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql" \
    >/dev/null 2>&1; then
    echo "una relacion V4 futura eludio el opt-in destructivo" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $conservada_futura$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.evidencia_futura_prueba
         WHERE evidencia_id = 1
    ) THEN
        RAISE EXCEPTION 'el down fallido no conservo la evidencia futura';
    END IF;
END
$conservada_futura$;
SQL

# Una dependencia externa no queda incluida por el opt-in V4. El desmontaje usa
# RESTRICT y debe abortar antes de expandir su radio destructivo a otro modulo.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
CREATE SCHEMA vec_v4_dependencia_externa_prueba;
CREATE VIEW vec_v4_dependencia_externa_prueba.vista_evidencia AS
    SELECT evidencia_id
      FROM vec_ejecucion_documental_v4.evidencia_futura_prueba;
SQL
if docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql" \
    >/dev/null 2>&1; then
    echo "el down V4 destruyo una dependencia externa con el opt-in local" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $dependencia_conservada$
BEGIN
    IF to_regclass(
        'vec_v4_dependencia_externa_prueba.vista_evidencia'
    ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.evidencia_futura_prueba
         WHERE evidencia_id = 1
    ) THEN
        RAISE EXCEPTION 'el down fallido no conservo la dependencia externa';
    END IF;
END
$dependencia_conservada$;
DROP VIEW vec_v4_dependencia_externa_prueba.vista_evidencia;
DROP SCHEMA vec_v4_dependencia_externa_prueba;
SQL
docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql"
# Se reaplica para continuar con las pruebas de ACL, ejecucion y borrado
# protegido sobre evidencia real.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/pruebas_sql/revalidacion_identidad_v1.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/pruebas_sql/privilegios_minimos_v2.sql"

docker exec --interactive \
    --env CLAVE_FUENTE="$clave_fuente" --env CLAVE_REGISTRO="$clave_registro" \
    --env CLAVE_EMISOR="$clave_emisor" \
    --env CLAVE_EJECUCION="$clave_ejecucion" \
    "$contenedor" psql --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
\getenv clave_fuente CLAVE_FUENTE
\getenv clave_registro CLAVE_REGISTRO
\getenv clave_emisor CLAVE_EMISOR
\getenv clave_ejecucion CLAVE_EJECUCION
CREATE ROLE vec_v4_fuente_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_fuente';
CREATE ROLE vec_v4_registro_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registro';
CREATE ROLE vec_v4_emisor_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_emisor';
CREATE ROLE vec_v4_ejecucion_prueba LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecucion';
GRANT vec_autorizacion_fuente TO vec_v4_fuente_prueba;
GRANT vec_autorizacion_registro TO vec_v4_registro_prueba;
GRANT vec_ejecucion_documental_v4_emisor_capacidad TO vec_v4_emisor_prueba;
GRANT vec_ejecucion_documental_v4_ejecutor_atestado TO vec_v4_ejecucion_prueba;
SQL

rechazar_consulta_runtime() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    local descripcion=$4

    if docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql --no-psqlrc --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" \
        --command "$consulta" >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

comprobar_tipo_sin_uso() {
    local usuario=$1
    local clave=$2
    local etiqueta=$3
    local privilegio

    privilegio=$(docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql --no-psqlrc --tuples-only --no-align --set ON_ERROR_STOP=1 \
        --host 127.0.0.1 --username "$usuario" --dbname "$base" \
        --command "SELECT has_type_privilege(current_user, 'vec_ejecucion_documental_v4.atestacion_pdp', 'USAGE')")
    if [[ "$privilegio" != "f" ]]; then
        echo "ACL invalida: $etiqueta conserva USAGE sobre un tipo V4" >&2
        exit 1
    fi
}

# Se prueba con las identidades LOGIN reales, ademas de inspeccionar catalogos:
# ningun runtime puede invocar pgcrypto ni resolver tipos internos de tablas.
for identidad in \
    "vec_v4_emisor_prueba:$clave_emisor:emisor" \
    "vec_v4_ejecucion_prueba:$clave_ejecucion:ejecutor"
do
    IFS=: read -r usuario clave etiqueta <<<"$identidad"
    rechazar_consulta_runtime "$usuario" "$clave" \
        "SELECT public.hmac(decode('00', 'hex'), decode('00', 'hex'), 'sha256')" \
        "$etiqueta ejecuto HMAC directamente"
    rechazar_consulta_runtime "$usuario" "$clave" \
        "SELECT public.digest(decode('00', 'hex'), 'sha256')" \
        "$etiqueta ejecuto otra funcion pgcrypto"
    comprobar_tipo_sin_uso "$usuario" "$clave" "$etiqueta"
done

# El propietario de las funciones SECURITY DEFINER solo puede ejecutar el
# overload HMAC bytea preciso. Se ejerce la llamada, no solo la ACL declarada.
docker exec --interactive "$contenedor" \
    psql --no-psqlrc --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
BEGIN;
SET LOCAL ROLE vec_ejecucion_documental_v4_propietario;
SELECT octet_length(public.hmac(
    decode('00', 'hex'), decode('00', 'hex'), 'sha256'
));
ROLLBACK;
SQL
if docker exec "$contenedor" \
    psql --no-psqlrc --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command "SET ROLE vec_ejecucion_documental_v4_propietario; SELECT public.hmac('dato'::text, 'clave'::text, 'sha256')" \
    >/dev/null 2>&1; then
    echo "el propietario ejecuto el overload HMAC text no autorizado" >&2
    exit 1
fi
if docker exec "$contenedor" \
    psql --no-psqlrc --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    --command "SET ROLE vec_ejecucion_documental_v4_propietario; SELECT public.digest(decode('00', 'hex'), 'sha256')" \
    >/dev/null 2>&1; then
    echo "el propietario ejecuto una funcion pgcrypto distinta de HMAC bytea" >&2
    exit 1
fi

puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
    echo "no se pudo resolver el puerto efimero" >&2
    exit 1
fi
export VEC_POSTGRES_TEST_FUENTE_DSN="postgresql://vec_v4_fuente_prueba:${clave_fuente}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_REGISTRO_DSN="postgresql://vec_v4_registro_prueba:${clave_registro}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_ADMIN_DSN="postgresql://postgres:${clave_admin}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_V4_EMISOR_DSN="postgresql://vec_v4_emisor_prueba:${clave_emisor}@127.0.0.1:${puerto}/${base}?sslmode=disable"
export VEC_POSTGRES_TEST_V4_EJECUCION_DSN="postgresql://vec_v4_ejecucion_prueba:${clave_ejecucion}@127.0.0.1:${puerto}/${base}?sslmode=disable"

if [[ "${VEC_POSTGRES_PRUEBA_SOLO_SQL:-0}" != "1" ]]; then
    (cd "$raiz" && go test ./internal/vec/adapters/postgres/confianzadocumental \
        -run '^TestIntegracionEjecucionDocumentalV4PostgreSQLReal$' -count=1)
fi

# En modo solo SQL no existe una orden real. Se marca en el contenedor efimero
# un estado de auditoria no vacio para ejercer igualmente la guarda destructiva.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $marca$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.atestacion_pdp
    ) AND NOT EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.orden_generacion_documental
    ) AND NOT EXISTS (
        SELECT 1 FROM vec_ejecucion_documental_v4.auditoria
    ) THEN
        UPDATE vec_ejecucion_documental_v4.control_cadena_auditoria
           SET ultima_secuencia = 1,
               ultima_huella_sha256 = repeat('1', 64)
         WHERE control_id = true;
    END IF;
END
$marca$;
SQL

if docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql" \
    >/dev/null 2>&1; then
    echo "el down V4 destruyo evidencia sin opt-in explicito" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $conservada$
BEGIN
    IF to_regnamespace('vec_ejecucion_documental_v4') IS NULL
       OR NOT EXISTS (
           SELECT 1
             FROM vec_ejecucion_documental_v4.control_cadena_auditoria
            WHERE ultima_secuencia > 0
       ) THEN
        RAISE EXCEPTION 'el down fallido no conservo esquema y evidencia';
    END IF;
END
$conservada$;
SQL

docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.down.sql"

docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
REVOKE vec_autorizacion_fuente FROM vec_v4_fuente_prueba;
REVOKE vec_autorizacion_registro FROM vec_v4_registro_prueba;
REVOKE vec_ejecucion_documental_v4_emisor_capacidad FROM vec_v4_emisor_prueba;
REVOKE vec_ejecucion_documental_v4_ejecutor_atestado FROM vec_v4_ejecucion_prueba;
DROP ROLE vec_v4_fuente_prueba;
DROP ROLE vec_v4_registro_prueba;
DROP ROLE vec_v4_emisor_prueba;
DROP ROLE vec_v4_ejecucion_prueba;
SQL

# Una cuenta LOGIN miembro del propietario es una autoridad residual. El down
# de roles debe abortar antes de revocar nada y conservar el enlace para que el
# operador pueda investigarlo y retirarlo expresamente.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_intruso_propietario_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_ejecucion_documental_v4_propietario
    TO vec_v4_intruso_propietario_prueba;
SQL
if docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto un LOGIN miembro del propietario" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $conservado$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname =
                   'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_v4_intruso_propietario_prueba'
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto autoridad antes de fallar';
    END IF;
END
$conservado$;
REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_v4_intruso_propietario_prueba;
DROP ROLE vec_v4_intruso_propietario_prueba;
SQL
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql"

# El desmontaje no vuelve a abrir a PUBLIC las funciones de pgcrypto.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $cierre_persistente$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_extension AS extension
          JOIN pg_catalog.pg_depend AS dependencia
            ON dependencia.refclassid = 'pg_catalog.pg_extension'::regclass
           AND dependencia.refobjid = extension.oid
           AND dependencia.classid = 'pg_catalog.pg_proc'::regclass
           AND dependencia.deptype = 'e'
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = dependencia.objid
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE extension.extname = 'pgcrypto'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down reabrio pgcrypto a PUBLIC';
    END IF;
END
$cierre_persistente$;
SQL
