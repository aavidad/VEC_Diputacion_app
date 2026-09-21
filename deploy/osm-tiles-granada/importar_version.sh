#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
estado="$raiz/estado"
directorio_fuentes="$raiz/../osrm-granada/data"
nombre_pbf="${OSM_SOURCE_BASENAME:-granada-buffer.osm.pbf}"

# Solo se seleccionan ficheros directamente bajo el directorio compartido. La
# ruta que llega a Compose se construye despues de esta lista positiva: ni una
# ruta relativa ni un enlace pueden sustituir la fuente declarada.
if [[ ! "$nombre_pbf" =~ ^[a-z0-9][a-z0-9_-]{0,63}\.osm\.pbf$ ]]; then
  echo "ERROR: OSM_SOURCE_BASENAME debe ser un basename .osm.pbf en minusculas." >&2
  exit 1
fi

omitir_integridad="${OSM_IMPORT_SKIP_INTEGRITY-false}"
argumentos_integridad=()
case "$omitir_integridad" in
  false)
    ;;
  true)
    argumentos_integridad=(--skip-integrity)
    ;;
  *)
    echo "ERROR: OSM_IMPORT_SKIP_INTEGRITY solo admite true o false." >&2
    exit 1
    ;;
esac

if [[ -L "$directorio_fuentes" || ! -d "$directorio_fuentes" ]]; then
  echo "ERROR: el directorio de fuentes PBF no es un directorio regular." >&2
  exit 1
fi

fuente="$directorio_fuentes/$nombre_pbf"
fuente_relativa="deploy/osrm-granada/data/$nombre_pbf"

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
# tilemaker decide el formato por la extension. Debe terminar en `.mbtiles`:
# `granada.mbtiles.parcial` se interpreta como un directorio de teselas.
parcial="$directorio_version/granada.parcial.mbtiles"
final="$directorio_version/granada.mbtiles"
trabajo="$estado/trabajo/$version"
bbox="${OSM_IMPORT_BBOX:--4.75,36.55,-1.75,38.25}"

# El rectangulo historico solo pertenece al PBF historico granada-buffer. Una
# fuente distinta debe declarar el rectangulo efectivo que se entrega a
# tilemaker; no se hereda ni se presenta como si fuera el ambito historico.
if [[ "$nombre_pbf" != "granada-buffer.osm.pbf" && -z "${OSM_IMPORT_BBOX:-}" ]]; then
  echo "ERROR: un PBF distinto exige OSM_IMPORT_BBOX explicito y verificable." >&2
  exit 1
fi

if [[ ! "$bbox" =~ ^-?[0-9]+([.][0-9]+)?,-?[0-9]+([.][0-9]+)?,-?[0-9]+([.][0-9]+)?,-?[0-9]+([.][0-9]+)?$ ]]; then
  echo "ERROR: OSM_IMPORT_BBOX debe ser minlon,minlat,maxlon,maxlat." >&2
  exit 1
fi

IFS=, read -r minlon minlat maxlon maxlat <<<"$bbox"
if ! awk -v minlon="$minlon" -v minlat="$minlat" -v maxlon="$maxlon" -v maxlat="$maxlat" '
  BEGIN { exit !(minlon >= -180 && maxlon <= 180 && minlat >= -90 && maxlat <= 90 && minlon < maxlon && minlat < maxlat) }
'; then
  echo "ERROR: OSM_IMPORT_BBOX no describe un rectangulo geografico valido." >&2
  exit 1
fi

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
OSM_SOURCE_BASENAME="$nombre_pbf" UID_GID="$(id -u):$(id -g)" docker compose --project-directory "$raiz" \
  --profile importacion run --rm importar-osm \
  /fuente/entrada.osm.pbf \
  --output "/salida/releases/$version/granada.parcial.mbtiles" \
  --store "/salida/trabajo/$version" \
  --bbox "$bbox" \
  "${argumentos_integridad[@]}" \
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
  --arg fuente "$fuente_relativa" \
  --arg pbf_basename "$nombre_pbf" \
  --arg sha256_fuente "$huella_fuente" \
  --arg sha256_mbtiles "$huella_mbtiles" \
  --arg bbox "$bbox" \
  --argjson importacion_omite_integridad "$omitir_integridad" \
  '{version:$version, creado_utc:$instante, fuente_ruta:$fuente, pbf_basename:$pbf_basename, sha256_fuente:$sha256_fuente, sha256_mbtiles:$sha256_mbtiles, importacion_omite_integridad:$importacion_omite_integridad, esquema:"OpenMapTiles 3.x", maxzoom:14, bbox_importacion_efectiva:($bbox | split(",") | map(tonumber))}' \
  >"$directorio_version/manifiesto.json"
chmod 0444 "$directorio_version/manifiesto.json"
rm -rf -- "$trabajo"

trap - EXIT
echo "OK: version inmutable preparada: $version"
echo "Active con: $raiz/activar_version.sh $version"
