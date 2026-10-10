#!/usr/bin/env bash
# Reutiliza la fixture B96 dentro de una sola transacción, con ROLLBACK final.
set -euo pipefail
cd "$(dirname "$0")/../../../.."
base=deploy/postgresql/bolsa_llamamientos/pruebas_sql/b96_solicitud_idempotencia_fixture.sql
b99=deploy/postgresql/bolsa_llamamientos/pruebas_sql/b99_lecturas_fixture.sql
contenedor=${1:?Indique el contenedor PostgreSQL 18 desechable, con B96 y B99 instaladas}
if [[ "$(tail -n 1 "$base")" != "ROLLBACK;" ]]; then
  printf '%s\n' 'La fixture B96 cambió; no se puede intercalar B99.' >&2
  exit 1
fi
{ sed '$d' "$base"; cat "$b99"; } |
  docker exec -i "$contenedor" psql -U postgres -d postgres -X -q -v ON_ERROR_STOP=1 -v B96_DISPOSABLE_CLONE=1
printf '%s\n' 'B99 fixture: OK; transacción revertida.'
