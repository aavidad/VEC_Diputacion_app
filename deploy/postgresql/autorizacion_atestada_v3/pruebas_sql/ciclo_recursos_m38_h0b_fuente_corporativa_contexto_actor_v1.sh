#!/usr/bin/env bash
# shellcheck disable=SC2034,SC2154 # estado compartido con el runner acreditado

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    printf 'el adaptador de recursos M38 solo puede cargarse desde su runner acreditado\n' >&2
    exit 64
fi
if [[ "${VEC_F0_CARGA_PRIVADA:-}" != '1' ]]; then
    printf 'carga no acreditada del adaptador de recursos M38\n' >&2
    return 64
fi
unset VEC_F0_CARGA_PRIVADA

readonly destino_m080_h0b='/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/080_consumidor_nominal.sql'
readonly destino_t080_h0b='/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/080_consumidor_nominal.sql'
readonly directorio_wrapper_h0b='/repo_h0b/deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/__h0b'
readonly destino_wrapper_h0b="${directorio_wrapper_h0b}/sin-r0.sql"
readonly destino_wrapper_nominal_h0b="${directorio_wrapper_h0b}/nominal/ensayo.sql"
readonly destino_wrapper_error_h0b="${directorio_wrapper_h0b}/error/ensayo.sql"
forma_padre_m_h0b='' forma_padre_wrapper_h0b=''
forma_pruebas_t_inicial_h0b='' forma_pruebas_t_actual_h0b=''
forma_componentes_t_h0b=''
forma_m_h0b='' forma_t_h0b='' forma_wrapper_sin_h0b=''
forma_wrapper_nominal_h0b='' forma_wrapper_error_h0b=''
forma_directorio_wrapper_h0b='' forma_directorio_nominal_h0b=''
forma_directorio_error_h0b=''
estado_m_h0b=nunca_creado estado_t_h0b=nunca_creado
estado_wrapper_sin_h0b=nunca_creado estado_wrapper_nominal_h0b=nunca_creado
estado_wrapper_error_h0b=nunca_creado estado_directorio_wrapper_h0b=nunca_creado
estado_directorio_nominal_h0b=nunca_creado estado_directorio_error_h0b=nunca_creado
estado_pruebas_t_h0b=nunca_creado estado_componentes_t_h0b=nunca_creado
identidad_activa_m38='' ruta_caso_m38='' forma_caso_m38=''
pid_hijo_m38='' pgid_hijo_m38='' hijo_esperado_m38=''
regimen_shell_m38='' monitor_previo_m38='' shellopts_exportado_m38=''
seccion_critica_m38='' senal_pendiente_m38='' generacion_senal_m38=0

iniciar_regimen_shell_m38_f0() {
    [[ -z "${regimen_shell_m38}" && -z "$(jobs -p)" ]] || return 65
    monitor_previo_m38=''; [[ "$-" != *m* ]] || monitor_previo_m38=1
    shellopts_exportado_m38=''; [[ "${SHELLOPTS@a}" != *x* ]] || shellopts_exportado_m38=1
    regimen_shell_m38=1
    set -m
    export -n SHELLOPTS
}
restaurar_regimen_shell_m38_f0() {
    [[ -n "${regimen_shell_m38}" ]] || return 0
    [[ -z "$(jobs -p)" ]] || return 65
    if [[ -n "${monitor_previo_m38}" ]]; then set -m; else set +m; fi
    if [[ -n "${shellopts_exportado_m38}" ]]; then export SHELLOPTS; else export -n SHELLOPTS; fi
    regimen_shell_m38=''
}
finalizar_seccion_critica_m38_f0() {
    seccion_critica_m38=''
    # La entrega al runner cierra C4b-1; el epílogo pertenece a C4b-3.
    [[ -z "${senal_pendiente_m38}" ]] || return "${senal_pendiente_m38}"
}
esperar_terminal_m38_f0() {
    local pid="$1" generacion
    while :; do
        generacion="${generacion_senal_m38}"
        wait -f "${pid}" 2>/dev/null || :
        ((generacion != generacion_senal_m38)) || break
    done
}
esperar_cliente_m38_f0() {
    local plazo="$1" cliente reloj terminado generacion estado=0 senal; shift
    [[ -n "${regimen_shell_m38}" && "$-" == *m* && "${SHELLOPTS@a}" != *x* &&
       -n "${identidad_activa_m38}" && -z "${seccion_critica_m38}${senal_pendiente_m38}" ]] || return 65
    seccion_critica_m38=1
    "$@" & cliente=$!
    sleep "${plazo}" & reloj=$!
    while :; do
        terminado='' generacion="${generacion_senal_m38}"
        if wait -n -f -p terminado "${cliente}" "${reloj}"; then estado=0; else estado=$?; fi
        if ((generacion == generacion_senal_m38)) || [[ -n "${terminado:-}" ]]; then break; fi
    done
    if [[ "${terminado:-}" == "${cliente}" ]]; then
        kill -TERM -- "-${reloj}" 2>/dev/null || :
        esperar_terminal_m38_f0 "${reloj}"
    elif [[ "${terminado:-}" == "${reloj}" ]]; then
        kill -TERM -- "-${cliente}" 2>/dev/null || :
        sleep 2
        kill -KILL -- "-${cliente}" 2>/dev/null || :
        esperar_terminal_m38_f0 "${cliente}"
        estado=65
    else
        kill -KILL -- "-${cliente}" "-${reloj}" 2>/dev/null || :
        esperar_terminal_m38_f0 "${cliente}"
        esperar_terminal_m38_f0 "${reloj}"
        estado=65
    fi
    if finalizar_seccion_critica_m38_f0; then :; else senal=$?; return "${senal}"; fi
    return "${estado}"
}

