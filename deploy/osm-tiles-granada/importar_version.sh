#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
estado="$raiz/estado"
fuente="$raiz/../osrm-granada/data/granada-buffer.osm.pbf"

for orden in docker sha256sum jq flock; do
  if ! command -v "$orden" >/dev/null 2>&1; then
    echo "ERROR: falta la dependencia local: $orden" >&2
    exit 1
  fi
done

if [[ ! -f "$fuente" || -L "$fuente" ]]; then
  echo "ERROR: no existe el PBF provincial regular esperado: $fuente" >&2
  exit 1
fi

mkdir -p "$estado/releases" "$estado/trabajo"
exec 9>"$estado/.bloqueo-importacion"
if ! flock -n 9; then
  echo "ERROR: ya hay una importacion en curso." >&2
  exit 1
fi

huella_fuente="$(sha256sum "$fuente" | awk '{print $1}')"
version="${1:-$(date -u +%Y%m%dT%H%M%SZ)-${huella_fuente:0:12}}"

if [[ ! "$version" =~ ^[0-9]{8}T[0-9]{6}Z-[a-f0-9]{12}$ ]]; then
  echo "ERROR: version no canonica. Use AAAAMMDDThhmmssZ-12hex." >&2
  exit 1
fi

directorio_version="$estado/releases/$version"
parcial="$directorio_version/granada.mbtiles.parcial"
final="$directorio_version/granada.mbtiles"
trabajo="$estado/trabajo/$version"

if [[ -e "$directorio_version" ]]; then
  echo "ERROR: la version ya existe y nunca se sobrescribe: $version" >&2
  exit 1
fi

mkdir -p "$directorio_version" "$trabajo"
limpiar_fallo() {
  codigo=$?
  if [[ $codigo -ne 0 ]]; then
    rm -rf -- "$directorio_version" "$trabajo"
  fi
  exit "$codigo"
}
trap limpiar_fallo EXIT

echo "Importando $version desde el PBF compartido (sin copiarlo)..."
UID_GID="$(id -u):$(id -g)" docker compose --project-directory "$raiz" \
  --profile importacion run --rm importar-osm \
  /fuente/granada-buffer.osm.pbf \
  --output "/salida/releases/$version/granada.mbtiles.parcial" \
  --store "/salida/trabajo/$version" \
  --config /usr/src/app/resources/config-openmaptiles.json \
  --process /usr/src/app/resources/process-openmaptiles.lua

"$raiz/validar_mbtiles.sh" "$parcial"
mv -- "$parcial" "$final"
chmod 0444 "$final"

huella_mbtiles="$(sha256sum "$final" | awk '{print $1}')"
instante="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
jq -n \
  --arg version "$version" \
  --arg instante "$instante" \
  --arg fuente "deploy/osrm-granada/data/granada-buffer.osm.pbf" \
  --arg sha256_fuente "$huella_fuente" \
  --arg sha256_mbtiles "$huella_mbtiles" \
  '{version:$version, creado_utc:$instante, fuente:$fuente, sha256_fuente:$sha256_fuente, sha256_mbtiles:$sha256_mbtiles, esquema:"OpenMapTiles 3.x", maxzoom:14}' \
  >"$directorio_version/manifiesto.json"
chmod 0444 "$directorio_version/manifiesto.json"
rm -rf -- "$trabajo"

trap - EXIT
echo "OK: version inmutable preparada: $version"
echo "Active con: $raiz/activar_version.sh $version"
