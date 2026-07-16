#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-baremacion-pg-${USER:-usuario}-$$"
base=vec_bolsa_baremacion_prueba

# Cada cuenta recibe 256 bits del generador criptografico del sistema. Los
# valores se asignan directamente a variables: nunca se escriben ni imprimen.
# Se evita depender de OpenSSL; od y tr forman parte del entorno base GNU/Linux.
generar_clave_prueba() {
    local destino=$1
    local valor

    if [[ ! -c /dev/urandom || ! -r /dev/urandom ]] \
        || ! command -v od >/dev/null 2>&1 \
        || ! command -v tr >/dev/null 2>&1; then
        echo "no hay una fuente local de entropia utilizable" >&2
        return 1
    fi
    if ! valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]'); then
        echo "no se pudo obtener entropia local" >&2
        return 1
    fi
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "la fuente local de entropia devolvio una clave invalida" >&2
        return 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_ejecutor=
clave_lector=
clave_registrador=
generar_clave_prueba clave_admin
generar_clave_prueba clave_ejecutor
generar_clave_prueba clave_lector
generar_clave_prueba clave_registrador
if [[ "$clave_admin" == "$clave_ejecutor" \
    || "$clave_admin" == "$clave_lector" \
    || "$clave_admin" == "$clave_registrador" \
    || "$clave_ejecutor" == "$clave_lector" \
    || "$clave_ejecutor" == "$clave_registrador" \
    || "$clave_lector" == "$clave_registrador" ]]; then
    echo "la fuente local de entropia produjo claves repetidas" >&2
    exit 1
fi
salida_uno=$(mktemp)
salida_dos=$(mktemp)

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -f "$salida_uno" "$salida_dos"
}
trap limpiar EXIT INT TERM

arrancar_postgres() {
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
}

psql_archivo() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
        < "$raiz/$1"
}

psql_archivo_con_destruccion() {
    docker exec --interactive \
        --env PGOPTIONS="-c vec.confirmar_destruccion_bolsa_baremacion=DESTRUIR_HISTORIA_BOLSA_BAREMACION_IRREVERSIBLE" \
        "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" < "$raiz/$1"
}

psql_archivo_con_reversion() {
    docker exec --interactive \
        --env PGOPTIONS="-c vec.confirmar_reversion_bolsa_baremacion=REVERTIR_MIGRACION_BOLSA_BAREMACION_V1" \
        "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" < "$raiz/$1"
}

exigir_fallo_archivo() {
    local archivo=$1
    local descripcion=$2
    if psql_archivo "$archivo" >/dev/null 2>&1; then
        echo "$descripcion" >&2
        exit 1
    fi
}

exigir_fallo_archivo_con_destruccion() {
    local archivo=$1
    local descripcion=$2
    if psql_archivo_con_destruccion "$archivo" >/dev/null 2>&1; then
        echo "$descripcion" >&2
        exit 1
    fi
}

exigir_fallo_archivo_con_reversion() {
    local archivo=$1
    local descripcion=$2
    if psql_archivo_con_reversion "$archivo" >/dev/null 2>&1; then
        echo "$descripcion" >&2
        exit 1
    fi
}

rechazar_consulta_runtime() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    local descripcion=$4

    if docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -X --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" \
        --command "$consulta" >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

instalar_bolsa() {
    psql_archivo deploy/postgresql/bolsa_baremacion/roles_up.sql
    psql_archivo deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.up.sql
    psql_archivo deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.up.sql
    psql_archivo deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.up.sql
    psql_archivo deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.up.sql
    psql_archivo deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.up.sql
}

desinstalar_bolsa_vacia() {
    psql_archivo_con_reversion deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.down.sql
    psql_archivo_con_reversion deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.down.sql
    psql_archivo_con_reversion deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.down.sql
    psql_archivo_con_destruccion deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql
    psql_archivo_con_reversion deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.down.sql
    psql_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql
}

(cd "$raiz" && go test \
    ./internal/modules/bolsa/adapters/postgres \
    ./internal/modules/bolsa/internal/transaccion \
    ./internal/modules/bolsa/ports)

arrancar_postgres

psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql

# Primera pasada: toda la prueba funcional y adversaria revierte. A
# continuacion se verifica que el down conservador funciona sobre un esquema
# realmente vacio y que la instalacion completa es repetible.
instalar_bolsa
psql_archivo deploy/postgresql/bolsa_baremacion/pruebas_sql/integracion_v1.sql

# Las ACL se ejercen con LOGIN independientes, no solo mediante SET ROLE desde
# postgres. El registrador reservado recibe su grupo pero debe seguir sin poder
# conectar hasta que exista una frontera criptografica productiva.
docker exec --interactive \
    --env CLAVE_EJECUTOR="$clave_ejecutor" \
    --env CLAVE_LECTOR="$clave_lector" \
    --env CLAVE_REGISTRADOR="$clave_registrador" \
    "$contenedor" \
    psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