preparar_recursos_m38_f0() {
    local -n identidad_salida="$1" forma_salida="$2" cid_salida="$3"
    local hallado id intento estado
    [[ -n "${regimen_shell_m38}" && "$-" == *m* && "${SHELLOPTS@a}" != *x* &&
       -z "$(jobs -p)" ]] || return 65
    identidad_activa_m38="$(openssl rand -hex 32)" || return 65
    [[ "${identidad_activa_m38}" =~ ^[0-9a-f]{64}$ && -d /tmp && ! -L /tmp ]] || return 65
    ruta_caso_m38="/tmp/vec-f0-h0-${identidad_activa_m38}"
    [[ ! -e "${ruta_caso_m38}" && ! -L "${ruta_caso_m38}" ]] || return 65
    mkdir --mode=0700 -- "${ruta_caso_m38}" || return 65
    forma_caso_m38="$(stat --printf='%d|%i|%u|%F|%a|%h' -- "${ruta_caso_m38}")" || return 65
    [[ "${forma_caso_m38}" =~ ^[0-9]+\|[0-9]+\|${EUID}\|directory\|700\|2$ ]] || return 65
    contenedor="vec-f0-h0-${identidad_activa_m38:0:32}"
    propietario_contenedor="${identidad_activa_m38}"
    cid_contenedor="${ruta_caso_m38}/contenedor.cid"; intencion_contenedor=1
    clave_postgres="$(openssl rand -hex 24)" || return 65
    esperar_cliente_m38_f0 30 docker run --detach --name "${contenedor}" --network none \
        --label "es.dipgra.vep.f0.propietario=${propietario_contenedor}" --cidfile "${cid_contenedor}" \
        --env POSTGRES_PASSWORD="${clave_postgres}" --env POSTGRES_INITDB_ARGS=--auth-local=trust \
        --tmpfs /var/lib/postgresql:rw,noexec,nosuid,size=768m "${imagen}" \
        -c max_prepared_transactions=0 >/dev/null || return $?
    chmod 0600 "${cid_contenedor}" || return 65
    hallado="$(descubrir_contenedor_propio_f0)" || return 65
    id="$(acreditar_hallazgo_contenedor_f0 "${hallado}")" || return 65
    acreditar_cidfile_f0 "${id}" || return 65
    for intento in {1..10}; do
        if esperar_cliente_m38_f0 5 docker exec "${id}" pg_isready --quiet \
            --username postgres --dbname postgres; then break; else estado=$?; fi
        ((estado != 130 && estado != 143)) || return "${estado}"
        sleep 1
    done
    esperar_cliente_m38_f0 5 docker exec "${id}" pg_isready --quiet \
        --username postgres --dbname postgres || return $?
    identidad_salida="${identidad_activa_m38}"; forma_salida="${forma_caso_m38}"; cid_salida="${id}"
}

