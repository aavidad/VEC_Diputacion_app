#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

modulo="$(go list -m -f '{{.Path}}')"
salida="$(mktemp)"
trap 'unlink "${salida}" 2>/dev/null || true' EXIT

comprobar_rechazo() {
	local objetivo="$1"
	shift
	if scripts/verificar_dependencias_superficie_interna.sh "${objetivo}" >"${salida}" 2>&1; then
		printf 'El verificador interno acepto por error %s.\n' "${objetivo}" >&2
		exit 1
	fi
	if ! grep -Fq 'dependencias no aprobadas' "${salida}"; then
		printf 'El verificador interno fallo por una causa distinta al aislamiento de %s:\n' "${objetivo}" >&2
		cat "${salida}" >&2
		exit 1
	fi
	for prohibida in "$@"; do
		if ! grep -Fxq "  - ${modulo}/${prohibida}" "${salida}"; then
			printf 'No se rechazo %s al comprobar %s:\n' "${prohibida}" "${objetivo}" >&2
			cat "${salida}" >&2
			exit 1
		fi
	done
}

# vec-server compone rutas ajenas, incluidas las de gestion. La presencia
# transitiva del paquete administracion en CT V2 no sustituye esta negativa.
comprobar_rechazo ./cmd/vec-server \
	cmd/vec-server \
	internal/app/bootstrap \
	internal/app/composicion/publica
comprobar_rechazo ./cmd/vec-publico \
	cmd/vec-publico \
	internal/app/composicion/publica

printf 'Autopruebas negativas de las superficies integrada y publica superadas.\n'
