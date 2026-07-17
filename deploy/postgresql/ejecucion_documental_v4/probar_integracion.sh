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
salida_observador_roles_v4=$(mktemp)
salida_grant_roles_v4=$(mktemp)

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -f "$salida_observador_roles_v4" "$salida_grant_roles_v4"
}
trap limpiar EXIT INT TERM

exigir_rechazo_roles_down_v4_con_mutacion() {
    local descripcion=$1
    local mutacion=$2

    # La mutacion y el down comparten conexion y transaccion. El rechazo deja
    # la transaccion abortada y el cierre de psql restaura la instalacion exacta
    # para el caso siguiente.
    if docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        --command "BEGIN; ${mutacion}" --file=- \
        < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
        >/dev/null 2>&1; then
        echo "roles_down V4 acepto ${descripcion}" >&2
        exit 1
    fi
}

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

# CREATEROLE y ser propietario de la base no convierten a una identidad en
# autoridad de bootstrap. Debe fallar antes de crear cualquier rol, guarda o
# disparador V4, aunque el resto de precondiciones sean validas.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_instalador_no_super_prueba NOLOGIN NOSUPERUSER
    NOCREATEDB CREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $propietario_temporal$
BEGIN
    EXECUTE format(
        'ALTER DATABASE %I OWNER TO vec_v4_instalador_no_super_prueba',
        current_database()
    );
END
$propietario_temporal$;
SQL
if docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command 'SET SESSION AUTHORIZATION vec_v4_instalador_no_super_prueba' \
    --file=- < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql" \
    >/dev/null 2>&1; then
    echo "roles_up V4 acepto un CREATEROLE no superusuario" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $sin_mutacion_no_super$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR pg_catalog.to_regnamespace(
        'vec_ejecucion_documental_v4_guardia'
    ) IS NOT NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
    ) THEN
        RAISE EXCEPTION 'el bootstrap V4 no superusuario dejo estado';
    END IF;
    EXECUTE format(
        'ALTER DATABASE %I OWNER TO postgres',
        current_database()
    );
END
$sin_mutacion_no_super$;
DROP ROLE vec_v4_instalador_no_super_prueba;
SQL

# Un bootstrap interrumpido antes de aplicar la primera migracion no ha creado
# aun filas de privilegios por defecto. Debe poder retirarse de forma exacta,
# sin adoptar estado ni exigir que la migracion haya llegado a ejecutarse.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql"
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $bootstrap_parcial_retirado$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR pg_catalog.to_regnamespace(
        'vec_ejecucion_documental_v4_guardia'
    ) IS NOT NULL OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          JOIN pg_catalog.pg_roles AS rol
            ON rol.oid = defecto.defaclrole
         WHERE rol.rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) THEN
        RAISE EXCEPTION 'la retirada del bootstrap V4 parcial dejo estado';
    END IF;
END
$bootstrap_parcial_retirado$;
SQL

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

# Tambien debe detectarse una membresia saliente: un rol V4 incorporado a un
# grupo ajeno es una dependencia compartida que el desmontaje no puede borrar.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_grupo_ajeno_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_v4_grupo_ajeno_prueba
    TO vec_ejecucion_documental_v4_emisor_capacidad;