lanzar_hijo_m38_f0() {
    local caso="$1" ticket="$2" salida="$3" error="$4" estado_ref="$5"
    local -n estado_salida="${estado_ref}"; local estado_proc intento
    [[ -z "$(jobs -p)" ]] || return 65
    /usr/bin/bash -p /proc/self/fd/8 --caso-inyeccion-h0b "${caso}" \
        8<&8 7<&7 9<<<"${ticket}" >"${salida}" 2>"${error}" &
    pid_hijo_m38=$!; hijo_esperado_m38=1
    for intento in {1..100}; do
        estado_proc="$(awk '{print $1"|"$3"|"$4"|"$5"|"$22}' "/proc/${pid_hijo_m38}/stat" 2>/dev/null)" || :
        [[ "${estado_proc}" == "${pid_hijo_m38}|T|${BASHPID}|${pid_hijo_m38}|"+([0-9]) ]] && break
        sleep 0.01
    done
    [[ "${estado_proc}" == "${pid_hijo_m38}|T|${BASHPID}|${pid_hijo_m38}|"+([0-9]) ]] || return 65
    pgid_hijo_m38="${pid_hijo_m38}"; kill -CONT -- "-${pgid_hijo_m38}" || return 65
    if esperar_hijo_con_plazo_m38_f0 180; then estado_salida=0; else estado_salida=$?; fi
}

esperar_hijo_con_plazo_m38_f0() {
    local reloj terminado estado=0
    sleep "$1" & reloj=$!
    if wait -n -f -p terminado "${pid_hijo_m38}" "${reloj}"; then estado=0; else estado=$?; fi
    if [[ "${terminado}" == "${pid_hijo_m38}" ]]; then
        hijo_esperado_m38=''; kill -TERM -- "-${reloj}" 2>/dev/null || :
        wait -f "${reloj}" 2>/dev/null || :; return "${estado}"
    fi
    return 65
}

retirar_recursos_m38_f0() {
    local actual hallado id
    if [[ -n "${pgid_hijo_m38}" ]] && kill -0 -- "-${pgid_hijo_m38}" 2>/dev/null; then
        kill -CONT -- "-${pgid_hijo_m38}" 2>/dev/null || :
        kill -TERM -- "-${pgid_hijo_m38}" 2>/dev/null || :; sleep 2
        kill -KILL -- "-${pgid_hijo_m38}" 2>/dev/null || :
    fi
    [[ -z "${hijo_esperado_m38}" ]] || { wait -f "${pid_hijo_m38}" 2>/dev/null || :; hijo_esperado_m38=''; }
    [[ -z "${identidad_activa_m38}" ]] && return 0
    hallado="$(descubrir_contenedor_propio_f0)" || return 65
    if [[ -n "${hallado}" ]]; then
        id="$(acreditar_hallazgo_contenedor_f0 "${hallado}")" || return 65
        acreditar_cidfile_f0 "${id}" || return 65
        esperar_cliente_m38_f0 30 docker rm --force --volumes "${id}" >/dev/null || return $?
        [[ -z "$(descubrir_contenedor_propio_f0)" ]] || return 65
    fi
    actual="$(stat --printf='%d|%i|%u|%F|%a|%h' -- "${ruta_caso_m38}")" || return 65
    [[ "${actual}" == "${forma_caso_m38}" && -d "${ruta_caso_m38}" && ! -L "${ruta_caso_m38}" ]] || return 65
    rm -rf -- "${ruta_caso_m38}" || return 65
    [[ ! -e "${ruta_caso_m38}" && ! -L "${ruta_caso_m38}" ]] || return 65
    identidad_activa_m38='' ruta_caso_m38='' forma_caso_m38=''
    pid_hijo_m38='' pgid_hijo_m38='' propietario_contenedor='' intencion_contenedor=''
}

forma_interior_h0b_f0() {
    local clase="$1" ruta="$2"
    docker exec "${contenedor}" bash -ceu '
ruta=$1; clase=$2
[[ ! -L $ruta ]] || exit 65
if [[ $clase == archivo ]]; then
  [[ -f $ruta ]] || exit 65
  forma=$(stat --printf="%d|%i|%u|%F|%a|%h|%s" -- "$ruta") || exit 65
  huella=$(sha256sum -- "$ruta" | awk "{print \$1}") || exit 65
  [[ $forma =~ ^[0-9]+\|[0-9]+\|[0-9]+\|regular\ file\|600\|1\|[0-9]+$ && $huella =~ ^[0-9a-f]{64}$ ]] || exit 65
  printf "%s|%s" "$forma" "$huella"
else
  [[ $clase == directorio && -d $ruta ]] || exit 65
  forma=$(stat --printf="%d|%i|%u|%F|%a|%h" -- "$ruta") || exit 65
  [[ $forma =~ ^[0-9]+\|[0-9]+\|[0-9]+\|directory\|700\|[0-9]+$ ]] || exit 65
  printf "%s" "$forma"
fi
' -- "${ruta}" "${clase}"
}

