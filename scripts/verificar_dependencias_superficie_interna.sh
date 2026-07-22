#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

modulo="$(go list -m -f '{{.Path}}')"
objetivo="${1:-./cmd/vec-interno}"
if (($# > 1)); then
	printf 'Uso: %s [paquete-objetivo]\n' "$0" >&2
	exit 2
fi
dependencias="$(mktemp)"
trap 'unlink "${dependencias}" 2>/dev/null || true' EXIT

# C4 admite exactamente la configuracion, el contrato de superficie y el
# dominio de identidad que valida dicho contrato. C5/C6 ampliaran esta lista
# de forma deliberada al aportar proveedores productivos reales.
LC_ALL=C go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' "${objetivo}" |
	LC_ALL=C sed '/^$/d' | LC_ALL=C sort -u >"${dependencias}"

prohibidas=()
while IFS= read -r paquete; do
	case "${paquete}" in
		"${modulo}/cmd/vec-interno" | \
			"${modulo}/config" | \
			"${modulo}/internal/app/composicion/interna" | \
			"${modulo}/internal/app/server" | \
			"${modulo}/internal/vec/adapters/httpseguridad" | \
			"${modulo}/internal/vec/domain" | \
			golang.org/x/text/transform | \
			golang.org/x/text/unicode/norm)
			;;
		*)
			prohibidas+=("${paquete}")
			;;
	esac
done <"${dependencias}"

if ((${#prohibidas[@]} != 0)); then
	printf 'El esqueleto interno arrastra dependencias no aprobadas:\n' >&2
	printf '  - %s\n' "${prohibidas[@]}" >&2
	exit 1
fi

for obligatoria in \
	"${modulo}/cmd/vec-interno" \
	"${modulo}/config" \
	"${modulo}/internal/app/composicion/interna" \
	"${modulo}/internal/app/server" \
	"${modulo}/internal/vec/adapters/httpseguridad" \
	"${modulo}/internal/vec/domain"; do
	if ! grep -Fxq "${obligatoria}" "${dependencias}"; then
		printf 'Falta una dependencia obligatoria del esqueleto interno: %s\n' "${obligatoria}" >&2
		exit 1
	fi
done

printf 'Grafo de dependencias de %s aislado: %s paquetes no estandar comprobados.\n' \
	"${objetivo}" "$(wc -l <"${dependencias}")"
