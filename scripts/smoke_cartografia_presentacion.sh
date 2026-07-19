#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

# La comprobación se ejecuta dentro de la red Docker de presentación. No exige
# curl, jq, Python ni utilidades cartográficas instaladas en el anfitrión.
docker compose --profile presentacion --profile herramientas-presentacion build \
  smoke-cartografia-presentacion
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm \
  --no-deps \
  smoke-cartografia-presentacion
