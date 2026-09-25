#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

archivos_sin_formato="$(gofmt -l cmd config internal tools/vecsilencio)"
if [[ -n "${archivos_sin_formato}" ]]; then
  printf 'Hay archivos Go sin formato:\n%s\n' "${archivos_sin_formato}" >&2
  exit 1
fi

go mod verify
# Timeouts explicitos: internal/vec/ports tarda >10 min bajo -race en
# maquinas de 2 nucleos (runners de CI) y el limite por defecto de go test
# es justamente 10 min. Vease H-01 de la auditoria 2026-07-16.
go test ./... -count=1 -timeout 20m
go test -race ./... -count=1 -timeout 30m
go vet ./...
# Guarda de fallos silenciosos (M2a): ningún error descartado sin registro
# fuera de la línea base congelada, que solo puede decrecer. La base del árbol
# se contrasta además con la del punto de bifurcación de la rama base
# (VECSILENCIO_RAMA_BASE; en CI origin/$GITHUB_BASE_REF; si no, origin/main):
# una huella nueva o un recuento mayor en base.txt falla aunque se haya
# editado a mano. En CI la referencia es obligatoria.
base_vecsilencio=tools/vecsilencio/base.txt
argumentos_vecsilencio=(-base "${base_vecsilencio}")
rama_base_vecsilencio="${VECSILENCIO_RAMA_BASE:-origin/${GITHUB_BASE_REF:-main}}"
anterior_vecsilencio=""
if git rev-parse --verify --quiet "${rama_base_vecsilencio}^{commit}" >/dev/null; then
  punto_vecsilencio="$(git merge-base "${rama_base_vecsilencio}" HEAD || true)"
  if [[ -z "${punto_vecsilencio}" ]]; then
    punto_vecsilencio="${rama_base_vecsilencio}"
  fi
  if git cat-file -e "${punto_vecsilencio}:${base_vecsilencio}" 2>/dev/null; then
    anterior_vecsilencio="$(mktemp)"
    git show "${punto_vecsilencio}:${base_vecsilencio}" >"${anterior_vecsilencio}"
    argumentos_vecsilencio+=(-base-anterior "${anterior_vecsilencio}")
    printf 'vecsilencio: línea base contrastada con %s (%s)\n' "${rama_base_vecsilencio}" "${punto_vecsilencio:0:12}"
  else
    printf 'vecsilencio: %s no tiene línea base en %s; alta inicial sin contraste\n' "${rama_base_vecsilencio}" "${punto_vecsilencio:0:12}"
  fi
elif [[ -n "${CI:-}" ]]; then
  printf 'vecsilencio: falta la referencia %s para contrastar la línea base (checkout con historia completa)\n' "${rama_base_vecsilencio}" >&2
  exit 1
else
  printf 'vecsilencio: aviso: sin referencia %s; la línea base no se contrasta con la rama base\n' "${rama_base_vecsilencio}" >&2
fi
estado_vecsilencio=0
go run ./tools/vecsilencio "${argumentos_vecsilencio[@]}" || estado_vecsilencio=$?
if [[ -n "${anterior_vecsilencio}" ]]; then
  rm -f "${anterior_vecsilencio}"
fi
if [[ "${estado_vecsilencio}" -ne 0 ]]; then
  exit "${estado_vecsilencio}"
fi
go build ./cmd/...
# Pruebas web (Node >= 20, sin dependencias): el portal y los clientes HTTP
# tienen su propia suite y hasta hoy no formaba parte de la puerta.
node --test $(git ls-files 'web/**/*.test.mjs')
scripts/verificar_dependencias_superficie_publica.sh
scripts/probar_verificador_dependencias_superficie_publica.sh
scripts/verificar_dependencias_superficie_interna.sh
scripts/probar_verificador_dependencias_superficie_interna.sh
scripts/aprovisionar_cartografia_osm.sh
scripts/tests/test_aprovisionar_cartografia_osm.sh
scripts/verificar_manifiestos_superficies_web.sh
scripts/probar_verificador_manifiestos_superficies_web.sh
scripts/probar_carga_tls_interna_root.sh
python3 -m unittest scripts.tests.test_generar_bases_demo_pdf scripts.tests.test_paquete_ejemplo
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
scripts/comprobar_tamano_ficheros.sh
git diff --check

printf 'Puerta de calidad superada.\n'
