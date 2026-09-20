#!/usr/bin/env bash
set -euo pipefail
raiz=$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)
contenedor="vec-bolsa-publica-b10-${USER:-usuario}-$$"
limpiar() { docker rm -f "$contenedor" >/dev/null 2>&1 || true; }
trap limpiar EXIT INT TERM
docker run --detach --rm --name "$contenedor" --publish 127.0.0.1::5432 --env POSTGRES_PASSWORD=prueba --env POSTGRES_DB=vec_bolsa_publica_b10 postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296 >/dev/null
for _ in $(seq 1 60); do docker exec --env PGPASSWORD=prueba "$contenedor" pg_isready --host 127.0.0.1 --username postgres --dbname vec_bolsa_publica_b10 >/dev/null 2>&1 && break; sleep 1; done
docker exec "$contenedor" psql -X --set ON_ERROR_STOP=1 --username postgres --dbname vec_bolsa_publica_b10 --command 'REVOKE CONNECT, TEMPORARY, CREATE ON DATABASE vec_bolsa_publica_b10 FROM PUBLIC; REVOKE CREATE ON SCHEMA public FROM PUBLIC' >/dev/null
docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 --username postgres --dbname vec_bolsa_publica_b10 <"$raiz/deploy/postgresql/bolsa_publica/roles_up.sql" >/dev/null
for m in 000001_proyeccion_publica.up.sql 000002_proyeccion_bolsas_v1.up.sql; do { printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'; cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/$m"; } | docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 --username postgres --dbname vec_bolsa_publica_b10 >/dev/null; done
docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 --username postgres --dbname vec_bolsa_publica_b10 <"$raiz/deploy/postgresql/bolsa_publica/pruebas_sql/000002_proyeccion_bolsas_v1.sql" >/dev/null
puerto=$(docker port "$contenedor" 5432/tcp | sed -n 's/.*:\([0-9][0-9]*\)$/\1/p' | head -n 1)
if [[ ! "$puerto" =~ ^[0-9]+$ ]]; then
  echo 'no se pudo resolver el puerto efimero de PostgreSQL B10' >&2
  exit 1
fi
VEC_PRUEBA_BOLSA_PUBLICA_DSN="postgres://postgres:prueba@127.0.0.1:${puerto}/vec_bolsa_publica_b10?sslmode=disable" \
  go test ./internal/modules/bolsa/adapters/postgrespublico -run '^TestIntegracionB10CierraCabecerasAntesDePosiciones$' -count=1
{ printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'; cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000002_proyeccion_bolsas_v1.down.sql"; } | docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 --username postgres --dbname vec_bolsa_publica_b10 >/dev/null
{ printf '%s\n' 'SET ROLE vec_bolsa_publica_migrador;'; cat "$raiz/deploy/postgresql/bolsa_publica/migraciones/000002_proyeccion_bolsas_v1.up.sql"; } | docker exec --interactive "$contenedor" psql -X --set ON_ERROR_STOP=1 --username postgres --dbname vec_bolsa_publica_b10 >/dev/null
docker exec "$contenedor" psql -X --tuples-only --no-align --username postgres --dbname vec_bolsa_publica_b10 --command "SELECT count(*) FROM information_schema.tables WHERE table_schema='vec_bolsa_publica_datos' AND table_name IN ('bolsa_publica','posicion_bolsa_publica')" | grep -Fx 2 >/dev/null
