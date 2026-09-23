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

# C4 partia de una capsula TLS cerrada. El primer montaje de lectura CT añade
# exactamente el cierre transitivo observado de sus servicios, handler HTTP y
# pool nominal PostgreSQL. Algunos paquetes Go contienen tambien operaciones
# de otros modulos: su importacion NO habilita rutas, autoridades ni datos. La
# lista positiva detecta cualquier arista nueva; la composicion exige ademas
# una guarda explicita de capacidades antes del dispatcher HTTP.
LC_ALL=C go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' "${objetivo}" |
	LC_ALL=C sed '/^$/d' | LC_ALL=C sort -u >"${dependencias}"

prohibidas=()
while IFS= read -r paquete; do
	case "${paquete}" in
		github.com/jackc/pgpassfile | \
		github.com/jackc/pgx/v5 | \
		github.com/jackc/pgx/v5/internal/sanitize | \
		github.com/jackc/pgx/v5/internal/stmtcache | \
		github.com/jackc/pgx/v5/pgxpool | \
		github.com/jackc/puddle/v2 | \
		github.com/jackc/puddle/v2/internal/genstack | \
		golang.org/x/sync/semaphore | \
		github.com/jackc/pgservicefile | \
			github.com/jackc/pgx/v5/internal/iobufpool | \
			github.com/jackc/pgx/v5/internal/pgio | \
			github.com/jackc/pgx/v5/pgconn | \
			github.com/jackc/pgx/v5/pgconn/ctxwatch | \
			github.com/jackc/pgx/v5/pgconn/internal/bgreader | \
			github.com/jackc/pgx/v5/pgproto3 | \
			github.com/jackc/pgx/v5/pgtype | \
			golang.org/x/text/cases | \
			golang.org/x/text/internal | \
			golang.org/x/text/internal/language | \
			golang.org/x/text/internal/language/compact | \
			golang.org/x/text/internal/tag | \
			golang.org/x/text/language | \
			golang.org/x/text/runes | \
			golang.org/x/text/secure/bidirule | \
			golang.org/x/text/secure/precis | \
			golang.org/x/text/unicode/bidi | \
			golang.org/x/text/width | \
			"${modulo}/cmd/vec-interno" | \
			"${modulo}/config" | \
		"${modulo}/internal/app/composicion/interna" | \
		"${modulo}/internal/modules/administracion" | \
		"${modulo}/internal/modules/bolsa" | \
		"${modulo}/internal/modules/contrataciontemporal/adapters/ginpixfichero" | \
		"${modulo}/internal/modules/contrataciontemporal/adapters/historiaincorporacion" | \
		"${modulo}/internal/modules/contrataciontemporal/adapters/httpinterno" | \
		"${modulo}/internal/modules/contrataciontemporal/adapters/postgres" | \
		"${modulo}/internal/modules/contrataciontemporal/application" | \
		"${modulo}/internal/modules/contrataciontemporal/application/diagnostico" | \
		"${modulo}/internal/modules/contrataciontemporal/cobertura" | \
		"${modulo}/internal/modules/contrataciontemporal/domain" | \
		"${modulo}/internal/modules/contrataciontemporal/ports" | \
		"${modulo}/internal/modules/cronos" | \
		"${modulo}/internal/modules/dietas" | \
		"${modulo}/internal/modules/personal" | \
		"${modulo}/internal/modules/personal/adapters/contrataciontemporal" | \
		"${modulo}/internal/modules/personal/adapters/fuenteejercicio" | \
		"${modulo}/internal/modules/personal/adapters/lecturaincorporacion" | \
		"${modulo}/internal/modules/personal/application" | \
		"${modulo}/internal/modules/personal/domain" | \
		"${modulo}/internal/modules/personal/ports" | \
		"${modulo}/internal/shared/i18n" | \
		"${modulo}/internal/vec/adapters/contextoactor/postgres" | \
		"${modulo}/internal/vec/adapters/httpapi" | \
		"${modulo}/internal/vec/adapters/httpseguridad/postgres" | \
		"${modulo}/internal/vec/adapters/postgres" | \
		"${modulo}/internal/vec/application" | \
		"${modulo}/internal/vec/canonico/almacen" | \
		"${modulo}/internal/vec/canonico/documental" | \
		"${modulo}/internal/vec/canonico/pagos" | \
		"${modulo}/internal/vec/canonico/recibomaterial" | \
		"${modulo}/internal/vec/ports" | \
			"${modulo}/internal/app/server" | \
			"${modulo}/internal/shared/limiteshttp" | \
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
	"${modulo}/internal/modules/contrataciontemporal/adapters/httpinterno" \
	"${modulo}/internal/modules/contrataciontemporal/adapters/postgres" \
	"${modulo}/internal/modules/contrataciontemporal/application" \
	"${modulo}/internal/vec/adapters/httpapi" \
	"${modulo}/internal/vec/adapters/httpseguridad" \
	"${modulo}/internal/vec/domain"; do
	if ! grep -Fxq "${obligatoria}" "${dependencias}"; then
		printf 'Falta una dependencia obligatoria del esqueleto interno: %s\n' "${obligatoria}" >&2
		exit 1
	fi
done

printf 'Grafo de dependencias de %s aislado: %s paquetes no estandar comprobados.\n' \
	"${objetivo}" "$(wc -l <"${dependencias}")"