\getenv clave_lector CLAVE_LECTOR
\getenv clave_registrador CLAVE_REGISTRADOR
CREATE ROLE vec_bolsa_ejecutor_acl_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_ejecutor';
CREATE ROLE vec_bolsa_lector_acl_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_lector';
CREATE ROLE vec_bolsa_registrador_acl_prueba LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD :'clave_registrador';
GRANT vec_bolsa_baremacion_ejecutor TO vec_bolsa_ejecutor_acl_prueba;
GRANT vec_bolsa_baremacion_lector_outbox TO vec_bolsa_lector_acl_prueba;
GRANT vec_bolsa_baremacion_registrador_atestacion
    TO vec_bolsa_registrador_acl_prueba;

SET ROLE vec_bolsa_baremacion_propietario;
CREATE FUNCTION vec_bolsa_baremacion.funcion_futura_acl_prueba()
RETURNS integer
LANGUAGE sql
IMMUTABLE
SET search_path = pg_catalog, pg_temp
AS 'SELECT 1';
CREATE TABLE vec_bolsa_baremacion.tabla_futura_acl_prueba (
    identificador bigint PRIMARY KEY
);
RESET ROLE;

DO $acl_cerrada$
DECLARE
    oid_propietario oid;
BEGIN
    SELECT oid INTO oid_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_bolsa_baremacion_propietario';
    IF oid_propietario IS NULL THEN
        RAISE EXCEPTION 'falta propietario Bolsa';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
       ) IS NULL OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_bolsa_baremacion_cerrar_acl_tipos'
           AND evtevent = 'ddl_command_end'
           AND evtenabled = 'O'
           AND evtfoid = pg_catalog.to_regprocedure(
               'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
           )
    ) THEN
        RAISE EXCEPTION 'falta la guarda automatica de tipos Bolsa';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclnamespace = 0
           AND defecto.defaclobjtype = 'f'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl)
              AS privilegio
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclnamespace = 0
           AND defecto.defaclobjtype = 'f'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclnamespace = 0
           AND defecto.defaclobjtype = 'T'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl)
              AS privilegio
         WHERE defecto.defaclrole = oid_propietario
           AND defecto.defaclnamespace = 0
           AND defecto.defaclobjtype = 'T'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'USAGE'
    ) THEN
        RAISE EXCEPTION 'defaults globales de funciones/tipos no cerrados';
    END IF;
    IF has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.texto_opaco_valido(text,integer)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_lector_outbox',
           'vec_bolsa_baremacion.huella_sha256_valida(text)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.funcion_futura_acl_prueba()',
           'EXECUTE'
       ) OR has_type_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.version_baremacion',
           'USAGE'
       ) OR has_type_privilege(
           'vec_bolsa_baremacion_lector_outbox',
           'vec_bolsa_baremacion.evento_outbox',
           'USAGE'
       ) OR has_type_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.tabla_futura_acl_prueba',
           'USAGE'
    ) THEN
        RAISE EXCEPTION 'un runtime hereda helpers o tipos internos';
    END IF;

    -- Inventario completo, no una muestra: cada funcion ejecutable debe estar
    -- en la lista positiva de su unico runtime y PUBLIC no ejecuta ninguna.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND has_function_privilege(
               'vec_bolsa_baremacion_ejecutor', funcion.oid, 'EXECUTE'
           ) IS DISTINCT FROM (funcion.oid IN (
               'vec_bolsa_baremacion.reservar_cambio(jsonb,jsonb,bytea,bytea)'::regprocedure,
               'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)'::regprocedure,
               'vec_bolsa_baremacion.abandonar_reserva(jsonb,jsonb,bytea,bytea)'::regprocedure,
               'vec_bolsa_baremacion.obtener_version_vigente(jsonb,jsonb,bytea,bytea)'::regprocedure,
               'vec_bolsa_baremacion.obtener_version(jsonb,jsonb,bytea,bytea)'::regprocedure,
               'vec_bolsa_baremacion.obtener_evidencia_transaccion(jsonb,jsonb,bytea,bytea)'::regprocedure
           ))
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND has_function_privilege(
               'vec_bolsa_baremacion_lector_outbox', funcion.oid, 'EXECUTE'
           ) IS DISTINCT FROM (funcion.oid IN (
               'vec_bolsa_baremacion.reclamar_evento_outbox(text,bytea,integer)'::regprocedure,
               'vec_bolsa_baremacion.finalizar_entrega_outbox(text,text,bytea,text,text)'::regprocedure
           ))
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND has_function_privilege(
               'vec_bolsa_baremacion_registrador_atestacion',
               funcion.oid,
               'EXECUTE'
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = funcion.pronamespace
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND privilegio.grantee = 0
           AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'la lista positiva de funciones runtime no es exacta';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
          CROSS JOIN (VALUES
              ('vec_bolsa_baremacion_ejecutor'::name),
              ('vec_bolsa_baremacion_lector_outbox'::name),
              ('vec_bolsa_baremacion_registrador_atestacion'::name)
          ) AS runtime(rol)
          CROSS JOIN (VALUES
              ('SELECT'::text), ('INSERT'::text), ('UPDATE'::text),
              ('DELETE'::text), ('TRUNCATE'::text), ('REFERENCES'::text),
              ('TRIGGER'::text)
          ) AS permiso(privilegio)
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND clase.relkind IN ('r', 'p', 'v', 'm', 'f')
           AND has_table_privilege(
               runtime.rol, clase.oid, permiso.privilegio
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
          CROSS JOIN (VALUES
              ('vec_bolsa_baremacion_ejecutor'::name),
              ('vec_bolsa_baremacion_lector_outbox'::name),
              ('vec_bolsa_baremacion_registrador_atestacion'::name)
          ) AS runtime(rol)
          CROSS JOIN (VALUES
              ('SELECT'::text), ('INSERT'::text),
              ('UPDATE'::text), ('REFERENCES'::text)
          ) AS permiso(privilegio)
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND clase.relkind IN ('r', 'p', 'v', 'm', 'f')
           AND has_any_column_privilege(
               runtime.rol, clase.oid, permiso.privilegio
           )
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = clase.relnamespace
          CROSS JOIN (VALUES
              ('vec_bolsa_baremacion_ejecutor'::name),
              ('vec_bolsa_baremacion_lector_outbox'::name),
              ('vec_bolsa_baremacion_registrador_atestacion'::name)
          ) AS runtime(rol)
          CROSS JOIN (VALUES
              ('USAGE'::text), ('SELECT'::text), ('UPDATE'::text)
          ) AS permiso(privilegio)
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND clase.relkind = 'S'
           AND has_sequence_privilege(
               runtime.rol, clase.oid, permiso.privilegio
           )
    ) THEN
        RAISE EXCEPTION 'un runtime conserva privilegios de relacion o columna';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          JOIN pg_catalog.pg_namespace AS espacio
            ON espacio.oid = tipo.typnamespace
         WHERE espacio.nspname = 'vec_bolsa_baremacion'
           AND tipo.typelem = 0
           AND tipo.typisdefined
           AND (
               has_type_privilege(
                   'vec_bolsa_baremacion_ejecutor', tipo.oid, 'USAGE'
               ) OR has_type_privilege(
                   'vec_bolsa_baremacion_lector_outbox', tipo.oid, 'USAGE'
               ) OR has_type_privilege(
                   'vec_bolsa_baremacion_registrador_atestacion',
                   tipo.oid,
                   'USAGE'
               )
           )
    ) THEN
        RAISE EXCEPTION 'un runtime conserva USAGE sobre tipos Bolsa';
    END IF;

    IF NOT has_schema_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion',
           'USAGE'
       ) OR has_schema_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion',
           'CREATE'
       ) OR NOT has_schema_privilege(
           'vec_bolsa_baremacion_lector_outbox',
           'vec_bolsa_baremacion',
           'USAGE'
       ) OR has_schema_privilege(
           'vec_bolsa_baremacion_lector_outbox',
           'vec_bolsa_baremacion',
           'CREATE'
       ) OR has_schema_privilege(
           'vec_bolsa_baremacion_registrador_atestacion',
           'vec_bolsa_baremacion',
           'USAGE'
       ) OR has_schema_privilege(
           'vec_bolsa_baremacion_registrador_atestacion',
           'vec_bolsa_baremacion',
           'CREATE'
       ) THEN
        RAISE EXCEPTION 'las ACL de esquema runtime no son minimas';
    END IF;

    IF NOT has_database_privilege(
           'vec_bolsa_baremacion_ejecutor', current_database(), 'CONNECT'
       ) OR has_database_privilege(
           'vec_bolsa_baremacion_ejecutor', current_database(), 'CREATE'
       ) OR has_database_privilege(
           'vec_bolsa_baremacion_ejecutor', current_database(), 'TEMP'
       ) OR NOT has_database_privilege(
           'vec_bolsa_baremacion_lector_outbox', current_database(), 'CONNECT'
       ) OR has_database_privilege(
           'vec_bolsa_baremacion_lector_outbox', current_database(), 'CREATE'
       ) OR has_database_privilege(
           'vec_bolsa_baremacion_lector_outbox', current_database(), 'TEMP'
       ) THEN
        RAISE EXCEPTION 'las ACL de base runtime no son minimas';
    END IF;
    IF has_database_privilege(
        'vec_bolsa_baremacion_registrador_atestacion',
        current_database(),
        'CONNECT'
    ) OR has_database_privilege(
        'vec_bolsa_baremacion_registrador_atestacion',
        current_database(),
        'CREATE'
    ) OR has_database_privilege(
        'vec_bolsa_baremacion_registrador_atestacion',
        current_database(),
        'TEMP'
    ) THEN
        RAISE EXCEPTION 'el registrador reservado conserva privilegios de base';
    END IF;
