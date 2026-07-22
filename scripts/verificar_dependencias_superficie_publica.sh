#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

artefacto_local="$(find . -type f \( -name '*.test' -o -name '*.prof' -o -name '*.pprof' \) -print -quit)"
if [[ -n "${artefacto_local}" ]]; then
	printf 'La superficie pública contiene un artefacto local no versionable: %s\n' "${artefacto_local}" >&2
	exit 1
fi

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
		"github.com/jackc/pgpassfile" | \
			"github.com/jackc/pgservicefile" | \
			"github.com/jackc/pgx/v5" | \
			"github.com/jackc/pgx/v5/internal/iobufpool" | \
			"github.com/jackc/pgx/v5/internal/pgio" | \
			"github.com/jackc/pgx/v5/internal/sanitize" | \
			"github.com/jackc/pgx/v5/internal/stmtcache" | \
			"github.com/jackc/pgx/v5/pgconn" | \
			"github.com/jackc/pgx/v5/pgconn/ctxwatch" | \
			"github.com/jackc/pgx/v5/pgconn/internal/bgreader" | \
			"github.com/jackc/pgx/v5/pgproto3" | \
			"github.com/jackc/pgx/v5/pgtype" | \
			"github.com/jackc/pgx/v5/pgxpool" | \
			"github.com/jackc/puddle/v2" | \
			"github.com/jackc/puddle/v2/internal/genstack" | \
			"golang.org/x/sync/semaphore" | \
			"golang.org/x/text/cases" | \
			"golang.org/x/text/internal" | \
			"golang.org/x/text/internal/language" | \
			"golang.org/x/text/internal/language/compact" | \
			"golang.org/x/text/internal/tag" | \
			"golang.org/x/text/language" | \
			"golang.org/x/text/runes" | \
			"golang.org/x/text/secure/bidirule" | \
			"golang.org/x/text/secure/precis" | \
			"golang.org/x/text/transform" | \
			"golang.org/x/text/unicode/bidi" | \
			"golang.org/x/text/unicode/norm" | \
			"golang.org/x/text/width" | \
			"${modulo}/cmd/vec-publico" | \
			"${modulo}/config" | \
			"${modulo}/internal/app/composicion/publica" | \
			"${modulo}/internal/app/server" | \
			"${modulo}/internal/modules/bolsa/adapters/postgrespublico" | \
			"${modulo}/internal/modules/bolsa/publico/aplicacion" | \
			"${modulo}/internal/modules/bolsa/publico/canonico" | \
			"${modulo}/internal/modules/bolsa/publico/dominio" | \
			"${modulo}/internal/modules/bolsa/publico/httpapi" | \
			"${modulo}/internal/modules/bolsa/publico/puertos" | \
			"${modulo}/internal/shared/limiteshttp" | \
			"${modulo}/internal/shared/postgresql")
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

for requerida in \
	"${modulo}/cmd/vec-publico" \
	"${modulo}/config" \
	"${modulo}/internal/app/composicion/publica" \
	"${modulo}/internal/modules/bolsa/adapters/postgrespublico" \
	"${modulo}/internal/modules/bolsa/publico/aplicacion" \
	"${modulo}/internal/modules/bolsa/publico/canonico" \
	"${modulo}/internal/modules/bolsa/publico/dominio" \
	"${modulo}/internal/modules/bolsa/publico/httpapi" \
	"${modulo}/internal/modules/bolsa/publico/puertos" \
	"${modulo}/internal/shared/limiteshttp" \
	"${modulo}/internal/shared/postgresql"; do
	if ! grep -Fxq "${requerida}" "${dependencias}"; then
		printf 'El binario publico no usa la dependencia autoritativa requerida: %s.\n' "${requerida}" >&2
		exit 1
	fi
done

if grep -Eq '/adapters/(fichero|memory|catalogosvec|httppublico)($|/)|publicatransitoria|/candidate($|/)|/modules/(cronos|dietas|personal)($|/)|/modules/bolsa/(application|domain|ports)($|/)|/internal/vec($|/)|/shared/baremacion($|/)' "${dependencias}"; then
	printf 'El binario publico no usa su raiz de composicion exclusiva.\n' >&2
	exit 1
fi

printf 'Grafo de dependencias de %s aislado: %s paquetes no estandar comprobados.\n' \
	"${objetivo}" "$(wc -l <"${dependencias}")"
