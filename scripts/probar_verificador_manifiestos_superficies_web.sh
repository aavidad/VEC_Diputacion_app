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

scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1

for catalogo in static/bolsa/i18n-publica.js static/verificar/i18n.js; do
	grep -Fxq "${catalogo}" "${temporal}/publico.manifest" || {
		printf 'La autoprueba requiere el catalogo i18n publico original: %s.\n' "${catalogo}" >&2
		exit 1
	}
	grep -Fxv "${catalogo}" "${temporal}/publico.manifest" >web/publico.manifest
	if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
		printf 'El verificador acepto la ausencia del catalogo i18n publico: %s.\n' "${catalogo}" >&2
		exit 1
	fi
	grep -Fxq "Falta recurso publico obligatorio: ${catalogo}" "${temporal}/salida" || {
		cat "${temporal}/salida" >&2
		exit 1
	}
	restaurar
done
restaurar

printf '%s\n' 'cartografia/granada-base-20260719-z8-z12.zip' >>web/publico.manifest
if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
	printf 'El verificador acepto el ZIP cartografico en la superficie publica.\n' >&2
	exit 1
fi
grep -Fq 'superficie publica incorpora cartografia' "${temporal}/salida" || {
	cat "${temporal}/salida" >&2
	exit 1
}
restaurar

printf '%s\n' 'cartografia/granada-base-20260719-z8-z12.zip' >>web/interno.manifest
if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
	printf 'El verificador acepto el ZIP cartografico duplicado.\n' >&2
	exit 1
fi
grep -Fq 'rutas duplicadas' "${temporal}/salida" || {
	cat "${temporal}/salida" >&2
	exit 1
}
restaurar

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

printf '%s\n' 'static/comun/tema-vec.js' >>web/publico.manifest
if scripts/verificar_manifiestos_superficies_web.sh >"${temporal}/salida" 2>&1; then
	printf 'El verificador acepto otro activo comun en la interseccion.\n' >&2
	exit 1
fi
grep -Fq 'Interseccion no autorizada entre manifiestos' "${temporal}/salida" || {
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

for manifiesto in publico.manifest interno.manifest interno.locales.manifest; do
	cmp -s "web/${manifiesto}" "${temporal}/${manifiesto}" || {
		printf 'La autoprueba no restauro exactamente %s.\n' "${manifiesto}" >&2
		exit 1
	}
done

printf 'Autoprueba negativa de manifiestos web superada.\n'
