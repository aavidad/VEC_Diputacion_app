#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C TZ=UTC

raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
cd "$raiz"

imagen=${VEC_POSTGRES_TEST_IMAGE:-postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}
contenedor="vec-vinculo-corporativo-pgx-${USER:-usuario}-$$"
base=c22b_vinculo_corporativo_pgx
clave=$(od -An -N24 -tx1 /dev/urandom | tr -d '[:space:]')
directorio=deploy/postgresql/contexto_actor_v1
roles="$directorio/roles_up.sql"
roles_selector="$directorio/roles_contexto_corporativo_rrhh_selector_v1_up.sql"
up_1="$directorio/migraciones/000001_contexto_actor_v1.up.sql"
up_2="$directorio/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql"
up_3="$directorio/migraciones/000003_organizacion_corporativa_v1.up.sql"
up_4="$directorio/migraciones/000004_vinculo_corporativo_rrhh_v1.up.sql"

limpiar() {
  local estado=$? proceso
  trap - EXIT INT TERM
  while IFS= read -r proceso; do
    kill "$proceso" >/dev/null 2>&1 || true
    wait "$proceso" >/dev/null 2>&1 || true
  done < <(jobs -pr)
  docker rm -f "$contenedor" >/dev/null 2>&1 || true
  if docker inspect "$contenedor" >/dev/null 2>&1; then
    echo 'quedo un contenedor PostgreSQL residual de B3' >&2
    estado=1
  fi
  exit "$estado"
}
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fallar() {
  echo "$1" >&2
  docker logs --tail 200 "$contenedor" >&2 2>/dev/null || true
  exit 1
}

esperar_postgres() {
  local consecutivas=0 respuesta
  for _ in $(seq 1 240); do
    if respuesta=$(docker exec --env PGPASSWORD="$clave" "$contenedor" \
      psql -XAt -h 127.0.0.1 -U postgres -d "$base" \
      -c "SELECT current_setting('server_version_num')||'|'||pg_is_in_recovery()" \
      2>/dev/null) && [[ $respuesta == '180004|false' ]]; then
      consecutivas=$((consecutivas + 1))
      [[ $consecutivas -eq 3 ]] && return 0
    else
      consecutivas=0
    fi
    sleep 0.25
  done
  fallar 'PostgreSQL 18.4 primario no quedo disponible tres veces consecutivas'
}

psql_archivo() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 \
    < "$raiz/$1"
}

psql_sql() {
  docker exec --interactive --env PGPASSWORD="$clave" "$contenedor" \
    psql -Xq -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1
}

consulta() {
  docker exec --env PGPASSWORD="$clave" "$contenedor" \
    psql -XAt -h 127.0.0.1 -U postgres -d "$base" -v ON_ERROR_STOP=1 -c "$1"
}

docker run -d --rm --name "$contenedor" --publish 127.0.0.1::5432 \
  -e POSTGRES_DB="$base" -e POSTGRES_PASSWORD="$clave" "$imagen" >/dev/null
esperar_postgres

psql_sql <<'SQL'
DO $base$
BEGIN
  CREATE ROLE c22b_pgx_dueno_base NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOINHERIT NOREPLICATION NOBYPASSRLS;
  EXECUTE format('ALTER DATABASE %I OWNER TO c22b_pgx_dueno_base',current_database());
  EXECUTE format('REVOKE ALL ON DATABASE %I FROM PUBLIC',current_database());
END
$base$;
REVOKE ALL ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
SQL
psql_archivo "$roles"
psql_archivo "$up_1"
psql_archivo "$up_2"
psql_archivo "$roles_selector"
psql_archivo "$up_3"
psql_archivo "$up_4"

puerto=$(docker port "$contenedor" 5432/tcp | head -n1)
puerto=${puerto##*:}
[[ $puerto =~ ^[0-9]+$ ]] || fallar 'no se pudo resolver el puerto TCP efimero'
dsn="postgres://postgres:${clave}@127.0.0.1:${puerto}/${base}?sslmode=disable"

VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn" \
  go test ./internal/vec/adapters/contextoactor/postgres \
    -run '^TestRetiradaVinculoCorporativoRRHHV1' -count=1
VEC_CONTEXTO_ACTOR_MIGRACION_POSTGRES_DSN="$dsn" \
  go test -race ./internal/vec/adapters/contextoactor/postgres \
    -run '^TestRetiradaVinculoCorporativoRRHHV1' -count=1

[[ $(consulta "SELECT current_setting('server_version_num')||'|'||pg_is_in_recovery()") == '180004|false' ]] ||
  fallar 'PostgreSQL dejo de ser el primario 18.4 esperado'
[[ $(consulta "SELECT count(*) FROM pg_stat_activity
 WHERE datname=current_database() AND pid<>pg_backend_pid()") == 0 ]] ||
  fallar 'quedaron conexiones PostgreSQL residuales tras B3'
[[ -z $(jobs -pr) ]] || fallar 'quedaron procesos en segundo plano tras B3'

echo 'ContextoActor: retirada literal pgx del vinculo corporativo RRHH V1 superada en PostgreSQL 18.4'