END
$acl_cerrada$;
SQL

estado=$(docker exec --env PGPASSWORD="$clave_ejecutor" "$contenedor" \
    psql -X --quiet --tuples-only --no-align --set ON_ERROR_STOP=1 \
    --host 127.0.0.1 --username vec_bolsa_ejecutor_acl_prueba \
    --dbname "$base" --command \
    "SELECT resultado FROM vec_bolsa_baremacion.reservar_cambio('{}'::jsonb,'{}'::jsonb,decode('','hex'),decode('','hex'))" \
    | tr -d '[:space:]')
if [[ "$estado" != "rechazada" ]]; then
    echo "el ejecutor no ejercio exclusivamente su funcion cerrada: $estado" >&2
    exit 1
fi
estado=$(docker exec --env PGPASSWORD="$clave_lector" "$contenedor" \
    psql -X --quiet --tuples-only --no-align --set ON_ERROR_STOP=1 \
    --host 127.0.0.1 --username vec_bolsa_lector_acl_prueba \
    --dbname "$base" --command \
    "SELECT resultado FROM vec_bolsa_baremacion.reclamar_evento_outbox('consumidor:no-registrado',decode(repeat('aa',32),'hex'),60)" \
    | tr -d '[:space:]')
if [[ "$estado" != "consumidor_no_autorizado" ]]; then
    echo "el lector no ejercio exclusivamente su funcion cerrada: $estado" >&2
    exit 1
