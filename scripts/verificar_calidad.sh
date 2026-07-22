#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

archivos_sin_formato="$(gofmt -l cmd config internal)"
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
go build ./cmd/...
scripts/verificar_dependencias_superficie_publica.sh
scripts/probar_verificador_dependencias_superficie_publica.sh
python3 -m unittest scripts.tests.test_generar_bases_demo_pdf
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
scripts/comprobar_tamano_ficheros.sh
git diff --check

printf 'Puerta de calidad superada.\n'
