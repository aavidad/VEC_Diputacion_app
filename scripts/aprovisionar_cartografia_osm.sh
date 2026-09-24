#!/usr/bin/env bash
# Preparacion de checkout/CI/imagen. El servidor nunca descarga cartografia.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

zip="web/cartografia/granada-base-20260719-z8-z12.zip"
indice="web/cartografia/granada-base-20260719-z8-z12.json"
sha_zip="0f0d78212832493c424699a42847069ae24b8b3717917780c4d64444aa250165"
sha_indice="9d69233ad62c4371a2248bef2f91db8340823cb2a618272627693805ad021f81"
url="https://github.com/aavidad/VEC_Diputacion_app/releases/download/osm-granada-base-20260719-z8-z12-v1/granada-base-20260719-z8-z12.zip"

fallar() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }
verificar_sha() {
  printf '%s  %s\n' "$1" "$2" | sha256sum --check --status
}

[[ -f "$indice" && ! -L "$indice" ]] || fallar "falta el indice OSM rastreado o es un enlace"
[[ ! -L web/cartografia ]] || fallar "el directorio OSM es un enlace"
verificar_sha "$sha_indice" "$indice" || fallar "el indice OSM no coincide con su huella"

if [[ -e "$zip" || -L "$zip" ]]; then
  [[ -f "$zip" && ! -L "$zip" ]] || fallar "el ZIP OSM local no es un archivo regular"
  verificar_sha "$sha_zip" "$zip" || fallar "el ZIP OSM local no coincide con la Release fijada"
  printf 'ZIP OSM local verificado: %s\n' "$sha_zip"
  exit 0
fi

command -v curl >/dev/null || fallar "curl no esta disponible para preparar el ZIP OSM"
mkdir -p "$(dirname "$zip")"
temporal="$(mktemp "${zip}.descarga.XXXXXXXX")"
trap 'rm -f "$temporal"' EXIT

curl --fail --location --silent --show-error \
  --proto '=https' --proto-redir '=https' \
  --connect-timeout 15 --max-time 180 --max-filesize 20000000 \
  --output "$temporal" "$url" || fallar "no se pudo descargar la Release OSM publica"
verificar_sha "$sha_zip" "$temporal" || fallar "el asset OSM descargado no coincide con su huella"
chmod 0644 "$temporal"
mv -n "$temporal" "$zip" || fallar "no se pudo instalar el ZIP OSM"
verificar_sha "$sha_zip" "$zip" || fallar "el ZIP OSM instalado no coincide con su huella"
printf 'ZIP OSM de Release verificado: %s\n' "$sha_zip"
