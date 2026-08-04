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

# shellcheck disable=SC2154
acreditar_snapshot_contenedor_f0() {
    local manifiesto="$1" raiz_contenedor="${2:-/repo}" reconstruido
    [[ "${raiz_contenedor}" == '/repo' || "${raiz_contenedor}" == '/repo_h0b' ]] || return 64
    reconstruido="${temporales}/manifiesto-contenedor-${raiz_contenedor#/}"
    docker exec "${contenedor}" bash -ceu '
export LC_ALL=C; set -o pipefail; raiz=$1; modo=$(stat --printf=%a "$raiz") || exit 65; [[ ! -L $raiz && -d $raiz && $modo == 700 ]] || exit 65
declare -a lineas=(); tab=$(printf "\t") || exit 65; salto=$(printf "\n_") || exit 65; salto=${salto%_}; lista=$(mktemp) || exit 65
find "$raiz" -mindepth 1 -print0 >"$lista" || exit 65
while IFS= read -r -d "" nodo; do
  relativa=${nodo#"$raiz"/}
  [[ $relativa != *"$tab"* && $relativa != *"$salto"* ]] || exit 65
  if [[ -L $nodo ]]; then exit 65
  elif [[ -d $nodo ]]; then
    modo=$(stat --printf=%a -- "$nodo") || exit 65; contiene=$(find "$nodo" -mindepth 1 -type f -print -quit) || exit 65
    [[ $modo == 700 && -n $contiene ]] || exit 65
  elif [[ -f $nodo ]]; then
    metadatos=$(stat --printf="%a|%h" -- "$nodo") || exit 65; [[ $metadatos == "600|1" ]] || exit 65
    huella=$(sha256sum -- "$nodo" | awk "{print \$1}") || exit 65; lineas+=("$relativa$tab$huella")
  else exit 65; fi
done <"$lista"
((${#lineas[@]} > 0)) || exit 65; printf "%s\n" "${lineas[@]}" | sort || exit 65
' -- "${raiz_contenedor}" >"${reconstruido}" || return 65
    cmp --silent -- "${manifiesto}" "${reconstruido}"
}
rechazar_snapshot_adverso_f0() { ! acreditar_snapshot_contenedor_f0 "$1" "$2" || fallar "$3"; }
probar_snapshot_adverso_f0() {
    local manifiesto="$1" raiz_contenedor="${2:-/repo}" fichero directorio nodo
    fichero="${raiz_contenedor}/deploy/postgresql/contexto_actor_v1/roles_up.sql"
    directorio="${raiz_contenedor}/deploy/postgresql/contexto_actor_v1"
    nodo="${raiz_contenedor}/__adverso_f0"
    docker exec "${contenedor}" ln -s "${raiz_contenedor}" "${nodo}"
    rechazar_snapshot_adverso_f0 "${manifiesto}" "${raiz_contenedor}" 'el snapshot admitió un enlace simbólico'
    docker exec "${contenedor}" rm -- "${nodo}"
    docker exec "${contenedor}" ln "${fichero}" "${nodo}"
    rechazar_snapshot_adverso_f0 "${manifiesto}" "${raiz_contenedor}" 'el snapshot admitió un enlace duro'
    docker exec "${contenedor}" rm -- "${nodo}"
    docker exec "${contenedor}" chmod 0644 "${fichero}"
    rechazar_snapshot_adverso_f0 "${manifiesto}" "${raiz_contenedor}" 'el snapshot admitió modo de fichero inseguro'
    docker exec "${contenedor}" chmod 0600 "${fichero}"
    docker exec "${contenedor}" chmod 0755 "${directorio}"
    rechazar_snapshot_adverso_f0 "${manifiesto}" "${raiz_contenedor}" 'el snapshot admitió modo de directorio inseguro'
    docker exec "${contenedor}" chmod 0700 "${directorio}"
    docker exec "${contenedor}" mkdir --mode=0700 "${nodo}"
    rechazar_snapshot_adverso_f0 "${manifiesto}" "${raiz_contenedor}" 'el snapshot admitió un directorio adicional'
    docker exec "${contenedor}" rmdir -- "${nodo}"
    acreditar_snapshot_contenedor_f0 "${manifiesto}" "${raiz_contenedor}" || fallar 'el snapshot no se restauró'
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