fi
rechazar_consulta_runtime vec_bolsa_ejecutor_acl_prueba "$clave_ejecutor" \
    "SELECT * FROM vec_bolsa_baremacion.version_baremacion" \
    "el ejecutor leyo tablas"
rechazar_consulta_runtime vec_bolsa_ejecutor_acl_prueba "$clave_ejecutor" \
    "SELECT vec_bolsa_baremacion.texto_opaco_valido('x',1)" \
    "el ejecutor invoco un helper"
rechazar_consulta_runtime vec_bolsa_ejecutor_acl_prueba "$clave_ejecutor" \
    "SELECT resultado FROM vec_bolsa_baremacion.reclamar_evento_outbox('x',decode(repeat('aa',32),'hex'),60)" \
    "el ejecutor consumio el outbox"
rechazar_consulta_runtime vec_bolsa_lector_acl_prueba "$clave_lector" \
    "SELECT resultado FROM vec_bolsa_baremacion.reservar_cambio('{}'::jsonb,'{}'::jsonb,decode('','hex'),decode('','hex'))" \
    "el lector invoco la operacion funcional"
rechazar_consulta_runtime vec_bolsa_registrador_acl_prueba \
    "$clave_registrador" "SELECT 1" \
    "el registrador reservado pudo conectar"

# Incluso vacio, el esquema no se destruye sin opt-in. Una dependencia de otro
# esquema tampoco queda incluida por la confirmacion destructiva local.
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql \
    "el down base acepto una retirada vacia sin opt-in"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $conservada_sin_opt_in$
BEGIN
    IF to_regclass(
           'vec_bolsa_baremacion.tabla_futura_acl_prueba'
       ) IS NULL OR to_regprocedure(
           'vec_bolsa_baremacion.funcion_futura_acl_prueba()'
       ) IS NULL THEN
        RAISE EXCEPTION 'el down sin opt-in muto el esquema';
    END IF;
END
$conservada_sin_opt_in$;
CREATE SCHEMA vec_bolsa_dependencia_externa_prueba;
CREATE VIEW vec_bolsa_dependencia_externa_prueba.vista_futura AS
    SELECT identificador
      FROM vec_bolsa_baremacion.tabla_futura_acl_prueba;
SQL
exigir_fallo_archivo_con_destruccion \
    deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql \
    "el down base destruyo una dependencia externa"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $dependencia_conservada$
BEGIN
    IF to_regclass(
           'vec_bolsa_dependencia_externa_prueba.vista_futura'
       ) IS NULL OR to_regclass(
           'vec_bolsa_baremacion.tabla_futura_acl_prueba'
       ) IS NULL THEN
        RAISE EXCEPTION 'el down con dependencia externa muto objetos';
    END IF;
