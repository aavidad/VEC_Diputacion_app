#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

manifiesto_publico="web/publico.manifest"
manifiesto_interno="web/interno.manifest"
temporales=()

limpiar() {
	rm -f "${temporales[@]}"
}
trap limpiar EXIT

fallar() {
	printf '%s\n' "$*" >&2
	exit 1
}

normalizar_manifiesto() {
	local nombre="$1"
	local manifiesto="$2"
	local normalizado
	normalizado="$(mktemp)"
	temporales+=("${normalizado}")

	test -s "${manifiesto}" || fallar "El manifiesto ${nombre} falta o esta vacio."
	while IFS= read -r ruta || [[ -n "${ruta}" ]]; do
		[[ -n "${ruta}" ]] || fallar "El manifiesto ${nombre} contiene lineas vacias."
		[[ "${ruta}" =~ ^static/[A-Za-z0-9._/-]+$ ]] ||
			fallar "Ruta no canonica en ${nombre}: ${ruta}"
		[[ "${ruta}" != *".."* && "${ruta}" != *"//"* ]] ||
			fallar "Ruta no canonica en ${nombre}: ${ruta}"
		[[ -f "web/${ruta}" ]] || fallar "Falta el recurso declarado por ${nombre}: ${ruta}"
		case "${ruta}" in
			*.test.js | *.test.mjs | *test-helper* | *datos-presentacion* | \
				*adaptador-presentacion* | *portal-presentacion* | */presentacion/* | \
				*demo* | *DEMO*)
				fallar "${nombre} incorpora un recurso de prueba o presentacion: ${ruta}"
				;;
		esac
		printf '%s\n' "${ruta}" >>"${normalizado}"
	done <"${manifiesto}"

	if [[ "$(LC_ALL=C sort -u "${normalizado}" | wc -l)" -ne "$(wc -l <"${normalizado}")" ]]; then
		fallar "El manifiesto ${nombre} contiene rutas duplicadas."
	fi
	LC_ALL=C sort -o "${normalizado}" "${normalizado}"
	printf '%s' "${normalizado}"
}

publico="$(normalizar_manifiesto publico "${manifiesto_publico}")"
interno="$(normalizar_manifiesto interno "${manifiesto_interno}")"

if grep -Eq '^static/(portal-empleado|area-personal|presentacion|modulos)/' "${publico}"; then
	fallar "La superficie publica incorpora recursos de una superficie autenticada."
fi
if grep -Eq '^static/(bolsa|verificar|area-personal|presentacion|modulos)/' "${interno}"; then
	fallar "La superficie interna incorpora recursos publicos, externos o de presentacion."
fi

for requerida in \
	static/bolsa/index.html \
	static/verificar/index.html \
	static/assets/logo-diputacion-granada.svg \
	static/styles.css \
	static/favicon.svg; do
	grep -Fxq "${requerida}" "${publico}" || fallar "Falta recurso publico obligatorio: ${requerida}"
done

for requerida in \
	static/portal-empleado/index.html \
	static/portal-empleado/portal.js \
	static/assets/logo-diputacion-granada.svg \
	static/styles.css \
	static/favicon.svg; do
	grep -Fxq "${requerida}" "${interno}" || fallar "Falta recurso interno obligatorio: ${requerida}"
done

compartidos="$(mktemp)"
esperados="$(mktemp)"
temporales+=("${compartidos}" "${esperados}")
LC_ALL=C comm -12 "${publico}" "${interno}" >"${compartidos}"
printf '%s\n' \
	static/assets/logo-diputacion-granada.svg \
	static/favicon.svg \
	static/styles.css | LC_ALL=C sort >"${esperados}"
if ! cmp -s "${compartidos}" "${esperados}"; then
	printf 'Interseccion no autorizada entre manifiestos:\n' >&2
	comm -3 "${esperados}" "${compartidos}" >&2 || true
	exit 1
fi

printf 'Manifiestos web aislados: %s recursos publicos, %s internos y %s compartidos autorizados.\n' \
	"$(wc -l <"${publico}")" "$(wc -l <"${interno}")" "$(wc -l <"${compartidos}")"