SQL
if docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto una membresia V4 saliente" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $saliente_conservada$
BEGIN
    IF NOT pg_has_role(
        'vec_ejecucion_documental_v4_emisor_capacidad',
        'vec_v4_grupo_ajeno_prueba',
        'MEMBER'
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto una membresia saliente antes de fallar';
    END IF;
END
$saliente_conservada$;
REVOKE vec_v4_grupo_ajeno_prueba
    FROM vec_ejecucion_documental_v4_emisor_capacidad;
DROP ROLE vec_v4_grupo_ajeno_prueba;
SQL

# Las tres opciones y el otorgante forman parte de la arista estructural. Una
# concesion semanticamente distinta no se adopta por coincidir sus extremos.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_ejecucion_documental_v4_migrador;
GRANT vec_ejecucion_documental_v4_propietario
    TO vec_ejecucion_documental_v4_migrador
    WITH ADMIN FALSE, INHERIT TRUE, SET TRUE;
SQL
if docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto opciones estructurales V4 alteradas" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $opciones_conservadas$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS TRUE
           AND membresia.set_option IS TRUE
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto la arista estructural antes de fallar';
    END IF;
END
$opciones_conservadas$;
REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_ejecucion_documental_v4_migrador;
GRANT vec_ejecucion_documental_v4_propietario
    TO vec_ejecucion_documental_v4_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL

# PostgreSQL atribuye al superusuario bootstrap (OID 10) toda membresia
# concedida por un superusuario. El inventario exige expresamente ese grantor;
# no depende del nombre de la LOGIN DBA que ejecute la ventana de retirada.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
DO $grantor_gobernado$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_roles AS otorgante
            ON otorgante.oid = membresia.grantor
         WHERE grupo.rolname = 'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND otorgante.oid = 10
           AND otorgante.rolsuper
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION 'la arista estructural no conserva el grantor bootstrap';
    END IF;
END
$grantor_gobernado$;
SQL

# El inventario inspecciona tambien la tercera coordenada. Esta deriva de
# catalogo controlada deja ambos extremos fuera de V4 y usa un rol V4 solo como
# grantor; despues del rechazo se restaura el OID bootstrap antes del REVOKE.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_grupo_grantor_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_v4_miembro_grantor_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_v4_grupo_grantor_prueba TO vec_v4_miembro_grantor_prueba;
UPDATE pg_catalog.pg_auth_members
   SET grantor = (
       SELECT oid
         FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_ejecucion_documental_v4_emisor_capacidad'
   )
 WHERE roleid = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_v4_grupo_grantor_prueba'
   )
   AND member = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_v4_miembro_grantor_prueba'
   );
SQL
if docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" \
    >/dev/null 2>&1; then
    echo "roles_down acepto un rol V4 usado solo como grantor" >&2
    exit 1
fi
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $grantor_solo_conservado$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_roles AS otorgante
            ON otorgante.oid = membresia.grantor
         WHERE grupo.rolname = 'vec_v4_grupo_grantor_prueba'
           AND miembro.rolname = 'vec_v4_miembro_grantor_prueba'
           AND otorgante.rolname =
                   'vec_ejecucion_documental_v4_emisor_capacidad'
    ) OR NOT has_function_privilege(
        'vec_ejecucion_documental_v4_propietario',
        'public.hmac(bytea,bytea,text)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar el grantor V4';
    END IF;
END
$grantor_solo_conservado$;
UPDATE pg_catalog.pg_auth_members
   SET grantor = 10
 WHERE roleid = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_v4_grupo_grantor_prueba'
   )
   AND member = (
       SELECT oid FROM pg_catalog.pg_roles
        WHERE rolname = 'vec_v4_miembro_grantor_prueba'
   );
REVOKE vec_v4_grupo_grantor_prueba FROM vec_v4_miembro_grantor_prueba;
DROP ROLE vec_v4_miembro_grantor_prueba;
DROP ROLE vec_v4_grupo_grantor_prueba;
SQL

# Atributos de rol, guarda, ACL y defaults son parte del contrato de retirada.
# Cada caso se ensaya en una transaccion independiente que debe abortar antes
# de la primera revocacion o eliminacion.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_guardia_propietario_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE SCHEMA vec_v4_acl_externa_prueba;
CREATE TABLE vec_v4_acl_externa_prueba.documento (identificador bigint);
SQL

mutaciones_atributos_v4=(
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado PASSWORD 'fixture_no_secreta';"
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado VALID UNTIL '2027-01-01 00:00:00+00';"
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado SET statement_timeout = '1s';"
    "ALTER ROLE vec_ejecucion_documental_v4_ejecutor_atestado IN DATABASE ${base} SET statement_timeout = '1s';"
)
for mutacion in "${mutaciones_atributos_v4[@]}"; do
    exigir_rechazo_roles_down_v4_con_mutacion \
        "atributos de rol ajenos al bootstrap" "$mutacion"
done

exigir_rechazo_roles_down_v4_con_mutacion \
    "otro propietario del esquema guarda" \
    "ALTER SCHEMA vec_ejecucion_documental_v4_guardia OWNER TO vec_v4_guardia_propietario_prueba;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "otro propietario de la funcion guarda" \
    "ALTER FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() OWNER TO vec_v4_guardia_propietario_prueba;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "USAGE publico en la guarda" \
    "GRANT USAGE ON SCHEMA vec_ejecucion_documental_v4_guardia TO PUBLIC;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "EXECUTE publico en la guarda" \
    "GRANT EXECUTE ON FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() TO PUBLIC;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "proconfig manipulado en la guarda" \
    "ALTER FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() SET search_path = pg_catalog;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "atributos manipulados en la funcion guarda" \
    "ALTER FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() SECURITY INVOKER;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "etiquetas manipuladas en la guarda" \
    "DROP EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos; CREATE EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos ON ddl_command_end WHEN TAG IN ('CREATE TABLE') EXECUTE FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos();"
