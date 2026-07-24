#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if (($# != 2)); then
	printf 'Uso: %s IMAGEN_PUBLICA IMAGEN_INTERNA\n' "$0" >&2
	exit 2
fi

fallar() {
	printf '%s\n' "$*" >&2
	exit 1
}

probar_fallo_cerrado() {
	local superficie="$1"
	local imagen="$2"
	local salida estado

	set +e
	salida="$(timeout 15s docker run --rm \
		--network none \
		--read-only \
		--cap-drop ALL \
		--security-opt no-new-privileges \
		"${imagen}" 2>&1)"
	estado=$?
	set -e

	if ((estado == 0)); then
		fallar "${superficie}: el artefacto arranco sin sus proveedores productivos."
	fi
	if ((estado == 124)); then
		fallar "${superficie}: el artefacto permanecio vivo sin configuracion productiva."
	fi
	if [[ -z "${salida}" ]]; then
		fallar "${superficie}: el fallo cerrado no produjo un diagnostico saneado."
	fi
	if rg -ni \
		'postgres(?:ql)?://|mysql://|oracle://|password=|contrase[nñ]a=|token=|secret=|dsn=|/home/|/run/secrets/|(?:10|127)\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}|192\.168\.[0-9]{1,3}\.[0-9]{1,3}|172\.(?:1[6-9]|2[0-9]|3[01])\.[0-9]{1,3}\.[0-9]{1,3}' \
		<<<"${salida}" >/dev/null; then
		fallar "${superficie}: el diagnostico de arranque revelo configuracion sensible."
	fi
}

probar_fallo_cerrado publico "$1"
probar_fallo_cerrado interno "$2"

printf 'Artefactos productivos fallan cerrados y sin revelar configuracion.\n'
