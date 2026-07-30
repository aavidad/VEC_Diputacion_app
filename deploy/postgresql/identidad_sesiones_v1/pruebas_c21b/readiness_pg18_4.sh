#!/usr/bin/env bash

esperar_postgresql_definitivo() {
  local contenedor=$1
  local base=$2
  local clave=$3
  local observado=
  local _=

  for _ in $(seq 1 120); do
    observado=$(
      docker exec --env PGPASSWORD="$clave" "$contenedor" \
        psql -XAtq --set ON_ERROR_STOP=1 \
        -h 127.0.0.1 -U postgres -d "$base" \
        -c "SELECT current_setting('server_version_num')||'|'||
pg_is_in_recovery()::text||'|'||current_database()" 2>/dev/null || true
    )
    if [[ $observado == "180004|false|$base" ]]; then
      return 0
    fi
    sleep 0.2
  done

  echo "PostgreSQL 18.4 primario no alcanzó disponibilidad definitiva por TCP: $observado" >&2
  return 1
}