END
$dependencia_conservada$;
DROP VIEW vec_bolsa_dependencia_externa_prueba.vista_futura;
DROP SCHEMA vec_bolsa_dependencia_externa_prueba;
SET ROLE vec_bolsa_baremacion_propietario;
DROP FUNCTION vec_bolsa_baremacion.funcion_futura_acl_prueba() RESTRICT;
DROP TABLE vec_bolsa_baremacion.tabla_futura_acl_prueba RESTRICT;
RESET ROLE;
REVOKE vec_bolsa_baremacion_ejecutor FROM vec_bolsa_ejecutor_acl_prueba;
REVOKE vec_bolsa_baremacion_lector_outbox FROM vec_bolsa_lector_acl_prueba;
REVOKE vec_bolsa_baremacion_registrador_atestacion
    FROM vec_bolsa_registrador_acl_prueba;
DROP ROLE vec_bolsa_ejecutor_acl_prueba;
DROP ROLE vec_bolsa_lector_acl_prueba;
DROP ROLE vec_bolsa_registrador_acl_prueba;
SQL
desinstalar_bolsa_vacia
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $desmontaje_completo$
BEGIN
    IF to_regnamespace('vec_bolsa_baremacion') IS NOT NULL
       OR to_regnamespace('vec_bolsa_baremacion_guardia') IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_event_trigger
            WHERE evtname = 'vec_bolsa_baremacion_cerrar_acl_tipos'
       ) OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname LIKE 'vec_bolsa_baremacion_%'
       ) OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL OR to_regnamespace('vec_autorizacion') IS NULL THEN
        RAISE EXCEPTION 'el desmontaje vacio no fue completo o afecto al nucleo';
    END IF;
END
$desmontaje_completo$;
SQL
instalar_bolsa

# Segunda pasada, solo dentro del contenedor desechable: conserva una fila de
# outbox con la que se prueba exclusion mutua entre dos sesiones reales.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --set CONFIRMAR_FIXTURE=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/bolsa_baremacion/pruebas_sql/integracion_v1.sql"

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_bolsa_outbox_concurrencia LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_baremacion_lector_outbox
    TO vec_bolsa_outbox_concurrencia;
INSERT INTO vec_bolsa_baremacion.consumidor_outbox_version (
    consumidor_ref, version, estado, rol_sesion, secuencia_inicial,
    registrada_en, acto_ref
) VALUES (
    'consumidor:bolsa:concurrencia', 1, 'activo',
    'vec_bolsa_outbox_concurrencia', 0, clock_timestamp(),
    'acto:consumidor:bolsa:concurrencia'
);
INSERT INTO vec_bolsa_baremacion.consumidor_outbox_actual (
    consumidor_ref, version, estado, actualizada_en
) VALUES (
    'consumidor:bolsa:concurrencia', 1, 'activo', clock_timestamp()
);
SQL

consulta_uno="BEGIN; SELECT resultado FROM vec_bolsa_baremacion.reclamar_evento_outbox('consumidor:bolsa:concurrencia', decode(repeat('aa',32),'hex'), 60); SELECT pg_sleep(1); COMMIT;"
consulta_dos="BEGIN; SELECT resultado FROM vec_bolsa_baremacion.reclamar_evento_outbox('consumidor:bolsa:concurrencia', decode(repeat('bb',32),'hex'), 60); SELECT pg_sleep(1); COMMIT;"

docker exec "$contenedor" psql -X --quiet --tuples-only --no-align \
    --set ON_ERROR_STOP=1 --username vec_bolsa_outbox_concurrencia \
    --dbname "$base" --command "$consulta_uno" >"$salida_uno" &
pid_uno=$!
docker exec "$contenedor" psql -X --quiet --tuples-only --no-align \
    --set ON_ERROR_STOP=1 --username vec_bolsa_outbox_concurrencia \
    --dbname "$base" --command "$consulta_dos" >"$salida_dos" &
pid_dos=$!
wait "$pid_uno"
wait "$pid_dos"

estados=$(grep -hE '^(reclamada|ocupada)$' \
    "$salida_uno" "$salida_dos" | sort | tr '\n' ' ')
if [[ "$estados" != "ocupada reclamada " ]]; then
    echo "exclusion outbox inesperada: $estados" >&2
    exit 1
fi
if grep -qx 'reclamada' "$salida_uno"; then
    token_ganador=aa
else
    token_ganador=bb
fi

resultado=$(docker exec "$contenedor" psql -X --quiet --tuples-only \
    --no-align --set ON_ERROR_STOP=1 \
    --username vec_bolsa_outbox_concurrencia --dbname "$base" \
    --command "SELECT vec_bolsa_baremacion.finalizar_entrega_outbox('consumidor:bolsa:concurrencia','evento:bolsa:001',decode(repeat('${token_ganador}',32),'hex'),'entregada',NULL)" \
    | tr -d '[:space:]')
