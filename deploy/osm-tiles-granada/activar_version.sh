#!/usr/bin/env bash
set -Eeuo pipefail

raiz="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
estado="$raiz/estado"
version="${1:-}"

if [[ ! "$version" =~ ^[0-9]{8}T[0-9]{6}Z-[a-f0-9]{12}$ ]]; then
  echo "ERROR: indique una version AAAAMMDDThhmmssZ-12hex." >&2
  exit 1
fi

for orden in docker jq sha256sum sqlite3 flock; do
  if ! command -v "$orden" >/dev/null 2>&1; then
    echo "ERROR: falta la dependencia local: $orden" >&2
    exit 1
  fi
done

mkdir -p "$estado"
exec 9>"$estado/.bloqueo-activacion"
if ! flock -n 9; then
  echo "ERROR: ya hay una activacion en curso." >&2
  exit 1
fi

directorio="$estado/releases/$version"
artefacto="$directorio/granada.mbtiles"
manifiesto="$directorio/manifiesto.json"
activo="$estado/activo"

"$raiz/validar_mbtiles.sh" "$artefacto"
if [[ ! -f "$manifiesto" || -L "$manifiesto" ]]; then
  echo "ERROR: falta el manifiesto inmutable de la version." >&2
  exit 1
fi

esperada="$(jq -er '.sha256_mbtiles' "$manifiesto")"
real="$(sha256sum "$artefacto" | awk '{print $1}')"
if [[ "$esperada" != "$real" ]]; then
  echo "ERROR: la huella del MBTiles no coincide con su manifiesto." >&2
  exit 1
fi

if [[ -e "$activo" && ! -L "$activo" ]]; then
  echo "ERROR: estado/activo debe ser un enlace simbolico o no existir." >&2
  exit 1
fi

anterior=""
if [[ -L "$activo" ]]; then
  anterior="$(readlink "$activo")"
fi

intercambiar() {
  destino="$1"
  temporal="$estado/.activo.$$.tmp"
  rm -f -- "$temporal"
  ln -s -- "$destino" "$temporal"
  mv -Tf -- "$temporal" "$activo"
}

intercambiar "releases/$version"

arrancar() {
  UID_GID="$(id -u):$(id -g)" docker compose --project-directory "$raiz" up -d --force-recreate --no-deps tiles-osm
}

saludable() {
  for _ in $(seq 1 60); do
    id="$(docker compose --project-directory "$raiz" ps -q tiles-osm 2>/dev/null || true)"
    if [[ -n "$id" ]]; then
      salud="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}sin-salud{{end}}' "$id" 2>/dev/null || true)"
      if [[ "$salud" == "healthy" ]]; then
        return 0
      fi
      if [[ "$(docker inspect --format '{{.State.Status}}' "$id" 2>/dev/null || true)" == "exited" ]]; then
        return 1
      fi
    fi
    sleep 2
  done
  return 1
}

if arrancar && saludable; then
  printf '%s\t%s\t%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$version" "activada" >>"$estado/activaciones.log"
  echo "OK: cartografia $version activa y saludable."
  exit 0
fi

echo "ERROR: la nueva version no ha superado salud; se revierte." >&2
if [[ -n "$anterior" ]]; then
  intercambiar "$anterior"
  arrancar || true
else
  rm -f -- "$activo"
  docker compose --project-directory "$raiz" stop tiles-osm >/dev/null 2>&1 || true
fi
printf '%s\t%s\t%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$version" "revertida" >>"$estado/activaciones.log"
exit 1
