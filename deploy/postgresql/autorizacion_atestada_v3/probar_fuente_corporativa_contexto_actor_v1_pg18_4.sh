#!/usr/bin/env bash
set -Eeuo pipefail
export LC_ALL=C
umask 077
directorio_entrada="$(dirname -- "${BASH_SOURCE[0]}")" || exit 65
directorio_script="$(cd -- "${directorio_entrada}" && pwd -P)" || exit 65
ruta_runner="${directorio_script}/${BASH_SOURCE[0]##*/}"
raiz="$(cd -- "${directorio_script}/../../.." && pwd -P)" || exit 65
cd -- "${raiz}" || exit 65
readonly ruta_helper_sql='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_helper_h0b='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_helper_operativo='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_capturador='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/capturar_snapshot_fuente_corporativa_contexto_actor_v1.go'
readonly destino_m080_h0b='/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/080_consumidor_nominal.sql'
readonly destino_t080_h0b='/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/080_consumidor_nominal.sql'
readonly destino_wrapper_h0b='/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/__ensayo_h0b.sql'
readonly sha256_helper_sql='a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5'
readonly sha256_helper_h0b='c5baff1e9d8febfa489a5cfd5aaf2004bdbc2f12b1b12881442902be8bdf1319'
readonly sha256_helper_operativo='8281ac2fe10a2c4609bfb7a87f68f69a1e71189d0d7a3ed946af231b866e2075'
readonly sha256_capturador='4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902'
readonly imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-f0-h0-${PPID}-${RANDOM}"
temporales='' raiz_base='' clave_postgres='' propietario_contenedor=''
intencion_contenedor='' cid_contenedor=''
etapa='H0' sustituto_autoprueba_bootstrap='' go_f0='' aleatorio_temporales=''
temporal_preausente='' temporal_propio='' identidad_temporales='' forma_temporales='' estado_mkdir_temporal=0
r0_posible='' finalizando_h0b='' wrapper_activo=''
fallar() { printf '[F0 H0] ERROR: %s\n' "$1" >&2; exit 1; }
paso() { printf '[F0 H0] %s\n' "$1"; }
limpiar() {
    local estado=$?
    trap - EXIT INT TERM
    if [[ "${r0_posible}" == '1' && -z "${finalizando_h0b}" ]]; then finalizar_h0b_f0 "${estado}" || estado=$?; fi
    if [[ "${intencion_contenedor}" == '1' ]] && ! retirar_contenedor_propio_f0; then
        printf '[F0 H0] ERROR: no se retiró el contenedor propio\n' >&2
        ((estado == 0)) && estado=1
    fi
    if ! retirar_directorio_temporal_f0; then
        printf '[F0 H0] ERROR: no se retiraron los temporales propios\n' >&2
        ((estado == 0)) && estado=1
    fi
    exit "${estado}"
}
limpiar_estricto_f0() {
    local estado=0
    retirar_contenedor_propio_f0 || estado=65
    retirar_directorio_temporal_f0 || estado=65
    return "${estado}"
}
metadatos_ruta_f0() { stat --printf='%d|%i|%f|%s|%y|%z|%h' -- "$1"; }
capturar_salida_f0() {
    local -n salida_f0="$1"; shift
    # shellcheck disable=SC2034
    salida_f0="$("$@")" || return 65
}
retirar_directorio_temporal_f0() {
    local actual forma
    [[ -n "${temporales}" ]] || return 0
    if [[ ! -e "${temporales}" && ! -L "${temporales}" ]]; then temporales='' temporal_preausente='' temporal_propio='' identidad_temporales=''; return 0; fi
    if [[ "${temporal_propio}" != '1' ]]; then
        [[ "${temporal_preausente}" == '1' ]] && capturar_salida_f0 forma stat --printf='%d|%i|%u|%F|%a|%h' -- "${temporales}" || return 65
        [[ "${forma}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|directory\|700\|2$ ]] || return 65
        identidad_temporales="${forma%|*}"; temporal_propio='1'
    fi
    capturar_salida_f0 actual stat --printf='%d|%i|%u|%F|%a' -- "${temporales}" || return 65
    [[ -d "${temporales}" && ! -L "${temporales}" && "${actual}" == "${identidad_temporales}" ]] || return 65
    if rm -rf -- "${temporales}"; then :; fi
    [[ ! -e "${temporales}" && ! -L "${temporales}" ]] || return 65
    temporales='' temporal_preausente='' temporal_propio='' identidad_temporales=''
}
huella_local_f0() { sha256sum -- "$1" | awk '{print $1}'; }
copiar_fuente_sin_enlaces_f0() {
    local fuente="$1" destino="$2" esperada="$3" gancho="${4:-}"
    local antes despues huella enlaces tamano
    [[ -f "${fuente}" && ! -L "${fuente}" ]] || return 65
    capturar_salida_f0 enlaces stat --printf='%h' -- "${fuente}" || return 65
    [[ "${enlaces}" == '1' ]] || return 65
    capturar_salida_f0 tamano stat --printf='%s' -- "${fuente}" || return 65
    [[ "${tamano}" =~ ^[0-9]+$ ]] && ((tamano > 0 && tamano <= 1048576)) || return 65
    antes="$(metadatos_ruta_f0 "${fuente}")" || return 65
    [[ -z "${gancho}" ]] || "${gancho}" "${fuente}" || return 65
    dd if="${fuente}" of="${destino}" iflag=nofollow,fullblock,count_bytes \
        count=1048577 conv=excl status=none || return 65
    chmod 0600 "${destino}" || return 65
    capturar_salida_f0 tamano stat --printf='%s' -- "${destino}" || return 65
    if ((tamano > 1048576)); then rm -- "${destino}" || return 65; return 65; fi
    despues="$(metadatos_ruta_f0 "${fuente}")" || return 65
    [[ ! -L "${fuente}" && "${antes}" == "${despues}" &&
       -f "${destino}" && ! -L "${destino}" ]] || return 65
    capturar_salida_f0 enlaces stat --printf='%h' -- "${destino}" || return 65
    [[ "${enlaces}" == '1' ]] || return 65
    capturar_salida_f0 huella huella_local_f0 "${destino}" || return 65
    [[ "${huella}" == "${esperada}" ]] || return 65
}
gancho_sustituir_por_enlace_f0() {
    mv -- "$1" "${1}.original" || return 65
    ln -s -- "${sustituto_autoprueba_bootstrap}" "$1" || return 65
    gancho_nofollow_construido=1
}
gancho_crecer_f0() { truncate -s 1048577 "$1" || return 65; }
probar_nofollow_bootstrap_f0() {
    local fuente="${temporales}/fuente-carrera.go" destino="${temporales}/destino-carrera.go" huella marca
    gancho_nofollow_construido=''
    printf 'package main\n' >"${fuente}" || return 65
    sustituto_autoprueba_bootstrap="${temporales}/sustituto-carrera.go"
    printf 'package main\n' >"${sustituto_autoprueba_bootstrap}" || return 65
    capturar_salida_f0 huella huella_local_f0 "${fuente}" || return 65
    if copiar_fuente_sin_enlaces_f0 "${fuente}" "${destino}" \
        "${huella}" gancho_sustituir_por_enlace_f0 2>/dev/null; then
        return 65
    fi
    [[ "${gancho_nofollow_construido}" == 1 && -L "${fuente}" &&
       "${fuente}" -ef "${sustituto_autoprueba_bootstrap}" && ! -e "${destino}" && ! -L "${destino}" ]] || return 65
    fuente="${temporales}/fuente-creciente.go"; destino="${temporales}/destino-creciente.go"
    printf 'package main\n' >"${fuente}" || return 65
    capturar_salida_f0 huella huella_local_f0 "${fuente}" || return 65
    if copiar_fuente_sin_enlaces_f0 "${fuente}" "${destino}" "${huella}" gancho_crecer_f0 2>/dev/null; then return 65; fi
    [[ -f "${fuente}" && ! -e "${destino}" ]] || return 65
    marca="${temporales}/hash-sobredimensionado"
    # shellcheck disable=SC2329
    sha256sum() { printf 'invocado\n' >"${marca}"; return 65; }
    if copiar_fuente_sin_enlaces_f0 "${fuente}" "${destino}" "${huella}" 2>/dev/null; then unset -f sha256sum; return 65; fi
    unset -f sha256sum
    [[ ! -e "${marca}" && ! -e "${destino}" ]] || return 65
}
seleccionar_toolchain_go_f0() {
    local go_base modulo_cache sistema arquitectura candidato goroot enlaces version
    go_base="$(command -v go)" || return 65
    modulo_cache="$(GOTOOLCHAIN=local "${go_base}" env GOMODCACHE)" || return 65
    sistema="$(GOTOOLCHAIN=local "${go_base}" env GOOS)" || return 65
    arquitectura="$(GOTOOLCHAIN=local "${go_base}" env GOARCH)" || return 65
    [[ "${sistema}/${arquitectura}" == 'linux/amd64' ]] || return 65
    candidato="${modulo_cache}/golang.org/toolchain@v0.0.1-go1.26.5.linux-amd64/bin/go"
    [[ -f "${candidato}" && ! -L "${candidato}" ]] || return 65
    capturar_salida_f0 enlaces stat --printf='%h' -- "${candidato}" || return 65
    [[ "${enlaces}" == '1' ]] || return 65
    capturar_salida_f0 version env GOTOOLCHAIN=local "${candidato}" version || return 65
    [[ "${version}" == 'go version go1.26.5 linux/amd64' ]] || return 65
    goroot="$(GOTOOLCHAIN=local "${candidato}" env GOROOT)" || return 65
    [[ "${goroot}/bin/go" -ef "${candidato}" ]] || return 65
    printf '%s' "${candidato}"
}
preparar_capturador_privado_f0() {
    local destino="${temporales}/capturador.go"
    local binario="${temporales}/capturador"
    copiar_fuente_sin_enlaces_f0 "${ruta_capturador}" "${destino}" \
        "${sha256_capturador}" || return 65
    env GOTOOLCHAIN=local GOWORK=off GOPROXY=off GOSUMDB=off \
        GONOSUMDB='*' GOFLAGS=-mod=readonly "${go_f0}" vet "${destino}" || return 65
    env GOTOOLCHAIN=local GOWORK=off GOPROXY=off GOSUMDB=off \
        GONOSUMDB='*' GOFLAGS=-mod=readonly "${go_f0}" build -race -trimpath \
        -o "${binario}" "${destino}" || return 65
    chmod 0700 "${binario}" || return 65
    "${binario}" --autoprueba >&2 || return 65
    printf '%s' "${binario}"
}
capturar_auxiliares_privados_f0() {
    local binario="$1"
    local snapshot="${temporales}/snapshot-auxiliares"
    local manifiesto="${temporales}/manifiesto-auxiliares" estado_directo
    local -a lineas=()
    "${binario}" --raiz . --destino "${snapshot}" \
        --manifiesto "${manifiesto}" -- "${ruta_helper_sql}" \
        "${ruta_helper_h0b}" "${ruta_helper_operativo}" || return 65
    mapfile -t lineas <"${manifiesto}" || return 65
    [[ ${#lineas[@]} -eq 3 &&
       "${lineas[0]}" == "${ruta_helper_sql}"$'\t'"${sha256_helper_sql}" &&
       "${lineas[1]}" == "${ruta_helper_h0b}"$'\t'"${sha256_helper_h0b}" &&
       "${lineas[2]}" == "${ruta_helper_operativo}"$'\t'"${sha256_helper_operativo}" ]] || return 65
    if bash "${snapshot}/${ruta_helper_sql}" >/dev/null 2>&1; then return 65; else estado_directo=$?; fi
    ((estado_directo == 64)) || return 65
    if bash "${snapshot}/${ruta_helper_operativo}" >/dev/null 2>&1; then return 65; else estado_directo=$?; fi
    ((estado_directo == 64)) || return 65
    if bash "${snapshot}/${ruta_helper_h0b}" >/dev/null 2>&1; then return 65; else estado_directo=$?; fi
    ((estado_directo == 64)) || return 65
    shellcheck -x "${ruta_runner}" "${snapshot}/${ruta_helper_sql}" \
        "${snapshot}/${ruta_helper_h0b}" "${snapshot}/${ruta_helper_operativo}" || return 65
    export VEC_F0_CARGA_PRIVADA=1
    # shellcheck source=/dev/null
    source "${snapshot}/${ruta_helper_sql}"
    export VEC_F0_CARGA_PRIVADA=1
    # shellcheck source=/dev/null
    source "${snapshot}/${ruta_helper_operativo}"
    export VEC_F0_CARGA_PRIVADA=1
    # shellcheck source=/dev/null
    source "${snapshot}/${ruta_helper_h0b}"
}
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

derivar_repo_base_h0_f0() {
    docker exec "${contenedor}" bash -ceu '
export LC_ALL=C
directorio=/repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes
archivos=(010_validadores.sql 020_canon_manifiesto.sql 030_canon_capacidad_mac.sql
  040_canon_consumo.sql 050_catalogo_fuente_checkpoint.sql 060_atestacion_consumo.sql
  070_acreditar_material_fuente.sql)
for archivo in "${archivos[@]}"; do
  nodo=$directorio/$archivo
  [[ -f $nodo && ! -L $nodo && $(stat --printf="%a|%h" -- "$nodo") == "600|1" ]] || exit 65
  rm -- "$nodo" || exit 65
done
[[ -d $directorio && ! -L $directorio && $(stat --printf=%a -- "$directorio") == 700 ]] || exit 65
[[ -z $(find "$directorio" -mindepth 1 -print -quit) ]] || exit 65
rmdir -- "$directorio" || exit 65
'
}

archivo() {
    docker exec "${contenedor}" psql -X --set ON_ERROR_STOP=1 --username "$1" --dbname postgres --file "/repo/$2"
}
sql() {
    docker exec "${contenedor}" psql -X --set ON_ERROR_STOP=1 --username "$1" --dbname postgres --command "$2"
}
valor() {
    docker exec "${contenedor}" psql -XAtq --set ON_ERROR_STOP=1 --username postgres --dbname postgres --command "$1"
}
probar_sqlstate_real_f0() {
    local codigo esperado ruta_error estado_salida obtenido
    paso 'clasificador contra errores reales de PostgreSQL 18.4'
    for codigo in 55P03 40P01 23505; do
        esperado='sqlstate=invalido'
        [[ "${codigo}" == '55P03' || "${codigo}" == '40P01' ]] &&
            esperado="sqlstate=${codigo}"
        ruta_error="$(mktemp "${temporales}/sqlstate-real.XXXXXX.err")" || fallar 'no se pudo reservar la captura SQLSTATE'
        if {
            printf '\\set VERBOSITY sqlstate\n'
            printf "DO \$bloque\$ BEGIN RAISE SQLSTATE '%s'; END \$bloque\$;\n" \
                "${codigo}"
        } | docker exec --env LC_ALL=C --interactive "${contenedor}" \
            psql -X --set ON_ERROR_STOP=1 --username postgres \
            --dbname postgres >/dev/null 2>"${ruta_error}"; then
            estado_salida=0
        else
            estado_salida=$?
        fi
        capturar_salida_f0 obtenido clasificar_sqlstate_psql_f0 \
            "${estado_salida}" "${ruta_error}" || fallar 'falló el clasificador SQLSTATE'
        [[ "${obtenido}" == "${esperado}" ]] ||
            fallar "SQLSTATE real mal clasificado: ${codigo}/${obtenido}"
    done
}
foto_catalogo() {
    docker exec "${contenedor}" pg_dump --schema-only --restrict-key=0000000000000000000000000000000000000000000000000000000000000000 --username postgres --dbname postgres | sha256sum | awk '{print $1}'
}

etapa_necesita_r0_f0() { [[ "$1" =~ ^(C2|C3|R1|R2a|R2b|T1|T2)$ ]]; }

ejecutar_etapa_dormida_f0() {
    local claves clave ruta relativa envoltorio destino estado=0 usuario
    claves="$(clausura_etapa_f0 "${etapa}")" || return 64
    envoltorio="$(mktemp "${temporales}/ensayo-etapa.XXXXXX.sql")" || return 65
    {
        printf '\\set ON_ERROR_STOP on\n\\set VERBOSITY sqlstate\n'
        printf '\\set AUTOCOMMIT off\nBEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;\n'
        printf "SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;\nSET LOCAL search_path=pg_catalog;\nSET LOCAL TimeZone='UTC';\nSET LOCAL lock_timeout='2s';\nSET LOCAL statement_timeout='10s';\nSET LOCAL transaction_timeout='15s';\nSET LOCAL idle_in_transaction_session_timeout='15s';\n"
        printf 'SELECT pg_catalog.txid_current() AS txid_f0 \\gset\n'
        for clave in ${claves}; do
            ruta="$(ruta_componente_f0 "${clave}")" || return 65
            if [[ "${clave}" == M* ]]; then
                relativa="../../migraciones/000007_componentes/${ruta##*/}"
            else
                relativa="./${ruta##*/}"
            fi
            printf 'SELECT 1/(pg_catalog.txid_current()=:txid_f0)::integer;\n'
            printf '\\ir %s\n' "${relativa}"
            printf 'SELECT 1/(pg_catalog.txid_current()=:txid_f0)::integer;\n'
        done
        printf 'ROLLBACK;\n'
    } >"${envoltorio}" || return 65
    destino='/repo/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/__ensayo_h0.sql'
    wrapper_activo="${destino}"
    docker cp "${envoltorio}" "${contenedor}:${destino}" || return 65
    comparar_huellas_f0 "${envoltorio}" "${destino}" || return 65
    usuario='vec_f0_h0_migrador'
    etapa_necesita_r0_f0 "${etapa}" && usuario='postgres'
    docker exec "${contenedor}" psql -X -v ON_ERROR_STOP=1 \
        --username "${usuario}" --dbname postgres --file "${destino}" || estado=$?
    docker exec "${contenedor}" rm -- "${destino}" || return 65
    wrapper_activo=''
    return "${estado}"
}

copiar_componente_sintetico_f0() {
    local origen="$1" destino="$2"
    validar_componentes_sql_f0 "${origen}" "${temporales}" || return 65
    docker cp "${origen}" "${contenedor}:${destino}" || return 65
    comparar_huellas_f0 "${origen}" "${destino}" || return 65
}

preparar_integracion_h0b_f0() {
    local modo="$1" directorio="${temporales}/integracion-h0b"
    local migracion="${directorio}/integracion-h0b-m080.sql" prueba="${directorio}/integracion-h0b-t080.sql" wrapper="${directorio}/integracion-h0b-wrapper.sql"
    docker exec "${contenedor}" mkdir --parents --mode=0700 "${destino_t080_h0b%/*}" && materializar_integracion_c2_virtual_h0b_f0 "${directorio}" "${modo}" || return 65
    copiar_componente_sintetico_f0 "${migracion}" "${destino_m080_h0b}" || return 65
    [[ "${modo}" == 'sin-r0' ]] || copiar_componente_sintetico_f0 "${prueba}" "${destino_t080_h0b}" || return 65
    wrapper_activo="${destino_wrapper_h0b}"
    docker cp "${wrapper}" "${contenedor}:${wrapper_activo}" || return 65
    comparar_huellas_f0 "${wrapper}" "${wrapper_activo}"
}

ejecutar_wrapper_h0b_f0() {
    local modo="$1" usuario='postgres' estado=0 salida="${temporales}/integracion-h0b.out" error="${temporales}/integracion-h0b.err"
    [[ "${modo}" != 'sin-r0' ]] || usuario='vec_f0_h0_migrador'
    docker exec "${contenedor}" psql -XAtq --set ON_ERROR_STOP=1 \
        --username "${usuario}" --dbname postgres --file "${wrapper_activo}" \
        >"${salida}" 2>"${error}" || estado=$?
    validar_resultado_wrapper_c2_virtual_h0b_f0 "${modo}" "${estado}" "${salida}" "${error}"
}

intentar_finalizacion_h0b_f0() {
    local -n estado_acumulado="$1"; local mensaje="$2"; shift 2
    if ! ("$@"); then
        printf '[F0 H0] ERROR: %s\n' "${mensaje}" >&2
        # shellcheck disable=SC2034
        estado_acumulado=65
    fi
}

finalizar_h0b_f0() {
    local causal="$1" estado_finalizacion=0 accion
    finalizando_h0b='1'
    [[ "${r0_posible}" != '1' ]] || intentar_finalizacion_h0b_f0 estado_finalizacion 'no se retiró R0' retirar_r0_sintetico_f0
    [[ -z "${wrapper_activo}" ]] || intentar_finalizacion_h0b_f0 estado_finalizacion 'no se retiró el wrapper' docker exec "${contenedor}" rm -f -- "${wrapper_activo}"
    [[ "${etapa}" != 'H0' ]] || intentar_finalizacion_h0b_f0 estado_finalizacion 'no se retiraron M080/T080' docker exec "${contenedor}" rm -f -- "${destino_m080_h0b}" "${destino_t080_h0b}"
    [[ "${etapa}" != 'H0' ]] || intentar_finalizacion_h0b_f0 estado_finalizacion 'no se retiró el directorio H0b' docker exec "${contenedor}" rmdir -- "${destino_t080_h0b%/*}" "${destino_t080_h0b%/*/*}"
    intentar_finalizacion_h0b_f0 estado_finalizacion 'R0 quedó presente' acreditar_r0_ausente_f0
    [[ "${etapa}" != 'H0' ]] || intentar_finalizacion_h0b_f0 estado_finalizacion 'la raíz H0b no volvió a base' acreditar_snapshot_contenedor_f0 "${manifiesto_sql}" /repo_h0b
    intentar_finalizacion_h0b_f0 estado_finalizacion 'la raíz base no volvió a base' acreditar_snapshot_contenedor_f0 "${manifiesto_sql_base}" /repo
    for accion in audiencia checkpoint catalogo roles objetos preparadas temporales sesiones; do
        intentar_finalizacion_h0b_f0 estado_finalizacion "no se acreditó ${accion}" acreditar_salida_base_h0b_f0 "${accion}" "${audiencia_base}" "${checkpoint_base}" "${catalogo_base}" "${roles_base}"
    done
    r0_posible=''; wrapper_activo=''; finalizando_h0b=''
    ((estado_finalizacion == 0)) || return 65
    return "${causal}"
}

probar_h0b_funcional_f0() {
    local estado=0
    mkdir --mode=0700 "${temporales}/autoprueba-h0b" "${temporales}/integracion-h0b" && probar_plantillas_c2_virtual_h0b_f0 "${temporales}/autoprueba-h0b" || return 65
    paso 'autoprueba pura H0b acreditada'
    preparar_integracion_h0b_f0 sin-r0 || estado=$?
    ((estado != 0)) || ejecutar_wrapper_h0b_f0 sin-r0 || estado=$?
    ((estado != 0)) || paso 'sin R0: SQLSTATE 42501 exacto acreditado'
    finalizar_h0b_f0 "${estado}" || return $?
    paso 'finalizador sin R0: rollback y línea base exacta acreditados'
    r0_posible='1'
    crear_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || acreditar_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || paso 'R0 canónico acreditado'
    ((estado != 0)) || preparar_integracion_h0b_f0 nominal || estado=$?
    ((estado != 0)) || ejecutar_wrapper_h0b_f0 nominal || estado=$?
    ((estado != 0)) || acreditar_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || paso 'integración virtual C2 nominal acreditada'
    ((estado != 0)) || preparar_integracion_h0b_f0 error || estado=$?
    ((estado != 0)) || ejecutar_wrapper_h0b_f0 error || estado=$?
    ((estado != 0)) || acreditar_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || paso 'error posterior: SQLSTATE 22012 exacto acreditado'
    finalizar_h0b_f0 "${estado}" || return $?
    paso 'finalizador R0: ausencia y ambas raíces/base exactas acreditadas'
}

probar_etapa_dormida_sintetica_f0() {
    local etapa_original="${etapa}" estado_error
    local migracion="${temporales}/010_validadores_m.sql" prueba="${temporales}/010_validadores_t.sql"
    local destino_m='/repo/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/010_validadores.sql'
    local destino_t='/repo/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/010_validadores.sql'
    docker exec "${contenedor}" mkdir --parents --mode=0700 "${destino_m%/*}" "${destino_t%/*}"
    printf '%s\n' 'CREATE TABLE vec_autorizacion_atestada_v3.autoprueba_etapa_h0(id integer);' >"${migracion}"
    printf '%s\n' 'INSERT INTO vec_autorizacion_atestada_v3.autoprueba_etapa_h0 VALUES (1);' >"${prueba}"
    copiar_componente_sintetico_f0 "${migracion}" "${destino_m}" || fallar 'falló M010 sintético: validación, copia o huella'
    copiar_componente_sintetico_f0 "${prueba}" "${destino_t}" || fallar 'falló T010 sintético: validación, copia o huella'
    etapa='A1'
    ejecutar_etapa_dormida_f0 >/dev/null ||
        fallar 'el camino nominal de etapa dormida falló'
    exigir_salida_f0 t 'el ROLLBACK nominal de etapa dejó residuos' valor \
        "SELECT pg_catalog.to_regclass('vec_autorizacion_atestada_v3.autoprueba_etapa_h0') IS NULL"
    printf '%s\n' 'INSERT INTO vec_autorizacion_atestada_v3.autoprueba_etapa_h0 VALUES (1); SELECT 1/0;' >"${prueba}"
    copiar_componente_sintetico_f0 "${prueba}" "${destino_t}" || fallar 'falló T010 sintético de error: validación, copia o huella'
    if ejecutar_etapa_dormida_f0 >/dev/null 2>&1; then fallar 'la etapa sintética con error fue aceptada'; else estado_error=$?; fi
    ((estado_error == 3)) || fallar 'el error sintético no procedía de psql'
    exigir_salida_f0 t 'el cierre de sesión tras error dejó residuos' valor \
        "SELECT pg_catalog.to_regclass('vec_autorizacion_atestada_v3.autoprueba_etapa_h0') IS NULL"
    etapa="${etapa_original}"
    docker exec "${contenedor}" rm -- "${destino_m}" "${destino_t}"
    docker exec "${contenedor}" rmdir -- "${destino_m%/*}" \
        "${destino_t%/*}" "${destino_t%/*/*}"
}
if (($# == 2)) && [[ "$1" == '--etapa' ]]; then
    etapa="$2"
elif (($# != 0)); then
    fallar 'uso: probar_fuente_corporativa_contexto_actor_v1_pg18_4.sh [--etapa ETAPA]'
fi
case "${etapa}" in
    H0|A1|A2|A3|A4|B1|B2|C1|C2|C3|R1|R2a|R2b|T1|T2) ;;
    *) fallar "etapa F0 no enumerada: ${etapa}" ;;
esac
for dependencia in bash chmod dd dirname docker env go ln mkdir mktemp mv openssl \
    awk cmp rm sha256sum shellcheck sleep stat tail truncate wc; do
    command -v "${dependencia}" >/dev/null 2>&1 ||
        fallar "dependencia local ausente: ${dependencia}"
done
[[ "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] ||
    fallar 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256'
docker image inspect "${imagen}" >/dev/null 2>&1 ||
    fallar 'la imagen PostgreSQL fijada no está disponible localmente'
capturar_salida_f0 aleatorio_temporales openssl rand -hex 32 || fallar 'no se pudo generar la reserva temporal'
[[ "${aleatorio_temporales}" =~ ^[0-9a-f]{64}$ && -d /tmp && ! -L /tmp ]] ||
    fallar 'reserva temporal inválida'
temporales="/tmp/vec-f0-h0-${aleatorio_temporales}"
trap limpiar EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
[[ ! -e "${temporales}" && ! -L "${temporales}" ]] || fallar 'la reserva temporal ya existe'
temporal_preausente='1'; estado_mkdir_temporal=0
mkdir --mode=0700 -- "${temporales}" || estado_mkdir_temporal=$?
capturar_salida_f0 forma_temporales stat --printf='%d|%i|%u|%F|%a|%h' -- "${temporales}" || fallar 'reserva temporal no acreditable'
[[ "${forma_temporales}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|directory\|700\|2$ ]] || fallar 'reserva temporal inválida'
identidad_temporales="${forma_temporales%|*}"; temporal_propio='1'
((estado_mkdir_temporal == 0)) || fallar 'mkdir falló después de reservar el temporal'
chmod 0700 "${temporales}" || fallar 'no se pudo proteger el directorio temporal'
capturar_salida_f0 raiz_base metadatos_ruta_f0 . ||
    fallar 'no se pudo acreditar la raíz física inicial'
probar_nofollow_bootstrap_f0 ||
    fallar 'la copia bootstrap siguió una sustitución simbólica'
go_f0="$(seleccionar_toolchain_go_f0)" ||
    fallar 'no está instalado el toolchain Go 1.26.5 local acreditado'
capturador="$(preparar_capturador_privado_f0)" ||
    fallar 'no se pudo acreditar y compilar el capturador privado'
capturar_auxiliares_privados_f0 "${capturador}" ||
    fallar 'no se pudieron acreditar y cargar los auxiliares privados'
paso 'autopruebas del analizador y clasificador'
probar_analizador_f0 "${temporales}"
probar_clasificador_f0 "${temporales}"
snapshot_sql=''
manifiesto_sql=''
manifiesto_sql_base=''
capturar_inventario_f0 "${capturador}" snapshot_sql manifiesto_sql ||
    fallar 'no se pudo capturar el inventario SQL exacto'
validar_componentes_snapshot_f0 "${snapshot_sql}" ||
    fallar 'la clausura SQL de etapa no es segura'
manifiesto_sql_base="${manifiesto_sql}"
if [[ "${etapa}" == 'H0' ]]; then
    manifiesto_sql_base="${temporales}/manifiesto-sql-base"
    derivar_manifiesto_base_h0_f0 "${manifiesto_sql}" "${manifiesto_sql_base}" ||
        fallar 'no se pudo derivar el manifiesto base H0'
fi
exigir_salida_f0 "${raiz_base}" 'la raíz física cambió durante el snapshot' metadatos_ruta_f0 .
paso "arranque PostgreSQL efímero sin red: ${imagen}"
capturar_salida_f0 clave_postgres openssl rand -hex 24 ||
    fallar 'no se pudo generar la clave efímera de PostgreSQL'
capturar_salida_f0 propietario_contenedor openssl rand -hex 32 ||
    fallar 'no se pudo generar la marca efímera del contenedor'
[[ "${propietario_contenedor}" =~ ^[0-9a-f]{64}$ ]] || fallar 'marca efímera de contenedor inválida'
cid_contenedor="${temporales}/contenedor.cid"
[[ ! -e "${cid_contenedor}" && ! -L "${cid_contenedor}" ]] || fallar 'ruta cidfile no exclusiva'
intencion_contenedor='1'
docker run --detach --name "${contenedor}" --network none \
    --label "es.dipgra.vep.f0.propietario=${propietario_contenedor}" \
    --cidfile "${cid_contenedor}" \
    --env POSTGRES_PASSWORD="${clave_postgres}" \
    --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
    --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=768m \
    "${imagen}" -c max_prepared_transactions=0 >/dev/null
chmod 0600 "${cid_contenedor}" || fallar 'no se pudo proteger el cidfile'
hallado_cid="$(descubrir_contenedor_propio_f0)" || fallar 'contenedor propio no descubrible'
id_contenedor="$(acreditar_hallazgo_contenedor_f0 "${hallado_cid}")" || fallar 'nombre, id y etiqueta del contenedor no coinciden'
acreditar_cidfile_f0 "${id_contenedor}" || fallar 'cidfile no coincide con el contenedor acreditado'
for _ in {1..60}; do
    docker exec "${contenedor}" pg_isready --quiet \
        --username postgres --dbname postgres && break
    sleep 1
done
docker exec "${contenedor}" pg_isready --quiet --username postgres --dbname postgres
docker exec "${contenedor}" mkdir --mode=0700 /repo
docker cp "${snapshot_sql}/." "${contenedor}:/repo"
if [[ "${etapa}" == 'H0' ]]; then
    docker exec "${contenedor}" mkdir --mode=0700 /repo_h0b
    docker cp "${snapshot_sql}/." "${contenedor}:/repo_h0b"
    derivar_repo_base_h0_f0 || fallar 'no se pudo derivar la raíz base H0'
fi
acreditar_snapshot_contenedor_f0 "${manifiesto_sql_base}" /repo ||
    fallar 'la raíz base no coincide byte a byte'
probar_snapshot_adverso_f0 "${manifiesto_sql_base}" /repo
if [[ "${etapa}" == 'H0' ]]; then
    acreditar_snapshot_contenedor_f0 "${manifiesto_sql}" /repo_h0b ||
        fallar 'la raíz H0b no coincide byte a byte'
    probar_snapshot_adverso_f0 "${manifiesto_sql}" /repo_h0b
fi
docker exec --interactive "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username postgres --dbname postgres <<'SQL'
REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC;
REVOKE ALL ON SCHEMA public FROM PUBLIC;
DO $version$
BEGIN
  IF current_setting('server_version_num')::integer <> 180004
     OR current_setting('max_prepared_transactions')::integer <> 0 THEN
    RAISE EXCEPTION 'se exige PostgreSQL 18.4 sin transacciones preparadas';
  END IF;
END
$version$;
SQL
paso 'dependencias reales del gobierno V3'
while IFS= read -r ruta; do archivo postgres "${ruta}" >/dev/null; done <<'RUTAS'
deploy/postgresql/contexto_actor_v1/roles_up.sql
deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
RUTAS
sql postgres "CREATE ROLE vec_contexto_f0_h0 LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS; GRANT vec_contexto_actor_v1_runtime TO vec_contexto_f0_h0 WITH ADMIN FALSE, INHERIT TRUE, SET FALSE" >/dev/null
docker exec --interactive "${contenedor}" psql -X --set ON_ERROR_STOP=1 \
    --username vec_contexto_f0_h0 --dbname postgres <<'SQL' >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT count(*) FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
 'oca_registro_v3_000000000000000000000000','rca_registro_v3_000000000000000000000000',
 'cta_sintetica_aaaaaaaaaaaaaaaaaaaaaaaa','prf_sintetico_cccccccccccccccccccccccc',
 'certificado','alto',clock_timestamp());
COMMIT;
SQL
sql postgres 'CREATE EXTENSION pgcrypto WITH SCHEMA public; REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM PUBLIC' >/dev/null
while IFS= read -r ruta; do archivo postgres "${ruta}" >/dev/null; done <<'RUTAS'
deploy/postgresql/autorizacion/roles_up.sql
deploy/postgresql/autorizacion/roles_v2_up.sql
deploy/postgresql/autorizacion/migraciones/000001_autorizacion.up.sql
deploy/postgresql/ejecucion_documental_v4/migraciones_autorizacion/000002_vinculo_autenticacion_actor_actual.up.sql
deploy/postgresql/autorizacion/migraciones/000003_proyeccion_motivos_autorizacion_v2.up.sql
deploy/postgresql/autorizacion/migraciones/000004_registro_decisiones_solicitud_ligada_v2.up.sql
deploy/postgresql/autorizacion/migraciones/000005_registro_decisiones_contexto_actor_v3.up.sql
deploy/postgresql/autorizacion/migraciones/000006_funcion_registro_decisiones_contexto_actor_v3.up.sql
deploy/postgresql/autorizacion/pruebas_sql/fixture_autorizacion_contexto_actor_v3.sql
deploy/postgresql/autorizacion/pruebas_sql/integracion_contexto_actor_v3.sql
deploy/postgresql/autorizacion/migraciones/000007_revalidacion_viva_decision_contexto_actor_v3.up.sql
deploy/postgresql/contratacion_temporal/roles_up.sql
RUTAS
sql postgres "CREATE ROLE vec_contratacion_temporal_consultor_rrhh NOLOGIN; GRANT CONNECT ON DATABASE postgres TO vec_contratacion_temporal_consultor_rrhh" >/dev/null
archivo postgres deploy/postgresql/autorizacion_atestada_v3/roles_up.sql >/dev/null
sql postgres "CREATE ROLE vec_f0_h0_migrador LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS; GRANT CONNECT ON DATABASE postgres TO vec_f0_h0_migrador; GRANT vec_autorizacion_atestada_v3_migrador TO vec_f0_h0_migrador WITH ADMIN FALSE, INHERIT FALSE, SET TRUE" >/dev/null
for migracion in 000001_gobierno_y_registro_v3 000002_consumidor_capacidad_v3 \
    000003_consumidor_consulta_cuadro_rrhh_v3 \
    000004_consumidor_consulta_detalle_rrhh_v3 \
    000005_revalidacion_final_consultas_rrhh_v3 \
    000006_prueba_consumo_consultas_rrhh_v3
do
    archivo vec_f0_h0_migrador "deploy/postgresql/autorizacion_atestada_v3/migraciones/${migracion}.up.sql" >/dev/null
done
audiencia_esperada="CHECK (audiencia_consumo = ANY (ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1'::text, 'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'::text, 'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'::text]))"
audiencia_base="$(definicion_audiencia)" || fallar 'no se pudo leer la audiencia base'
[[ "${audiencia_base}" == "${audiencia_esperada}" ]] ||
    fallar '000001..000006 no dejan exactamente tres audiencias'
checkpoint_base="$(foto_checkpoint)" || fallar 'no se pudo capturar el checkpoint base'
catalogo_base="$(foto_catalogo)" || fallar 'no se pudo capturar el catálogo base'
roles_base="$(foto_roles)" || fallar 'no se pudo capturar la topología de roles base'
exigir_salida_f0 '0' 'línea base H0 inválida' contar_objetos_f0
[[ -n "${checkpoint_base}" && -n "${catalogo_base}" &&
   -n "${roles_base}" ]] || fallar 'línea base H0 inválida'
if [[ "${etapa}" == 'H0' ]]; then
    paso 'autoprueba nominal y de error del arnés de etapas dormidas'
    probar_etapa_dormida_sintetica_f0
    acreditar_snapshot_contenedor_f0 "${manifiesto_sql_base}" /repo ||
        fallar 'H0a dejó residuos en la raíz base'
    acreditar_snapshot_contenedor_f0 "${manifiesto_sql}" /repo_h0b ||
        fallar 'H0a alteró la raíz H0b'
fi
acreditar_limpieza "${audiencia_base}" "${checkpoint_base}" \
    "${catalogo_base}" "${roles_base}"
[[ "${etapa}" != 'H0' ]] || probar_h0b_funcional_f0 || fallar 'falló el flujo funcional R0/H0b'
probar_sqlstate_real_f0
estado_etapa=0
if [[ "${etapa}" != 'H0' ]]; then
    if etapa_necesita_r0_f0 "${etapa}"; then
        r0_posible='1'
        crear_r0_sintetico_f0 || estado_etapa=$?
        ((estado_etapa != 0)) || ejecutar_etapa_dormida_f0 || estado_etapa=$?
        finalizar_h0b_f0 "${estado_etapa}" || estado_etapa=$?
    elif ejecutar_etapa_dormida_f0; then :; else estado_etapa=$?; fi
fi
acreditar_limpieza "${audiencia_base}" "${checkpoint_base}" \
    "${catalogo_base}" "${roles_base}"
((estado_etapa == 0)) || fallar "la etapa ${etapa} falló y fue revertida"
paso "${etapa} verde: snapshot y arnés transaccional acreditados"
limpiar_estricto_f0 || fallar 'no se pudo retirar el contenedor o los temporales'
trap - EXIT INT TERM
exigir_salida_f0 "${raiz_base}" 'la limpieza final cambió la raíz' metadatos_ruta_f0 .
[[ ! -e "${temporales}" ]] || fallar 'la limpieza final dejó recursos'