if [[ "$resultado" != "entregada" ]]; then
    echo "finalizacion concurrente inesperada: $resultado" >&2
    exit 1
fi

integridad=$(docker exec "$contenedor" psql -X --quiet --tuples-only \
    --no-align --set ON_ERROR_STOP=1 --username postgres --dbname "$base" \
    --command "SELECT ((SELECT estado = 'pendiente' FROM vec_bolsa_baremacion.evento_outbox WHERE referencia = 'evento:bolsa:001') AND (SELECT count(*) = 2 FROM vec_bolsa_baremacion.entrega_outbox_version WHERE consumidor_ref = 'consumidor:bolsa:concurrencia') AND (SELECT count(*) = 1 FROM vec_bolsa_baremacion.cursor_outbox_version WHERE consumidor_ref = 'consumidor:bolsa:concurrencia') AND position(repeat('${token_ganador}',32) IN (SELECT string_agg(to_jsonb(v)::text,'') FROM vec_bolsa_baremacion.entrega_outbox_version AS v WHERE consumidor_ref = 'consumidor:bolsa:concurrencia')) = 0)::int" \
    | tr -d '[:space:]')
if [[ "$integridad" != "1" ]]; then
    echo "historia concurrente o secreto persistido incorrectamente" >&2
    exit 1
fi

# Toda migracion descendente intermedia debe detectar la historia del esquema
# completo antes de revocar una ACL, retirar una funcion o eliminar una tabla.
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.down.sql \
    "000004 down acepto historia sin opt-in"
exigir_fallo_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.down.sql \
    "000004 down desmonto parcialmente una Bolsa con historia y opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.down.sql \
    "000003 down acepto historia sin opt-in"
exigir_fallo_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.down.sql \
    "000003 down desmonto parcialmente una Bolsa con historia y opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.down.sql \
    "000002 down acepto historia sin opt-in"
exigir_fallo_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.down.sql \
    "000002 down desmonto parcialmente una Bolsa con historia y opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.down.sql \
    "el down de autorizacion acepto Bolsa instalada sin opt-in"
exigir_fallo_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.down.sql \
    "el down de autorizacion retiro la frontera con Bolsa instalada y opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql \
    "el down base destruyo historia sin opt-in"
exigir_fallo_archivo_con_destruccion \
    deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql \
    "el opt-in del down base autorizo borrar historia"

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $historia_y_superficie_conservadas$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.version_baremacion
    ) OR NOT EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.evento_outbox
    ) OR NOT EXISTS (
        SELECT 1 FROM vec_bolsa_baremacion.entrega_outbox_version
    ) THEN
        RAISE EXCEPTION 'un down fallido perdio historia durable';
    END IF;
    IF to_regprocedure(
           'vec_bolsa_baremacion.finalizar_entrega_outbox(text,text,bytea,text,text)'
       ) IS NULL OR to_regprocedure(
           'vec_bolsa_baremacion.obtener_evidencia_transaccion(jsonb,jsonb,bytea,bytea)'
       ) IS NULL OR to_regprocedure(
           'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)'
       ) IS NULL OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION 'un down fallido retiro funciones';
    END IF;
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_constraint AS restriccion
         WHERE restriccion.conrelid =
                   'vec_bolsa_baremacion.version_baremacion'::regclass
           AND restriccion.conname IN (
               'version_evento_exacto',
               'version_auditoria_exacta'
           )
         GROUP BY restriccion.conrelid
        HAVING count(*) = 2
    ) THEN
        RAISE EXCEPTION 'un down fallido retiro restricciones';
    END IF;
    IF NOT has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)',
           'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_bolsa_baremacion_lector_outbox',
           'vec_bolsa_baremacion.finalizar_entrega_outbox(text,text,bytea,text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'un down fallido revoco ACL runtime';
    END IF;
END
$historia_y_superficie_conservadas$;
SQL

# La instancia con historia se descarta intacta. El desmontaje y las pruebas
# adversarias de roles se ejecutan en una segunda instalacion realmente vacia;
# el runner no incorpora TRUNCATE, DELETE ni desactivacion de salvaguardas.
docker rm -f "$contenedor" >/dev/null
arrancar_postgres
psql_archivo deploy/postgresql/autorizacion/roles_up.sql
psql_archivo deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
psql_archivo deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
instalar_bolsa

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_bolsa_outbox_concurrencia LOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_baremacion_lector_outbox
    TO vec_bolsa_outbox_concurrencia;
SQL

