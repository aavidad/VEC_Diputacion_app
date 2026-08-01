#!/usr/bin/env bash

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    printf 'el auxiliar operativo F0 solo puede cargarse desde su runner acreditado\n' >&2
    exit 64
fi
if [[ "${VEC_F0_CARGA_PRIVADA:-}" != '1' ]]; then
    printf 'carga no acreditada del auxiliar operativo F0\n' >&2
    return 64
fi
unset VEC_F0_CARGA_PRIVADA

huella_contenedor_f0() {
    # shellcheck disable=SC2154
    docker exec "${contenedor}" sha256sum "$1" | awk '{print $1}'
}

descubrir_contenedor_propio_f0() {
    # shellcheck disable=SC2154
    docker container ls --all --no-trunc --filter "name=^/${contenedor}$" \
        --filter "label=es.dipgra.vep.f0.propietario=${propietario_contenedor}" \
        --format '{{.ID}}|{{.Names}}'
}

acreditar_hallazgo_contenedor_f0() {
    local hallado="$1" id inspeccion
    # shellcheck disable=SC2154
    [[ -n "${hallado}" && "${hallado}" != *$'\n'* ]] || return 65
    id="${hallado%%|*}"
    [[ "${id}" =~ ^[0-9a-f]{64}$ &&
       "${hallado}" == "${id}|${contenedor}" ]] || return 65
    capturar_salida_f0 inspeccion docker inspect --format \
        '{{.Id}}|{{.Name}}|{{ index .Config.Labels "es.dipgra.vep.f0.propietario" }}' \
        "${id}" || return 65
    [[ "${inspeccion}" == "${id}|/${contenedor}|${propietario_contenedor}" ]] ||
        return 65
    printf '%s' "${id}"
}

acreditar_cidfile_f0() {
    local id="$1" metadatos modo enlaces bytes
    # shellcheck disable=SC2154
    [[ -e "${cid_contenedor}" || -L "${cid_contenedor}" ]] || return 2
    [[ -f "${cid_contenedor}" && ! -L "${cid_contenedor}" ]] || return 65
    capturar_salida_f0 metadatos stat --printf='%a|%h|%s' -- \
        "${cid_contenedor}" || return 65
    IFS='|' read -r modo enlaces bytes <<<"${metadatos}" || return 65
    [[ "${modo}" == '600' && "${enlaces}" == '1' &&
       ("${bytes}" == '64' || "${bytes}" == '65') ]] || return 65
    if [[ "${bytes}" == '64' ]]; then
        dd if="${cid_contenedor}" iflag=nofollow,count_bytes count=66 \
            status=none | cmp --silent -- - <(printf '%s' "${id}")
    else
        dd if="${cid_contenedor}" iflag=nofollow,count_bytes count=66 \
            status=none | cmp --silent -- - <(printf '%s\n' "${id}")
    fi
}

retirar_contenedor_propio_f0() {
    local hallado id estado_cid=0
    # shellcheck disable=SC2154
    [[ "${intencion_contenedor}" == '1' &&
       "${propietario_contenedor}" =~ ^[0-9a-f]{64}$ ]] || return 0
    capturar_salida_f0 hallado descubrir_contenedor_propio_f0 || return 65
    [[ -z "${hallado}" || "${hallado}" != *$'\n'* ]] || return 65
    if [[ -z "${hallado}" ]]; then
        propietario_contenedor=''
        intencion_contenedor=''
        return 0
    fi
    capturar_salida_f0 id acreditar_hallazgo_contenedor_f0 "${hallado}" || return 65
    acreditar_cidfile_f0 "${id}" || {
        estado_cid=$?
        ((estado_cid == 2)) && estado_cid=0
    }
    if docker rm --force --volumes "${id}" >/dev/null 2>&1; then :; fi
    capturar_salida_f0 hallado descubrir_contenedor_propio_f0 || return 65
    [[ -z "${hallado}" ]] || return 65
    propietario_contenedor=''
    intencion_contenedor=''
    ((estado_cid == 0)) || return 65
}

comparar_huellas_f0() {
    local local_f0 contenedor_f0
    capturar_salida_f0 local_f0 huella_local_f0 "$1" || return 65
    capturar_salida_f0 contenedor_f0 huella_contenedor_f0 "$2" || return 65
    [[ "${local_f0}" == "${contenedor_f0}" ]]
}

exigir_salida_f0() {
    local esperada="$1" mensaje="$2" obtenida
    shift 2
    capturar_salida_f0 obtenida "$@" || fallar "${mensaje}"
    [[ "${obtenida}" == "${esperada}" ]] || fallar "${mensaje}"
}
