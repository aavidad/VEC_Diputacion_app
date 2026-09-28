#!/usr/bin/env bash
set -euo pipefail

raiz=$(git -C "$(dirname -- "${BASH_SOURCE[0]}")" rev-parse --show-toplevel)
export GOCACHE=${GOCACHE:-/dev/shm/vec-rrhh-plantillas-web-gocache}
export GOMAXPROCS=${GOMAXPROCS:-2}

"$raiz/deploy/postgresql/contratacion_temporal/pruebas_sql/ct137_ad3_100_pg18.sh"
(
  cd "$raiz"
  go test ./internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo \
    ./internal/modules/contrataciontemporal/application/plantillascatalogo
  go test ./internal/app/bootstrap -run Plantillas -count=1
)
printf 'SQL/Go focal de plantillas superado; navegador y recuperación requieren entorno mTLS separado.\n'