# Sin el literal comun, cada down se cierra incluso sobre una instalacion vacia.
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.down.sql \
    "000004 down acepto una instalacion vacia sin opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.down.sql \
    "000003 down acepto una instalacion vacia sin opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.down.sql \
    "000002 down acepto una instalacion vacia sin opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.down.sql \
    "el down de autorizacion acepto una instalacion vacia sin opt-in"
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql \
    "el down base acepto una instalacion vacia sin opt-in"

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $superficie_vacia_conservada$
BEGIN
    IF to_regprocedure(
           'vec_bolsa_baremacion.finalizar_entrega_outbox(text,text,bytea,text,text)'
       ) IS NULL OR to_regprocedure(
           'vec_bolsa_baremacion.obtener_evidencia_transaccion(jsonb,jsonb,bytea,bytea)'
       ) IS NULL OR to_regprocedure(
           'vec_bolsa_baremacion.confirmar_cambio(jsonb,jsonb,bytea,bytea,bytea)'
       ) IS NULL OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION 'un down sin opt-in muto la instalacion vacia';
    END IF;
END
$superficie_vacia_conservada$;
SQL

psql_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones/000004_entrega_outbox.down.sql
psql_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones/000003_abandono_y_lecturas.down.sql
psql_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones/000002_operaciones_baremacion.down.sql
psql_archivo_con_destruccion \
    deploy/postgresql/bolsa_baremacion/migraciones/000001_bolsa_baremacion.down.sql

# Aun sin el consumidor, la frontera de autorizacion exige su propio opt-in.
exigir_fallo_archivo \
    deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.down.sql \
    "el down de autorizacion acepto ausencia del consumidor sin opt-in"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $frontera_conservada_sin_opt_in$
BEGIN
    IF to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NULL THEN
        RAISE EXCEPTION 'el down de autorizacion sin opt-in retiro la frontera';
    END IF;
END
$frontera_conservada_sin_opt_in$;
SQL
psql_archivo_con_reversion \
    deploy/postgresql/bolsa_baremacion/migraciones_autorizacion/000001_revalidacion_bolsa_baremacion.down.sql

exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto un LOGIN miembro del lector outbox"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $membresia_entrante_conservada$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_bolsa_baremacion_lector_outbox'
           AND miembro.rolname = 'vec_bolsa_outbox_concurrencia'
    ) OR to_regnamespace('vec_bolsa_baremacion_guardia') IS NULL
       OR NOT has_database_privilege(
           'vec_bolsa_baremacion_ejecutor', current_database(), 'CONNECT'
       ) THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar membresia entrante';
    END IF;
END
$membresia_entrante_conservada$;
REVOKE vec_bolsa_baremacion_lector_outbox
    FROM vec_bolsa_outbox_concurrencia;
DROP ROLE vec_bolsa_outbox_concurrencia;
SQL

# Propietario, ACL, search_path y etiquetas forman parte de la identidad de la
# guarda. Cada manipulacion debe abortar antes de retirar el disparador o
# revocar privilegios de runtime, y se restaura antes de ensayar la siguiente.
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
CREATE ROLE vec_bolsa_guardia_propietario_prueba NOLOGIN NOSUPERUSER
    NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
ALTER SCHEMA vec_bolsa_baremacion_guardia
    OWNER TO vec_bolsa_guardia_propietario_prueba;
ALTER FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos()
    OWNER TO vec_bolsa_guardia_propietario_prueba;
SQL
exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto propietarios manipulados en la guarda"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $propietarios_conservados$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          JOIN pg_catalog.pg_roles AS rol ON rol.oid = espacio.nspowner
         WHERE espacio.nspname = 'vec_bolsa_baremacion_guardia'
           AND rol.rolname = 'vec_bolsa_guardia_propietario_prueba'
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          JOIN pg_catalog.pg_roles AS rol ON rol.oid = funcion.proowner
         WHERE funcion.oid = to_regprocedure(
                   'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
               )
           AND rol.rolname = 'vec_bolsa_guardia_propietario_prueba'
    ) OR NOT has_database_privilege(
        'vec_bolsa_baremacion_ejecutor', current_database(), 'CONNECT'
    ) THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar propietarios';
    END IF;
END
$propietarios_conservados$;
ALTER FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos()
    OWNER TO postgres;
ALTER SCHEMA vec_bolsa_baremacion_guardia OWNER TO postgres;
DROP ROLE vec_bolsa_guardia_propietario_prueba;

GRANT USAGE ON SCHEMA vec_bolsa_baremacion_guardia TO PUBLIC;
GRANT EXECUTE ON FUNCTION
    vec_bolsa_baremacion_guardia.cerrar_acl_tipos() TO PUBLIC;
