#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

temporal="$(mktemp -d)"
trap 'rm -rf "${temporal}"' EXIT

cp web/publico.manifest "${temporal}/publico.manifest"
cp web/interno.manifest "${temporal}/interno.manifest"
cp web/interno.locales.manifest "${temporal}/interno.locales.manifest"

restaurar() {
	cp "${temporal}/publico.manifest" web/publico.manifest
	cp "${temporal}/interno.manifest" web/interno.manifest
	cp "${temporal}/interno.locales.manifest" web/interno.locales.manifest
}
trap 'restaurar; rm -rf "${temporal}"' EXIT

printf '%s\n' 'static/portal-empleado/index.html' >>web/publico.manifest
if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
	printf 'El verificador acepto un recurso interno en la superficie publica.\n' >&2
	exit 1
fi
grep -Fq 'superficie publica incorpora recursos' "${temporal}/salida" || {
	cat "${temporal}/salida" >&2
	exit 1
}
restaurar

printf '%s\n' 'static/bolsa/index.html' >>web/interno.manifest
if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
	printf 'El verificador acepto un recurso publico en la superficie interna.\n' >&2
	exit 1
fi
grep -Fq 'superficie interna incorpora recursos' "${temporal}/salida" || {
	cat "${temporal}/salida" >&2
	exit 1
}
restaurar

printf '%s\n' '../config/secreto.json' >>web/interno.locales.manifest
if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
	printf 'El verificador acepto una ruta no canonica de traduccion interna.\n' >&2
	exit 1
fi
grep -Fq 'traduccion interna no canonica' "${temporal}/salida" || {
	cat "${temporal}/salida" >&2
	exit 1
}
restaurar

printf 'Autoprueba negativa de manifiestos web superada.\n'