acreditar_forma_interior_h0b_f0() {
    local obtenida
    capturar_salida_f0 obtenida forma_interior_h0b_f0 "$1" "$2" || return 65
    [[ "${obtenida}" == "$3" ]]
}

acreditar_ausencia_interior_h0b_f0() {
    # shellcheck disable=SC2016 # condición evaluada en el Bash privado del contenedor
    docker exec "${contenedor}" bash -ceu '[[ ! -e $1 && ! -L $1 ]]' -- "$1"
}

variable_forma_archivo_h0b_f0() {
    case "$1" in
        "${destino_m080_h0b}") printf forma_m_h0b ;;
        "${destino_t080_h0b}") printf forma_t_h0b ;;
        "${destino_wrapper_h0b}") printf forma_wrapper_sin_h0b ;;
        "${destino_wrapper_nominal_h0b}") printf forma_wrapper_nominal_h0b ;;
        "${destino_wrapper_error_h0b}") printf forma_wrapper_error_h0b ;;
        *) return 64 ;;
    esac
}

variable_estado_archivo_h0b_f0() {
    case "$1" in
        "${destino_m080_h0b}") printf estado_m_h0b ;;
        "${destino_t080_h0b}") printf estado_t_h0b ;;
        "${destino_wrapper_h0b}") printf estado_wrapper_sin_h0b ;;
        "${destino_wrapper_nominal_h0b}") printf estado_wrapper_nominal_h0b ;;
        "${destino_wrapper_error_h0b}") printf estado_wrapper_error_h0b ;;
        *) return 64 ;;
    esac
}

crear_directorio_wrapper_h0b_f0() {
    local ruta="$1" variable variable_estado padre forma_padre forma_nueva
    if [[ "${ruta}" == "${directorio_wrapper_h0b}" ]]; then
        variable=forma_directorio_wrapper_h0b
        variable_estado=estado_directorio_wrapper_h0b
        padre="${ruta%/*}"
        capturar_salida_f0 forma_padre forma_interior_h0b_f0 directorio "${padre}" || return 65
        [[ -z "${forma_padre_wrapper_h0b}" ]] || [[ "${forma_padre}" == "${forma_padre_wrapper_h0b}" ]] || return 65
        forma_padre_wrapper_h0b="${forma_padre}"
    elif [[ "${ruta}" == "${destino_wrapper_nominal_h0b%/*}" ]]; then
        variable=forma_directorio_nominal_h0b
        variable_estado=estado_directorio_nominal_h0b
        padre="${directorio_wrapper_h0b}"
    elif [[ "${ruta}" == "${destino_wrapper_error_h0b%/*}" ]]; then
        variable=forma_directorio_error_h0b
        variable_estado=estado_directorio_error_h0b
        padre="${directorio_wrapper_h0b}"
    else
        return 64
    fi
    local -n forma_destino="${variable}"
    local -n estado_destino="${variable_estado}"
    [[ -z "${forma_destino}" && "${estado_destino}" == nunca_creado ]] || return 65
    docker exec "${contenedor}" bash -ceu '
padre=$1; ruta=$2
[[ -d $padre && ! -L $padre && ! -e $ruta && ! -L $ruta ]] || exit 65
mkdir --mode=0700 -- "$ruta"
' -- "${padre}" "${ruta}" || return 65
    capturar_salida_f0 forma_nueva forma_interior_h0b_f0 directorio "${ruta}" || return 65
    [[ "${forma_nueva}" =~ ^[0-9]+\|[0-9]+\|[0-9]+\|directory\|700\|2$ ]] || return 65
    forma_destino="${forma_nueva}"
    estado_destino=registrado_presente
    [[ "${padre}" != "${directorio_wrapper_h0b}" ]] ||
        capturar_salida_f0 forma_directorio_wrapper_h0b \
            forma_interior_h0b_f0 directorio "${padre}" || return 65
}

