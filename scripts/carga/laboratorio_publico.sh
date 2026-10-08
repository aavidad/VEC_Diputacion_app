#!/usr/bin/env bash
# Laboratorio local de carga del portal público de Bolsa.
#
# Levanta un PostgreSQL 18.4 desechable (memoria 2 GB, datos en disco,
# max_connections=100 como la principal), instala la proyección pública
# (roles_up + 000001 + 000002), publica el escenario sintético de las pruebas
# de integración más un volumen realista de bolsas con miles de posiciones
# enmascaradas y arranca cmd/vec-publico con TLS en 127.0.0.1.
#
# Solo datos sintéticos. Nunca contacta con la principal ni con servicios
# externos. Todo el material (TLS, claves, binario) vive en el directorio de
# estado, fuera de Git, y se borra con «retirar».
#
# Uso:
#   scripts/carga/laboratorio_publico.sh preparar   # PG + datos + binario + servidor
#   scripts/carga/laboratorio_publico.sh servidor   # (re)arranca solo vec-publico
#   scripts/carga/laboratorio_publico.sh retirar    # para todo y borra el estado
#
# Variables: VEC_CARGA_ESTADO (directorio privado), VEC_CARGA_PUERTO_PG,
# VEC_CARGA_PUERTO_WEB, VEC_CARGA_BOLSAS (número de bolsas, máx. 128),
# VEC_CARGA_POSICIONES_MAX (posiciones de la bolsa más grande).
set -euo pipefail
umask 077

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
estado=${VEC_CARGA_ESTADO:-$HOME/.cache/vec-carga-publico}
contenedor=vec-carga-publico-pg
puerto_pg=${VEC_CARGA_PUERTO_PG:-55480}
puerto_web=${VEC_CARGA_PUERTO_WEB:-18480}
num_bolsas=${VEC_CARGA_BOLSAS:-40}
posiciones_max=${VEC_CARGA_POSICIONES_MAX:-4000}
base=vec_bolsa_publica_carga
accion=${1:-preparar}

if ! [[ "$num_bolsas" =~ ^[0-9]+$ ]] || ((num_bolsas < 1 || num_bolsas > 128)); then
    echo 'VEC_CARGA_BOLSAS entre 1 y 128' >&2; exit 2
fi
if ! [[ "$posiciones_max" =~ ^[0-9]+$ ]] || ((posiciones_max < 10 || posiciones_max > 100000)); then
    echo 'VEC_CARGA_POSICIONES_MAX entre 10 y 100000' >&2; exit 2
fi

psql_admin() {
    docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 -q \
        --username postgres --dbname "$base" "$@"
}

esperar_pg() {
    local seguidas=0
    for _ in $(seq 1 240); do
        if docker exec "$contenedor" psql -XAt -h 127.0.0.1 -U postgres -d "$base" \
            -c 'SELECT NOT pg_is_in_recovery()' 2>/dev/null | grep -qx t; then
            seguidas=$((seguidas + 1)); [[ $seguidas -eq 3 ]] && return 0
        else
            seguidas=0
        fi
        sleep 0.25
    done
    echo 'PostgreSQL no quedó disponible' >&2; return 1
}

generar_tls() {
    mkdir -p "$estado/tls"
    local d="$estado/tls"
    openssl req -x509 -newkey rsa:3072 -sha256 -nodes -days 2 -subj '/CN=CA laboratorio carga VEC' \
        -keyout "$d/ca.key" -out "$d/ca.crt" >/dev/null 2>&1
    printf '%s\n' 'subjectAltName=DNS:localhost,IP:127.0.0.1' 'extendedKeyUsage=serverAuth' \
        'keyUsage=digitalSignature,keyEncipherment' >"$d/servidor.ext"
    openssl req -newkey rsa:3072 -nodes -sha256 -subj '/CN=localhost' \
        -keyout "$d/servidor.key" -out "$d/servidor.csr" >/dev/null 2>&1
    openssl x509 -req -sha256 -days 2 -in "$d/servidor.csr" -CA "$d/ca.crt" -CAkey "$d/ca.key" \
        -CAcreateserial -extfile "$d/servidor.ext" -out "$d/servidor.crt" >/dev/null 2>&1
    chmod 644 "$d/ca.crt" "$d/servidor.crt"
}

