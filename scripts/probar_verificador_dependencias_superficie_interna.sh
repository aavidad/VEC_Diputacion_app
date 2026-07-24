#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

salida="$(mktemp)"
trap 'unlink "${salida}" 2>/dev/null || true' EXIT

if scripts/verificar_dependencias_superficie_interna.sh ./cmd/vec-server >"${salida}" 2>&1; then
	printf 'El verificador interno acepto por error el binario integrado.\n' >&2
	exit 1
fi
if ! grep -Fq 'dependencias no aprobadas' "${salida}"; then
	printf 'El verificador interno fallo por una causa distinta al aislamiento:\n' >&2
	cat "${salida}" >&2
	exit 1
fi

printf 'Autoprueba negativa del grafo interno superada.\n'