retirar_directorios_wrapper_h0b_f0() {
    local basal="${forma_padre_wrapper_h0b}" raiz="${forma_directorio_wrapper_h0b}"
    local nominal="${forma_directorio_nominal_h0b}" error="${forma_directorio_error_h0b}"
    [[ -n "${basal}" && -n "${raiz}" ]] || return 65
    # shellcheck disable=SC2016 # el bloque se evalúa en el Bash privado del contenedor
    docker exec "${contenedor}" bash -ceu '
forma() { [[ -d $1 && ! -L $1 ]] && stat --printf="%d|%i|%u|%F|%a|%h" -- "$1"; }
padre=$1; basal=$2; raiz=$3; forma_raiz=$4; nominal=$5; forma_nominal=$6; error=$7; forma_error=$8
[[ $(forma "$raiz") == "$forma_raiz" && -z $(find "$raiz" -mindepth 1 ! -type d -print -quit) ]] || exit 65
[[ -n $forma_nominal && $(forma "$nominal") == "$forma_nominal" || -z $forma_nominal && ! -e $nominal && ! -L $nominal ]] || exit 65
[[ -n $forma_error && $(forma "$error") == "$forma_error" || -z $forma_error && ! -e $error && ! -L $error ]] || exit 65
[[ -z $forma_nominal ]] || { [[ $(forma "$nominal") == "$forma_nominal" ]]; rmdir -- "$nominal"; }
[[ -z $forma_error ]] || { [[ $(forma "$error") == "$forma_error" ]]; rmdir -- "$error"; }
identidad=${forma_raiz%|*}; [[ $(forma "$raiz") == "$identidad|2" ]] || exit 65
rmdir -- "$raiz" || exit 65
[[ $(forma "$padre") == "$basal" && ! -e $raiz && ! -L $raiz ]] || exit 65
' -- "${directorio_wrapper_h0b%/*}" "${basal}" "${directorio_wrapper_h0b}" "${raiz}" \
        "${destino_wrapper_nominal_h0b%/*}" "${nominal}" \
        "${destino_wrapper_error_h0b%/*}" "${error}" || return 65
    estado_directorio_wrapper_h0b=retirado_por_la_accion
    [[ -z "${forma_directorio_nominal_h0b}" ]] || estado_directorio_nominal_h0b=retirado_por_la_accion
    [[ -z "${forma_directorio_error_h0b}" ]] || estado_directorio_error_h0b=retirado_por_la_accion
}

preparar_padres_t_h0b_f0() {
    local salida pruebas="${destino_t080_h0b%/*/*}" componentes="${destino_t080_h0b%/*}"
    if [[ "${estado_pruebas_t_h0b}" == registrado_presente &&
          "${estado_componentes_t_h0b}" == registrado_presente ]]; then
        acreditar_forma_interior_h0b_f0 directorio "${pruebas}" \
            "${forma_pruebas_t_actual_h0b}" &&
            acreditar_forma_interior_h0b_f0 directorio "${componentes}" \
                "${forma_componentes_t_h0b}"
        return
    fi
    [[ "${estado_pruebas_t_h0b}" == nunca_creado &&
       "${estado_componentes_t_h0b}" == nunca_creado ]] || return 65
    # shellcheck disable=SC2016 # el bloque se evalúa en el Bash privado del contenedor
    capturar_salida_f0 salida docker exec "${contenedor}" bash -ceu '
pruebas=$1; componentes=$2
[[ ! -e $pruebas && ! -L $pruebas && ! -e $componentes && ! -L $componentes ]] || exit 65
mkdir --mode=0700 -- "$pruebas"
inicial=$(stat --printf="%d|%i|%u|%F|%a|%h" -- "$pruebas") || exit 65
mkdir --mode=0700 -- "$componentes"
actual=$(stat --printf="%d|%i|%u|%F|%a|%h" -- "$pruebas") || exit 65
forma_componentes=$(stat --printf="%d|%i|%u|%F|%a|%h" -- "$componentes") || exit 65
printf "%s;%s;%s" "$inicial" "$actual" "$forma_componentes"
' -- "${pruebas}" "${componentes}" || return 65
    IFS=';' read -r forma_pruebas_t_inicial_h0b forma_pruebas_t_actual_h0b \
        forma_componentes_t_h0b <<<"${salida}"
    [[ "${forma_pruebas_t_inicial_h0b}" =~ ^[0-9]+\|[0-9]+\|[0-9]+\|directory\|700\|2$ &&
       "${forma_pruebas_t_actual_h0b}" =~ ^[0-9]+\|[0-9]+\|[0-9]+\|directory\|700\|3$ &&
       "${forma_pruebas_t_inicial_h0b%|*}" == "${forma_pruebas_t_actual_h0b%|*}" &&
       "${forma_componentes_t_h0b}" =~ ^[0-9]+\|[0-9]+\|[0-9]+\|directory\|700\|2$ ]] || return 65
    estado_pruebas_t_h0b=registrado_presente
    estado_componentes_t_h0b=registrado_presente
}

