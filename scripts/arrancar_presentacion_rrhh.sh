#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

# El perfil base mantiene Contratación temporal y Bolsas disponibles aunque
# Dietas o sus artefactos cartográficos no estén instalados. Un nombre estable
# permite reutilizar las redes locales entre worktrees sin solaparlas.
export COMPOSE_PROJECT_NAME="${COMPOSE_PROJECT_NAME:-vec-ct-local}"

perfiles="--profile presentacion"
if [ "${VEC_PRESENTACION_CON_CARTOGRAFIA:-false}" = "true" ]; then
  perfiles="$perfiles --profile presentacion-cartografia"
fi

# shellcheck disable=SC2086
docker compose $perfiles up \
  --detach \
  --build \
  --wait \
  --wait-timeout "${VEC_PRESENTACION_WAIT_TIMEOUT:-180}"

if [ "${VEC_PRESENTACION_CON_CARTOGRAFIA:-false}" = "true" ]; then
  scripts/smoke_cartografia_presentacion.sh
fi

echo "Presentación RRHH disponible en http://127.0.0.1:${VEC_PRESENTACION_PUBLISHED_PORT:-8081}/presentacion/"
