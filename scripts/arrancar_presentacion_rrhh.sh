#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

# La presentación completa se ejecuta exclusivamente con su composición
# Docker: portal, proxy, mediador Go, OSRM y teselas OSM. No arranca binarios,
# proxies ni servicios cartográficos directamente en el anfitrión.
docker compose --profile presentacion up \
  --detach \
  --build \
  --wait \
  --wait-timeout "${VEC_PRESENTACION_WAIT_TIMEOUT:-180}"

scripts/smoke_cartografia_presentacion.sh

echo "Presentación RRHH disponible en http://127.0.0.1:${VEC_PRESENTACION_PUBLISHED_PORT:-8081}/presentacion/"
