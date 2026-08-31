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

psql_admin() {
    docker exec --interactive "$contenedor" psql --no-psqlrc --quiet \
        --set ON_ERROR_STOP=1 --username postgres --dbname "$base" "$@"
}

aplicar() {
    psql_admin < "$raiz/$1"
}

psql_login() {
    local usuario=$1
    local clave=$2
    shift 2
    docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 --host 127.0.0.1 \
        --username "$usuario" --dbname "$base" "$@"
}

exigir_rechazo_login() {
    local usuario=$1
    local clave=$2
    local consulta=$3
    local descripcion=$4
    if psql_login "$usuario" "$clave" --command "$consulta" \
        >/dev/null 2>&1; then
        echo "ACL invalida: $descripcion" >&2
        exit 1
    fi
}

docker run --detach --rm --name "$contenedor" --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null
for _ in $(seq 1 60); do
    if docker exec "$contenedor" pg_isready --username postgres \
        --dbname "$base" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec "$contenedor" pg_isready --username postgres --dbname "$base" \
    >/dev/null
version_postgresql=$(psql_admin --tuples-only --no-align \
    --command 'SHOW server_version_num')
if [[ "$version_postgresql" != "180004" ]]; then
    echo "se requiere PostgreSQL 18.4, no ${version_postgresql}" >&2
    exit 1
fi

psql_admin <<'SQL'
CREATE EXTENSION IF NOT EXISTS pgcrypto WITH SCHEMA public;
SQL
aplicar deploy/postgresql/autorizacion/roles_up.sql
aplicar deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql

# El bootstrap V4 sigue reservado al superusuario. Una cuenta CREATEROLE que
# sea propietaria de la base debe fallar sin dejar roles ni guarda parciales.
psql_admin <<'SQL'
CREATE ROLE vec_v4_instalador_no_super_prueba NOLOGIN NOSUPERUSER
    NOCREATEDB CREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
DO $propietario$
BEGIN
    EXECUTE format(
        'ALTER DATABASE %I OWNER TO vec_v4_instalador_no_super_prueba',
        current_database()
    );
END
$propietario$;
SQL
if psql_admin \
    --command 'SET SESSION AUTHORIZATION vec_v4_instalador_no_super_prueba' \
    --file=- \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/roles_up.sql" \
    >/dev/null 2>&1; then
    echo "roles_up V4 acepto un CREATEROLE no superusuario" >&2
    exit 1
fi
psql_admin <<'SQL'
DO $sin_estado$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
    ) OR to_regnamespace('vec_ejecucion_documental_v4_guardia') IS NOT NULL THEN
        RAISE EXCEPTION 'el bootstrap no superusuario dejo estado';
    END IF;
    EXECUTE format('ALTER DATABASE %I OWNER TO postgres', current_database());
END
$sin_estado$;
DROP ROLE vec_v4_instalador_no_super_prueba;
SQL

# Un bootstrap interrumpido antes de la migracion debe retirarse y reinstalarse.
aplicar deploy/postgresql/ejecucion_documental_v4/roles_up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/roles_down.sql
aplicar deploy/postgresql/ejecucion_documental_v4/roles_up.sql

# Conserva una decision historica, aplica la autoridad actual de identidad y
# monta despues el esquema V4. El orden es parte del contrato reproducible.
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.down.sql
aplicar deploy/postgresql/ejecucion_documental_v4/pruebas_sql/sembrar_decision_legacy_v1.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.up.sql
aplicar deploy/postgresql/ejecucion_documental_v4/pruebas_sql/revalidacion_identidad_v1.sql
aplicar deploy/postgresql/ejecucion_documental_v4/pruebas_sql/privilegios_minimos_v2.sql

# La retirada base permanece protegida incluso sin evidencia nueva.
if aplicar \
    deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql \
    >/dev/null 2>&1; then
    echo "el down V4 acepto una retirada sin consentimiento" >&2
    exit 1
fi

psql_admin \
    --set clave_fuente="$clave_fuente" \
    --set clave_registro="$clave_registro" \
    --set clave_emisor="$clave_emisor" \
    --set clave_ejecucion="$clave_ejecucion" <<'SQL'
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

# Las dos identidades runtime siguen separadas y no ejecutan pgcrypto.
for identidad in \
    "vec_v4_emisor_prueba:$clave_emisor:emisor" \
    "vec_v4_ejecucion_prueba:$clave_ejecucion:ejecutor"
do
    IFS=: read -r usuario clave etiqueta <<<"$identidad"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT public.hmac(decode('00','hex'),decode('00','hex'),'sha256')" \
        "$etiqueta ejecuto HMAC directamente"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT * FROM vec_ejecucion_documental_v4.atestacion_pdp" \
        "$etiqueta leyo evidencia V4"
done

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
    (cd "$raiz" && go test \
        ./internal/vec/adapters/postgres/confianzadocumental \
        -run '^TestIntegracionEjecucionDocumentalV4PostgreSQLReal$' -count=1)
fi

aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000002_registro_autoridad_objeto_esperado_v1.up.sql
prueba_autoridad=deploy/postgresql/ejecucion_documental_v4/pruebas_sql/registro_autoridad_objeto_esperado_v1.sql

