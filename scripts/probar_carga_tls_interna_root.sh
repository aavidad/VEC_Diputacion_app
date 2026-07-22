#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

binario="$(mktemp)"
trap 'rm -f "${binario}"' EXIT
CGO_ENABLED=0 go test -c -o "${binario}" ./internal/app/composicion/interna

if [[ "$(id -u)" == 0 ]]; then
	"${binario}" -test.run '^TestConstruirServidorInternoCargaRealDesdeArbolRootSoloLectura$' -test.v
	exit 0
fi
if ! command -v docker >/dev/null 2>&1; then
	printf 'Se requiere UID 0 o Docker para probar carga TLS root-owned.\n' >&2
	exit 1
fi

docker run --rm \
	--read-only \
	--network none \
	--cap-drop ALL \
	--security-opt no-new-privileges \
	--tmpfs /root:rw,mode=0700 \
	--tmpfs /tmp:rw,mode=1777 \
	--mount "type=bind,src=${binario},dst=/prueba,readonly" \
	debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818 \
	/prueba -test.run '^TestConstruirServidorInternoCargaRealDesdeArbolRootSoloLectura$' -test.v
