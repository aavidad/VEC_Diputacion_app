#!/usr/bin/env bash
# Inspección de manifiestos; el modo final verifica también ZIP y árbol extraído.
set -Eeuo pipefail
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
repo=$(git -C "$script_dir" rev-parse --show-toplevel)
modo=${1:-}
[[ $# == 0 || ( $# == 1 && $modo == --provisional ) ]] || { echo 'uso: validar_web.sh [--provisional]' >&2; exit 2; }
fallar() { printf 'ERROR: web corte3: %s\n' "$*" >&2; exit 1; }
if [[ $modo == --provisional ]]; then
  "$script_dir/validar_plan.sh" --provisional >/dev/null
else
  "$script_dir/validar_plan.sh" >/dev/null
fi
for manifest in produccion.manifest interno.manifest; do
  [[ -f $repo/web/$manifest ]] || fallar "falta $manifest"
  sort -u "$repo/web/$manifest" | cmp -s - <(sort "$repo/web/$manifest") || fallar "duplicados en $manifest"
  while IFS= read -r ruta; do
    [[ -n $ruta && $ruta != \#* ]] || continue
    [[ $ruta != /* && $ruta != *../* && $ruta != *//* ]] || fallar "ruta insegura en $manifest: $ruta"
    if [[ $ruta == cartografia/granada-base-20260719-z8-z12.zip && $modo == --provisional ]]; then
      continue
    fi
    [[ -f $repo/web/$ruta && ! -L $repo/web/$ruta ]] || fallar "falta entrada de $manifest: $ruta"
  done <"$repo/web/$manifest"
done
for ruta in \
  static/area-personal/mi-bolsa-historial.js \
  static/portal-empleado/portal-bolsas-reincorporaciones.js \
  static/portal-empleado/modulos/contratacion-temporal/rrhh-plantillas-vista.js \
  static/portal-empleado/modulos/contratacion-temporal/cliente-http-borradores-publicados.js \
  static/portal-empleado/modulos/contratacion-temporal/vista-borradores-publicados.js; do
  grep -Fxq -- "$ruta" "$repo/web/produccion.manifest" || fallar "falta en producción: $ruta"
done
if [[ $modo == --provisional ]]; then
  printf 'WEB_CORTE3_PROVISIONAL_OK: rutas presentes; ZIP y árbol extraído pendientes\n'
  exit 0
fi
command -v rsync >/dev/null || fallar 'falta rsync'
tmp=$(mktemp -d /dev/shm/vec-piden-corte3-web.XXXXXXXX)
trap 'rm -rf -- "$tmp"' EXIT
rsync -a --files-from="$repo/web/produccion.manifest" -- "$repo/web/" "$tmp/"
"$repo/scripts/verificar_web_produccion.sh" "$tmp" "$repo/web/produccion.manifest" >/dev/null \
  || fallar 'árbol web/ZIP no supera verificación productiva'
printf 'WEB_CORTE3_FINAL_OK: manifiesto y ZIP verificados\n'