if [[ "${VEC_POSTGRES_PRUEBA_SOLO_SQL:-0}" == "1" ]]; then
    aplicar "$prueba_autoridad"
else
    psql_admin --set exigir_autoridad=1 --set solo_registro=1 \
        --set retener_registro=1 --file="$raiz/$prueba_autoridad" &
    pid_registro_uno=$!
    registro_retenido=false
    for _ in $(seq 1 50); do
        estado=$(psql_admin --tuples-only --no-align --command \
            "SELECT count(*) FROM pg_catalog.pg_stat_activity WHERE pid <> pg_backend_pid() AND state = 'active' AND query LIKE 'SELECT pg_sleep(3)%'")
        if [[ "$estado" == "1" ]]; then
            registro_retenido=true
            break
        fi
        sleep 0.1
    done
    if [[ "$registro_retenido" != true ]]; then
        wait "$pid_registro_uno" || true
        echo "no se observo el registro de autoridad retenido" >&2
        exit 1
    fi
    psql_admin --set exigir_autoridad=1 --set solo_registro=1 \
        --file="$raiz/$prueba_autoridad"
    wait "$pid_registro_uno"
    psql_admin --set exigir_autoridad=1 --file="$raiz/$prueba_autoridad"
fi

# Ninguna identidad existente puede fabricar el nuevo registro ni leerlo.
for identidad in \
    "vec_v4_fuente_prueba:$clave_fuente:fuente" \
    "vec_v4_registro_prueba:$clave_registro:registro" \
    "vec_v4_emisor_prueba:$clave_emisor:emisor" \
    "vec_v4_ejecucion_prueba:$clave_ejecucion:ejecutor"
do
    IFS=: read -r usuario clave etiqueta <<<"$identidad"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT * FROM vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(0,convert_to('{}','UTF8'))" \
        "$etiqueta fabrico una autoridad de objeto"
    exigir_rechazo_login "$usuario" "$clave" \
        "SELECT * FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1" \
        "$etiqueta leyo el registro de autoridad"
done

historia=$(psql_admin --tuples-only --no-align --command \
    "SELECT count(*) FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1")
down_autoridad=deploy/postgresql/ejecucion_documental_v4/migraciones/000002_registro_autoridad_objeto_esperado_v1.down.sql
if [[ "$historia" != "0" ]]; then
    if aplicar "$down_autoridad" >/dev/null 2>&1; then
        echo "el down de autoridad destruyo historia sin limpieza" >&2
        exit 1
    fi
    docker exec --interactive \
        --env PGOPTIONS="-c vec.limpiar_registro_autoridad_objeto_esperado_v1_prueba=LIMPIAR_REGISTRO_AUTORIDAD_OBJETO_ESPERADO_V1_PRUEBA" \
        "$contenedor" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" < "$raiz/$down_autoridad"
else
    aplicar "$down_autoridad"
fi

# Reinstalacion limpia y retirada final sin historia.
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones/000002_registro_autoridad_objeto_esperado_v1.up.sql
psql_admin <<'SQL'
DO $reinstalada$
BEGIN
    IF to_regprocedure(
        'vec_ejecucion_documental_v4.registrar_autoridad_objeto_esperado_v1(numeric,bytea)'
    ) IS NULL OR EXISTS (
        SELECT 1
          FROM vec_ejecucion_documental_v4.registro_autoridad_objeto_esperado_v1
    ) THEN
        RAISE EXCEPTION 'la reinstalacion no quedo limpia';
    END IF;
END
$reinstalada$;
SQL
aplicar "$down_autoridad"

docker exec --interactive \
    --env PGOPTIONS="-c vec.confirmar_destruccion_ejecucion_documental_v4=DESTRUIR_EVIDENCIA_V4_IRREVERSIBLE" \
    "$contenedor" psql --no-psqlrc --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" \
    < "$raiz/deploy/postgresql/ejecucion_documental_v4/migraciones/000001_ejecucion_documental_v4.down.sql"
aplicar deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000003_revalidacion_ejecucion_documental_v4.down.sql

psql_admin <<'SQL'
REVOKE vec_autorizacion_fuente FROM vec_v4_fuente_prueba;
REVOKE vec_autorizacion_registro FROM vec_v4_registro_prueba;
REVOKE vec_ejecucion_documental_v4_emisor_capacidad
    FROM vec_v4_emisor_prueba;
REVOKE vec_ejecucion_documental_v4_ejecutor_atestado
    FROM vec_v4_ejecucion_prueba;
DROP ROLE vec_v4_fuente_prueba;
DROP ROLE vec_v4_registro_prueba;
DROP ROLE vec_v4_emisor_prueba;
DROP ROLE vec_v4_ejecucion_prueba;
SQL
aplicar deploy/postgresql/ejecucion_documental_v4/roles_down.sql

psql_admin <<'SQL'
DO $retirada_final$
BEGIN
    IF to_regnamespace('vec_ejecucion_documental_v4') IS NOT NULL
       OR to_regnamespace('vec_ejecucion_documental_v4_guardia') IS NOT NULL
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname LIKE 'vec_ejecucion_documental_v4_%'
       ) THEN
        RAISE EXCEPTION 'la retirada final V4 dejo residuos';
    END IF;
END
$retirada_final$;
SQL

echo "integracion autoridad objeto esperado V1/PostgreSQL 18.4: correcta"