exigir_rechazo_roles_down_v4_con_mutacion \
    "una funcion adicional en la guarda" \
    "CREATE FUNCTION vec_ejecucion_documental_v4_guardia.extra_prueba() RETURNS event_trigger LANGUAGE plpgsql AS 'BEGIN END;'; REVOKE ALL ON FUNCTION vec_ejecucion_documental_v4_guardia.extra_prueba() FROM PUBLIC;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "un uso adicional de la funcion guarda" \
    "CREATE EVENT TRIGGER vec_ejecucion_documental_v4_extra_prueba ON ddl_command_end WHEN TAG IN ('CREATE TABLE') EXECUTE FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos();"
exigir_rechazo_roles_down_v4_con_mutacion \
    "TEMPORARY adicional en la base" \
    "GRANT TEMPORARY ON DATABASE ${base} TO vec_ejecucion_documental_v4_emisor_capacidad;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "CREATE adicional sobre public" \
    "GRANT CREATE ON SCHEMA public TO vec_ejecucion_documental_v4_propietario;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "HMAC con opcion de concesion" \
    "GRANT EXECUTE ON FUNCTION public.hmac(bytea,bytea,text) TO vec_ejecucion_documental_v4_propietario WITH GRANT OPTION;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "una ACL de tabla externa" \
    "GRANT SELECT ON vec_v4_acl_externa_prueba.documento TO vec_ejecucion_documental_v4_emisor_capacidad;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "un beneficiario adicional en defaults" \
    "ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario GRANT EXECUTE ON FUNCTIONS TO vec_ejecucion_documental_v4_emisor_capacidad;"
exigir_rechazo_roles_down_v4_con_mutacion \
    "otra clase de default ACL" \
    "ALTER DEFAULT PRIVILEGES FOR ROLE vec_ejecucion_documental_v4_propietario GRANT USAGE ON SCHEMAS TO vec_ejecucion_documental_v4_emisor_capacidad;"

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $estado_canonico_tras_negativas$
DECLARE
    oid_ejecutor oid;
BEGIN
    SELECT oid INTO STRICT oid_ejecutor
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_ejecucion_documental_v4_ejecutor_atestado'
       AND rolpassword IS NULL
       AND rolvaliduntil IS NULL;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_db_role_setting
         WHERE setrole = oid_ejecutor
    ) OR pg_catalog.to_regprocedure(
        'vec_ejecucion_documental_v4_guardia.extra_prueba()'
    ) IS NOT NULL OR EXISTS (
        SELECT 1 FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_extra_prueba'
    ) OR NOT EXISTS (
        SELECT 1
         FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_ejecucion_documental_v4_cerrar_acl_tipos'
           AND cardinality(evttags) = 12
    ) THEN
        RAISE EXCEPTION 'una prueba negativa V4 no revirtio su transaccion';
    END IF;
END
$estado_canonico_tras_negativas$;
DROP TABLE vec_v4_acl_externa_prueba.documento;
DROP SCHEMA vec_v4_acl_externa_prueba;
DROP ROLE vec_v4_guardia_propietario_prueba;
SQL

# Carrera real entre bases. Se conservan los OID antes del DROP para auditar
# tambien roleid/member/grantor. El bloqueo de pg_database abre una ventana en
# la que roles_down ya retiene pg_authid/pg_auth_members y el GRANT espera; no
# se cancela ninguna sesion: tras COMMIT debe fallar porque el rol desaparecio.
docker exec --interactive "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_grant_concurrente_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
SQL
oids_roles_v4=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname "$base" \
    --command "SELECT string_agg(oid::text, ',' ORDER BY rolname) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_ejecucion_documental_v4_propietario', 'vec_ejecucion_documental_v4_migrador', 'vec_ejecucion_documental_v4_emisor_capacidad', 'vec_ejecucion_documental_v4_ejecutor_atestado')")
