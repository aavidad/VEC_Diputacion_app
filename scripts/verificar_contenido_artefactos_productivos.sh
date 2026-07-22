#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

if (($# < 1 || $# > 2)); then
	printf 'Uso: %s IMAGEN_PUBLICA [IMAGEN_INTERNA]\n' "$0" >&2
	exit 2
fi

imagen_publica="$1"
imagen_interna="${2:-}"
temporal="$(mktemp -d)"
contenedores=()
declare -A contenedor_por_superficie=()

limpiar() {
	for contenedor in "${contenedores[@]}"; do
		docker rm -f "${contenedor}" >/dev/null 2>&1 || true
	done
	rm -rf "${temporal}"
}
trap limpiar EXIT

fallar() {
	printf '%s\n' "$*" >&2
	exit 1
}

extraer_superficie() {
	local superficie="$1"
	local imagen="$2"
	local destino="${temporal}/${superficie}"
	local contenedor
	mkdir -p "${destino}/web" "${destino}/bin"
	contenedor="$(docker create "${imagen}")"
	contenedores+=("${contenedor}")
	contenedor_por_superficie["${superficie}"]="${contenedor}"
	docker cp "${contenedor}:/app/web/." "${destino}/web"
	docker cp "${contenedor}:/usr/local/bin/." "${destino}/bin"
}

verificar_locales_internos() {
	local contenedor="${contenedor_por_superficie[interno]}"
	local destino="${temporal}/interno/locales"
	local esperado="${temporal}/interno/locales-esperados"
	local real="${temporal}/interno/locales-reales"
	mkdir -p "${destino}"
	docker cp "${contenedor}:/app/locales/." "${destino}"
	LC_ALL=C sort -u web/interno.locales.manifest >"${esperado}"
	find "${destino}" -type f -printf '%P\n' | LC_ALL=C sort >"${real}"
	cmp -s "${esperado}" "${real}" ||
		fallar "interno: el inventario de traducciones no coincide con su manifiesto."
	if find "${destino}" -type l -print -quit | grep -q . ||
		find "${destino}" -type f -perm /022 -print -quit | grep -q .; then
		fallar "interno: una traduccion es enlazable o modificable en ejecucion."
	fi
}

verificar_configuracion() {
	local superficie="$1"
	local imagen="$2"
	local binario="$3"
	local usuario entrada entorno historial
	usuario="$(docker image inspect "${imagen}" --format '{{.Config.User}}')"
	entrada="$(docker image inspect "${imagen}" --format '{{join .Config.Entrypoint " "}}')"
	entorno="$(docker image inspect "${imagen}" --format '{{range .Config.Env}}{{println .}}{{end}}')"
	historial="$(docker history --no-trunc --format '{{.CreatedBy}}' "${imagen}")"

	[[ "${usuario}" == app ]] || fallar "${superficie}: la imagen no usa el usuario no privilegiado app."
	[[ "${entrada}" == "/usr/local/bin/${binario}" ]] ||
		fallar "${superficie}: punto de entrada inesperado: ${entrada}"
	if grep -Eqi '(^|_)(PASSWORD|CONTRASENA|SECRET|TOKEN|PRIVATE_KEY|DSN|KMS|TSA|POSTGRESQL)=' <<<"${entorno}"; then
		fallar "${superficie}: la configuracion de imagen incorpora secretos o destinos sensibles."
	fi
	if grep -Eqi '(PASSWORD|CONTRASENA|SECRET|TOKEN|PRIVATE_KEY)=' <<<"${historial}"; then
		fallar "${superficie}: el historial de capas contiene una asignacion sensible."
	fi
}

verificar_inventario() {
	local superficie="$1"
	local manifiesto="$2"
	local binario="$3"
	local destino="${temporal}/${superficie}"
	local esperado real
	esperado="${destino}/esperado"
	real="${destino}/real"
	if find "${destino}/web" "${destino}/bin" -type l -print -quit | grep -q .; then
		fallar "${superficie}: el artefacto contiene enlaces simbolicos en su aplicacion."
	fi
	if find "${destino}/web" -type f -perm /022 -print -quit | grep -q .; then
		fallar "${superficie}: un recurso web puede ser modificado por el usuario de ejecucion."
	fi

	cmp -s "${manifiesto}" "${destino}/web/produccion.manifest" ||
		fallar "${superficie}: el manifiesto incluido no coincide con el revisado."
	{
		printf '%s\n' produccion.manifest
		cat "${manifiesto}"
	} | LC_ALL=C sort >"${esperado}"
	find "${destino}/web" -type f -printf '%P\n' | LC_ALL=C sort >"${real}"
	if ! cmp -s "${esperado}" "${real}"; then
		printf '%s: inventario web distinto del manifiesto:\n' "${superficie}" >&2
		comm -3 "${esperado}" "${real}" >&2 || true
		exit 1
	fi

	mapfile -t binarios < <(find "${destino}/bin" -maxdepth 1 -type f -printf '%f\n' | LC_ALL=C sort)
	((${#binarios[@]} == 1)) && [[ "${binarios[0]}" == "${binario}" ]] ||
		fallar "${superficie}: el artefacto no contiene exclusivamente ${binario}."
	[[ -x "${destino}/bin/${binario}" ]] || fallar "${superficie}: el binario no es ejecutable."
	if find "${destino}/bin/${binario}" -perm /022 -print -quit | grep -q .; then
		fallar "${superficie}: el binario extraido conserva permiso de escritura."
	fi
}

extraer_superficie publico "${imagen_publica}"
verificar_configuracion publico "${imagen_publica}" vec-publico
verificar_inventario publico web/publico.manifest vec-publico

if docker cp "${contenedor_por_superficie[publico]}:/app/locales/." \
	"${temporal}/publico/locales-no-autorizados" >/dev/null 2>&1; then
	fallar "publico: el artefacto incorpora traducciones de la superficie interna."
fi

if rg -n '/api/vec|/portal-empleado|/area-personal|credentials[[:space:]]*:[[:space:]]*.include|document\.cookie|localStorage|sessionStorage' \
	"${temporal}/publico/web" >/dev/null; then
	fallar "publico: el cliente contiene una ruta interna o estado de sesion prohibido."
fi

if [[ -z "${imagen_interna}" ]]; then
	printf 'Artefacto publico verificado; la comprobacion C2 completa requiere tambien la imagen interna.\n'
	exit 0
fi

extraer_superficie interno "${imagen_interna}"
verificar_configuracion interno "${imagen_interna}" vec-interno
verificar_inventario interno web/interno.manifest vec-interno
verificar_locales_internos

# Una superficie puede enlazar a otra a traves del proxy de borde (por
# ejemplo, abrir la consulta publica o verificar un recibo). Lo que no puede
# hacer el cliente interno es consumir directamente la API anonima ni incluir
# sus recursos: esto ultimo ya queda cerrado por el manifiesto exacto.
if rg -n '/api/publico(?:/|[?"'"'"'])' "${temporal}/interno/web" >/dev/null; then
	fallar "interno: el cliente intenta consumir directamente la API publica."
fi

printf 'Artefactos productivos publico e interno aislados y conformes con sus manifiestos.\n'
