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

verificar_artefactos() {
    cantidad=0
    for archivo in "/data/${nombre_datos}.osrm" "/data/${nombre_datos}.osrm."*; do
        if [ ! -f "$archivo" ] || [ -L "$archivo" ]; then
            echo "artefacto OSRM ausente, irregular o simbolico: $archivo" >&2
            exit 66
        fi
        if [ ! -r "$archivo" ] || [ -w "$archivo" ]; then
            echo "artefacto OSRM no es de solo lectura para el proceso: $archivo" >&2
            exit 66
        fi
        cantidad=$((cantidad + 1))
    done
    if [ "$cantidad" -lt 4 ]; then
        echo "conjunto de artefactos OSRM incompleto" >&2
        exit 66
    fi
}

normalizar_artefactos() {
    if [ "$(id -u)" -ne 0 ]; then
        echo "la normalizacion de artefactos exige el usuario root efimero" >&2
        exit 77
    fi
    cantidad=0
    for archivo in "/data/${nombre_datos}.osrm" "/data/${nombre_datos}.osrm."*; do
        if [ ! -f "$archivo" ] || [ -L "$archivo" ]; then
            echo "artefacto OSRM ausente, irregular o simbolico: $archivo" >&2
            exit 66
        fi
        chmod 0444 -- "$archivo"
        cantidad=$((cantidad + 1))
    done
    if [ "$cantidad" -lt 4 ]; then
        echo "conjunto de artefactos OSRM incompleto" >&2
        exit 66
    fi
}

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
    normalizar)
        normalizar_artefactos
        ;;
    verificar_lectura)
        verificar_artefactos
        ;;
    servir)
        verificar_artefactos
        exec osrm-routed --algorithm mld --max-table-size 50000 \
            --verbosity WARNING \
            "/data/${nombre_datos}.osrm"
        ;;
    *)
        echo "operacion OSRM no permitida" >&2
        exit 64
        ;;
esac
