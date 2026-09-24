#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

manifiesto_publico="web/publico.manifest"
manifiesto_interno="web/interno.manifest"
manifiesto_locales_internos="web/interno.locales.manifest"
manifiesto_produccion="web/produccion.manifest"
cartografia_zip="cartografia/granada-base-20260719-z8-z12.zip"
cartografia_indice="cartografia/granada-base-20260719-z8-z12.json"
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
		if [[ "${ruta}" == produccion.manifest && "${nombre}" == produccion ]]; then
			:
		elif [[ "${ruta}" != "${cartografia_zip}" && "${ruta}" != "${cartografia_indice}" ]]; then
			[[ "${ruta}" =~ ^static/[A-Za-z0-9._/-]+$ ]] ||
				fallar "Ruta no canonica en ${nombre}: ${ruta}"
		elif [[ "${nombre}" == publico ]]; then
			fallar "La superficie publica incorpora cartografia interna: ${ruta}"
		fi
		[[ "${ruta}" != *".."* && "${ruta}" != *"//"* ]] ||
			fallar "Ruta no canonica en ${nombre}: ${ruta}"
		[[ -f "web/${ruta}" ]] || fallar "Falta el recurso declarado por ${nombre}: ${ruta}"
		[[ ! -L "web/${ruta}" ]] || fallar "Recurso enlazado en ${nombre}: ${ruta}"
		case "${ruta}" in
			*.zip)
				[[ "${ruta}" == "${cartografia_zip}" ]] ||
					fallar "ZIP no autorizado en ${nombre}: ${ruta}"
				;;
			*.json)
				case "${ruta}" in
					"${cartografia_indice}" | static/acceso/locales/es.json | \
						static/area-personal/locales/es.json | \
						static/portal-empleado/modulos/contratacion-temporal/formalizacion-desarrollo.json)
						;;
					*) fallar "JSON no autorizado en ${nombre}: ${ruta}" ;;
				esac
				;;
		esac
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
produccion="$(normalizar_manifiesto produccion "${manifiesto_produccion}")"

for ruta in "${cartografia_zip}" "${cartografia_indice}"; do
	grep -Fxq "${ruta}" "${interno}" || fallar "Falta cartografia interna: ${ruta}"
	grep -Fxq "${ruta}" "${produccion}" || fallar "Falta cartografia productiva: ${ruta}"
done
[[ ! -L web/cartografia ]] || fallar "El directorio cartografico es un enlace simbolico."

# La base es una copia cerrada del MBTiles historico; el indice fija la
# procedencia y las huellas de las 1040 teselas, sin prometer cobertura total.
if ! python3 - "web/${cartografia_zip}" "web/${cartografia_indice}" <<'PY'
import hashlib
import json
import re
import stat
import sys
import zipfile

zip_path, index_path = sys.argv[1:]
with open(index_path, encoding="utf-8") as source:
    index = json.load(source)
with open(zip_path, "rb") as source:
    zip_hash = hashlib.file_digest(source, "sha256").hexdigest()
with open(index_path, "rb") as source:
    index_hash = hashlib.file_digest(source, "sha256").hexdigest()
assert index_hash == "9d69233ad62c4371a2248bef2f91db8340823cb2a618272627693805ad021f81"
assert zip_hash == index["sha256_zip"] == "0f0d78212832493c424699a42847069ae24b8b3717917780c4d64444aa250165"
assert index["esquema"] == "vec.osm.png-export.v1"
assert index["sha256_mbtiles"] == "1ad538ef1f9331eca95137edc078d3a3b77336d54cabc838524172fba6547ebe"
assert (index["zoom_min"], index["zoom_max"], index["total"]) == (8, 12, 1040)
assert index["posiciones_bbox_importacion"] == 1246
assert len(index["sin_vector_en_bbox_importacion"]) == 206
tiles = index["teselas"]
assert len(tiles) == 1040
expected = {entry["ruta"]: entry["sha256"] for entry in tiles}
assert len(expected) == 1040
with zipfile.ZipFile(zip_path) as archive:
    members = archive.infolist()
    assert len(members) == 1040
    assert {member.filename for member in members} == set(expected)
    for member in members:
        assert re.fullmatch(r"tiles/(?:8|9|10|11|12)/[0-9]+/[0-9]+\.png", member.filename)
        assert stat.S_ISREG(member.external_attr >> 16)
        data = archive.read(member)
        assert data[:16] == b"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"
        assert int.from_bytes(data[16:20], "big") == int.from_bytes(data[20:24], "big") == 256
        assert hashlib.sha256(data).hexdigest() == expected[member.filename]
PY
then
	fallar "La cartografia empaquetada no coincide con su indice historico."
fi

if grep -Eq '^static/(portal-empleado|area-personal|presentacion|modulos)/' "${publico}"; then
	fallar "La superficie publica incorpora recursos de una superficie autenticada."
fi
if grep -Eq '^static/(bolsa|verificar|area-personal|presentacion|modulos)/' "${interno}"; then
	fallar "La superficie interna incorpora recursos publicos, externos o de presentacion."
fi

for requerida in \
	static/bolsa/index.html \
	static/bolsa/i18n-publica.js \
	static/verificar/i18n.js \
	static/verificar/index.html \
	static/assets/logo-diputacion-granada.svg \
	static/comun/tema-vec.css \
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
	static/comun/tema-vec.css \
	static/favicon.svg \
	static/styles.css | LC_ALL=C sort >"${esperados}"
if ! cmp -s "${compartidos}" "${esperados}"; then
	printf 'Interseccion no autorizada entre manifiestos:\n' >&2
	comm -3 "${esperados}" "${compartidos}" >&2 || true
	exit 1
fi

locales_internos="$(mktemp)"
temporales+=("${locales_internos}")
test -s "${manifiesto_locales_internos}" ||
	fallar "El manifiesto de traducciones internas falta o esta vacio."
while IFS= read -r ruta || [[ -n "${ruta}" ]]; do
	[[ "${ruta}" =~ ^[A-Za-z0-9._/-]+\.json$ ]] ||
		fallar "Ruta de traduccion interna no canonica: ${ruta}"
	[[ "${ruta}" != *".."* && "${ruta}" != *"//"* ]] ||
		fallar "Ruta de traduccion interna no canonica: ${ruta}"
	[[ -f "locales/${ruta}" ]] || fallar "Falta la traduccion interna: ${ruta}"
	printf '%s\n' "${ruta}" >>"${locales_internos}"
done <"${manifiesto_locales_internos}"
if [[ "$(LC_ALL=C sort -u "${locales_internos}" | wc -l)" -ne "$(wc -l <"${locales_internos}")" ]]; then
	fallar "El manifiesto de traducciones internas contiene rutas duplicadas."
fi
grep -Fxq es.json "${locales_internos}" || fallar "Falta la traduccion castellana obligatoria."

printf 'Manifiestos aislados: %s recursos publicos, %s internos, %s compartidos y %s traducciones internas.\n' \
	"$(wc -l <"${publico}")" "$(wc -l <"${interno}")" "$(wc -l <"${compartidos}")" \
	"$(wc -l <"${locales_internos}")"
