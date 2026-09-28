#!/usr/bin/env bash
# Ensaya el delta DBA del calculador B51 sin conservar ningún cambio.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
sql="$repo/deploy/postgresql/bolsa_llamamientos/roles_calculador_politica_up.sql"
fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
[[ $# == 1 && ( $1 == --clon || $1 == --destino ) ]] \
  || fallar 'uso: preflight_roles_calculador.sh --clon|--destino'
"$script_dir/preflight_no_go.sh" "$1"
[[ -f $sql && $(tail -n 1 -- "$sql") == 'COMMIT;' ]] \
  || fallar 'delta de rol calculador sin COMMIT final exacto'
[[ $(grep -c '^COMMIT;$' "$sql") == 1 ]] \
  || fallar 'delta de rol calculador tiene más de un COMMIT'
sed '$s/^COMMIT;$/ROLLBACK;/' "$sql" \
  | psql -X --no-psqlrc --set=ON_ERROR_STOP=1 >/dev/null \
  || fallar 'preflight del rol calculador B51 rechazado'
printf 'B51: rol calculador compatible en ROLLBACK\n'
