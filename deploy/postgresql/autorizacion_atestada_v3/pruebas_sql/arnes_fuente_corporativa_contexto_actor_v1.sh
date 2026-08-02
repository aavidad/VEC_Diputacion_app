#!/usr/bin/env bash

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    printf 'el arnés F0 solo puede cargarse desde su runner acreditado\n' >&2
    exit 64
fi
if [[ "${VEC_F0_CARGA_PRIVADA:-}" != '1' ]]; then
    printf 'carga no acreditada del arnés F0\n' >&2
    return 64
fi
unset VEC_F0_CARGA_PRIVADA

validar_componentes_sql_f0() {
    local ruta="${1:-}"
    local directorio_temporal="${2:-}"
    local depurado lineas bytes bytes_despues
    if [[ -z "${ruta}" || -z "${directorio_temporal}" ||
          ! -f "${ruta}" || ! -r "${ruta}" || -L "${ruta}" ]]; then
        printf 'componente SQL F0 ausente, ilegible o no regular\n' >&2
        return 65
    fi
    bytes="$(stat --printf='%s' -- "${ruta}")" || return 65
    if [[ ! "${bytes}" =~ ^[0-9]+$ ]] || ((bytes == 0 || bytes > 1048576)); then
        printf 'componente SQL F0 excede los límites del artefacto\n' >&2
        return 65
    fi
    dd if="${ruta}" of=/dev/null iflag=nofollow,fullblock,count_bytes \
        count=1048577 status=none || return 65
    bytes_despues="$(stat --printf='%s' -- "${ruta}")" || return 65
    [[ "${bytes_despues}" == "${bytes}" ]] || return 65
    lineas="$(awk 'END { print NR }' "${ruta}")" || return 65
    if ((lineas == 0 || lineas >= 800)); then
        printf 'componente SQL F0 excede los límites del artefacto\n' >&2
        return 65
    fi
    depurado="$(mktemp "${directorio_temporal}/lexico-f0.XXXXXX")" || return 65
    if ! awk -v nombre="${ruta##*/}" '
function espacios(cantidad, j) {
    for (j = 0; j < cantidad; j++) salida = salida " "
}
function rechazar(motivo) {
    print "componente SQL F0 rechazado (" nombre "): " motivo > "/dev/stderr"
    fallo = 1
    exit 65
}
BEGIN {
    estado = "normal"
    profundidad = 0
    delimitador = ""
    fallo = 0
    continuacion_identificador = \
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_$"
}
{
    linea = $0 "\n"
    salida = ""
    i = 1
    while (i <= length(linea)) {
        c = substr(linea, i, 1)
        siguiente = substr(linea, i + 1, 1)
        if (estado == "bloque") {
            if (c == "/" && siguiente == "*") {
                profundidad++
                espacios(2)
                i += 2
            } else if (c == "*" && siguiente == "/") {
                profundidad--
                espacios(2)
                i += 2
                if (profundidad == 0) estado = "normal"
            } else {
                salida = salida (c == "\n" ? "\n" : " ")
                i++
            }
            continue
        }
        if (estado == "dolar") {
            if (substr(linea, i, length(delimitador)) == delimitador) {
                espacios(length(delimitador))
                i += length(delimitador)
                estado = "normal"
            } else {
                salida = salida (c == "\n" ? "\n" : " ")
                i++
            }
            continue
        }
        if (estado == "simple") {
            if (c == "\047" && siguiente == "\047") {
                espacios(2)
                i += 2
            } else if (c == "\047") {
                espacios(1)
                i++
                estado = "normal"
            } else {
                salida = salida (c == "\n" ? "\n" : " ")
                i++
            }
            continue
        }
        if (estado == "escape") {
            if (c == "\\" && i < length(linea)) {
                espacios(2)
                i += 2
            } else if (c == "\047" && siguiente == "\047") {
                espacios(2)
                i += 2
            } else if (c == "\047") {
                espacios(1)
                i++
                estado = "normal"
            } else {
                salida = salida (c == "\n" ? "\n" : " ")
                i++
            }
            continue
        }
        if (estado == "doble") {
            if (c == "\042" && siguiente == "\042") {
                espacios(2)
                i += 2
            } else if (c == "\042") {
                espacios(1)
                i++
                estado = "normal"
            } else {
                salida = salida (c == "\n" ? "\n" : " ")
                i++
            }
            continue
        }
        if (c == "-" && siguiente == "-") {
            while (i <= length(linea) && substr(linea, i, 1) != "\n") {
                salida = salida " "
                i++
            }
        } else if (c == "/" && siguiente == "*") {
            estado = "bloque"
            profundidad = 1
            espacios(2)
            i += 2
        } else if ((c == "E" || c == "e") && siguiente == "\047") {
            estado = "escape"
            espacios(2)
            i += 2
        } else if (c == "\047") {
            estado = "simple"
            espacios(1)
            i++
        } else if (c == "\042") {
            estado = "doble"
            espacios(1)
            i++
        } else if (c == "$") {
            resto = substr(linea, i)
            delimitador = ""
            previo = (i == 1 ? "" : substr(linea, i - 1, 1))
            frontera = (previo == "" || previo ~ /[[:space:]]/ ||
                        (previo ~ /^[ -~]$/ &&
                         index(continuacion_identificador, previo) == 0))
            if (frontera && substr(resto, 1, 2) == "$$") {
                delimitador = "$$"
            } else if (frontera &&
                       match(resto, /^\$[A-Za-z_][A-Za-z0-9_]*\$/)) {
                delimitador = substr(resto, RSTART, RLENGTH)
            }
            if (delimitador != "") {
                estado = "dolar"
                espacios(length(delimitador))
                i += length(delimitador)
            } else {
                salida = salida c
                i++
            }
        } else if (c == "\\") {
            rechazar("metacomando psql fuera de literal")
        } else {
            salida = salida c
            i++
        }
    }
    printf "%s", salida
}
END {
    if (!fallo && estado != "normal") {
        print "componente SQL F0 rechazado (" nombre "): literal o comentario sin cerrar" > "/dev/stderr"
        exit 65
    }
}
' "${ruta}" >"${depurado}"; then
        rm -f -- "${depurado}" || return 65
        return 65
    fi
    if ! awk -v nombre="${ruta##*/}" '
BEGIN { RS = ";"; fallo = 0 }
{
    sentencia = toupper($0)
    gsub(/[[:space:]]+/, " ", sentencia)
    sub(/^ /, "", sentencia)
    sub(/ $/, "", sentencia)
    if (sentencia == "") next
    if (sentencia ~ /^(BEGIN|COMMIT|END|ROLLBACK|ABORT)( |$)/ ||
        sentencia ~ /^START TRANSACTION( |$)/ ||
        sentencia ~ /^SAVEPOINT( |$)/ ||
        sentencia ~ /^RELEASE( SAVEPOINT)?( |$)/ ||
        sentencia ~ /^PREPARE TRANSACTION( |$)/) {
        print "componente SQL F0 rechazado (" nombre "): control transaccional superior" > "/dev/stderr"
        fallo = 1
        exit 65
    }
}
END { if (fallo) exit 65 }
' "${depurado}"; then
        rm -f -- "${depurado}" || return 65
        return 65
    fi
    rm -f -- "${depurado}" || return 65
}

