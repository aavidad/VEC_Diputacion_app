#!/usr/bin/env bash
set -Eeuo pipefail

: "${VEC_PRINCIPAL_ADMIN_DATABASE_URL:?Falta VEC_PRINCIPAL_ADMIN_DATABASE_URL}"
: "${VEC_BOLSA_AUDITORIA_FRONTERA_LOGIN_PASSWORD:?Falta VEC_BOLSA_AUDITORIA_FRONTERA_LOGIN_PASSWORD}"

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
artefacto=${VEC_PRINCIPAL_ARTIFACT_DIR:-$HOME/.local/state/vec-desarrollo-20260906/incorporacion-servidor-20260910/artefacto}
binario=$artefacto/vec-server
app=${VEC_PRINCIPAL_APP_CONTAINER:-vec-aplicacion-incorporacion-20260910}
postgres=${VEC_PRINCIPAL_POSTGRES_CONTAINER:-vec-postgresql-20260906}
marca=$(date -u +%Y%m%dT%H%M%SZ)
respaldo=$artefacto/respaldo-d6-$marca
nuevo=$artefacto/.vec-server.d6.$$
app_parada=false

reanudar_si_falla() {
  local estado=$?
  rm -f -- "$nuevo"
  if (( estado != 0 )) && [[ $app_parada == true ]]; then
    podman start "$app" >/dev/null 2>&1 || true
  fi
  exit "$estado"
}
trap reanudar_si_falla EXIT

git -C "$repo" status --porcelain | grep -q . && {
  echo 'El checkout contiene cambios; se rechaza desplegar.' >&2
  exit 1
}
git -C "$repo" checkout main
git -C "$repo" pull --ff-only origin main

podman stop --time 30 "$app"
app_parada=true

psql_base=(psql "$VEC_PRINCIPAL_ADMIN_DATABASE_URL" --no-psqlrc --set=ON_ERROR_STOP=1)
"${psql_base[@]}" --set=finalizar=ROLLBACK \
  --set=bolsa_auditoria_password="$VEC_BOLSA_AUDITORIA_FRONTERA_LOGIN_PASSWORD" \
  --file "$script_dir/01_roles.sql"
"${psql_base[@]}" --set=finalizar=COMMIT \
  --set=bolsa_auditoria_password="$VEC_BOLSA_AUDITORIA_FRONTERA_LOGIN_PASSWORD" \
  --file "$script_dir/01_roles.sql"
"${psql_base[@]}" --set=finalizar=ROLLBACK --file "$script_dir/02_migraciones.sql"
"${psql_base[@]}" --set=finalizar=COMMIT --file "$script_dir/02_migraciones.sql"

mkdir -p -- "$artefacto" "$respaldo"
GOTOOLCHAIN=auto go -C "$repo" build -buildvcs=false -o "$nuevo" ./cmd/vec-server
if [[ -f $binario ]]; then
  cp -a -- "$binario" "$respaldo/vec-server"
fi
if [[ -d $artefacto/web ]]; then
  cp -a -- "$artefacto/web" "$respaldo/web"
fi
install -m 0755 -- "$nuevo" "$binario"
rm -f -- "$nuevo"
rsync -a --delete "$repo/web/" "$artefacto/web/"

podman start "$app" >/dev/null
app_parada=false
podman ps --filter "name=^${app}$" --filter status=running --format '{{.Names}}' | grep -Fx "$app"
podman ps --filter "name=^${postgres}$" --filter status=running --format '{{.Names}}' | grep -Fx "$postgres"
podman logs --tail 20 "$app"
"$script_dir/verificar.sh"

trap - EXIT
printf 'D6 desplegado; respaldo: %s\n' "$respaldo"
