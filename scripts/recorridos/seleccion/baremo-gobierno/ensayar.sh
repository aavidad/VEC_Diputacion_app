#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  preparar|alta|recuperar|reinicio) export VEC_BAREMO_PG_FASE="$1" ;;
  *) echo 'Uso: ensayar.sh preparar|alta|recuperar|reinicio' >&2; exit 2 ;;
esac
: "${VEC_BAREMO_PG_CONFIG:?Falta configuración privada del ensayo}"
if [[ "${VEC_BAREMO_PG_DESECHABLE:-}" != si ]]; then
  echo 'Falta activación explícita de clon desechable' >&2
  exit 2
fi
raiz=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
cd -- "$raiz"
ensayo_go=${VEC_ENSAYO_GO:-go}
export GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=4
export TMPDIR=${TMPDIR:-/var/tmp/va-baremo-pg}
export GOCACHE=${GOCACHE:-/tmp/vec-codexa-go-build-20261002}
mkdir -p -- "$TMPDIR" "$GOCACHE"
timeout 300s "$ensayo_go" test -p 8 ./internal/app/bootstrap \
  -run '^TestGobiernoReglasBaremoIntegracionPostgreSQL$' -count=1 -v