clasificar_sqlstate_psql_f0() {
    local estado_salida="${1:-}"
    local ruta_error="${2:-}"
    local bytes ultimo_byte
    if [[ "${estado_salida}" != '3' || ! -f "${ruta_error}" ||
          -L "${ruta_error}" ]]; then
        printf 'sqlstate=invalido\n'
        return 0
    fi
    bytes="$(wc -c <"${ruta_error}")" || { printf 'sqlstate=invalido\n'; return 0; }
    ultimo_byte="$(tail -c 1 -- "${ruta_error}")" || { printf 'sqlstate=invalido\n'; return 0; }
    if [[ ! "${bytes}" =~ ^[0-9]+$ ]] || ((bytes == 0 || bytes > 4096)) ||
       [[ -n "${ultimo_byte}" ]]; then
        printf 'sqlstate=invalido\n'
        return 0
    fi
    awk '
BEGIN { valido = 1; cantidad = 0; codigo = "" }
{
    if ($0 ~ /^ERROR:[[:space:]]+(55P03|40P01)$/ ||
        $0 ~ /^psql:[^[:cntrl:]]+:[0-9]+:[[:space:]]+ERROR:[[:space:]]+(55P03|40P01)$/) {
        cantidad++
        codigo = substr($0, length($0) - 4)
    } else valido = 0
}
END {
    if (valido && cantidad == 1) print "sqlstate=" codigo
    else print "sqlstate=invalido"
}
' "${ruta_error}"
}

