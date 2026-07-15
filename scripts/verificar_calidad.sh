#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

archivos_sin_formato="$(gofmt -l cmd config internal)"
if [[ -n "${archivos_sin_formato}" ]]; then
  printf 'Hay archivos Go sin formato:\n%s\n' "${archivos_sin_formato}" >&2
  exit 1
fi

go mod verify
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./cmd/...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
git diff --check

printf 'Puerta de calidad superada.\n'