if [[ ! "$oids_roles_v4" =~ ^[0-9]+,[0-9]+,[0-9]+,[0-9]+$ ]]; then
    echo "no se pudieron fijar los OID de roles V4" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_v4_roles_down_bloqueo \
    "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL' &
\set ON_ERROR_STOP 1
SELECT pg_advisory_lock(
    hashtextextended('vec_ejecucion_documental_v4:roles_down:v2', 0)
);
SELECT pg_sleep(4);
BEGIN;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;
SELECT pg_advisory_unlock(
    hashtextextended('vec_ejecucion_documental_v4:roles_down:v2', 0)
);
SELECT pg_sleep(4);
ROLLBACK;
SQL
pid_bloqueo_roles_v4=$!
bloqueo_roles_v4_observado=false
for _ in $(seq 1 40); do
    estado=$(docker exec "$contenedor" psql --tuples-only --no-align \
        --username postgres --dbname "$base" \
        --command "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE application_name = 'vec_v4_roles_down_bloqueo' AND state = 'active' AND query LIKE 'SELECT pg_sleep(4)%'")
    if [[ "$estado" == "1" ]]; then
        bloqueo_roles_v4_observado=true
        break
    fi
    sleep 0.1
done
if [[ "$bloqueo_roles_v4_observado" != true ]]; then
    wait "$pid_bloqueo_roles_v4" || true
    echo "no se observo el bloqueo preparatorio de roles_down V4" >&2
    exit 1
fi

docker exec --interactive --env PGAPPNAME=vec_v4_roles_down_carrera \
    "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql" &
pid_roles_down_v4=$!
docker exec --interactive --env PGAPPNAME=vec_v4_grant_carrera \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --set VERBOSITY=verbose --username postgres --dbname postgres \
    >"$salida_grant_roles_v4" 2>&1 <<'SQL' &
\set ON_ERROR_STOP 1
\set VERBOSITY verbose
SELECT pg_sleep(5);
GRANT vec_ejecucion_documental_v4_ejecutor_atestado
    TO vec_v4_grant_concurrente_prueba;
SQL
pid_grant_v4=$!

# El observador conecta antes de que pg_database quede congelado. Consulta las
# funciones de estadisticas directamente: pg_stat_activity lee pg_authid y se
# bloquearia por la misma barrera que se esta intentando demostrar.
docker exec --interactive --env PGAPPNAME=vec_v4_observador_carrera \
    "$contenedor" psql -X --quiet --tuples-only --no-align \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    >"$salida_observador_roles_v4" <<'SQL' &
SELECT pg_sleep(7);
WITH actividad AS MATERIALIZED (
    SELECT *
      FROM pg_catalog.pg_stat_get_activity(NULL::integer)
)
SELECT count(*)
  FROM actividad AS concesion
  JOIN actividad AS retirada
    ON retirada.application_name = 'vec_v4_roles_down_carrera'
   AND retirada.pid = ANY (pg_catalog.pg_blocking_pids(concesion.pid))
 WHERE concesion.application_name = 'vec_v4_grant_carrera'
   AND concesion.state = 'active'
   AND concesion.wait_event_type = 'Lock'
   AND (
       SELECT count(*)
         FROM pg_catalog.pg_lock_status() AS bloqueo
        WHERE bloqueo.pid = retirada.pid
          AND bloqueo.locktype = 'relation'
          AND bloqueo.relation IN (
              'pg_catalog.pg_authid'::regclass::oid,
              'pg_catalog.pg_auth_members'::regclass::oid
          )
          AND bloqueo.mode = 'AccessExclusiveLock'
          AND bloqueo.granted
   ) = 2;
SQL
pid_observador_roles_v4=$!

if ! wait "$pid_observador_roles_v4"; then
    wait "$pid_bloqueo_roles_v4" || true
    wait "$pid_roles_down_v4" || true
    wait "$pid_grant_v4" || true
    echo "fallo el observador preconectado de la carrera roles_down V4" >&2
    exit 1
fi
if ! grep -qx '1' "$salida_observador_roles_v4"; then
    wait "$pid_bloqueo_roles_v4" || true
    wait "$pid_roles_down_v4" || true
    wait "$pid_grant_v4" || true
    echo "no se demostro que el GRANT esperase al roles_down V4" >&2
    exit 1
fi

wait "$pid_bloqueo_roles_v4"
wait "$pid_roles_down_v4"
grant_v4_rechazado=false
if ! wait "$pid_grant_v4"; then
    grant_v4_rechazado=true
