#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

binario="$(mktemp)"
volumen="vec-c4-tls-root-${RANDOM}-$$"
trap 'rm -f "${binario}"; docker volume rm -f "${volumen}" >/dev/null 2>&1 || true' EXIT
CGO_ENABLED=0 go test -c -o "${binario}" ./internal/app/composicion/interna

if ! command -v docker >/dev/null 2>&1; then
	printf 'Se requiere Docker para probar provisionador root y runtime 10001.\n' >&2
	exit 1
fi
docker volume create "${volumen}" >/dev/null

contenedor() {
	docker run --rm \
	--read-only \
	--network none \
	--cap-drop ALL \
	--security-opt no-new-privileges \
	--tmpfs /tmp:rw,mode=1777 \
	--env VEC_PRUEBA_TLS_ROOT_DIR=/material \
	--mount "type=bind,src=${binario},dst=/prueba,readonly" \
	--mount "type=volume,src=${volumen},dst=/material${1}" \
	--user "${2}" \
	debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 \
	/prueba -test.run "${3}" -test.v
}

contenedor "" "0:10001" '^TestPrepararMaterialTLSRootParaRuntimeNoPrivilegiado$'
contenedor ",readonly" "0:10001" '^TestConstruirServidorInternoCargaRealComoRuntimeNoPrivilegiado$'
contenedor ",readonly" "10001:10001" '^TestConstruirServidorInternoCargaRealComoRuntimeNoPrivilegiado$'
