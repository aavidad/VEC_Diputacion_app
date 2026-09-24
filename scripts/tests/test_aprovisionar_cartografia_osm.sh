#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
raiz="$(pwd -P)"
temporal="$(mktemp -d)"
trap 'rm -rf "$temporal"' EXIT
mkdir -p "$temporal/repositorio/scripts" "$temporal/repositorio/web/cartografia" "$temporal/bin"
cp scripts/aprovisionar_cartografia_osm.sh "$temporal/repositorio/scripts/"
cp web/cartografia/granada-base-20260719-z8-z12.json "$temporal/repositorio/web/cartografia/"
zip="$temporal/repositorio/web/cartografia/granada-base-20260719-z8-z12.zip"
indice="$temporal/repositorio/web/cartografia/granada-base-20260719-z8-z12.json"

cat >"$temporal/bin/curl" <<'SH'
#!/usr/bin/env bash
set -euo pipefail
destino=""
for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == --output ]]; then
    siguiente=$((i+1))
    destino="${!siguiente}"
  fi
done
[[ "${*: -1}" == "$VEC_TEST_URL" && -n "$destino" ]] || exit 9
cp "$VEC_TEST_SOURCE" "$destino"
SH
chmod +x "$temporal/bin/curl"

export VEC_TEST_URL="https://github.com/aavidad/VEC_Diputacion_app/releases/download/osm-granada-base-20260719-z8-z12-v1/granada-base-20260719-z8-z12.zip"
export VEC_TEST_SOURCE="$raiz/web/cartografia/granada-base-20260719-z8-z12.zip"
export PATH="$temporal/bin:$PATH"
aprovisionar() { "$temporal/repositorio/scripts/aprovisionar_cartografia_osm.sh" >"$temporal/salida" 2>&1; }
debe_fallar() { if aprovisionar; then echo "ERROR: acepto $1" >&2; exit 1; fi; }

# Sin ZIP en un checkout limpio, solo se admite el asset completo de la URL fija.
aprovisionar
cmp -s "$zip" "$VEC_TEST_SOURCE"

# La copia local valida permite trabajar antes de publicar la Release.
aprovisionar
grep -Fq 'ZIP OSM local verificado' "$temporal/salida"
rm "$zip"
printf 'invalido' >"$temporal/asset-corrupto.zip"
export VEC_TEST_SOURCE="$temporal/asset-corrupto.zip"
debe_fallar "un asset remoto alterado"
[[ ! -e "$zip" ]] || { echo 'ERROR: quedo un ZIP parcial' >&2; exit 1; }

printf 'invalido' >"$zip"
debe_fallar "un ZIP local alterado"
rm "$zip"
ln -s "$raiz/web/cartografia/granada-base-20260719-z8-z12.zip" "$zip"
debe_fallar "un ZIP local enlazado"
rm "$zip"
printf 'invalido' >"$indice"
debe_fallar "un indice alterado"

echo 'Aprovisionamiento OSM local, descarga y fallos cerrados verificados.'
