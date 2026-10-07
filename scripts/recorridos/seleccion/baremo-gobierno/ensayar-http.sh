#!/usr/bin/env bash
set -euo pipefail

fase=${1:-}
case "$fase" in
  compilar) filtro='^$' ;;
  preparar|alta|recuperar|reinicio)
    : "${VEC_BAREMO_HTTP_PG_CONFIG:?Falta configuración privada del ensayo HTTP}"
    [[ ${VEC_BAREMO_PG_DESECHABLE:-} == si ]] || { echo 'Falta activación explícita de clon desechable' >&2; exit 2; }
    export VEC_BAREMO_PG_FASE="$fase"
    filtro='^TestGobiernoReglasBaremoHTTPIntegracionPostgreSQL$'
    ;;
  *) echo 'Uso: ensayar-http.sh compilar|preparar|alta|recuperar|reinicio' >&2; exit 2 ;;
esac
raiz=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../../.." && pwd)
cd -- "$raiz"
ensayo_go=${VEC_ENSAYO_GO:-go}
export GOTOOLCHAIN=local GOPROXY=off GOMAXPROCS=8
export TMPDIR=${TMPDIR:-/var/tmp/codexa-baremo-http-20261003}
mkdir -p -- "$TMPDIR"
scratch=$(mktemp -d "$TMPDIR/overlay-baremo-http.XXXXXXXX")
trap 'rm -rf -- "$scratch"' EXIT
harness=29905e989478a0480a43127bbe4058f7a0e33a17
for fichero in bolsa_reglas_baremo_gobierno_integracion_test.go bolsa_reglas_baremo_gobierno_helpers_integracion_test.go; do
  git show "$harness:internal/app/bootstrap/$fichero" > "$scratch/$fichero"
done
python3 - "$raiz" "$scratch" <<'PY'
import json, pathlib, sys
raiz, scratch = map(pathlib.Path, sys.argv[1:])
archivos = ["bolsa_reglas_baremo_gobierno_integracion_test.go", "bolsa_reglas_baremo_gobierno_helpers_integracion_test.go"]
overlay = {"Replace": {str(raiz / "internal/app/bootstrap" / f): str(scratch / f) for f in archivos}}
(scratch / "overlay.json").write_text(json.dumps(overlay))
PY
timeout 1800s "$ensayo_go" test -p 8 -tags baremo_pg_http -overlay "$scratch/overlay.json" \
  ./internal/app/bootstrap -run "$filtro" -count=1 -v