preparar_pg() {
    local clave_admin clave_lector clave_publicador
    clave_admin=$(openssl rand -hex 24)
    clave_lector=$(openssl rand -hex 24)
    clave_publicador=$(openssl rand -hex 24)
    docker rm -f "$contenedor" >/dev/null 2>&1 || true
    # Datos en un volumen anónimo del disco de Docker (no /dev/shm); --rm lo borra.
    docker run --detach --rm --name "$contenedor" --memory 2g --cpus 4 \
        --publish "127.0.0.1:${puerto_pg}:5432" \
        --env POSTGRES_DB="$base" --env POSTGRES_PASSWORD="$clave_admin" \
        postgres:18.4 -c max_connections=100 -c shared_buffers=512MB \
        -c track_io_timing=on -c shared_preload_libraries=pg_stat_statements -c pg_stat_statements.track=all >/dev/null
    esperar_pg
    local conf
    conf=$(docker exec "$contenedor" psql -XAt -U postgres -d "$base" -c 'SHOW config_file')
    docker cp "$estado/tls/servidor.crt" "$contenedor:/tmp/srv.crt"
    docker cp "$estado/tls/servidor.key" "$contenedor:/tmp/srv.key"
    docker exec --user root --env CONF="$conf" "$contenedor" sh -ceu '
        chown postgres:postgres /tmp/srv.crt /tmp/srv.key; chmod 600 /tmp/srv.key
        printf "%s\n" "ssl = on" "ssl_cert_file = '\''/tmp/srv.crt'\''" "ssl_key_file = '\''/tmp/srv.key'\''" >> "$CONF"'
    docker restart "$contenedor" >/dev/null
    esperar_pg
    psql_admin -c "REVOKE CONNECT, TEMPORARY ON DATABASE ${base} FROM PUBLIC" \
        -c 'REVOKE CREATE ON SCHEMA public FROM PUBLIC' \
        -c 'REVOKE ALL PRIVILEGES ON DATABASE postgres, template1 FROM PUBLIC'
    psql_admin <"$raiz/deploy/postgresql/bolsa_publica/roles_up.sql" >/dev/null
    {
        printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'
        cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000001_proyeccion_publica.up.sql"
        cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000002_proyeccion_bolsas_v1.up.sql"
    } | psql_admin >/dev/null
    # Escenario de convocatorias: el mismo JSON sintético y ancla que usa la
    # prueba de integración de la proyección (huellas ya calculadas).
    # shellcheck disable=SC2016 # los $ son literales del fichero de pruebas
    sed -n '/^\$proyeccion\$$/,/^\$proyeccion\$::jsonb$/p' \
        "$raiz/deploy/postgresql/bolsa_publica/probar_integracion.sh" \
        | sed '1d;$d' >"$estado/proyeccion.json"
    docker cp "$estado/proyeccion.json" "$contenedor:/tmp/proyeccion.json"
    docker exec --user root "$contenedor" chown postgres /tmp/proyeccion.json
    docker exec --interactive --env CL="$clave_lector" --env CP="$clave_publicador" "$contenedor" \
        psql -X -q --set ON_ERROR_STOP=1 -U postgres -d "$base" \
        -v bolsas="$num_bolsas" -v pmax="$posiciones_max" <<'SQL' >/dev/null
\getenv cl CL
\getenv cp CP
CREATE ROLE vec_bolsa_publica_carga_login LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS PASSWORD :'cl';
GRANT vec_bolsa_publica_consulta TO vec_bolsa_publica_carga_login WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
ALTER ROLE vec_bolsa_publica_publicador_login PASSWORD :'cp';
CREATE SCHEMA laboratorio;
CREATE TABLE laboratorio.entrada(proyeccion jsonb, bolsas jsonb);
GRANT USAGE ON SCHEMA laboratorio TO vec_bolsa_publica_publicador_login;
INSERT INTO laboratorio.entrada(proyeccion) SELECT pg_read_file('/tmp/proyeccion.json')::jsonb;
-- Bolsas sintéticas: tamaños repartidos entre pmax/20 y pmax, documentos
-- enmascarados con el formato público y situaciones mezcladas.
UPDATE laboratorio.entrada SET bolsas = jsonb_build_object(
  'generado_en', proyeccion#>>'{fuente,actualizada_en}',
  'bolsas', (
    SELECT jsonb_agg(b ORDER BY b->>'bolsa_ref') FROM (
      SELECT jsonb_build_object(
        'bolsa_ref', format('bolsa:carga-%s', lpad(i::text, 3, '0')),
        'categoria', format('Categoría sintética %s', i),
        'categoria_clave', format('categoria-carga-%s', i),
        'grupos', jsonb_build_array((ARRAY['A1','A2','C1','C2','E'])[1 + i % 5]),
        'tipo_lista', 'definitiva',
        'vigente_desde', '2026-01-01T00:00:00Z',
        'vigente_hasta', NULL,
        'total', n,
        'posiciones', (
          SELECT jsonb_agg(jsonb_build_object(
                   'orden', o,
                   'documento_enmascarado', format('***%s**', lpad(((o * 7919 + i * 104729) % 10000)::text, 4, '0')),
                   'estado_clave', (ARRAY['disponible','disponible','disponible','ocupado','no_disponible','excluido'])[1 + (o + i) % 6])
                 ORDER BY o)
            FROM generate_series(1, n) AS o)
      ) AS b
      FROM (SELECT i, greatest(10, (:pmax::int * (1 + (i * 37) % 20)) / 20) AS n
              FROM generate_series(1, :bolsas::int) AS i) AS t
    ) AS s));
GRANT SELECT ON laboratorio.entrada TO vec_bolsa_publica_publicador_login;
-- Límites gobernados que exige la función de publicación al LOGIN publicador.
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v3(proyeccion, bolsas, repeat('c', 64)) FROM laboratorio.entrada;
RESET SESSION AUTHORIZATION;
SQL
    docker exec "$contenedor" rm -f /tmp/proyeccion.json
    printf 'postgres://vec_bolsa_publica_carga_login:%s@localhost:%s/%s?sslmode=verify-full&sslrootcert=%s\n' \
        "$clave_lector" "$puerto_pg" "$base" "$estado/tls/ca.crt" >"$estado/dsn_lector"
    printf 'postgres://postgres:%s@localhost:%s/%s?sslmode=verify-full&sslrootcert=%s\n' \
        "$clave_admin" "$puerto_pg" "$base" "$estado/tls/ca.crt" >"$estado/dsn_admin"
    calcular_y_publicar_ancla
    docker exec "$contenedor" psql -XAt -U postgres -d "$base" -c \
        "SELECT 'bolsas=' || count(*) || ' posiciones=' || sum(total) FROM vec_bolsa_publica_datos.bolsa_publica"
}

