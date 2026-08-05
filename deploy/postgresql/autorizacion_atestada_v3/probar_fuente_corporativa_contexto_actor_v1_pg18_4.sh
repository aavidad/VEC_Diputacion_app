#!/usr/bin/bash -p
set -Eeuo pipefail
[[ "$-" == *p* ]] || exit 65
for ruta_sistema_f0 in /usr/bin/bash /usr/bin/env /usr/bin/grep; do
    [[ -f "${ruta_sistema_f0}" && -x "${ruta_sistema_f0}" && ! -w "${ruta_sistema_f0}" ]] || exit 65
done
unset ruta_sistema_f0
estados_entorno=()
if /usr/bin/env -0 |
    /usr/bin/grep -zE '^(BASH_FUNC_|LD_)' >/dev/null
then
    estados_entorno=("${PIPESTATUS[@]}")
    exit 65
else
    estados_entorno=("${PIPESTATUS[@]}")
fi
((${#estados_entorno[@]} == 2)) || exit 65
[[ "${estados_entorno[0]}" == 0 &&
   "${estados_entorno[1]}" == 1 ]] || exit 65
unset estados_entorno
export LC_ALL=C
umask 077
unset BASH_ENV ENV
modo_m38='' selector_inyeccion_h0b='' identidad_m38='' cid_esperado_m38=''
forma_temporal_m38='' forma_runner_m38='' forma_raiz_m38=''
if (($# > 0)) && [[ "$1" == '--caso-inyeccion-h0b' ]]; then
    (($# == 2)) || exit 64
    selector_inyeccion_h0b="$2"
    IFS='|' read -r padre_m38 caso_m38 identidad_m38 cid_esperado_m38 td ti tu tt tm th rd ri ru rt rm rh rs rsha zd zi zu zt zm zh <&9 2>/dev/null || exit 64
    if IFS= read -r _ <&9 2>/dev/null; then exit 64; fi
    exec 9<&-
    [[ -n "${zh}" && "${zh}" != *'|'* && "${padre_m38}" == "${PPID}" &&
       "${caso_m38}" == "${selector_inyeccion_h0b}" &&
       "${identidad_m38}" =~ ^[0-9a-f]{64}$ &&
       "${cid_esperado_m38}" =~ ^[0-9a-f]{64}$ &&
       "${td}|${ti}|${tu}|${tt}|${tm}|${th}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|directory\|700\|2$ &&
       "${rd}|${ri}|${ru}|${rt}|${rm}|${rh}|${rs}|${rsha}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|regular\ file\|600\|0\|[0-9]+\|[0-9a-f]{64}$ &&
       "${zd}|${zi}|${zu}|${zt}|${zm}|${zh}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|directory\|[0-7]{3,4}\|[0-9]+$ &&
       "${selector_inyeccion_h0b}" =~ ^(A0[1-3]|[NE](0[1-9]|10)|F(0[1-9]|1[0-5])|NOMINAL)$ ]] || exit 64
    forma_temporal_m38="${td}|${ti}|${tu}|${tt}|${tm}|${th}"
    forma_runner_m38="${rd}|${ri}|${ru}|${rt}|${rm}|${rh}|${rs}|${rsha}"
    forma_raiz_m38="${zd}|${zi}|${zu}|${zt}|${zm}|${zh}"
    set +m
    [[ "$-" != *m* ]] || exit 64
    builtin kill -STOP "${BASHPID}"
    modo_m38='hijo'; set -- --etapa H0
fi
if [[ "${modo_m38}" == hijo ]]; then
    ruta_runner=/proc/self/fd/8; raiz=/proc/self/fd/7
else
    directorio_entrada="$(dirname -- "${BASH_SOURCE[0]}")" || exit 65
    directorio_script="$(cd -- "${directorio_entrada}" && pwd -P)" || exit 65
    ruta_runner="${directorio_script}/${BASH_SOURCE[0]##*/}"
    raiz="$(cd -- "${directorio_script}/../../.." && pwd -P)" || exit 65
fi
cd -- "${raiz}" || exit 65
readonly ruta_helper_sql='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_helper_h0b='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/arnes_r0_sintetico_h0b_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_helper_operativo='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/operaciones_runner_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_adaptador_m38='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/ciclo_recursos_m38_h0b_fuente_corporativa_contexto_actor_v1.sh'
readonly ruta_supervisor_m38='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go'
readonly ruta_capturador='deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/capturar_snapshot_fuente_corporativa_contexto_actor_v1.go'
readonly sha256_helper_sql='a07057fb15315c5d2d0d10d6f3beea85f196fc78598cfcc4d1f63918bcbadde5'
readonly sha256_helper_h0b='02a00f2fc49e181d1cf8ed147a927155899956dbdbd7f36f3443ee4d7cbafded'
readonly sha256_helper_operativo='9b137f1302c5672e9fd5c0c8df169810cbc7e57a11fa2129bf79a777e92c5e81'
readonly sha256_adaptador_m38='98d22a302bfd8ad3964b9135ce78c655f7a31171088ad9c5c49c285f647a8cb7'
readonly sha256_supervisor_m38='a1bbb941baa91cbc6e32f565542cb68b1548a31214d17d4b8a8c9ee87f764f95'
readonly sha256_binario_supervisor_m38='4bd9e83093a345528b690197eec129fbe22cb029dc5f0ace40e82dd0333079a8'
readonly sha256_capturador='4a967fd13bac213ea7ebf7316af98dcc9a9dfb39b9b3b28f68e0c91958878902'
readonly imagen="${VEC_POSTGRES_TEST_IMAGE:-postgres@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296}"
contenedor="vec-f0-h0-${PPID}-${RANDOM}"
[[ -z "${identidad_m38}" ]] || contenedor="vec-f0-h0-${identidad_m38:0:32}"
temporales='' raiz_base='' clave_postgres='' propietario_contenedor="${identidad_m38}"
intencion_contenedor='' cid_contenedor=''
etapa='H0' sustituto_autoprueba_bootstrap='' go_f0=''
aleatorio_temporales='' supervisor_m38=''
temporal_preausente='' temporal_propio='' identidad_temporales='' forma_temporales='' estado_mkdir_temporal=0
r0_posible='' finalizando_h0b='' wrapper_activo='' inyeccion_h0b_activa='' traza_m38_activa=''
traza_finalizador_h0b='' caso_observado_h0b='' recuperacion_interna_h0b='' seccion_critica_m38='' senal_pendiente_m38='' generacion_senal_m38=0
fallar() { printf '[F0 H0] ERROR: %s\n' "$1" >&2; exit 1; }
paso() { printf '[F0 H0] %s\n' "$1"; }
gestionar_senal_m38_f0() {
    [[ " ${FUNCNAME[*]:1} " != *" gestionar_senal_m38_f0 "* ]] || return 0
    [[ -n "${identidad_activa_m38:-}" && ( -n "${seccion_critica_m38:-}" || -n "${senal_pendiente_m38:-}" ) ]] || exit "$1"
    [[ -n "${senal_pendiente_m38:-}" ]] || senal_pendiente_m38="$1"
    generacion_senal_m38=$((generacion_senal_m38 + 1))
}
limpiar() {
    local estado=$?
    trap - EXIT INT TERM
    senal_pendiente_m38=''
    if [[ "${r0_posible}" == '1' && -z "${finalizando_h0b}" ]]; then finalizar_h0b_f0 "${estado}" || estado=$?; fi
    if [[ "${modo_m38}" == hijo ]]; then exit "${estado}"; fi
    if declare -F retirar_recursos_m38_f0 >/dev/null && ! retirar_recursos_m38_f0; then estado=65; fi
    if declare -F restaurar_regimen_shell_m38_f0 >/dev/null && ! restaurar_regimen_shell_m38_f0; then estado=65; fi
    if [[ "${intencion_contenedor}" == '1' ]] && ! retirar_contenedor_propio_f0; then
        printf '[F0 H0] ERROR: no se retiró el contenedor propio\n' >&2
        estado=65
    fi
    if ! retirar_directorio_temporal_f0; then
        printf '[F0 H0] ERROR: no se retiraron los temporales propios\n' >&2
        estado=65
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
entorno_go_aislado_f0() {
    [[ -z "${GOAMD64:-}" ]] || return 65
    /usr/bin/env -i HOME="${temporales}" TMPDIR="${temporales}" GOCACHE="${temporales}/cache-go" \
        PATH=/usr/local/go/bin:/usr/bin:/bin LC_ALL=C GOENV=off GOAMD64=v1 GOTELEMETRY=off \
        GOTOOLCHAIN=local GOWORK=off GOPROXY=off GOSUMDB=off GONOSUMDB='*' \
        GOFLAGS=-mod=readonly "$@"
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
    local go_base=go modulo_cache sistema_arquitectura candidato goroot enlaces version home_sistema
    home_sistema="$(/usr/bin/awk -F: -v uid="${EUID}" '$3 == uid {print $6}' /etc/passwd)" || return 65
    modulo_cache="$(entorno_go_aislado_f0 GOMODCACHE="${home_sistema}/go/pkg/mod" "${go_base}" env GOMODCACHE)" || return 65
    sistema_arquitectura="$(entorno_go_aislado_f0 "${go_base}" env GOOS GOARCH)" || return 65
    [[ "${sistema_arquitectura}" == $'linux\namd64' ]] || return 65
    candidato="${modulo_cache}/golang.org/toolchain@v0.0.1-go1.26.5.linux-amd64/bin/go"
    [[ -f "${candidato}" && ! -L "${candidato}" ]] || return 65
    capturar_salida_f0 enlaces stat --printf='%h' -- "${candidato}" || return 65
    [[ "${enlaces}" == '1' ]] || return 65
    capturar_salida_f0 version entorno_go_aislado_f0 "${candidato}" version || return 65
    [[ "${version}" == 'go version go1.26.5 linux/amd64' ]] || return 65
    goroot="$(entorno_go_aislado_f0 "${candidato}" env GOROOT)" || return 65
    [[ "${goroot}/bin/go" -ef "${candidato}" ]] || return 65
    printf '%s' "${candidato}"
}
preparar_capturador_privado_f0() {
    local destino="${temporales}/capturador.go"
    local binario="${temporales}/capturador"
    copiar_fuente_sin_enlaces_f0 "${ruta_capturador}" "${destino}" \
        "${sha256_capturador}" || return 65
    entorno_go_aislado_f0 GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
        "${go_f0}" vet "${destino}" || return 65
    entorno_go_aislado_f0 GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
        "${go_f0}" build -race -trimpath -o "${binario}" "${destino}" || return 65
    chmod 0700 "${binario}" || return 65
    "${binario}" --autoprueba >&2 || return 65
    printf '%s' "${binario}"
}
capturar_auxiliares_privados_f0() {
    local binario="$1" snapshot="${temporales}/snapshot-auxiliares"
    local manifiesto="${temporales}/manifiesto-auxiliares" estado_directo fuente fuente_go forma_binario huella_binario huella_fuente_post
    local -a lineas=()
    "${binario}" --raiz . --destino "${snapshot}" \
        --manifiesto "${manifiesto}" -- "${ruta_helper_sql}" "${ruta_helper_h0b}" \
        "${ruta_adaptador_m38}" \
        "${ruta_helper_operativo}" "${ruta_supervisor_m38}" || return 65
    mapfile -t lineas <"${manifiesto}" || return 65
    [[ ${#lineas[@]} -eq 5 &&
       "${lineas[0]}" == "${ruta_helper_sql}"$'\t'"${sha256_helper_sql}" &&
       "${lineas[1]}" == "${ruta_helper_h0b}"$'\t'"${sha256_helper_h0b}" &&
       "${lineas[2]}" == "${ruta_adaptador_m38}"$'\t'"${sha256_adaptador_m38}" &&
       "${lineas[3]}" == "${ruta_helper_operativo}"$'\t'"${sha256_helper_operativo}" &&
       "${lineas[4]}" == "${ruta_supervisor_m38}"$'\t'"${sha256_supervisor_m38}" ]] || return 65
    shellcheck -x "${ruta_runner}" "${snapshot}/${ruta_helper_sql}" "${snapshot}/${ruta_helper_h0b}" \
        "${snapshot}/${ruta_helper_operativo}" "${snapshot}/${ruta_adaptador_m38}" || return 65
    [[ ! -v destino_m080_h0b && ! -v destino_t080_h0b && ! -v directorio_wrapper_h0b &&
       ! -v destino_wrapper_h0b &&
       ! -v destino_wrapper_nominal_h0b && ! -v destino_wrapper_error_h0b ]] || return 65
    for fuente in "${ruta_helper_sql}" "${ruta_helper_operativo}" "${ruta_helper_h0b}" "${ruta_adaptador_m38}"; do
        if bash "${snapshot}/${fuente}" >/dev/null 2>&1; then
            return 65
        else
            estado_directo=$?
        fi
        ((estado_directo == 64)) || return 65
        [[ ! -v VEC_F0_CARGA_PRIVADA ]] || return 65
        export VEC_F0_CARGA_PRIVADA=1
        # shellcheck source=/dev/null
        source "${snapshot}/${fuente}" || return 65
        [[ ! -v VEC_F0_CARGA_PRIVADA ]] || return 65
    done
    [[ -v destino_m080_h0b && -v destino_t080_h0b && -v directorio_wrapper_h0b &&
       -v destino_wrapper_h0b &&
       -v destino_wrapper_nominal_h0b && -v destino_wrapper_error_h0b ]] || return 65
    [[ "${destino_m080_h0b@a}${destino_t080_h0b@a}${directorio_wrapper_h0b@a}${destino_wrapper_h0b@a}${destino_wrapper_nominal_h0b@a}${destino_wrapper_error_h0b@a}" == rrrrrr ]] || return 65
    [[
       "${destino_m080_h0b}|${destino_t080_h0b}|${directorio_wrapper_h0b}" == '/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/080_consumidor_nominal.sql|/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/080_consumidor_nominal.sql|/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/__h0b' &&
       "${destino_wrapper_h0b}|${destino_wrapper_nominal_h0b}|${destino_wrapper_error_h0b}" == '/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/__h0b/sin-r0.sql|/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/__h0b/nominal/ensayo.sql|/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/__h0b/error/ensayo.sql'
    ]] || return 65
    fuente_go="${snapshot}/${ruta_supervisor_m38}"
    supervisor_m38="${temporales}/supervisor-m38"
    [[ -f "${fuente_go}" && ! -L "${fuente_go}" &&
       "$(stat --printf='%a|%u|%F|%h' -- "${fuente_go}")" == "600|${EUID}|regular file|1" &&
       "$(huella_local_f0 "${fuente_go}")" == "${sha256_supervisor_m38}" ]] || return 65
    entorno_go_aislado_f0 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
        "${go_f0}" vet "${fuente_go}" || return 65
    entorno_go_aislado_f0 GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
        "${go_f0}" build -trimpath -o "${supervisor_m38}" "${fuente_go}" || return 65
    chmod 0700 "${supervisor_m38}" || return 65
    forma_binario="$(/usr/bin/stat --printf='%a|%u|%F|%h' -- "${supervisor_m38}")" || return 65
    huella_fuente_post="$(/usr/bin/sha256sum -- "${fuente_go}")" || return 65
    huella_binario="$(/usr/bin/sha256sum -- "${supervisor_m38}")" || return 65
    [[ "${huella_fuente_post}" == "${sha256_supervisor_m38}  ${fuente_go}" ]] || return 65
    [[ ! -L "${supervisor_m38}" ]] || return 65
    [[ "${forma_binario}" == "700|${EUID}|regular file|1" ]] || return 65
    [[ "${huella_binario}" == "${sha256_binario_supervisor_m38}  ${supervisor_m38}" ]] || return 65
    "${supervisor_m38}" --autoprueba >&2 || return 65
    if "${supervisor_m38}" --modo-desconocido >/dev/null 2>&1; then
        return 65
    else
        estado_directo=$?
    fi
    ((estado_directo == 64)) || return 65
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
inyectar_frontera_h0b_f0() {
    [[ -n "${1:-}" && "${inyeccion_h0b_activa}" == 1 &&
       "${selector_inyeccion_h0b}" == "$1" ]] || return 0
    caso_observado_h0b="$1"
    return 79
}
intentar_finalizacion_h0b_f0() {
    local -n estado_acumulado="$1"; local caso="$2" mensaje="$3"; shift 3
    if [[ "${traza_m38_activa}" == 1 &&
          ( -n "${traza_finalizador_h0b}" || "${selector_inyeccion_h0b}" != F* ||
            "${selector_inyeccion_h0b}" == "${caso}" ) ]]; then
        traza_finalizador_h0b+="${traza_finalizador_h0b:+,}${caso}"
    fi
    if [[ "${inyeccion_h0b_activa}" == 1 && "${selector_inyeccion_h0b}" == "${caso}" ]]; then
        [[ -n "${caso_observado_h0b}" ]] || caso_observado_h0b="${caso}"
        estado_acumulado=65
    elif ! "$@"; then
        printf '[F0 H0] ERROR: %s\n' "${mensaje}" >&2
        # shellcheck disable=SC2034 # asignación mediante nameref
        estado_acumulado=65
        [[ "${selector_inyeccion_h0b}" != F* || -n "${caso_observado_h0b}" ]] || caso_observado_h0b=INVALIDO
    fi
}
finalizar_h0b_f0() {
    local causal="$1" estado_finalizacion=0 accion accion_f0=8
    # shellcheck disable=SC2154 # constantes del adaptador privado
    local -a wrappers=("${destino_wrapper_h0b}" "${destino_wrapper_nominal_h0b}" "${destino_wrapper_error_h0b}")
    [[ "${etapa}" == H0 ]] || wrappers=("${wrapper_activo}")
    finalizando_h0b=1
    [[ "${r0_posible}" != 1 && "${inyeccion_h0b_activa}" != 1 ]] || intentar_finalizacion_h0b_f0 estado_finalizacion F01 'no se retiró R0' retirar_r0_sintetico_f0
    [[ -z "${wrapper_activo}" && "${inyeccion_h0b_activa}" != 1 ]] || intentar_finalizacion_h0b_f0 estado_finalizacion F02 'no se retiró el wrapper' retirar_archivos_h0b_f0 "${wrappers[@]}"
    # shellcheck disable=SC2154 # constantes del adaptador privado
    [[ "${etapa}" != H0 ]] || intentar_finalizacion_h0b_f0 estado_finalizacion F03 'no se retiraron M080/T080' retirar_archivos_h0b_f0 "${destino_m080_h0b}" "${destino_t080_h0b}"
    [[ "${etapa}" != H0 ]] || intentar_finalizacion_h0b_f0 estado_finalizacion F04 'no se retiró el directorio H0b' retirar_directorios_wrapper_h0b_f0
    intentar_finalizacion_h0b_f0 estado_finalizacion F05 'R0 quedó presente' acreditar_r0_ausente_f0
    [[ "${etapa}" != H0 ]] || intentar_finalizacion_h0b_f0 estado_finalizacion F06 'la raíz H0b no volvió a base' acreditar_snapshot_contenedor_f0 "${manifiesto_sql}" /repo_h0b
    intentar_finalizacion_h0b_f0 estado_finalizacion F07 'la raíz base no volvió a base' acreditar_snapshot_contenedor_f0 "${manifiesto_sql_base}" /repo
    for accion in audiencia checkpoint catalogo roles objetos preparadas temporales sesiones; do
        intentar_finalizacion_h0b_f0 estado_finalizacion "F$(printf '%02d' "${accion_f0}")" "no se acreditó ${accion}" acreditar_salida_base_h0b_f0 "${accion}" "${audiencia_base}" "${checkpoint_base}" "${catalogo_base}" "${roles_base}"
        accion_f0=$((accion_f0 + 1))
    done
    [[ "${traza_m38_activa}" != 1 ]] || traza_finalizador_h0b+="${traza_finalizador_h0b:+,}TERMINAL"
    r0_posible='' wrapper_activo='' finalizando_h0b=''
    ((estado_finalizacion == 0)) || { inyeccion_h0b_activa=''; return 65; }
    recuperacion_interna_h0b=1; inyeccion_h0b_activa=''
    return "${causal}"
}
probar_h0b_funcional_f0() {
    local estado=0 recuperacion=exterior
    mkdir --mode=0700 "${temporales}/autoprueba-h0b" &&
        probar_plantillas_c2_virtual_h0b_f0 "${temporales}/autoprueba-h0b" || return 65
    paso 'autoprueba pura H0b acreditada'
    mkdir --mode=0700 "${temporales}/integracion-h0b-sin-r0" || return 65
    # shellcheck disable=SC2154 # constante del adaptador privado
    crear_directorio_wrapper_h0b_f0 "${directorio_wrapper_h0b}" || return 65
    preparar_integracion_h0b_f0 sin-r0 || estado=$?
    ((estado != 0)) || ejecutar_wrapper_h0b_f0 sin-r0 || estado=$?
    ((estado != 0)) || paso 'sin R0: SQLSTATE 42501 exacto acreditado'
    finalizar_h0b_f0 "${estado}" || return $?
    reiniciar_ledger_h0b_f0 || return 65
    if [[ "${modo_m38}" == hijo ]]; then
        recuperacion_interna_h0b='' traza_finalizador_h0b='' caso_observado_h0b=''
        traza_m38_activa=1
        [[ "${selector_inyeccion_h0b}" == NOMINAL ]] || inyeccion_h0b_activa=1
    fi
    paso 'finalizador sin R0: rollback y línea base exacta acreditados'
    mkdir --mode=0700 "${temporales}/integracion-h0b-nominal" \
        "${temporales}/integracion-h0b-error" || return 65
    crear_directorio_wrapper_h0b_f0 "${directorio_wrapper_h0b}" || return 65
    r0_posible=1
    crear_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || acreditar_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || inyectar_frontera_h0b_f0 A01 || estado=$?
    ((estado != 0)) || paso 'R0 canónico acreditado'
    ((estado != 0)) || preparar_integracion_h0b_f0 nominal || estado=$?
    ((estado != 0)) || ejecutar_wrapper_h0b_f0 nominal || estado=$?
    ((estado != 0)) || acreditar_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || inyectar_frontera_h0b_f0 A02 || estado=$?
    ((estado != 0)) || paso 'integración virtual C2 nominal acreditada'
    ((estado != 0)) || preparar_integracion_h0b_f0 error || estado=$?
    ((estado != 0)) || ejecutar_wrapper_h0b_f0 error || estado=$?
    ((estado != 0)) || acreditar_r0_sintetico_f0 || estado=$?
    ((estado != 0)) || inyectar_frontera_h0b_f0 A03 || estado=$?
    ((estado != 0)) || paso 'error posterior: SQLSTATE 22012 exacto acreditado'
    if finalizar_h0b_f0 "${estado}"; then estado=0; else estado=$?; fi
    if [[ "${modo_m38}" == hijo ]]; then
        [[ "${recuperacion_interna_h0b}" != 1 ]] || recuperacion=interna-exterior
        [[ "${selector_inyeccion_h0b}" != NOMINAL ]] || caso_observado_h0b=NOMINAL
        printf '[F0 M38] RESULTADO|%s|%s|%s|%s\n' "${caso_observado_h0b}" "${estado}" "${traza_finalizador_h0b}" "${recuperacion}"
        return "${estado}"
    fi
    ((estado == 0)) || return "${estado}"
    paso 'finalizador R0: ausencia y ambas raíces/base exactas acreditadas'
}
acreditar_runner_m38_f0() { [[ "$(stat --printf='%d|%i|%u|%F|%a|%h|%s' -- /proc/self/fd/8)|$(huella_local_f0 /proc/self/fd/8)" == "$1" ]]; }
probar_rechazos_m38_f0() {
    local defecto caso ticket estado traza="${temporales}/rechazo-m38.trace"
    local -a argumentos
    for defecto in vacio desconocido repetido sin-ticket discrepante; do
        caso=''; ticket=x; argumentos=(--caso-inyeccion-h0b "${caso}")
        [[ "${defecto}" != desconocido ]] || argumentos[1]=Z99
        [[ "${defecto}" != repetido ]] || argumentos+=(sobrante)
        [[ "${defecto}" != discrepante ]] || { argumentos[1]=A01; ticket="${BASHPID}|A02|0000000000000000000000000000000000000000000000000000000000000000|0000000000000000000000000000000000000000000000000000000000000000|1|1|${EUID}|directory|700|2|1|1|${EUID}|regular file|600|0|1|0000000000000000000000000000000000000000000000000000000000000000|1|1|${EUID}|directory|700|2"; }
        exec 6>"${traza}"
        if [[ "${defecto}" == sin-ticket ]]; then
            if /usr/bin/env -i PATH=/ruta-no-resoluble LC_ALL=C BASH_XTRACEFD=6 \
                /usr/bin/bash -p -x /proc/self/fd/8 "${argumentos[@]}" 8<&8 7<&7 9<&- >/dev/null 2>&1; then
                estado=0
            else
                estado=$?
            fi
        else
            if /usr/bin/env -i PATH=/ruta-no-resoluble LC_ALL=C BASH_XTRACEFD=6 \
                /usr/bin/bash -p -x /proc/self/fd/8 "${argumentos[@]}" 8<&8 7<&7 9<<<"${ticket}" >/dev/null 2>&1; then
                estado=0
            else
                estado=$?
            fi
        fi
        exec 6>&-
        ((estado == 64)) && ! grep -Eqv '^\+ (set |for ruta_sistema_f0 in |/usr/bin/env -0$|/usr/bin/grep -zE '\''\^\(BASH_FUNC_\|LD_\)'\''$|estados_entorno=|export |LC_ALL=|umask |unset |modo_m38=|selector_inyeccion_h0b=|(identidad|cid_esperado)_m38=|forma_(temporal|runner|raiz)_m38=|IFS=|read |exec( |$)|exit |\[\[ |\(\()' "${traza}" || return 65
    done
}
ejecutar_matriz_m38_f0() {
    local casos='A01 A02 A03 N01 N02 N03 N04 N05 N06 N07 N08 N09 N10 E01 E02 E03 E04 E05 E06 E07 E08 E09 E10 F01 F02 F03 F04 F05 F06 F07 F08 F09 F10 F11 F12 F13 F14 F15'
    local original huella copia forma_runner forma_raiz catalogo='' caso identidad cid forma ticket salida error estado linea resultado cantidad esperado observado declarado traza recuperacion extra forma_restaurada
    iniciar_regimen_shell_m38_f0 || return 65
    probar_oraculo_inyeccion_h0b_f0 || return 65
    huella="$(huella_local_f0 "${ruta_runner}")"; original="$(stat --printf='%d|%i|%u|%F|%a|%h|%s' -- "${ruta_runner}")|${huella}" || return 65
    [[ "${original}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|regular\ file\|[0-7]{3,4}\|1\|[0-9]+\|[0-9a-f]{64}$ ]] || return 65
    copia="${temporales}/runner-m38"; copiar_fuente_sin_enlaces_f0 "${ruta_runner}" "${copia}" "${huella}" || return 65
    exec 8<"${copia}" 7<"${raiz}"; [[ "$(stat --printf='%u|%F|%a|%h' -- /proc/self/fd/8)" == "${EUID}|regular file|600|1" ]] || return 65; rm -- "${copia}" || return 65
    forma_runner="$(stat --printf='%d|%i|%u|%F|%a|%h|%s' -- /proc/self/fd/8)|${huella}"; forma_raiz="$(stat --printf='%d|%i|%u|%F|%a|%h' -- /proc/self/fd/7)"
    acreditar_runner_m38_f0 "${forma_runner}" || return 65
    printf 'adverso\n' >"${copia}"; acreditar_runner_m38_f0 "${forma_runner}" && rm -- "${copia}" || return 65
    copiar_fuente_sin_enlaces_f0 "${ruta_runner}" "${copia}" "${huella}" || return 65
    forma_restaurada="$(stat --printf='%d|%i' -- "${copia}")"; [[ "${forma_runner}" != "${forma_restaurada}|"* ]] || return 65
    rm -- "${copia}"; acreditar_runner_m38_f0 "${forma_runner}" || return 65
    [[ "$(stat --printf='%d|%i|%u|%F|%a|%h|%s' -- "${ruta_runner}")|$(huella_local_f0 "${ruta_runner}")" == "${original}" ]] && probar_rechazos_m38_f0 || return 65
    for caso in ${casos}; do catalogo+="${catalogo:+$'\n'}${caso}"; done
    validar_catalogo_inyecciones_h0b_f0 "${catalogo}" || return 65
    for caso in NOMINAL ${casos}; do
        acreditar_runner_m38_f0 "${forma_runner}" && [[ "$(stat --printf='%d|%i|%u|%F|%a|%h' -- /proc/self/fd/7)" == "${forma_raiz}" ]] || return 65
        preparar_recursos_m38_f0 identidad forma cid || return $?
        salida="${temporales}/${caso}.out"; error="${temporales}/${caso}.err"
        ticket="${BASHPID}|${caso}|${identidad}|${cid}|${forma}|${forma_runner}|${forma_raiz}"
        lanzar_hijo_m38_f0 "${caso}" "${ticket}" "${salida}" "${error}" estado || return 65
        resultado=''; cantidad=0; while IFS= read -r linea; do [[ "${linea}" != '[F0 M38] RESULTADO|'* ]] || { resultado="${linea#*RESULTADO|}"; cantidad=$((cantidad + 1)); }; done <"${salida}"
        IFS='|' read -r observado declarado traza recuperacion extra <<<"${resultado}"
        retirar_recursos_m38_f0 || return $?
        esperado="$(oraculo_inyeccion_h0b_f0 "${caso}")" || return 65
        ((cantidad == 1)) && [[ -z "${extra}" && "${estado}" == "${esperado%%|*}" ]] && validar_observacion_inyeccion_h0b_f0 "${caso}" "${observado}" "${declarado}" "${traza}" "${recuperacion}" || return 65
        acreditar_runner_m38_f0 "${forma_runner}" && [[ "$(stat --printf='%d|%i|%u|%F|%a|%h' -- /proc/self/fd/7)" == "${forma_raiz}" ]] || return 65
    done
    restaurar_regimen_shell_m38_f0 || return 65
    [[ "$(stat --printf='%d|%i|%u|%F|%a|%h|%s' -- "${ruta_runner}")|$(huella_local_f0 "${ruta_runner}")" == "${original}" && "$(stat --printf='%d|%i|%u|%F|%a|%h' -- /proc/self/fd/7)" == "${forma_raiz}" ]] || return 65
    exec 8<&- 7<&-
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
elif (($# == 1)) && [[ "$1" == '--matriz-inyeccion-h0b' ]]; then
    modo_m38=conductor; etapa=H0
elif (($# != 0)); then
    fallar 'uso: runner [--etapa ETAPA|--matriz-inyeccion-h0b]'
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
if [[ "${modo_m38}" == hijo ]]; then
    [[ "$(stat --printf='%d|%i|%u|%F|%a|%h|%s' -- /proc/self/fd/8)|$(huella_local_f0 /proc/self/fd/8)" == "${forma_runner_m38}" &&
       "$(stat --printf='%d|%i|%u|%F|%a|%h' -- /proc/self/fd/7)" == "${forma_raiz_m38}" ]] || exit 65
fi
[[ "${imagen}" =~ ^[^[:space:]@]+@sha256:[0-9a-f]{64}$ ]] ||
    fallar 'VEC_POSTGRES_TEST_IMAGE debe fijarse por digest sha256'
docker image inspect "${imagen}" >/dev/null 2>&1 ||
    fallar 'la imagen PostgreSQL fijada no está disponible localmente'
if [[ "${modo_m38}" == hijo ]]; then
    temporales="/tmp/vec-f0-h0-${identidad_m38}"
    capturar_salida_f0 forma_temporales stat --printf='%d|%i|%u|%F|%a|%h' -- "${temporales}" || exit 65
    [[ "${forma_temporales}" == "${forma_temporal_m38}" ]] || exit 65
else
    capturar_salida_f0 aleatorio_temporales openssl rand -hex 32 || fallar 'no se pudo generar la reserva temporal'
    [[ "${aleatorio_temporales}" =~ ^[0-9a-f]{64}$ && -d /tmp && ! -L /tmp ]] || fallar 'reserva temporal inválida'
    temporales="/tmp/vec-f0-h0-${aleatorio_temporales}"
fi
trap limpiar EXIT
trap 'gestionar_senal_m38_f0 130' INT
trap 'gestionar_senal_m38_f0 143' TERM
if [[ "${modo_m38}" != hijo ]]; then
    [[ ! -e "${temporales}" && ! -L "${temporales}" ]] || fallar 'la reserva temporal ya existe'
    temporal_preausente='1'; mkdir --mode=0700 -- "${temporales}" || estado_mkdir_temporal=$?
    capturar_salida_f0 forma_temporales stat --printf='%d|%i|%u|%F|%a|%h' -- "${temporales}" || fallar 'reserva temporal no acreditable'
fi
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
if [[ "${modo_m38}" == conductor ]]; then
    ejecutar_matriz_m38_f0 || {
        estado_matriz=$?
        ((estado_matriz == 130 || estado_matriz == 143)) && exit "${estado_matriz}"
        fallar 'la matriz funcional M38 no quedó acreditada'
    }
    paso 'matriz funcional M38 acreditada en 39 procesos aislados'
    limpiar_estricto_f0 || fallar 'la matriz M38 dejó recursos propios'
    trap - EXIT INT TERM; exit 0
fi
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
[[ "${modo_m38}" == hijo ]] || capturar_salida_f0 clave_postgres openssl rand -hex 24 ||
    fallar 'no se pudo generar la clave efímera de PostgreSQL'
[[ -n "${propietario_contenedor}" ]] || capturar_salida_f0 propietario_contenedor openssl rand -hex 32 || fallar 'no se pudo generar la marca efímera del contenedor'
[[ "${propietario_contenedor}" =~ ^[0-9a-f]{64}$ ]] || fallar 'marca efímera de contenedor inválida'
cid_contenedor="${temporales}/contenedor.cid"
intencion_contenedor='1'
if [[ "${modo_m38}" != hijo ]]; then
    [[ ! -e "${cid_contenedor}" && ! -L "${cid_contenedor}" ]] || fallar 'ruta cidfile no exclusiva'
    docker run --detach --name "${contenedor}" --network none \
        --label "es.dipgra.vep.f0.propietario=${propietario_contenedor}" \
        --cidfile "${cid_contenedor}" --env POSTGRES_PASSWORD="${clave_postgres}" \
        --env POSTGRES_INITDB_ARGS='--auth-local=trust' \
        --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=768m \
        "${imagen}" -c max_prepared_transactions=0 >/dev/null
    chmod 0600 "${cid_contenedor}" || fallar 'no se pudo proteger el cidfile'
fi
hallado_cid="$(descubrir_contenedor_propio_f0)" || fallar 'contenedor propio no descubrible'
id_contenedor="$(acreditar_hallazgo_contenedor_f0 "${hallado_cid}")" || fallar 'nombre, id y etiqueta del contenedor no coinciden'
acreditar_cidfile_f0 "${id_contenedor}" || fallar 'cidfile no coincide con el contenedor acreditado'
[[ "${modo_m38}" != hijo || "${id_contenedor}" == "${cid_esperado_m38}" ]] || fallar 'el hijo no adoptó el CID acreditado'
[[ "$(docker inspect --format '{{.Config.Image}}|{{.State.Running}}' "${id_contenedor}")" == "${imagen}|true" ]] || fallar 'imagen o estado Docker discrepante'
if [[ "${modo_m38}" != hijo ]]; then
    for _ in {1..60}; do
        docker exec "${contenedor}" pg_isready --quiet \
            --username postgres --dbname postgres && break
        sleep 1
    done
fi
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
if [[ "${etapa}" == H0 && "${modo_m38}" == hijo ]]; then
    if probar_h0b_funcional_f0; then exit 0; else estado_hijo=$?; exit "${estado_hijo}"; fi
fi
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
