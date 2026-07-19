#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
temporal="$(mktemp)"
trap 'rm -f "$temporal"' EXIT

for script in "$raiz"/*.sh "$raiz"/tests/*.sh; do
  bash -n "$script"
done

jq -e . "$raiz/config/tileserver.json" >/dev/null
jq -e . "$raiz/config/estilos/osm-granada.json" >/dev/null

UID_GID="$(id -u):$(id -g)" docker compose --project-directory "$raiz" \
  --profile importacion config --format json >"$temporal"

exigir() {
  patron="$1"
  fichero="$2"
  if ! rg -q -- "$patron" "$fichero"; then
    echo "ERROR: falta el contrato '$patron' en $fichero" >&2
    exit 1
  fi
}

exigir 'ghcr\.io/systemed/tilemaker:master@sha256:[a-f0-9]{64}' "$raiz/compose.yaml"
exigir 'maptiler/tileserver-gl:v5\.6\.0@sha256:[a-f0-9]{64}' "$raiz/compose.yaml"
exigir 'nginx:1\.27-alpine@sha256:[a-f0-9]{64}' "$raiz/compose.yaml"
exigir 'mbtiles://\{granada\}' "$raiz/config/estilos/osm-granada.json"
exigir '© OpenStreetMap contributors' "$raiz/README.md"
exigir 'Leaflet 1\.9\.4' "$raiz/README.md"

jq -e '
  .services["importar-osm"].network_mode == "none" and
  .services["importar-osm"].read_only == true and
  .services["importar-osm"].pull_policy == "never" and
  (.services["importar-osm"].cap_drop | index("ALL")) != null and
  (.services["importar-osm"].security_opt | index("no-new-privileges:true")) != null and
  ([.services["importar-osm"].volumes[] |
    select(.target == "/fuente/granada-buffer.osm.pbf" and .read_only == true)] | length) == 1 and
  .services["tiles-osm"].read_only == true and
  .services["tiles-osm"].pull_policy == "never" and
  (.services["tiles-osm"].cap_drop | index("ALL")) != null and
  (.services["tiles-osm"].security_opt | index("no-new-privileges:true")) != null and
  (.services["tiles-osm"].command | index("--no-cors")) != null and
  (.services["tiles-osm"].command | index("--silent")) != null and
  (.services["tiles-osm"].tmpfs | index("/home/node/.cache:rw,noexec,nosuid,nodev,size=32m,mode=0700,uid=999,gid=999")) != null and
  ((.services["tiles-osm"].ports // []) | length) == 0 and
  .services["proxy-osm"].read_only == true and
  .services["proxy-osm"].pull_policy == "never" and
  (.services["proxy-osm"].cap_drop | index("ALL")) != null and
  (.services["proxy-osm"].security_opt | index("no-new-privileges:true")) != null and
  ([.services["proxy-osm"].ports[] |
    select(.host_ip == "127.0.0.1" and .target == 8080)] | length) == 1 and
  (.services["proxy-osm"].networks | has("cartografia_sin_egreso")) and
  (.services["proxy-osm"].networks | has("borde_cartografia")) and
  .networks.cartografia_sin_egreso.internal == true and
  ((.networks.borde_cartografia.internal // false) == false) and
  .networks.borde_cartografia.driver_opts["com.docker.network.bridge.enable_ip_masquerade"] == "false"
' "$temporal" >/dev/null

if find "$raiz" -path "$raiz/estado" -prune -o \( -name '*.pbf' -o -name '*.mbtiles' \) -print | grep -q .; then
  echo "ERROR: el despliegue ha copiado datos cartograficos pesados." >&2
  exit 1
fi

if rg -n 'tile\.openstreetmap\.org|unpkg\.com|cdn\.jsdelivr\.net' \
  "$raiz/compose.yaml" "$raiz/config" "$raiz/nginx-osm-same-origin.conf"; then
  echo "ERROR: runtime cartografico con dependencia publica." >&2
  exit 1
fi

echo "OK: contrato OSM on-premise validado sin ejecutar la importacion."
