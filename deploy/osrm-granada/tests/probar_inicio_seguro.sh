#!/bin/sh

set -eu

directorio=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
inicio="$directorio/iniciar_osrm.sh"
compose="$directorio/docker-compose.yml"

OSRM_DATA_BASENAME=granada-buffer /bin/sh "$inicio" validar
OSRM_DATA_BASENAME=jaen_2026 /bin/sh "$inicio" validar

for nombre_hostil in \
    'granada; id' \
    '../granada' \
    'granada.buffer' \
    'Granada' \
    'granada buffer' \
    '-granada' \
    'granada-' \
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
do
    if OSRM_DATA_BASENAME="$nombre_hostil" /bin/sh "$inicio" validar 2>/dev/null; then
        echo "se acepto un nombre de datos OSRM no permitido" >&2
        exit 1
    fi
done

if grep -En -- 'sh[[:space:]]+-c' "$compose"; then
    echo "el compose OSRM no puede ejecutar comandos mediante sh -c" >&2
    exit 1
fi

if ! grep -Fq -- '--verbosity WARNING' "$inicio"; then
    echo "OSRM debe ocultar las coordenadas de las rutas en sus registros" >&2
    exit 1
fi

if ! grep -Fq 'user: "65534:65534"' "$compose"; then
    echo "el servicio OSRM debe ejecutarse sin privilegios" >&2
    exit 1
fi

docker compose -f "$compose" config --quiet

configuracion=$(OSRM_DATA_BASENAME=jaen_2026 \
    docker compose -f "$compose" --profile build config)
apariciones=$(printf '%s\n' "$configuracion" | \
    grep -Fc 'OSRM_DATA_BASENAME: jaen_2026')
if [ "$apariciones" -ne 5 ]; then
    echo "el nombre del conjunto no llega a todos los servicios OSRM" >&2
    exit 1
fi
