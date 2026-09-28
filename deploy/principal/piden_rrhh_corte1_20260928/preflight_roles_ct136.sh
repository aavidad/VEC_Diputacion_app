#!/usr/bin/env bash
# Ensaya la preimagen exacta del rol CT136 y revierte toda la transacción.
set -Eeuo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
sql="$repo/deploy/postgresql/contratacion_temporal/roles_registrador_auditoria_up.sql"
fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
[[ $# == 1 && ( $1 == --clon || $1 == --destino ) ]] \
  || fallar 'uso: preflight_roles_ct136.sh --clon|--destino'
"$script_dir/preflight_no_go.sh" "$1"
[[ -f $sql && $(tail -n 1 -- "$sql") == 'COMMIT;' ]] \
  || fallar 'delta de rol CT136 sin COMMIT final exacto'
[[ $(grep -c '^COMMIT;$' "$sql") == 1 ]] \
  || fallar 'delta de rol CT136 tiene más de un COMMIT'

# CREATE ROLE/GRANT y todas sus guardas de ACL se ejecutan con el DBA real,
# pero ROLLBACK es el único final admitido. Un error de psql aborta la tanda.
sed '$s/^COMMIT;$/ROLLBACK;/' "$sql" \
  | psql -X --no-psqlrc --set=ON_ERROR_STOP=1 >/dev/null \
  || fallar 'preflight del rol/ACL CT136 rechazado'
printf 'CT136: rol y ACL compatibles en ROLLBACK\n'