fi
if [[ "$grant_v4_rechazado" != true ]]; then
    echo "un GRANT V4 concurrente sobrevivio al DROP ROLE" >&2
    exit 1
fi
if ! grep -Eq 'ERROR: +42704:' "$salida_grant_roles_v4" \
   || ! grep -Fq \
       'vec_ejecucion_documental_v4_ejecutor_atestado' \
       "$salida_grant_roles_v4"; then
    echo "el GRANT V4 no acredito SQLSTATE 42704 y el rol retirado" >&2
    sed -n '1,20p' "$salida_grant_roles_v4" >&2
    exit 1
fi

restos_roles_v4=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS membresia WHERE membresia.roleid IN ($oids_roles_v4) OR membresia.member IN ($oids_roles_v4) OR membresia.grantor IN ($oids_roles_v4)")
huerfanas_roles_v4=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_auth_members AS membresia LEFT JOIN pg_catalog.pg_authid AS grupo ON grupo.oid = membresia.roleid LEFT JOIN pg_catalog.pg_authid AS miembro ON miembro.oid = membresia.member LEFT JOIN pg_catalog.pg_authid AS otorgante ON otorgante.oid = membresia.grantor WHERE grupo.oid IS NULL OR miembro.oid IS NULL OR otorgante.oid IS NULL")
roles_v4_restantes=$(docker exec "$contenedor" psql --tuples-only --no-align \
    --username postgres --dbname postgres \
    --command "SELECT count(*) FROM pg_catalog.pg_roles WHERE rolname IN ('vec_ejecucion_documental_v4_propietario', 'vec_ejecucion_documental_v4_migrador', 'vec_ejecucion_documental_v4_emisor_capacidad', 'vec_ejecucion_documental_v4_ejecutor_atestado')")
if [[ "$restos_roles_v4" != "0" || "$huerfanas_roles_v4" != "0" \
      || "$roles_v4_restantes" != "0" ]]; then
    echo "roles_down V4 dejo roles o membresias huerfanas" >&2
    exit 1
fi
docker exec "$contenedor" psql --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres \
    --command "DROP ROLE vec_v4_grant_concurrente_prueba"

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

# La guarda puede pertenecer a un DBA nominativo distinto del bootstrap. La
# membresia estructural conserva grantor OID 10, mientras esquema y funcion son
# propiedad del DBA que hizo el bootstrap; ese mismo DBA completa la retirada.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_v4_dba_alternativo_prueba NOLOGIN SUPERUSER;
SQL
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command 'SET SESSION AUTHORIZATION vec_v4_dba_alternativo_prueba' \
    --file=- < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $instalacion_dba_alternativo$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          JOIN pg_catalog.pg_roles AS propietario
            ON propietario.oid = espacio.nspowner
         WHERE espacio.nspname = 'vec_ejecucion_documental_v4_guardia'
           AND propietario.rolname = 'vec_v4_dba_alternativo_prueba'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_roles AS propietario
            ON propietario.oid = funcion.proowner
         WHERE funcion.oid = pg_catalog.to_regprocedure(
                   'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
               )
           AND propietario.rolname = 'vec_v4_dba_alternativo_prueba'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
          JOIN pg_catalog.pg_authid AS otorgante
            ON otorgante.oid = membresia.grantor
         WHERE grupo.rolname = 'vec_ejecucion_documental_v4_propietario'
           AND miembro.rolname = 'vec_ejecucion_documental_v4_migrador'
           AND otorgante.oid = 10
           AND otorgante.rolsuper
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION 'el DBA alternativo no separo propiedad y grantor V4';
    END IF;
END
$instalacion_dba_alternativo$;
SQL
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.up.sql"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql"
docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.down.sql"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command 'SET SESSION AUTHORIZATION vec_v4_dba_alternativo_prueba' \
    --file=- < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_down.sql"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $retirada_dba_alternativo$
BEGIN
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR pg_catalog.to_regnamespace(
        'vec_ejecucion_documental_v4_guardia'
    ) IS NOT NULL THEN
        RAISE EXCEPTION 'el DBA alternativo no completo la retirada V4';
    END IF;
END
$retirada_dba_alternativo$;
DROP ROLE vec_v4_dba_alternativo_prueba;
SQL

echo "integracion ejecucion documental V4/PostgreSQL 18.4: correcta"