es_sqlstate_exacto_f0() {
    local estado_salida="${1:-}" ruta_error="${2:-}" esperado="${3:-}"
    local bytes ultimo_byte
    [[ "${estado_salida}" == '3' && "${esperado}" =~ ^[0-9A-Z]{5}$ &&
       -f "${ruta_error}" && ! -L "${ruta_error}" ]] || return 1
    bytes="$(wc -c <"${ruta_error}")" || return 1
    ultimo_byte="$(tail -c 1 -- "${ruta_error}")" || return 1
    [[ "${bytes}" =~ ^[0-9]+$ ]] && ((bytes > 0 && bytes <= 4096)) &&
        [[ -z "${ultimo_byte}" ]] || return 1
    awk -v esperado="${esperado}" '
BEGIN { cantidad = 0; valido = 1 }
{
    if ($0 ~ "^ERROR:[[:space:]]+" esperado "$" ||
        $0 ~ "^psql:[^[:cntrl:]]+:[0-9]+:[[:space:]]+ERROR:[[:space:]]+" esperado "$") cantidad++
    else valido = 0
}
END { exit !(valido && cantidad == 1) }
' "${ruta_error}"
}

ruta_componente_f0() {
    case "$1" in
        M010) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/010_validadores.sql' ;; T010) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/010_validadores.sql' ;;
        M020) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/020_canon_manifiesto.sql' ;; T020) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/020_canon_manifiesto.sql' ;;
        M030) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/030_canon_capacidad_mac.sql' ;; T030) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/030_canon_capacidad_mac.sql' ;;
        M040) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/040_canon_consumo.sql' ;; T040) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/040_canon_consumo.sql' ;;
        M050) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/050_catalogo_fuente_checkpoint.sql' ;; T050) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/050_catalogo_fuente_checkpoint.sql' ;;
        M060) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/060_atestacion_consumo.sql' ;; T060) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/060_atestacion_consumo.sql' ;;
        M070) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/070_acreditar_material_fuente.sql' ;; T070) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/070_acreditar_material_fuente.sql' ;;
        M080) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/080_consumidor_nominal.sql' ;; T080) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/080_consumidor_nominal.sql' ;;
        M090) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/090_acl_audiencias_centinela.sql' ;; T090) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/090_acl_audiencias_centinela.sql' ;;
        M810) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/810_acreditar_retirada.sql' ;; T810) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/810_acreditar_retirada.sql' ;;
        M820) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/820_retirar_objetos.sql' ;; T820) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/820_retirar_objetos.sql' ;;
        M830) printf 'deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/830_restaurar_audiencias.sql' ;; T830) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/830_restaurar_audiencias.sql' ;;
        T100) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/100_estructura_acl.sql' ;; T110) printf 'deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/110_consumo_replay_rollback.sql' ;;
        *) return 64 ;;
    esac
}

clausura_etapa_f0() {
    case "$1" in
        A1) printf 'M010 T010' ;; A2) printf 'M010 M020 T020' ;;
        A3) printf 'M010 M030 T030' ;; A4) printf 'M010 M040 T040' ;;
        B1) printf 'M010 M050 T050' ;; B2) printf 'M010 M050 M060 T060' ;;
        C1) printf 'M010 M020 M030 M050 M070 T070' ;;
        C2) printf 'M010 M020 M030 M040 M050 M060 M070 M080 T080' ;;
        C3|T1|T2|R1|R2a|R2b)
            printf 'M010 M020 M030 M040 M050 M060 M070 M080 M090 '
            case "$1" in
                C3) printf 'T090' ;; T1) printf 'T100' ;; T2) printf 'T110' ;;
                R1) printf 'M810 T810' ;; R2a) printf 'M810 M820 T820' ;;
                R2b) printf 'M810 M820 M830 T830' ;;
            esac ;;
        *) return 64 ;;
    esac
}

inventario_base_f0() {
    command cat <<'RUTAS'
deploy/postgresql/contexto_actor_v1/roles_up.sql
deploy/postgresql/contexto_actor_v1/migraciones/000001_contexto_actor_v1.up.sql
deploy/postgresql/contexto_actor_v1/migraciones/000002_acreditacion_uso_registro_contexto_actor_v2.up.sql
deploy/postgresql/contexto_actor_v1/pruebas_sql/fixtures_sinteticos.sql
deploy/postgresql/autorizacion/pruebas_sql/fixture_contexto_actor_v3.sql
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
deploy/postgresql/autorizacion_atestada_v3/roles_up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000001_gobierno_y_registro_v3.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000002_consumidor_capacidad_v3.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000005_revalidacion_final_consultas_rrhh_v3.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
RUTAS
}