copiar_componente_sintetico_f0() {
    local origen="$1" destino="$2" caso_copia="${3:-}" caso_huella="${4:-}"
    local variable variable_estado forma_nueva padre forma_padre
    variable="$(variable_forma_archivo_h0b_f0 "${destino}")" || variable=''
    padre="${destino%/*}"
    if [[ "${destino}" == "${destino_m080_h0b}" && -z "${forma_padre_m_h0b}" ]]; then
        capturar_salida_f0 forma_padre forma_interior_h0b_f0 directorio "${padre}" || return 65
        forma_padre_m_h0b="${forma_padre}"
    elif [[ "${destino}" == "${destino_t080_h0b}" ]]; then
        preparar_padres_t_h0b_f0 || return 65
    fi
    if [[ -n "${variable}" ]]; then
        local -n forma_destino="${variable}"
        variable_estado="$(variable_estado_archivo_h0b_f0 "${destino}")" || return 65
        local -n estado_destino="${variable_estado}"
        [[ "${estado_destino}" == nunca_creado ||
           "${estado_destino}" == retirado_para_sustitucion ]] || return 65
        acreditar_ausencia_interior_h0b_f0 "${destino}" || return 65
    fi
    validar_componentes_sql_f0 "${origen}" "${temporales}" || return 65
    docker cp "${origen}" "${contenedor}:${destino}" || return 65
    if [[ -n "${variable}" ]]; then
        capturar_salida_f0 forma_nueva forma_interior_h0b_f0 archivo "${destino}" || return 65
        forma_destino="${forma_nueva}"
        estado_destino=registrado_presente
    fi
    inyectar_frontera_h0b_f0 "${caso_copia}" || return $?
    comparar_huellas_f0 "${origen}" "${destino}" || return 65
    inyectar_frontera_h0b_f0 "${caso_huella}"
}

retirar_para_sustitucion_h0b_f0() {
    local destino="$1" variable variable_estado
    variable="$(variable_forma_archivo_h0b_f0 "${destino}")" || return 65
    variable_estado="$(variable_estado_archivo_h0b_f0 "${destino}")" || return 65
    local -n forma_destino="${variable}" estado_destino="${variable_estado}"
    [[ "${estado_destino}" == registrado_presente && -n "${forma_destino}" ]] || return 65
    # shellcheck disable=SC2016 # validación y retirada se ejecutan en el contenedor
    docker exec "${contenedor}" bash -ceu '
ruta=$1; esperada=$2
[[ -f $ruta && ! -L $ruta ]] || exit 65
actual=$(stat --printf="%d|%i|%u|%F|%a|%h|%s" -- "$ruta") || exit 65
huella=$(sha256sum -- "$ruta" | awk "{print \$1}") || exit 65
[[ "$actual|$huella" == "$esperada" ]] || exit 65
rm -- "$ruta"
[[ ! -e $ruta && ! -L $ruta ]]
' -- "${destino}" "${forma_destino}" || return 65
    estado_destino=retirado_para_sustitucion
}

retirar_archivos_h0b_f0() {
    local argumento variable variable_estado gestionar_t=0
    local -a recursos=()
    for argumento; do
        variable="$(variable_forma_archivo_h0b_f0 "${argumento}")" || return 65
        variable_estado="$(variable_estado_archivo_h0b_f0 "${argumento}")" || return 65
        local -n forma_registrada="${variable}" estado_registrado="${variable_estado}"
        recursos+=("${argumento}" "${estado_registrado}" "${forma_registrada}")
        [[ "${argumento}" != "${destino_t080_h0b}" ]] || gestionar_t=1
    done
    # shellcheck disable=SC2016 # las dos pasadas se ejecutan dentro del contenedor
    docker exec "${contenedor}" bash -ceu '
gestionar_t=$1; padre_m=$2; forma_m=$3; pruebas_t=$4; componentes_t=$5
forma_pruebas_inicial=$6; forma_pruebas_actual=$7; forma_componentes=$8
estado_pruebas=$9; estado_componentes=${10}; shift 10
validar() {
  ruta=$1; esperada=$2
  [[ -f $ruta && ! -L $ruta ]] || return 65
  actual=$(stat --printf="%d|%i|%u|%F|%a|%h|%s" -- "$ruta") || return 65
  huella=$(sha256sum -- "$ruta" | awk "{print \$1}") || return 65
  [[ "$actual|$huella" == "$esperada" ]]
}
forma_dir() { [[ -d $1 && ! -L $1 ]] && stat --printf="%d|%i|%u|%F|%a|%h" -- "$1"; }
[[ -z $forma_m || $(forma_dir "$padre_m") == "$forma_m" ]] || exit 65
if [[ $gestionar_t == 1 ]]; then
  [[ $estado_pruebas == registrado_presente && $estado_componentes == registrado_presente ]] ||
    [[ $estado_pruebas == nunca_creado && $estado_componentes == nunca_creado ]] || exit 65
  if [[ $estado_pruebas == registrado_presente ]]; then
    [[ $(forma_dir "$pruebas_t") == "$forma_pruebas_actual" && $(forma_dir "$componentes_t") == "$forma_componentes" ]] || exit 65
  else [[ ! -e $pruebas_t && ! -L $pruebas_t && ! -e $componentes_t && ! -L $componentes_t ]] || exit 65; fi
fi
for ((i=1;i<=$#;i+=3)); do
  j=$((i+1)); k=$((i+2)); estado=${!j}
  case $estado in
    registrado_presente) validar "${!i}" "${!k}" || exit 65 ;;
    nunca_creado|retirado_para_sustitucion|retirado_por_la_accion)
      [[ ! -e ${!i} && ! -L ${!i} ]] || exit 65 ;;
    *) exit 65 ;;
  esac
