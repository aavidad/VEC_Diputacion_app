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
declare -g temporales
# shellcheck disable=SC2154  # Variable aportada por el runner acreditado.
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
# shellcheck disable=SC2034  # Variable consumida por el runner acreditado.
    wrapper_activo=''
    return "${estado}"
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

# shellcheck disable=SC2154  # Aportado por el runner acreditado.
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
