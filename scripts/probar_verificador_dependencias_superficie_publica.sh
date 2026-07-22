#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

salida="$(mktemp)"
trap 'rm -f "${salida}"' EXIT

# La raiz integrada importa deliberadamente dominios internos. Si la puerta
# dejase de detectarlos, esta autoprueba negativa impediria aprobar la CI.
if scripts/verificar_dependencias_superficie_publica.sh ./cmd/vec-server >"${salida}" 2>&1; then
	printf 'El verificador acepto por error el binario integrado.\n' >&2
	exit 1
fi
if ! grep -Fq 'dependencias prohibidas' "${salida}"; then
	printf 'El verificador fallo por una causa distinta a la frontera de dependencias:\n' >&2
	cat "${salida}" >&2
	exit 1
fi

printf 'Autoprueba negativa del grafo publico superada.\n'