done
for ((i=1;i<=$#;i+=3)); do
  j=$((i+1)); k=$((i+2))
  [[ ${!j} != registrado_presente ]] || {
    validar "${!i}" "${!k}" && rm -- "${!i}"
  } || exit 65
done
for ((i=1;i<=$#;i+=3)); do
  [[ ! -e ${!i} && ! -L ${!i} ]] || exit 65
done
[[ -z $forma_m || $(forma_dir "$padre_m") == "$forma_m" ]] || exit 65
if [[ $gestionar_t == 1 && $estado_componentes == registrado_presente ]]; then
  [[ $(forma_dir "$componentes_t") == "$forma_componentes" && -z $(find "$componentes_t" -mindepth 1 -print -quit) ]] || exit 65
  rmdir -- "$componentes_t"; [[ $(forma_dir "$pruebas_t") == "$forma_pruebas_inicial" && -z $(find "$pruebas_t" -mindepth 1 -print -quit) ]] || exit 65
  rmdir -- "$pruebas_t"; [[ ! -e $pruebas_t && ! -L $pruebas_t && ! -e $componentes_t && ! -L $componentes_t ]] || exit 65
fi
' -- "${gestionar_t}" "${destino_m080_h0b%/*}" "${forma_padre_m_h0b}" \
        "${destino_t080_h0b%/*/*}" "${destino_t080_h0b%/*}" \
        "${forma_pruebas_t_inicial_h0b}" "${forma_pruebas_t_actual_h0b}" \
        "${forma_componentes_t_h0b}" "${estado_pruebas_t_h0b}" \
        "${estado_componentes_t_h0b}" "${recursos[@]}" || return 65
    for argumento; do
        variable_estado="$(variable_estado_archivo_h0b_f0 "${argumento}")"
        local -n estado_retirado="${variable_estado}"
        [[ "${estado_retirado}" != registrado_presente ]] || estado_retirado=retirado_por_la_accion
    done
    if ((gestionar_t == 1)) && [[ "${estado_componentes_t_h0b}" == registrado_presente ]]; then
        estado_componentes_t_h0b=retirado_por_la_accion
        estado_pruebas_t_h0b=retirado_por_la_accion
    fi
}

reiniciar_ledger_h0b_f0() {
    [[ "${estado_m_h0b}" == retirado_por_la_accion &&
       "${estado_t_h0b}" == nunca_creado &&
       "${estado_wrapper_sin_h0b}" == retirado_por_la_accion &&
       "${estado_wrapper_nominal_h0b}" == nunca_creado &&
       "${estado_wrapper_error_h0b}" == nunca_creado &&
       "${estado_directorio_wrapper_h0b}" == retirado_por_la_accion &&
       "${estado_directorio_nominal_h0b}" == nunca_creado &&
       "${estado_directorio_error_h0b}" == nunca_creado &&
       "${estado_pruebas_t_h0b}" == nunca_creado &&
       "${estado_componentes_t_h0b}" == nunca_creado ]] || return 65
    acreditar_ausencia_interior_h0b_f0 "${destino_m080_h0b}" || return 65
    acreditar_ausencia_interior_h0b_f0 "${destino_t080_h0b%/*/*}" || return 65
    acreditar_ausencia_interior_h0b_f0 "${directorio_wrapper_h0b}" || return 65
    forma_m_h0b='' forma_t_h0b='' forma_wrapper_sin_h0b=''
    forma_wrapper_nominal_h0b='' forma_wrapper_error_h0b=''
    forma_directorio_wrapper_h0b='' forma_directorio_nominal_h0b=''
    forma_directorio_error_h0b=''
    estado_m_h0b=nunca_creado; estado_t_h0b=nunca_creado
    estado_wrapper_sin_h0b=nunca_creado; estado_wrapper_nominal_h0b=nunca_creado
    estado_wrapper_error_h0b=nunca_creado; estado_directorio_wrapper_h0b=nunca_creado
    estado_directorio_nominal_h0b=nunca_creado; estado_directorio_error_h0b=nunca_creado
}

preparar_integracion_h0b_f0() {
    local modo="$1" prefijo='' directorio destino_wrapper="${destino_wrapper_h0b}"
    directorio="${temporales}/integracion-h0b-${modo}"
    local migracion="${directorio}/integracion-h0b-m080.sql"
    local prueba="${directorio}/integracion-h0b-t080.sql"
    local wrapper="${directorio}/integracion-h0b-wrapper.sql"
    [[ "${modo}" != nominal ]] || prefijo=N
    [[ "${modo}" != error ]] || prefijo=E
    [[ "${modo}" != nominal ]] || destino_wrapper="${destino_wrapper_nominal_h0b}"
    [[ "${modo}" != error ]] || destino_wrapper="${destino_wrapper_error_h0b}"
    [[ -z "${prefijo}" ]] || crear_directorio_wrapper_h0b_f0 "${destino_wrapper%/*}" || return 65
    inyectar_frontera_h0b_f0 "${prefijo}01" || return $?
    materializar_integracion_c2_virtual_h0b_f0 "${directorio}" "${modo}" || return 65
    inyectar_frontera_h0b_f0 "${prefijo}02" || return $?
    if [[ "${modo}" == error ]]; then
        retirar_para_sustitucion_h0b_f0 "${destino_m080_h0b}" || return 65
        retirar_para_sustitucion_h0b_f0 "${destino_t080_h0b}" || return 65
    fi
    copiar_componente_sintetico_f0 "${migracion}" "${destino_m080_h0b}" \
        "${prefijo}03" "${prefijo}04" || return $?
    [[ "${modo}" == 'sin-r0' ]] || copiar_componente_sintetico_f0 \
        "${prueba}" "${destino_t080_h0b}" "${prefijo}05" "${prefijo}06" || return $?
    wrapper_activo="${destino_wrapper}"
    local variable_wrapper variable_estado_wrapper forma_wrapper
    variable_wrapper="$(variable_forma_archivo_h0b_f0 "${wrapper_activo}")" || return 65
    variable_estado_wrapper="$(variable_estado_archivo_h0b_f0 "${wrapper_activo}")" || return 65
    local -n estado_wrapper="${variable_estado_wrapper}"
    [[ "${estado_wrapper}" == nunca_creado ]] || return 65
    acreditar_ausencia_interior_h0b_f0 "${wrapper_activo}" || return 65
    docker cp "${wrapper}" "${contenedor}:${wrapper_activo}" || return 65
    capturar_salida_f0 forma_wrapper forma_interior_h0b_f0 archivo "${wrapper_activo}" || return 65
    printf -v "${variable_wrapper}" %s "${forma_wrapper}"
    estado_wrapper=registrado_presente
    inyectar_frontera_h0b_f0 "${prefijo}07" || return $?
    comparar_huellas_f0 "${wrapper}" "${wrapper_activo}" || return 65
    inyectar_frontera_h0b_f0 "${prefijo}08"
}

ejecutar_wrapper_h0b_f0() {
    local modo="$1" prefijo=N usuario=postgres estado=0
    local salida="${temporales}/integracion-h0b.out"
    local error="${temporales}/integracion-h0b.err"
    [[ "${modo}" != 'sin-r0' ]] || usuario=vec_f0_h0_migrador
    [[ "${modo}" != error ]] || prefijo=E
    docker exec "${contenedor}" psql -XAtq --set ON_ERROR_STOP=1 \
        --username "${usuario}" --dbname postgres --file "${wrapper_activo}" \
        >"${salida}" 2>"${error}" || estado=$?
    inyectar_frontera_h0b_f0 "${prefijo}09" || return $?
    validar_resultado_wrapper_c2_virtual_h0b_f0 \
        "${modo}" "${estado}" "${salida}" "${error}" || return 65
    inyectar_frontera_h0b_f0 "${prefijo}10"
}
