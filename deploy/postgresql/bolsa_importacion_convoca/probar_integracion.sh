#!/usr/bin/env bash
set -euo pipefail

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-bolsa-convoca-b1-${USER:-usuario}-$$"
base=vec_bolsa_convoca_b1_prueba
directorio_tls=$(mktemp -d /tmp/vec-bolsa-convoca-b1-tls-XXXXXX)
cache_go=$(mktemp -d /tmp/vec-bolsa-convoca-b1-go-XXXXXX)

generar_clave() {
    local destino=$1 valor
    valor=$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')
    if [[ ${#valor} -ne 64 || $valor == *[!0-9a-f]* ]]; then
        echo "no se pudo generar una clave efimera de prueba" >&2
        exit 1
    fi
    printf -v "$destino" '%s' "$valor"
}

clave_admin=
clave_ejecutor=
clave_recuperador=
clave_conciliador=
clave_retencion=
clave_gobernanza=
generar_clave clave_admin
generar_clave clave_ejecutor
generar_clave clave_recuperador
generar_clave clave_conciliador
generar_clave clave_retencion
generar_clave clave_gobernanza

limpiar() {
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    rm -rf "$directorio_tls" "$cache_go"
}
trap limpiar EXIT INT TERM

openssl req -x509 -newkey rsa:3072 -sha256 -nodes -days 1 \
    -subj '/CN=CA integracion VEC Bolsa Convoca B1' \
    -keyout "$directorio_tls/ca.key" \
    -out "$directorio_tls/ca.crt" >/dev/null 2>&1
openssl req -newkey rsa:3072 -nodes -sha256 -subj '/CN=localhost' \
    -addext 'subjectAltName=DNS:localhost,IP:127.0.0.1' \
    -addext 'extendedKeyUsage=serverAuth' \
    -addext 'keyUsage=digitalSignature,keyEncipherment' \
    -keyout "$directorio_tls/servidor.key" \
    -out "$directorio_tls/servidor.csr" >/dev/null 2>&1
openssl x509 -req -sha256 -days 1 \
    -in "$directorio_tls/servidor.csr" \
    -CA "$directorio_tls/ca.crt" \
    -CAkey "$directorio_tls/ca.key" \
    -CAcreateserial \
    -copy_extensions copy \
    -out "$directorio_tls/servidor.crt" >/dev/null 2>&1

docker run --detach --rm --name "$contenedor" \
    --publish 127.0.0.1::5432 \
    --env POSTGRES_DB="$base" \
    --env POSTGRES_PASSWORD="$clave_admin" \
    "$imagen" >/dev/null

esperar_postgres() {
    for _ in $(seq 1 60); do
        if docker exec "$contenedor" psql -X --tuples-only --no-align \
            --username postgres --dbname "$base" \
            --command 'SELECT 1' >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    docker exec "$contenedor" psql -X --tuples-only --no-align \
        --username postgres --dbname "$base" --command 'SELECT 1' >/dev/null
}
esperar_postgres

configuracion=$(docker exec "$contenedor" psql -X --tuples-only --no-align \
    --username postgres --dbname "$base" --command 'SHOW config_file')
docker cp "$directorio_tls/servidor.crt" "$contenedor:/tmp/vec-convoca-server.crt"
docker cp "$directorio_tls/servidor.key" "$contenedor:/tmp/vec-convoca-server.key"
docker cp "$directorio_tls/ca.crt" "$contenedor:/tmp/vec-convoca-ca.crt"
docker exec --user root --env VEC_CONFIGURACION="$configuracion" \
    "$contenedor" sh -ceu '
    chown postgres:postgres /tmp/vec-convoca-server.crt \
        /tmp/vec-convoca-server.key /tmp/vec-convoca-ca.crt
    chmod 600 /tmp/vec-convoca-server.key
    chmod 644 /tmp/vec-convoca-server.crt /tmp/vec-convoca-ca.crt
    printf "%s\n" \
      "ssl = on" \
      "ssl_cert_file = '\''/tmp/vec-convoca-server.crt'\''" \
      "ssl_key_file = '\''/tmp/vec-convoca-server.key'\''" \
      "ssl_ca_file = '\''/tmp/vec-convoca-ca.crt'\''" \
      "ssl_min_protocol_version = '\''TLSv1.2'\''" \
      >> "$VEC_CONFIGURACION"
'
docker restart "$contenedor" >/dev/null
esperar_postgres

psql_admin() {
    docker exec --interactive "$contenedor" psql -X --quiet \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname "$base"
}

psql_archivo() {
    psql_admin < "$raiz/$1"
}

docker exec "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command \
    "REVOKE CONNECT, TEMPORARY ON DATABASE ${base} FROM PUBLIC;
     REVOKE CREATE ON SCHEMA public FROM PUBLIC;
     REVOKE ALL PRIVILEGES ON DATABASE postgres, template1 FROM PUBLIC"

psql_archivo deploy/postgresql/bolsa_importacion_convoca/roles_up.sql
for migracion in \
    "$raiz"/deploy/postgresql/bolsa_importacion_convoca/migraciones/*.up.sql
do
    {
        printf '%s\n' 'SET ROLE vec_bolsa_importacion_convoca_migrador;'
        sed -n '1,$p' "$migracion"
    } | psql_admin
done

docker exec --interactive \
    --env CLAVE_EJECUTOR="$clave_ejecutor" \
    --env CLAVE_RECUPERADOR="$clave_recuperador" \
    --env CLAVE_CONCILIADOR="$clave_conciliador" \
    --env CLAVE_RETENCION="$clave_retencion" \
    --env CLAVE_GOBERNANZA="$clave_gobernanza" \
    "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" <<'SQL'
\getenv clave_ejecutor CLAVE_EJECUTOR
\getenv clave_recuperador CLAVE_RECUPERADOR
\getenv clave_conciliador CLAVE_CONCILIADOR
\getenv clave_retencion CLAVE_RETENCION
\getenv clave_gobernanza CLAVE_GOBERNANZA
CREATE ROLE vec_convoca_ejecutor_prueba LOGIN PASSWORD :'clave_ejecutor'
    NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_convoca_conciliador_prueba LOGIN PASSWORD :'clave_conciliador'
    NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_convoca_retencion_prueba LOGIN PASSWORD :'clave_retencion'
    NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_convoca_recuperador_prueba LOGIN PASSWORD :'clave_recuperador'
    NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE vec_convoca_gobernanza_prueba LOGIN PASSWORD :'clave_gobernanza'
    NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_bolsa_importacion_convoca_ejecutor TO vec_convoca_ejecutor_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_bolsa_importacion_convoca_conciliador TO vec_convoca_conciliador_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_bolsa_importacion_convoca_retencion TO vec_convoca_retencion_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_bolsa_importacion_convoca_recuperador TO vec_convoca_recuperador_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT vec_bolsa_importacion_convoca_gobernanza TO vec_convoca_gobernanza_prueba
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
ALTER ROLE vec_convoca_ejecutor_prueba
    SET application_name = 'vec-bolsa-convoca-importacion';
ALTER ROLE vec_convoca_conciliador_prueba
    SET application_name = 'vec-bolsa-convoca-conciliacion';
ALTER ROLE vec_convoca_retencion_prueba
    SET application_name = 'vec-bolsa-convoca-retencion';
ALTER ROLE vec_convoca_recuperador_prueba
    SET application_name = 'vec-bolsa-convoca-recuperacion-vec-t13';
ALTER ROLE vec_convoca_gobernanza_prueba
    SET application_name = 'vec-bolsa-convoca-gobernanza-retencion';
ALTER ROLE vec_convoca_ejecutor_prueba SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_convoca_conciliador_prueba SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_convoca_retencion_prueba SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_convoca_recuperador_prueba SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_convoca_gobernanza_prueba SET search_path = 'pg_catalog,pg_temp';
ALTER ROLE vec_convoca_ejecutor_prueba SET statement_timeout = '45s';
ALTER ROLE vec_convoca_conciliador_prueba SET statement_timeout = '45s';
ALTER ROLE vec_convoca_retencion_prueba SET statement_timeout = '45s';
ALTER ROLE vec_convoca_recuperador_prueba SET statement_timeout = '45s';
ALTER ROLE vec_convoca_gobernanza_prueba SET statement_timeout = '45s';
ALTER ROLE vec_convoca_ejecutor_prueba SET lock_timeout = '3s';
ALTER ROLE vec_convoca_conciliador_prueba SET lock_timeout = '3s';
ALTER ROLE vec_convoca_retencion_prueba SET lock_timeout = '3s';
ALTER ROLE vec_convoca_recuperador_prueba SET lock_timeout = '3s';
ALTER ROLE vec_convoca_gobernanza_prueba SET lock_timeout = '3s';
ALTER ROLE vec_convoca_ejecutor_prueba SET idle_in_transaction_session_timeout = '20s';
ALTER ROLE vec_convoca_conciliador_prueba SET idle_in_transaction_session_timeout = '20s';
ALTER ROLE vec_convoca_retencion_prueba SET idle_in_transaction_session_timeout = '20s';
ALTER ROLE vec_convoca_recuperador_prueba SET idle_in_transaction_session_timeout = '20s';
ALTER ROLE vec_convoca_gobernanza_prueba SET idle_in_transaction_session_timeout = '20s';
ALTER ROLE vec_convoca_ejecutor_prueba SET transaction_timeout = '60s';
ALTER ROLE vec_convoca_conciliador_prueba SET transaction_timeout = '60s';
ALTER ROLE vec_convoca_retencion_prueba SET transaction_timeout = '60s';
ALTER ROLE vec_convoca_recuperador_prueba SET transaction_timeout = '60s';
ALTER ROLE vec_convoca_gobernanza_prueba SET transaction_timeout = '60s';
ALTER ROLE vec_convoca_ejecutor_prueba SET log_parameter_max_length_on_error = 0;
ALTER ROLE vec_convoca_conciliador_prueba SET log_parameter_max_length_on_error = 0;
ALTER ROLE vec_convoca_retencion_prueba SET log_parameter_max_length_on_error = 0;
ALTER ROLE vec_convoca_recuperador_prueba SET log_parameter_max_length_on_error = 0;
ALTER ROLE vec_convoca_gobernanza_prueba SET log_parameter_max_length_on_error = 0;
SQL

if docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
SET ROLE vec_convoca_gobernanza_prueba;
SELECT vec_bolsa_importacion_convoca.publicar_politica_retencion_v1(
    'politica:retencion:convoca:version-inicial-invalida',
    999, 3600, 'actor:gobernanza:integracion'
);
SQL
then
    echo "politica Convoca acepto una primera version distinta de 1" >&2
    exit 1
fi
residuos_politica_invalida=$(docker exec "$contenedor" psql -X --quiet \
    --tuples-only --no-align --set ON_ERROR_STOP=1 \
    --username postgres --dbname "$base" --command "
SELECT count(*)
  FROM vec_bolsa_importacion_convoca.politica_retencion
 WHERE politica_retencion_ref =
       'politica:retencion:convoca:version-inicial-invalida'")
if [[ "$residuos_politica_invalida" != "0" ]]; then
    echo "rechazo de politica inicial Convoca dejo residuos" >&2
    exit 1
fi

esperar_puerto_publicado() {
    for _ in $(seq 1 60); do
        if timeout 1 bash -c \
            "exec 3<>/dev/tcp/127.0.0.1/${puerto}; exec 3>&-; exec 3<&-" \
            >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    timeout 1 bash -c \
        "exec 3<>/dev/tcp/127.0.0.1/${puerto}; exec 3>&-; exec 3<&-"
}
exportar_dsns() {
    puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*://p' | head -1)
    esperar_puerto_publicado
    export VEC_PRUEBA_BOLSA_CONVOCA_EJECUTOR_DSN="postgresql://vec_convoca_ejecutor_prueba:${clave_ejecutor}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
    export VEC_PRUEBA_BOLSA_CONVOCA_RECUPERADOR_DSN="postgresql://vec_convoca_recuperador_prueba:${clave_recuperador}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
    export VEC_PRUEBA_BOLSA_CONVOCA_CONCILIADOR_DSN="postgresql://vec_convoca_conciliador_prueba:${clave_conciliador}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
    export VEC_PRUEBA_BOLSA_CONVOCA_RETENCION_DSN="postgresql://vec_convoca_retencion_prueba:${clave_retencion}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
    export VEC_PRUEBA_BOLSA_CONVOCA_GOBERNANZA_DSN="postgresql://vec_convoca_gobernanza_prueba:${clave_gobernanza}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
    export VEC_PRUEBA_BOLSA_CONVOCA_ADMIN_DSN="postgresql://postgres:${clave_admin}@localhost:${puerto}/${base}?sslmode=verify-full&sslrootcert=${directorio_tls}/ca.crt"
}
exportar_dsns
export GOCACHE="$cache_go"

(
    cd "$raiz"
    if [[ -n ${VEC_PRUEBA_BOLSA_CONVOCA_RUN:-} ]]; then
        go test -count=1 -run "$VEC_PRUEBA_BOLSA_CONVOCA_RUN" \
            ./internal/modules/bolsa/adapters/postgresimportacionconvoca
    else
        go test -count=1 ./internal/modules/bolsa/adapters/postgresimportacionconvoca
    fi
)

if [[ -n ${VEC_PRUEBA_BOLSA_CONVOCA_RUN:-} ]]; then
    echo "integracion PostgreSQL 18/TLS seleccionada: OK"
    exit 0
fi

docker restart "$contenedor" >/dev/null
esperar_postgres
exportar_dsns
export VEC_PRUEBA_BOLSA_CONVOCA_TRAS_REINICIO=1
(
    cd "$raiz"
    go test -count=1 -run '^TestRecuperacionPostgreSQLTrasReinicio$' \
        ./internal/modules/bolsa/adapters/postgresimportacionconvoca
)

if psql_archivo \
    deploy/postgresql/bolsa_importacion_convoca/migraciones/000001_importacion_convoca_durable.down.sql \
    >/dev/null 2>&1; then
    echo "down destruyo historia sin confirmacion" >&2
    exit 1
fi
if psql_archivo deploy/postgresql/bolsa_importacion_convoca/roles_down.sql \
    >/dev/null 2>&1; then
    echo "roles_down acepto esquema o identidades vivas" >&2
    exit 1
fi

docker exec --interactive "$contenedor" psql -X --quiet \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" <<'SQL'
REVOKE vec_bolsa_importacion_convoca_ejecutor FROM vec_convoca_ejecutor_prueba;
REVOKE vec_bolsa_importacion_convoca_recuperador FROM vec_convoca_recuperador_prueba;
REVOKE vec_bolsa_importacion_convoca_conciliador FROM vec_convoca_conciliador_prueba;
REVOKE vec_bolsa_importacion_convoca_retencion FROM vec_convoca_retencion_prueba;
REVOKE vec_bolsa_importacion_convoca_gobernanza FROM vec_convoca_gobernanza_prueba;
DROP ROLE vec_convoca_ejecutor_prueba;
DROP ROLE vec_convoca_recuperador_prueba;
DROP ROLE vec_convoca_conciliador_prueba;
DROP ROLE vec_convoca_retencion_prueba;
DROP ROLE vec_convoca_gobernanza_prueba;
SQL

while IFS= read -r migracion
do
    {
        printf '%s\n' 'SET ROLE vec_bolsa_importacion_convoca_migrador;'
        sed -n '1,$p' "$migracion"
    } | docker exec --interactive \
        --env PGOPTIONS='-c vec.confirmar_destruccion_bolsa_importacion_convoca=DESTRUIR_HISTORIA_IMPORTACION_CONVOCA_IRREVERSIBLE' \
        "$contenedor" psql -X --quiet --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base"
done < <(
    find "$raiz/deploy/postgresql/bolsa_importacion_convoca/migraciones" \
        -maxdepth 1 -type f -name '*.down.sql' | sort --reverse
)

psql_archivo deploy/postgresql/bolsa_importacion_convoca/roles_down.sql

residuos=$(docker exec "$contenedor" psql -X --quiet --tuples-only --no-align \
    --set ON_ERROR_STOP=1 --username postgres --dbname "$base" --command "
SELECT
    (SELECT count(*) FROM pg_catalog.pg_namespace
      WHERE nspname = 'vec_bolsa_importacion_convoca')
  + (SELECT count(*) FROM pg_catalog.pg_roles
      WHERE rolname LIKE 'vec_bolsa_importacion_convoca_%')")
if [[ "$residuos" != "0" ]]; then
    echo "reversion dejo objetos de importacion Convoca" >&2
    exit 1
fi

echo "integracion PostgreSQL 18/TLS de importacion Convoca B1: OK"
