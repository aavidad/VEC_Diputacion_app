#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

# Smoke autocontenido de CI. Usa un proyecto Docker efímero y no instala ni
# ejecuta servidores, navegadores o utilidades de red en el anfitrión.
export COMPOSE_PROJECT_NAME="vec-presentacion-smoke-$$"
export VEC_PRESENTACION_PUBLISHED_PORT="${VEC_SMOKE_PRESENTACION_PORT:-18091}"

limpiar() {
  docker compose --profile presentacion --profile presentacion-cartografia \
    --profile herramientas-presentacion down --remove-orphans --volumes \
    >/dev/null 2>&1 || true
}
trap limpiar EXIT INT TERM

docker compose --profile presentacion --profile presentacion-cartografia up \
  --detach \
  --build \
  --wait \
  --wait-timeout "${VEC_SMOKE_WAIT_TIMEOUT:-180}"

docker compose --profile presentacion --profile presentacion-cartografia \
  --profile herramientas-presentacion run \
  --rm \
  --no-deps \
  smoke-cartografia-presentacion

echo "Smoke Docker de la presentación RRHH superado."
