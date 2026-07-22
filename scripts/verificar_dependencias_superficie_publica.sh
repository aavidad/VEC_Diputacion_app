#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

modulo="$(go list -m -f '{{.Path}}')"
objetivo="${1:-./cmd/vec-publico}"
if (($# > 1)); then
	printf 'Uso: %s [paquete-objetivo]\n' "$0" >&2
	exit 2
fi
dependencias="$(mktemp)"
trap 'rm -f "${dependencias}"' EXIT

# Se inspeccionan todos los paquetes no estandar, no solo internal. Asi una
# dependencia futura de config hacia un SDK de nube, Vault, KMS o base de datos
# tambien exige ampliar conscientemente la lista positiva.
LC_ALL=C go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' "${objetivo}" |
	LC_ALL=C sed '/^$/d' | LC_ALL=C sort -u >"${dependencias}"

prohibidas=()
while IFS= read -r paquete; do
	case "${paquete}" in
		"${modulo}/cmd/vec-publico" | \
			"${modulo}/config" | \
			"${modulo}/internal/app/composicion/publica")
			;;
		*)
			prohibidas+=("${paquete}")
			;;
	esac
done <"${dependencias}"

if ((${#prohibidas[@]} != 0)); then
	printf 'El binario publico arrastra dependencias prohibidas:\n' >&2
	printf '  - %s\n' "${prohibidas[@]}" >&2
	exit 1
fi

if ! grep -Fxq "${modulo}/cmd/vec-publico" "${dependencias}" ||
	! grep -Fxq "${modulo}/config" "${dependencias}" ||
	! grep -Fxq "${modulo}/internal/app/composicion/publica" "${dependencias}"; then
	printf 'El binario publico no usa su raiz de composicion exclusiva.\n' >&2
	exit 1
fi

printf 'Grafo de dependencias de %s aislado: %s paquetes no estandar comprobados.\n' \
	"${objetivo}" "$(wc -l <"${dependencias}")"
