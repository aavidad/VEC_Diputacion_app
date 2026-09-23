#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

salida="$(mktemp)"
trap 'unlink "${salida}" 2>/dev/null || true' EXIT

for binario in ./cmd/vec-server ./cmd/vec-publico; do
	if scripts/verificar_dependencias_superficie_interna.sh "${binario}" >"${salida}" 2>&1; then
		printf 'El verificador interno acepto por error %s.\n' "${binario}" >&2
		exit 1
	fi
	if ! grep -Fq 'dependencias no aprobadas' "${salida}"; then
		printf 'El verificador interno fallo por una causa distinta al aislamiento (%s):\n' "${binario}" >&2
		cat "${salida}" >&2
		exit 1
	fi
done

printf 'Autoprueba negativa del grafo interno superada.\n'
