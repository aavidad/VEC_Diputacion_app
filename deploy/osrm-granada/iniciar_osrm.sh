#!/bin/sh

set -eu

LC_ALL=C
export LC_ALL
umask 077

nombre_datos=${OSRM_DATA_BASENAME:-granada-buffer}

case "$nombre_datos" in
    *[!a-z0-9_-]*)
        echo "nombre del conjunto de datos OSRM no permitido" >&2
        exit 64
        ;;
esac

case "$nombre_datos" in
    [a-z0-9]|[a-z0-9]*[a-z0-9])
        ;;
    *)
        echo "nombre del conjunto de datos OSRM no permitido" >&2
        exit 64
        ;;
esac

if [ "${#nombre_datos}" -gt 64 ]; then
    echo "nombre del conjunto de datos OSRM no permitido" >&2
    exit 64
fi

case "${1:-}" in
    validar)
        exit 0
        ;;
    extraer)
        exec osrm-extract -p /opt/car.lua "/data/${nombre_datos}.osm.pbf"
        ;;
    particionar)
        exec osrm-partition "/data/${nombre_datos}.osrm"
        ;;
    personalizar)
        exec osrm-customize "/data/${nombre_datos}.osrm"
        ;;
    servir)
        exec osrm-routed --algorithm mld --max-table-size 50000 \
            "/data/${nombre_datos}.osrm"
        ;;
    *)
        echo "operacion OSRM no permitida" >&2
        exit 64
        ;;
esac