SQL
exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto ACL manipuladas en la guarda"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $acl_conservada$
BEGIN
    IF NOT has_schema_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion_guardia',
           'USAGE'
       ) OR NOT has_function_privilege(
           'vec_bolsa_baremacion_ejecutor',
           'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()',
           'EXECUTE'
       ) OR to_regprocedure(
           'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
       ) IS NULL THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar ACL de guarda';
    END IF;
END
$acl_conservada$;
REVOKE USAGE ON SCHEMA vec_bolsa_baremacion_guardia FROM PUBLIC;
REVOKE EXECUTE ON FUNCTION
    vec_bolsa_baremacion_guardia.cerrar_acl_tipos() FROM PUBLIC;

ALTER FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos()
    SET search_path = pg_catalog;
SQL
exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto un search_path manipulado en la guarda"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $configuracion_conservada$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc
         WHERE oid = to_regprocedure(
                   'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
               )
           AND proconfig = ARRAY['search_path=pg_catalog']::text[]
    ) THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar search_path';
    END IF;
END
$configuracion_conservada$;
ALTER FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos()
    SET search_path = pg_catalog, pg_temp;

DROP EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos;
CREATE EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos
    ON ddl_command_end
    WHEN TAG IN ('CREATE TABLE')
    EXECUTE FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos();
SQL
exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto etiquetas manipuladas en la guarda"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $etiquetas_conservadas$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtname = 'vec_bolsa_baremacion_cerrar_acl_tipos'
           AND evttags = ARRAY['CREATE TABLE']::text[]
    ) THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar etiquetas';
    END IF;
END
$etiquetas_conservadas$;
DROP EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos;
CREATE EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos
    ON ddl_command_end
    WHEN TAG IN (
        'CREATE TABLE',
        'CREATE TABLE AS',
        'CREATE FOREIGN TABLE',
        'CREATE VIEW',
        'CREATE MATERIALIZED VIEW',
        'CREATE TYPE',
        'CREATE DOMAIN',
        'ALTER TABLE',
        'ALTER VIEW',
        'ALTER MATERIALIZED VIEW',
        'ALTER TYPE',
        'ALTER DOMAIN'
    )
    EXECUTE FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos();

CREATE ROLE vec_bolsa_grupo_externo_prueba NOLOGIN NOSUPERUSER NOCREATEDB
    NOCREATEROLE NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_grupo_externo_prueba
    TO vec_bolsa_baremacion_ejecutor;
SQL
exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto un rol Bolsa dentro de un grupo externo"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $membresia_saliente_conservada$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_bolsa_grupo_externo_prueba'
           AND miembro.rolname = 'vec_bolsa_baremacion_ejecutor'
    ) OR to_regnamespace('vec_bolsa_baremacion_guardia') IS NULL THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar membresia saliente';
    END IF;
END
$membresia_saliente_conservada$;
REVOKE vec_bolsa_grupo_externo_prueba
    FROM vec_bolsa_baremacion_ejecutor;
DROP ROLE vec_bolsa_grupo_externo_prueba;

GRANT vec_bolsa_baremacion_propietario
    TO vec_bolsa_baremacion_migrador
    WITH ADMIN TRUE, INHERIT TRUE, SET TRUE;
SQL
exigir_fallo_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql \
    "roles_down acepto opciones estructurales alteradas"
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $opciones_conservadas$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
          JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname = 'vec_bolsa_baremacion_propietario'
           AND miembro.rolname = 'vec_bolsa_baremacion_migrador'
           AND membresia.admin_option
           AND membresia.inherit_option
           AND membresia.set_option
    ) OR to_regnamespace('vec_bolsa_baremacion_guardia') IS NULL THEN
        RAISE EXCEPTION 'roles_down muto antes de rechazar opciones alteradas';
    END IF;
END
$opciones_conservadas$;
GRANT vec_bolsa_baremacion_propietario
    TO vec_bolsa_baremacion_migrador
    WITH ADMIN FALSE, INHERIT FALSE, SET TRUE;
SQL

psql_archivo deploy/postgresql/bolsa_baremacion/roles_down.sql
docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
DO $desmontaje_final$
BEGIN
    IF to_regnamespace('vec_bolsa_baremacion') IS NOT NULL
       OR to_regnamespace('vec_bolsa_baremacion_guardia') IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_event_trigger
            WHERE evtname = 'vec_bolsa_baremacion_cerrar_acl_tipos'
       ) OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname LIKE 'vec_bolsa_baremacion_%'
       ) OR to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL OR to_regnamespace('vec_autorizacion') IS NULL THEN
        RAISE EXCEPTION 'desmontaje final incompleto o destructivo para el nucleo';
    END IF;
END
$desmontaje_final$;
SQL
"$raiz/deploy/postgresql/bolsa_baremacion/probar_integracion_v3.sh"
echo "integracion Bolsa/PostgreSQL 18.4: correcta"