# La huella del manifiesto V3 cubre también las cabeceras de las bolsas, así
# que depende del volumen generado: se publica primero con un ancla
# provisional, se calcula la huella con el propio adaptador y se republica.
calcular_y_publicar_ancla() {
    local paquete="$raiz/internal/modules/bolsa/adapters/postgrespublico"
    local ayuda="$paquete/zz_laboratorio_carga_huella_test.go"
    # La ayuda nunca debe quedarse en el paquete, ni siquiera si se interrumpe.
    trap 'rm -f "$ayuda"' EXIT
    cp "$raiz/scripts/carga/huella_manifiesto_laboratorio.go.txt" "$ayuda"
    local salida
    salida=$(cd "$raiz" && VEC_CARGA_DSN_LECTOR="$(cat "$estado/dsn_lector")" \
        go test -count=1 -run '^TestLaboratorioCargaHuellaManifiesto$' -v ./internal/modules/bolsa/adapters/postgrespublico/ 2>&1) || true
    rm -f "$ayuda"
    ancla=$(printf '%s\n' "$salida" | sed -n 's/^HUELLA_MANIFIESTO=\([0-9a-f]\{64\}\)$/\1/p')
    [[ -n "$ancla" ]] || { printf '%s\n' "$salida" | tail -20 >&2; echo 'no se pudo calcular la huella' >&2; return 1; }
    printf '%s\n' "$ancla" >"$estado/ancla"
    docker exec --interactive "$contenedor" psql -X -q --set ON_ERROR_STOP=1 -U postgres -d "$base" \
        -v ancla="$ancla" <<'SQL' >/dev/null
SET application_name = 'vec-bolsa-publicador';
SET search_path = 'pg_catalog,pg_temp';
SET statement_timeout = '60s';
SET lock_timeout = '5s';
SET idle_in_transaction_session_timeout = '5s';
SET transaction_timeout = '2min';
SET log_parameter_max_length_on_error = 0;
SET SESSION AUTHORIZATION vec_bolsa_publica_publicador_login;
SELECT vec_bolsa_publica_publicacion.publicar_proyeccion_v3(proyeccion, bolsas, :'ancla') FROM laboratorio.entrada;
RESET SESSION AUTHORIZATION;
SQL
    docker exec "$contenedor" psql -XAt -U postgres -d "$base" -c 'ANALYZE' \
        >/dev/null
    # En otra base: la cuenta lectora no debe ver objetos ajenos a la proyección.
    docker exec "$contenedor" psql -XAtq -U postgres -d postgres -c 'CREATE EXTENSION IF NOT EXISTS pg_stat_statements'
}