inventario_etapa_f0() {
    local claves clave
    inventario_base_f0 || return 65
    if [[ "$1" == 'H0' ]]; then
        claves='M010 M020 M030 M040 M050 M060 M070'
    else
        claves="$(clausura_etapa_f0 "$1")" || return 64
    fi
    for clave in ${claves}; do
        ruta_componente_f0 "${clave}" || return 64
        printf '\n' || return 65
    done
}

capturar_inventario_f0() {
    local binario="$1"
    local -n salida_snapshot="$2"
    local -n salida_manifiesto="$3"
    local inventario="${temporales}/inventario-rutas"
    local -a rutas=()
    # shellcheck disable=SC2154
    inventario_etapa_f0 "${etapa}" >"${inventario}" || return 65
    mapfile -t rutas <"${inventario}" || return 65
    ((${#rutas[@]} > 0)) || return 65
    salida_snapshot="${temporales}/snapshot-sql"
    salida_manifiesto="${temporales}/manifiesto-sql"
    "${binario}" --raiz . --destino "${salida_snapshot}" \
        --manifiesto "${salida_manifiesto}" -- "${rutas[@]}" || return 65
    validar_manifiesto_snapshot_f0 "${salida_manifiesto}"
}

validar_componentes_snapshot_f0() {
    local snapshot="$1" claves clave ruta
    if [[ "${etapa}" == 'H0' ]]; then
        claves='M010 M020 M030 M040 M050 M060 M070'
    else
        claves="$(clausura_etapa_f0 "${etapa}")" || return 64
    fi
    for clave in ${claves}; do
        ruta="$(ruta_componente_f0 "${clave}")" || return 64
        validar_componentes_sql_f0 "${snapshot}/${ruta}" "${temporales}" ||
            return 65
    done
}

# shellcheck disable=SC2154
clausura_migraciones_etapa_f0() {
    local claves clave
    claves="$(clausura_etapa_f0 "$1")" || return 64
    for clave in ${claves}; do [[ "${clave}" == M* ]] && printf '%s ' "${clave}"; done
    return 0
}

derivar_manifiesto_base_h0_f0() {
    local origen="${1:-}" destino="${2:-}" temporal ruta huella adicional
    local retiradas=0
    [[ -n "${destino}" && ! -e "${destino}" && ! -L "${destino}" ]] || return 65
    validar_manifiesto_snapshot_f0 "${origen}" || return 65
    temporal="$(mktemp "${destino}.XXXXXX")" || return 65
    while IFS=$'\t' read -r ruta huella adicional; do
        case "${ruta}" in
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/010_validadores.sql|\
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/020_canon_manifiesto.sql|\
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/030_canon_capacidad_mac.sql|\
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/040_canon_consumo.sql|\
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/050_catalogo_fuente_checkpoint.sql|\
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/060_atestacion_consumo.sql|\
            deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_componentes/070_acreditar_material_fuente.sql)
                ((retiradas += 1)) ;;
            *) printf '%s\t%s\n' "${ruta}" "${huella}" >>"${temporal}" || {
                rm -f -- "${temporal}"; return 65;
            } ;;
        esac
    done <"${origen}"
    if ((retiradas != 7)) || ! validar_manifiesto_snapshot_f0 "${temporal}"; then
        rm -f -- "${temporal}"; return 65
    fi
    mv -- "${temporal}" "${destino}" || { rm -f -- "${temporal}"; return 65; }
}

inventario_i0_f0() {
    command cat <<'RUTAS'
deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_fuente_corporativa_contexto_actor_v1.up.sql
deploy/postgresql/autorizacion_atestada_v3/migraciones/000007_fuente_corporativa_contexto_actor_v1.down.sql
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/fuente_corporativa_contexto_actor_v1.sql
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/900_concurrencia_consumo_revocacion.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/910_retirada_dependencias_componentes.sh
deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/000007_componentes/920_regresion_consumidores_v3.sql
internal/vec/adapters/seguridad/confianzaatestacion/capacidad_fuente_corporativa_v1_vector_test.go
internal/vec/adapters/seguridad/confianzaatestacion/testdata/manifiesto_fuente_corporativa_v1.json
internal/vec/adapters/seguridad/confianzaatestacion/testdata/capacidad_fuente_corporativa_v1.json
internal/vec/adapters/seguridad/confianzaatestacion/testdata/consumo_fuente_corporativa_v1.json
RUTAS
}

inventario_completo_i0_f0() {
    local clave
    inventario_base_f0 || return 65
    for clave in M010 T010 M020 T020 M030 T030 M040 T040 M050 T050 \
        M060 T060 M070 T070 M080 T080 M090 T090 T100 T110 \
        M810 T810 M820 T820 M830 T830
    do ruta_componente_f0 "${clave}" || return 64; printf '\n' || return 65; done
    inventario_i0_f0 || return 65
}

validar_manifiesto_snapshot_f0() {
    local manifiesto="${1:-}"
    local anterior='' ruta huella adicional cantidad=0 final
    final="$(tail -c 1 -- "${manifiesto}")" || return 65
    [[ -f "${manifiesto}" && ! -L "${manifiesto}" &&
       -z "${final}" ]] || return 65
    while IFS=$'\t' read -r ruta huella adicional; do
        [[ -z "${adicional}" && "${ruta}" =~ ^[^/[:space:]][^[:space:]]*$ &&
           ! "${ruta}" =~ (^|/)\.\.?(/|$) && ! "${ruta}" =~ // &&
           "${huella}" =~ ^[0-9a-f]{64}$ ]] || return 65
        [[ -z "${anterior}" || "${anterior}" < "${ruta}" ]] || return 65
        anterior="${ruta}"
        ((cantidad += 1))
    done <"${manifiesto}"
    ((cantidad > 0))
}

probar_sql_valido_f0() {
    local descripcion="$1" temporales="$2" ruta
    ruta="$(mktemp "${temporales}/valido.XXXXXX.sql")" || return 1
    command cat >"${ruta}" || return 1
    validar_componentes_sql_f0 "${ruta}" "${temporales}" >/dev/null 2>&1 || {
        printf 'falso positivo del analizador: %s\n' "${descripcion}" >&2
        return 1
    }
}

probar_sql_rechazado_f0() {
    local descripcion="$1" temporales="$2" ruta
    ruta="$(mktemp "${temporales}/rechazado.XXXXXX.sql")" || return 1
    command cat >"${ruta}" || return 1
    if validar_componentes_sql_f0 "${ruta}" "${temporales}" >/dev/null 2>&1; then
        printf 'evasión aceptada por el analizador: %s\n' "${descripcion}" >&2
        return 1
    fi
}

probar_analizador_f0() {
    local temporales="$1" sentencia vacio sobredimensionado creciente marca
    probar_sql_valido_f0 'literales e identificadores' "${temporales}" <<'SQL'
SELECT 'BEGIN; COMMIT; ABORT;', E'ROLLBACK\nSAVEPOINT', "END";
SQL
    probar_sql_valido_f0 'comentarios y bloque anidado' "${temporales}" <<'SQL'
/* BEGIN; /* COMMIT; */ ROLLBACK; */
SELECT 1; -- ABORT; \ir oculto.sql
SQL
    probar_sql_valido_f0 'cuerpo PL/pgSQL' "${temporales}" <<'SQL'
CREATE FUNCTION ejemplo_f0() RETURNS void LANGUAGE plpgsql AS $cuerpo$
BEGIN
    PERFORM E'COMMIT\n\ir no_ejecutable.sql';
END
$cuerpo$;
SQL
    probar_sql_valido_f0 'dólar tras puntuación' "${temporales}" <<'SQL'
SELECT $texto$START TRANSACTION; \include no_ejecutable.sql$texto$;
SELECT ARRAY[$q$texto \ir no_ejecutable$q$];
SQL
    for sentencia in \
        'BEGIN;' ' START /* evasión */ TRANSACTION;' 'COMMIT;' 'END;' \
        'ROLLBACK;' 'SAVEPOINT s;' 'RELEASE /* evasión */ SAVEPOINT s;' \
        "PREPARE /* evasión */ TRANSACTION 'x';" "COMMIT PREPARED 'x';" \
        "ROLLBACK PREPARED 'x';" 'ABORT;' 'ABORT WORK;' \
        'ABORT TRANSACTION;' 'ABORT /* evasión */ WORK;' \
        '/* exterior /* interior */ */ \ir componente.sql' \
        'SELECT 1; \include componente.sql' "SELECT 'BEGIN;" \
        "CREATE TABLE vec_autorizacion_atestada_v3.evasion_f0 AS SELECT 1 AS foo\$tag\$col; COMMIT; SELECT 1 AS \$tag\$;"
    do
        printf '%s\n' "${sentencia}" |
            probar_sql_rechazado_f0 "${sentencia}" "${temporales}" || return
    done
    vacio="$(mktemp "${temporales}/vacio.XXXXXX.sql")" || return 1
    if validar_componentes_sql_f0 "${vacio}" "${temporales}" >/dev/null 2>&1; then
        printf 'el analizador aceptó un componente vacío\n' >&2
        return 1
    fi
    sobredimensionado="${temporales}/sobredimensionado.sql"
    marca="${temporales}/awk-invocado"
    truncate -s 1048577 "${sobredimensionado}" || return 1
    awk() { printf 'invocado\n' >"${marca}"; return 65; }
    if validar_componentes_sql_f0 "${sobredimensionado}" "${temporales}" >/dev/null 2>&1; then
        unset -f awk
        return 1
    fi
    unset -f awk
    [[ ! -e "${marca}" ]] || return 1
    creciente="${temporales}/creciente.sql"
    marca="${temporales}/awk-invocado-creciente"
    printf 'SELECT 1;\n' >"${creciente}" || return 1
    # shellcheck disable=SC2329
    dd() { truncate -s 1048577 "${creciente}" || return 65; command dd "$@"; }
    awk() { printf 'invocado\n' >"${marca}"; return 65; }
    if validar_componentes_sql_f0 "${creciente}" "${temporales}" >/dev/null 2>&1; then
        unset -f dd awk
        return 1
    fi
    unset -f dd awk
    [[ ! -e "${marca}" ]] || return 1
}

probar_clasificacion_f0() {
    local esperado="$1" estado_salida="$2" temporales="$3" contenido="$4"
    local ruta obtenido
    ruta="$(mktemp "${temporales}/sqlstate.XXXXXX.err")" || return 1
    printf '%s' "${contenido}" >"${ruta}" || return 1
    obtenido="$(clasificar_sqlstate_psql_f0 "${estado_salida}" "${ruta}")" || return 1
    [[ "${obtenido}" == "${esperado}" ]] || {
        printf 'clasificación SQLSTATE incorrecta: %s/%s\n' \
            "${esperado}" "${obtenido}" >&2
        return 1
    }
}

probar_fallo_dependencia_clasificador_f0() {
    local dependencia="$1" temporales="$2" ruta obtenido estado=0
    ruta="$(mktemp "${temporales}/sqlstate-fallo.XXXXXX.err")" || return 1
    printf 'ERROR:  55P03\n' >"${ruta}" || return 1
    case "${dependencia}" in
        wc) wc() { printf '14\n'; return 65; } ;;
        tail) tail() { return 65; } ;;
        *) return 1 ;;
    esac
    obtenido="$(clasificar_sqlstate_psql_f0 3 "${ruta}")" || estado=$?
    unset -f -- "${dependencia}" || return 1
    [[ "${estado}" == 0 && "${obtenido}" == 'sqlstate=invalido' ]] || {
        printf 'fallo de %s produjo una clasificación admisible\n' "${dependencia}" >&2
        return 1
    }
}

probar_clasificador_f0() {
    local temporales="$1"
    probar_clasificacion_f0 'sqlstate=55P03' 3 "${temporales}" $'ERROR:  55P03\n'
    probar_clasificacion_f0 'sqlstate=40P01' 3 "${temporales}" \
        $'psql:/repo/ensayo.sql:7: ERROR:  40P01\n'
    probar_clasificacion_f0 'sqlstate=invalido' 3 "${temporales}" $'ERROR:  23505\n'
    probar_clasificacion_f0 'sqlstate=invalido' 3 "${temporales}" \
        $'ERROR:  55P03\nERROR:  40P01\n'
    probar_clasificacion_f0 'sqlstate=invalido' 3 "${temporales}" \
        $'ERROR:  55P03\ndetalle humano\n'
    probar_clasificacion_f0 'sqlstate=invalido' 0 "${temporales}" $'ERROR:  55P03\n'
    probar_clasificacion_f0 'sqlstate=invalido' 3 "${temporales}" ''
    probar_clasificacion_f0 'sqlstate=invalido' 3 "${temporales}" 'ERROR:  55P03'
    probar_fallo_dependencia_clasificador_f0 wc "${temporales}"
    probar_fallo_dependencia_clasificador_f0 tail "${temporales}"
}