compilar() {
    (cd "$raiz" && GOFLAGS=-trimpath go build -o "$estado/vec-publico" ./cmd/vec-publico)
}

arrancar_servidor() {
    if [[ -f "$estado/servidor.pid" ]] && kill -0 "$(cat "$estado/servidor.pid")" 2>/dev/null; then
        kill "$(cat "$estado/servidor.pid")"; sleep 1
    fi
    (
        cd "$raiz"
        env -i PATH="$PATH" HOME="$HOME" \
            VEC_EXECUTION_PROFILE=produccion VEC_AUTH_MODE=disabled \
            VEC_HTTP_ADDR="127.0.0.1:${puerto_web}" \
            VEC_TLS_CERT_FILE="$estado/tls/servidor.crt" VEC_TLS_KEY_FILE="$estado/tls/servidor.key" \
            VEC_BOLSA_PUBLICA_DATABASE_URL="$(cat "$estado/dsn_lector")" \
            VEC_BOLSA_PUBLICA_MANIFIESTO_SHA256="$(cat "$estado/ancla")" \
            VEC_BOLSA_CATEGORIES_CATALOG_ID=categorias-profesionales \
            VEC_BOLSA_CATEGORIES_CATALOG_VERSION=2 \
            VEC_BOLSA_CATEGORIES_CATALOG_SHA256="$(printf 'b%.0s' $(seq 64))" \
            VEC_BOLSA_CATEGORIES_PUBLIC_PROJECTION_SHA256=b661b37ca7323fa168734899038f8fa99cb77ff07d114e4a7d787d62b5d36593 \
            GOMAXPROCS="${VEC_CARGA_GOMAXPROCS:-4}" \
            nohup "$estado/vec-publico" >"$estado/servidor.log" 2>&1 &
        echo $! >"$estado/servidor.pid"
    )
    for _ in $(seq 1 120); do
        if curl -fsS --cacert "$estado/tls/ca.crt" "https://localhost:${puerto_web}/readyz" >/dev/null 2>&1; then
            echo "vec-publico listo en https://localhost:${puerto_web} (pid $(cat "$estado/servidor.pid"))"
            return 0
        fi
        sleep 0.5
    done
    echo 'vec-publico no arrancó; últimas líneas del registro:' >&2
    tail -20 "$estado/servidor.log" >&2
    return 1
}

case "$accion" in
    preparar)
        mkdir -p "$estado"; chmod 700 "$estado"
        generar_tls
        preparar_pg
        compilar
        arrancar_servidor
        ;;
    servidor)
        compilar
        arrancar_servidor
        ;;
    retirar)
        if [[ -f "$estado/servidor.pid" ]]; then kill "$(cat "$estado/servidor.pid")" 2>/dev/null || true; fi
        docker rm -f "$contenedor" >/dev/null 2>&1 || true
        rm -rf -- "$estado"
        ;;
    *)
        echo 'Uso: laboratorio_publico.sh preparar|servidor|retirar' >&2; exit 2
        ;;
esac
