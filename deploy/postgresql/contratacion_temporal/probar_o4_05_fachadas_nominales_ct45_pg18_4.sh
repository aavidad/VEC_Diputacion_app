#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
patron_material_argumento='--env(=|[[:space:]])VEC_(CAPACIDAD|DECISION|MOTIVO|CONTEXTO|PAYLOAD|SOBRE|EVIDENCIA|RAIZ)[^[:space:]]*='
estado_material_argumentos=0
rg -q -- "$patron_material_argumento" "${BASH_SOURCE[0]}" ||
    estado_material_argumentos=$?
if (( estado_material_argumentos != 1 )); then
    printf 'el guion expone material VEC en argumentos de Docker\n' >&2
    exit 1
fi

# Composición: CT44 conserva su PostgreSQL 18.4 fijado por digest y acredita
# íntegramente motor, cursores, carreras y retirada antes de abrir CT45.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_motor_consultas_rrhh_pg18_4.sh"
: "${contenedor:?el ejecutor CT44 debe exponer PostgreSQL}"

paso() {
    printf '[O4-05:CT-000045:PG18.4] %s\n' "$1"
}

estado_ct45() {
    valor "SELECT pg_catalog.concat_ws('|',
        cobertura.version_esquema::text,
        consultas.version_esquema::text,
        (SELECT pg_catalog.count(*)::text
           FROM pg_catalog.pg_proc funcion
          WHERE funcion.pronamespace =
                'vec_contratacion_temporal'::regnamespace
            AND funcion.proname = ANY(ARRAY[
                'consultar_cuadro_rrhh_atestado_v1',
                'consultar_detalle_rrhh_atestado_v1'
            ]::name[]))
    )
      FROM vec_contratacion_temporal.control_migracion_cobertura_o4 cobertura
      CROSS JOIN
           vec_contratacion_temporal.control_migracion_consultas_rrhh consultas
     WHERE cobertura.control AND consultas.control"
}

comprobar_estado_ct45() {
    local esperado=$1
    local contexto=$2
    local obtenido
    obtenido="$(estado_ct45)"
    if [[ $obtenido != "$esperado" ]]; then
        printf 'estado CT45 alterado tras %s\n' "$contexto" >&2
        printf 'esperado=%s\nobtenido=%s\n' "$esperado" "$obtenido" >&2
        return 1
    fi
}

preparar_paquete_temporal() {
    docker exec "$contenedor" sh -eu -c '
        rm -rf /tmp/ct000045
        mkdir -p /tmp/ct000045
        cp /repo/contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.up.sql /tmp/ct000045/
        cp -a /repo/contratacion_temporal/migraciones/000045_componentes /tmp/ct000045/
    '
}

ejecutar_paquete_temporal() {
    docker exec --workdir /tmp/ct000045 "$contenedor" \
        psql -X -v ON_ERROR_STOP=1 -v VERBOSITY=verbose \
        --username postgres --dbname postgres \
        --file 000045_fachadas_nominales_consultas_rrhh.up.sql
}

esperar_fallo_inclusion() {
    local componente=$1
    local salida
    salida="$(mktemp "${TMPDIR:-/tmp}/vec-ct45-inclusion.XXXXXX")"
    temporales+=("$salida")
    if ejecutar_paquete_temporal >"$salida" 2>&1; then
        printf 'se esperaba fallo por componente ausente: %s\n' \
            "$componente" >&2
        return 1
    fi
    if ! rg -Fq "$componente" "$salida"; then
        printf 'fallo inesperado al omitir %s\n' "$componente" >&2
        tail -20 "$salida" >&2
        return 1
    fi
}
huella_catalogo_ct45() {
    local base=${1:-postgres}
    docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" --command "WITH catalogo AS (
        SELECT funcion.proname::text AS nombre,
               pg_catalog.pg_get_function_identity_arguments(
                   funcion.oid
               ) AS identidad,
               pg_catalog.pg_get_function_result(funcion.oid) AS resultado,
               funcion.prosrc, funcion.proconfig::text,
               funcion.proacl::text,
               pg_catalog.obj_description(
                   funcion.oid, 'pg_proc'
               ) AS comentario
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::regnamespace
           AND funcion.proname = ANY(ARRAY[
               'consultar_cuadro_rrhh_atestado_v1',
               'consultar_detalle_rrhh_atestado_v1'
           ]::name[])
    )
    SELECT pg_catalog.encode(pg_catalog.sha256(
               pg_catalog.convert_to(pg_catalog.string_agg(
                   pg_catalog.concat_ws(
                       E'\\x1f', nombre, identidad, resultado, prosrc,
                       proconfig, proacl, comentario
                   ), E'\\x1e' ORDER BY nombre
               ), 'UTF8')
           ), 'hex')
      FROM catalogo"
}
oides_catalogo_ct45() {
    local base=$1
    docker exec "$contenedor" psql -XAtq --set ON_ERROR_STOP=1 \
        --username postgres --dbname "$base" --command "
        SELECT pg_catalog.string_agg(
                   funcion.oid::text, ',' ORDER BY funcion.proname
               )
          FROM pg_catalog.pg_proc funcion
         WHERE funcion.pronamespace =
               'vec_contratacion_temporal'::regnamespace
           AND funcion.proname = ANY(ARRAY[
               'consultar_cuadro_rrhh_atestado_v1',
               'consultar_detalle_rrhh_atestado_v1'
           ]::name[])"
}

probar_deriva() {
    local descripcion=$1
    local mensaje=$2
    local mutacion=$3
    local restauracion=$4
    psql_admin --command "$mutacion" >/dev/null
    esperar_fallo "$descripcion" 55000 \
        "$mensaje" \
        archivo \
        contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.down.sql
    comprobar_estado_ct45 '25|9|2' "$descripcion"
    psql_admin --command "$restauracion" >/dev/null
}

extraer_vector() {
    local caso=$1
    local perfil=$2
    local preparar=${3:-true}
    local fila
    if [[ $preparar == true ]]; then
        psql_admin --command \
            "SELECT public.preparar_vector_cierre_ct43('$caso','$perfil')" \
            >/dev/null
    fi
    fila="$(valor "SELECT pg_catalog.concat_ws('|',
        capacidad::text, decision::text, motivo::text, contexto::text,
        persona_version::text, perfil_version::text, payload::text,
        cose::text, evidencia::text, spki::text
    ) FROM public.vectores_consulta_rrhh_v3
    WHERE caso='$caso' AND perfil='$perfil'")"
    IFS='|' read -r capacidad decision motivo contexto persona_version \
        perfil_version payload sobre evidencia raiz <<<"$fila"
    if [[ -z $capacidad || -z $decision || -z $motivo ||
        -z $contexto || -z $persona_version || -z $perfil_version ||
        -z $payload || -z $sobre || -z $evidencia || -z $raiz ]]; then
        printf 'vector CT45 incompleto: %s/%s\n' "$caso" "$perfil" >&2
        return 1
    fi
}

efectos_decision() {
    local caso=$1
    valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.consumo_decision_v3
          WHERE decision_ref='decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
          WHERE decision_ref='decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal.registro_acceso_rrhh
          WHERE decision_ref='decision:consulta-rrhh:$caso'),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2
          WHERE decision_ref='decision:consulta-rrhh:$caso')
    )"
}

efectos_totales() {
    valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.consumo_decision_v3),
        (SELECT pg_catalog.count(*)
           FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal.registro_acceso_rrhh),
        (SELECT pg_catalog.count(*)
           FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2)
    )"
}

probar_limites_frontera() {
    local perfil
    local variante
    local comunes=(
        forma_valida_no_utf8
        alcance_nulo organizacion_superior
        clase_ambito_superior ambito_superior
        consulta_nula
        capacidad_nula capacidad_inferior capacidad_superior
        decision_nula decision_inferior decision_superior
        motivo_nulo motivo_inferior motivo_superior
        contexto_nulo contexto_inferior contexto_superior
        persona_nula persona_inferior persona_superior persona_fraccionaria
        perfil_nulo perfil_inferior perfil_superior perfil_fraccionario
        payload_nulo payload_inferior payload_superior
        sobre_nulo sobre_inferior sobre_superior
        evidencia_nula evidencia_inferior evidencia_superior
        raiz_nula raiz_inferior raiz_superior
    )
    for perfil in cuadro detalle; do
        for variante in "${comunes[@]}"; do
            esperar_fallo "frontera $perfil/$variante CT45" 42501 \
                'consulta RRHH rechazada' \
                psql_runtime --command "
                    BEGIN TRANSACTION
                        ISOLATION LEVEL SERIALIZABLE READ WRITE;
                    SELECT vec_contratacion_temporal.invocar_limite_fachada_ct45(
                        '$perfil', '$variante'
                    )"
        done
    done
    for variante in cuadro_texto_superior cuadro_estado_superior \
        cuadro_fase_superior cuadro_cursor_superior
    do
        esperar_fallo "frontera cuadro/$variante CT45" 42501 \
            'consulta RRHH rechazada' \
            psql_runtime --command "
                BEGIN TRANSACTION
                    ISOLATION LEVEL SERIALIZABLE READ WRITE;
                SELECT vec_contratacion_temporal.invocar_limite_fachada_ct45(
                    'cuadro', '$variante'
                )"
    done
    esperar_fallo 'frontera detalle/expediente superior CT45' 42501 \
        'consulta RRHH rechazada' \
        psql_runtime --command "
            BEGIN TRANSACTION
                ISOLATION LEVEL SERIALIZABLE READ WRITE;
            SELECT vec_contratacion_temporal.invocar_limite_fachada_ct45(
                'detalle', 'detalle_expediente_superior'
            )"
}

invocar_cuadro_escalar() {
    local caso=$1
    local cierre=${2:-true}
    local preparar=${3:-true}
    local estado=${4:-}
    extraer_vector "$caso" cuadro "$preparar"
    VEC_CAPACIDAD="$capacidad" VEC_DECISION="$decision" \
    VEC_MOTIVO="$motivo" VEC_CONTEXTO="$contexto" \
    VEC_PERSONA="$persona_version" VEC_PERFIL="$perfil_version" \
    VEC_PAYLOAD="$payload" VEC_SOBRE="$sobre" \
    VEC_EVIDENCIA="$evidencia" VEC_RAIZ="$raiz" \
    VEC_CASO="$caso" VEC_CIERRE="$cierre" VEC_ESTADO="$estado" \
        docker exec --interactive \
        --env VEC_CAPACIDAD --env VEC_DECISION --env VEC_MOTIVO \
        --env VEC_CONTEXTO --env VEC_PERSONA --env VEC_PERFIL \
        --env VEC_PAYLOAD --env VEC_SOBRE --env VEC_EVIDENCIA \
        --env VEC_RAIZ --env VEC_CASO --env VEC_CIERRE --env VEC_ESTADO \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv capacidad VEC_CAPACIDAD
\getenv decision VEC_DECISION
\getenv motivo VEC_MOTIVO
\getenv contexto VEC_CONTEXTO
\getenv persona VEC_PERSONA
\getenv perfil VEC_PERFIL
\getenv payload VEC_PAYLOAD
\getenv sobre VEC_SOBRE
\getenv evidencia VEC_EVIDENCIA
\getenv raiz VEC_RAIZ
\getenv caso VEC_CASO
\getenv cierre VEC_CIERRE
\getenv estado VEC_ESTADO
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=public,pg_catalog;
SET LOCAL row_security=off;
SELECT pg_catalog.set_config('vec.prueba_ct43_sqlstate', $1, true)
\parse configurar_estado
\bind_named configurar_estado :estado
\g /dev/null
SELECT (
    pg_catalog.encode(
        pg_catalog.sha256(resultado.contenido_canonico), 'hex'
    ) = resultado.contenido_huella_sha256
    AND resultado.esquema =
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2'
    AND resultado.total >= 0
)
  FROM vec_contratacion_temporal
       .consultar_cuadro_rrhh_atestado_v1(
           ROW($1::text,$2::text,$3::text)::
               vec_contratacion_temporal.alcance_consulta_rrhh_v1,
           ROW($4::text,$5::text,$6::text,$7::smallint,$8::text)::
               vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
           $9::bytea,$10::bytea,$11::bytea,$12::bytea,
           $13::numeric,$14::numeric,$15::bytea,$16::bytea,
           $17::bytea,$18::bytea
       ) resultado
\parse consultar_cuadro
\bind_named consultar_cuadro 'organizacion:diputacion-granada' 'organizacion' 'organizacion:diputacion-granada' '' '' '' '10' '' :capacidad :decision :motivo :contexto :persona :perfil :payload :sobre :evidencia :raiz
\g
\if :cierre
COMMIT;
\else
ROLLBACK;
\endif
SQL
}

invocar_detalle_escalar() {
    local caso=$1
    local cierre=${2:-true}
    local preparar=${3:-true}
    local estado=${4:-}
    extraer_vector "$caso" detalle "$preparar"
    VEC_CAPACIDAD="$capacidad" VEC_DECISION="$decision" \
    VEC_MOTIVO="$motivo" VEC_CONTEXTO="$contexto" \
    VEC_PERSONA="$persona_version" VEC_PERFIL="$perfil_version" \
    VEC_PAYLOAD="$payload" VEC_SOBRE="$sobre" \
    VEC_EVIDENCIA="$evidencia" VEC_RAIZ="$raiz" \
    VEC_CASO="$caso" VEC_CIERRE="$cierre" VEC_ESTADO="$estado" \
        docker exec --interactive \
        --env VEC_CAPACIDAD --env VEC_DECISION --env VEC_MOTIVO \
        --env VEC_CONTEXTO --env VEC_PERSONA --env VEC_PERFIL \
        --env VEC_PAYLOAD --env VEC_SOBRE --env VEC_EVIDENCIA \
        --env VEC_RAIZ --env VEC_CASO --env VEC_CIERRE --env VEC_ESTADO \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv capacidad VEC_CAPACIDAD
\getenv decision VEC_DECISION
\getenv motivo VEC_MOTIVO
\getenv contexto VEC_CONTEXTO
\getenv persona VEC_PERSONA
\getenv perfil VEC_PERFIL
\getenv payload VEC_PAYLOAD
\getenv sobre VEC_SOBRE
\getenv evidencia VEC_EVIDENCIA
\getenv raiz VEC_RAIZ
\getenv caso VEC_CASO
\getenv cierre VEC_CIERRE
\getenv estado VEC_ESTADO
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=public,pg_catalog;
SET LOCAL row_security=off;
SELECT pg_catalog.set_config('vec.prueba_ct43_sqlstate', $1, true)
\parse configurar_estado
\bind_named configurar_estado :estado
\g /dev/null
SELECT (
    pg_catalog.encode(
        pg_catalog.sha256(resultado.contenido_canonico), 'hex'
    ) = resultado.contenido_huella_sha256
    AND resultado.esquema =
        'vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2'
    AND resultado.expediente_ref = 'expediente:rrhh:minimizado'
    AND resultado.version_expediente = 1
)
  FROM vec_contratacion_temporal
       .consultar_detalle_rrhh_atestado_v1(
           ROW($1::text,$2::text,$3::text)::
               vec_contratacion_temporal.alcance_consulta_rrhh_v1,
           ROW($4::text,$5::numeric)::
               vec_contratacion_temporal.consulta_detalle_rrhh_v1,
           $6::bytea,$7::bytea,$8::bytea,$9::bytea,
           $10::numeric,$11::numeric,$12::bytea,$13::bytea,
           $14::bytea,$15::bytea
       ) resultado
\parse consultar_detalle
\bind_named consultar_detalle 'organizacion:diputacion-granada' 'organizacion' 'organizacion:diputacion-granada' 'expediente:rrhh:minimizado' '1' :capacidad :decision :motivo :contexto :persona :perfil :payload :sobre :evidencia :raiz
\g
\if :cierre
COMMIT;
\else
ROLLBACK;
\endif
SQL
}

for requerido in \
    migraciones/000045_fachadas_nominales_consultas_rrhh.up.sql \
    migraciones/000045_fachadas_nominales_consultas_rrhh.down.sql \
    migraciones/000045_componentes/010_guardas_frontera.sql \
    migraciones/000045_componentes/020_fachada_cuadro.sql \
    migraciones/000045_componentes/030_fachada_detalle.sql \
    migraciones/000045_componentes/090_acl_catalogo.sql \
    migraciones/000045_componentes/095_avance_barreras.sql
do
    [[ -f "$directorio/$requerido" ]] || {
        printf 'falta artefacto CT45: %s\n' "$requerido" >&2
        exit 1
    }
done

paso 'omisión y mutación individual revierten el paquete completo'
estado_base='24|8|0'
componentes=(
    010_guardas_frontera.sql
    020_fachada_cuadro.sql
    030_fachada_detalle.sql
    090_acl_catalogo.sql
    095_avance_barreras.sql
)
for componente in "${componentes[@]}"; do
    preparar_paquete_temporal
    docker exec "$contenedor" rm \
        "/tmp/ct000045/000045_componentes/$componente"
    esperar_fallo_inclusion "$componente"
    comprobar_estado_ct45 "$estado_base" "ausencia de $componente"
    preparar_paquete_temporal
    docker exec --env VEC_COMPONENTE="$componente" \
        "$contenedor" sh -eu -c \
        'printf "\\nSELECT 1 / 0;\\n" >> "/tmp/ct000045/000045_componentes/$VEC_COMPONENTE"'
    esperar_fallo "mutación de $componente CT45" 22012 \
        'division by zero' ejecutar_paquete_temporal
    comprobar_estado_ct45 "$estado_base" "mutación de $componente"
done
paso 'tres bases frescas divergen en OID y conservan el mismo manifiesto'
huellas_frescas=()
oides_frescos=()
docker exec "$contenedor" psql -X -v ON_ERROR_STOP=1 \
    --username postgres --dbname template1 --command \
    'CREATE DATABASE ct45_base_fresca TEMPLATE postgres' >/dev/null
for ciclo in 1 2 3; do
    base_fresca="ct45_fresca_$ciclo"
    docker exec --interactive --env VEC_BASE_FRESCA="$base_fresca" \
        "$contenedor" psql -Xv ON_ERROR_STOP=1 -U postgres -d template1 <<'SQL' >/dev/null
\getenv base_fresca VEC_BASE_FRESCA
CREATE DATABASE :"base_fresca" TEMPLATE ct45_base_fresca;
REVOKE ALL PRIVILEGES ON DATABASE :"base_fresca" FROM PUBLIC, vec_contratacion_temporal_consultor_rrhh;
GRANT CONNECT ON DATABASE :"base_fresca" TO vec_contratacion_temporal_consultor_rrhh;
SQL
    docker exec "$contenedor" psql -X -v ON_ERROR_STOP=1 \
        --username postgres --dbname "$base_fresca" \
        --file \
        /repo/contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.up.sql \
        >/dev/null
    huellas_frescas+=("$(huella_catalogo_ct45 "$base_fresca")")
    oides_frescos+=("$(oides_catalogo_ct45 "$base_fresca")")
done
[[ ${huellas_frescas[0]} == "${huellas_frescas[1]}" &&
    ${huellas_frescas[1]} == "${huellas_frescas[2]}" ]]
[[ ${oides_frescos[0]} != "${oides_frescos[1]}" &&
    ${oides_frescos[1]} != "${oides_frescos[2]}" ]]
for ciclo in 1 2 3; do
    docker exec "$contenedor" psql -X -v ON_ERROR_STOP=1 \
        --username postgres --dbname template1 --command \
        "DROP DATABASE ct45_fresca_$ciclo" >/dev/null
done
docker exec "$contenedor" psql -X -v ON_ERROR_STOP=1 \
    --username postgres --dbname template1 --command \
    'DROP DATABASE ct45_base_fresca' >/dev/null
paso 'UP, reentradas, DOWN y tres huellas semánticas iguales'
huellas=()
for ciclo in 1 2 3; do
    archivo \
        contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.up.sql
    comprobar_estado_ct45 '25|9|2' "UP $ciclo"
    huellas+=("$(huella_catalogo_ct45)")
    esperar_fallo "reentrada UP CT45 ciclo $ciclo" 55000 \
        'estado incompatible para fachadas nominales RRHH' \
        archivo \
        contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.up.sql
    if (( ciclo < 3 )); then
        archivo \
            contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.down.sql
        comprobar_estado_ct45 "$estado_base" "DOWN $ciclo"
        esperar_fallo "reentrada DOWN CT45 ciclo $ciclo" 55000 \
            'estado incompatible para revertir fachadas nominales RRHH' \
            archivo \
            contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.down.sql
    fi
done
[[ ${huellas[0]} == "${huellas[1]}" &&
    ${huellas[1]} == "${huellas[2]}" ]]

paso 'firmas, ACL, topología y límites hostiles'
archivo contratacion_temporal/pruebas_sql/o405_ct45_contrato_acl.sql
archivo contratacion_temporal/pruebas_sql/o405_ct45_limites_frontera.sql
estado_antes_limites="$(efectos_totales)"
probar_limites_frontera
[[ "$(efectos_totales)" == "$estado_antes_limites" ]]
psql_admin <<'SQL' >/dev/null
REVOKE EXECUTE ON FUNCTION vec_contratacion_temporal.invocar_limite_fachada_ct45(text, text)
FROM vec_c2d2_registro_runtime;
DROP FUNCTION vec_contratacion_temporal.invocar_limite_fachada_ct45(text, text);
GRANT TEMPORARY ON DATABASE postgres TO vec_c2d2_registro_runtime;
SQL
esperar_fallo 'runtime intenta almacenar un tipo CT40' 42501 \
    'permission denied for type vec_contratacion_temporal.alcance_consulta_rrhh_v1' \
    psql_admin --set ON_ERROR_STOP=1 --command "
        SET SESSION AUTHORIZATION vec_c2d2_registro_runtime;
        CREATE TEMP TABLE no_autorizada_ct45 (
            alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1
        )"
psql_admin --command \
    'REVOKE TEMPORARY ON DATABASE postgres FROM vec_c2d2_registro_runtime' \
    >/dev/null
[[ "$(valor "SELECT pg_catalog.has_database_privilege(
    'vec_c2d2_registro_runtime', 'postgres', 'TEMP'
)")" == 'f' ]]

paso 'la fachada exige SERIALIZABLE READ WRITE'
for aislamiento in 'READ COMMITTED' 'REPEATABLE READ'; do
    esperar_fallo "aislamiento $aislamiento" 42501 \
        'consulta RRHH rechazada' \
        psql_runtime --command "
            BEGIN TRANSACTION ISOLATION LEVEL $aislamiento READ WRITE;
            SELECT * FROM vec_contratacion_temporal
                .consultar_cuadro_rrhh_atestado_v1(
                    NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL
                )"
done
esperar_fallo 'transacción de solo lectura' 42501 \
    'consulta RRHH rechazada' \
    psql_runtime --command "
        BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ ONLY;
        SELECT * FROM vec_contratacion_temporal
            .consultar_detalle_rrhh_atestado_v1(
                NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL
        )"

paso 'puentes e identidad indirecta se rechazan y son reversibles'
psql_admin <<'SQL' >/dev/null
CREATE ROLE puente_hostil_ct45 NOLOGIN;
GRANT puente_hostil_ct45
    TO vec_contratacion_temporal_consultor_rrhh
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
esperar_fallo 'puente de salida del grupo consultor: cuadro' 42501 \
    'consulta RRHH rechazada' \
    invocar_cuadro_escalar ct45_puente_cuadro
esperar_fallo 'puente de salida del grupo consultor: detalle' 42501 \
    'consulta RRHH rechazada' \
    invocar_detalle_escalar ct45_puente_detalle
psql_admin <<'SQL' >/dev/null
REVOKE puente_hostil_ct45
    FROM vec_contratacion_temporal_consultor_rrhh;
DROP ROLE puente_hostil_ct45;
CREATE ROLE puente_indirecto_ct45 NOLOGIN;
CREATE ROLE login_indirecto_ct45
    LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
GRANT vec_contratacion_temporal_consultor_rrhh
    TO puente_indirecto_ct45
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
GRANT puente_indirecto_ct45
    TO login_indirecto_ct45
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
SQL
for nombre in cuadro detalle; do
    esperar_fallo "login indirecto: $nombre" 42501 \
        'consulta RRHH rechazada' \
        docker exec "$contenedor" psql -X \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username login_indirecto_ct45 --dbname postgres \
        --command "
            BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
            SELECT * FROM vec_contratacion_temporal
                .consultar_${nombre}_rrhh_atestado_v1(
                    NULL,NULL,NULL,NULL,NULL,NULL,
                    NULL,NULL,NULL,NULL,NULL,NULL
                )"
done
psql_admin <<'SQL' >/dev/null
DROP OWNED BY login_indirecto_ct45;
DROP ROLE login_indirecto_ct45;
DROP OWNED BY puente_indirecto_ct45;
DROP ROLE puente_indirecto_ct45;
SQL

paso 'binds escalares sin USAGE, éxito atómico y rollback sin efectos'
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION public.consultar_cuadro_rrhh_atestado_v1(text)
RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT $1';
CREATE FUNCTION public.consultar_detalle_rrhh_atestado_v1(text)
RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT $1';
CREATE FUNCTION
vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(text)
RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT $1';
CREATE FUNCTION
vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(text)
RETURNS text LANGUAGE sql IMMUTABLE AS 'SELECT $1';
REVOKE ALL ON FUNCTION
    public.consultar_cuadro_rrhh_atestado_v1(text),
    public.consultar_detalle_rrhh_atestado_v1(text),
    vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(text),
    vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(text)
FROM PUBLIC;
SQL
[[ "$(invocar_cuadro_escalar ct45_cuadro_confirmado)" == 't' ]]
[[ "$(efectos_decision ct45_cuadro_confirmado)" == '1|1|1|1' ]]
[[ "$(invocar_detalle_escalar ct45_detalle_confirmado)" == 't' ]]
[[ "$(efectos_decision ct45_detalle_confirmado)" == '1|1|1|1' ]]
[[ "$(invocar_cuadro_escalar ct45_cuadro_revertido false)" == 't' ]]
[[ "$(efectos_decision ct45_cuadro_revertido)" == '0|0|0|0' ]]
[[ "$(invocar_detalle_escalar ct45_detalle_revertido false)" == 't' ]]
[[ "$(efectos_decision ct45_detalle_revertido)" == '0|0|0|0' ]]
psql_admin <<'SQL' >/dev/null
DROP FUNCTION public.consultar_cuadro_rrhh_atestado_v1(text);
DROP FUNCTION public.consultar_detalle_rrhh_atestado_v1(text);
DROP FUNCTION
    vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(text);
DROP FUNCTION
    vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(text);
SQL

paso 'cada pieza VEC mutada y cada constante cruzada fallan sin efectos'
for especificacion in \
    "capacidad|capacidad || pg_catalog.decode('00','hex')" \
    "decision|decision || pg_catalog.decode('00','hex')" \
    "motivo|motivo || pg_catalog.decode('00','hex')" \
    "contexto|contexto || pg_catalog.decode('00','hex')" \
    'persona_version|persona_version + 1' \
    'perfil_version|perfil_version + 1' \
    "payload|payload || pg_catalog.decode('00','hex')" \
    "cose|cose || pg_catalog.decode('00','hex')" \
    "evidencia|evidencia || pg_catalog.decode('00','hex')" \
    'spki|pg_catalog.set_byte(spki,0,pg_catalog.get_byte(spki,0)#1)'
do
    IFS='|' read -r columna expresion <<<"$especificacion"
    caso="ct45_mut_${columna}"
    psql_admin --command "
        SELECT public.preparar_vector_cierre_ct43('$caso','cuadro');
        UPDATE public.vectores_consulta_rrhh_v3
           SET $columna = $expresion
         WHERE caso='$caso' AND perfil='cuadro'" >/dev/null
    esperar_fallo "pieza VEC mutada: $columna" 42501 \
        'consulta RRHH rechazada' \
        invocar_cuadro_escalar "$caso" true false
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
done
for especificacion in \
    'accion|accion.ajena' 'modulo_id|modulo_ajeno' \
    'tipo_recurso|recurso_ajeno' 'finalidad|finalidad_ajena'
do
    IFS='|' read -r campo valor_ajeno <<<"$especificacion"
    caso="ct45_const_${campo}"
    psql_admin --command "
        SELECT public.preparar_vector_cierre_ct43('$caso','cuadro');
        SELECT public.adulterar_decision_consulta_rrhh_v3(
            '$caso','$campo','$valor_ajeno'
        )" >/dev/null
    esperar_fallo "constante de decisión: $campo" 42501 \
        'consulta RRHH rechazada' \
        invocar_cuadro_escalar "$caso" true false
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
done
for especificacion in \
    'audiencia_consumo|audiencia.ajena' 'operacion|operacion.ajena'
do
    IFS='|' read -r campo valor_ajeno <<<"$especificacion"
    caso="ct45_const_${campo}"
    psql_admin --command "
        SELECT public.preparar_vector_cierre_ct43('$caso','cuadro');
        SELECT public.adulterar_capacidad_consulta_rrhh_v3(
            '$caso','$campo','$valor_ajeno'
        )" >/dev/null
    esperar_fallo "constante de capacidad: $campo" 42501 \
        'consulta RRHH rechazada' \
        invocar_cuadro_escalar "$caso" true false
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
done

paso 'replay, SQLSTATE transitorios y carrera dejan un único efecto'
[[ "$(invocar_cuadro_escalar ct45_replay)" == 't' ]]
esperar_fallo 'replay de capacidad CT45' 42501 \
    'consulta RRHH rechazada' \
    invocar_cuadro_escalar ct45_replay true false
[[ "$(efectos_decision ct45_replay)" == '1|1|1|1' ]]
psql_admin --command 'CREATE TRIGGER forzar_sqlstate_ct45 BEFORE INSERT ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 FOR EACH ROW EXECUTE FUNCTION public.forzar_sqlstate_prueba_ct43()' >/dev/null
for estado in 40001 40P01 55P03 57014; do
    caso="ct45_estado_$estado"
    esperar_fallo "propagación SQLSTATE $estado CT45" "$estado" \
        'estado transitorio sintético CT43' \
        invocar_detalle_escalar "$caso" true true "$estado"
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
done
psql_admin --command 'DROP TRIGGER forzar_sqlstate_ct45 ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2' >/dev/null
extraer_vector ct45_carrera cuadro true
salida_a="$(mktemp "${TMPDIR:-/tmp}/ct45-carrera-a.XXXXXX")"
salida_b="$(mktemp "${TMPDIR:-/tmp}/ct45-carrera-b.XXXXXX")"
temporales+=("$salida_a" "$salida_b")
(invocar_cuadro_escalar ct45_carrera true false >"$salida_a" 2>&1) &
pid_a=$!
(invocar_cuadro_escalar ct45_carrera true false >"$salida_b" 2>&1) &
pid_b=$!
set +e
wait "$pid_a"; estado_a=$?
wait "$pid_b"; estado_b=$?
set -e
if (( (estado_a == 0) + (estado_b == 0) != 1 )); then
    printf 'la carrera CT45 no dejó un único ganador: %s/%s\n' \
        "$estado_a" "$estado_b" >&2
    exit 1
fi
if (( estado_a == 0 )); then salida_perdedor=$salida_b
else salida_perdedor=$salida_a
fi
if rg -Fq '40001' "$salida_perdedor"; then
    esperar_fallo 'reintento tras serialización CT45' 42501 \
        'consulta RRHH rechazada' \
        invocar_cuadro_escalar ct45_carrera true false
else
    rg -Fq '42501' "$salida_perdedor"
    rg -Fq 'consulta RRHH rechazada' "$salida_perdedor"
fi
[[ "$(efectos_decision ct45_carrera)" == '1|1|1|1' ]]

paso 'cinco salidas NULL del motor se rechazan y revierten'
archivo contratacion_temporal/pruebas_sql/o405_ct45_salidas_nulas.sql
for especificacion in 'cuadro|cursor_huella' 'cuadro|expediente' 'cuadro|version' 'detalle|cursor_huella' 'detalle|alcance_huella'; do
    IFS='|' read -r perfil variante <<<"$especificacion"; caso="ct45_nulo_${perfil}_${variante}"
    psql_admin --command "SELECT public.preparar_vector_cierre_ct43('$caso','$perfil')" >/dev/null
    esperar_fallo "salida NULL $perfil/$variante" 42501 'consulta RRHH rechazada' psql_runtime --command "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE; SELECT vec_contratacion_temporal.invocar_salida_nula_ct45('$caso','$perfil','${perfil}_${variante}')"
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
done
archivo contratacion_temporal/pruebas_sql/o405_ct45_salidas_nulas_retirar.sql
paso 'la barrera CT45 protege el safe-down de CT44'
esperar_fallo 'safe-down CT44 con CT45 instalada' 55000 \
    'estado incompatible para revertir motor RRHH' \
    archivo \
    contratacion_temporal/migraciones/000044_motor_consultas_rrhh.down.sql
comprobar_estado_ct45 '25|9|2' 'safe-down CT44'

paso 'barrera y dependencia futuras bloquean DOWN sin estado parcial'
psql_admin <<'SQL' >/dev/null
BEGIN;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 10 WHERE control AND version_esquema = 9;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 26 WHERE control AND version_esquema = 25;
COMMIT;
SQL
esperar_fallo 'barrera futura CT46' 55000 \
    'estado incompatible para revertir fachadas nominales RRHH' \
    archivo \
    contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.down.sql
comprobar_estado_ct45 '26|10|2' 'barrera futura CT46'
psql_admin <<'SQL' >/dev/null
BEGIN;
UPDATE vec_contratacion_temporal.control_migracion_consultas_rrhh
   SET version_esquema = 9 WHERE control AND version_esquema = 10;
UPDATE vec_contratacion_temporal.control_migracion_cobertura_o4
   SET version_esquema = 25 WHERE control AND version_esquema = 26;
COMMIT;
CREATE VIEW public.dependencia_futura_ct45 AS
SELECT *
  FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
       NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL
  );
SQL
esperar_fallo 'dependencia futura CT45' 2BP01 \
    'cannot drop function' \
    archivo \
    contratacion_temporal/migraciones/000045_fachadas_nominales_consultas_rrhh.down.sql
comprobar_estado_ct45 '25|9|2' 'dependencia futura CT45'
psql_admin --command 'DROP VIEW public.dependencia_futura_ct45' >/dev/null

paso 'derivas de ACL y configuración bloquean la retirada'
firma_cuadro='vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
probar_deriva 'ACL añadida CT45' \
    'ACL de fachadas nominales RRHH incompatible' \
    "GRANT EXECUTE ON FUNCTION $firma_cuadro TO vec_contratacion_temporal_ejecutor" \
    "REVOKE EXECUTE ON FUNCTION $firma_cuadro FROM vec_contratacion_temporal_ejecutor"
probar_deriva 'configuración hostil CT45' \
    'ACL de fachadas nominales RRHH incompatible' \
    "ALTER FUNCTION $firma_cuadro SET statement_timeout='3s'" \
    "ALTER FUNCTION $firma_cuadro SET statement_timeout='4s'"

paso 'CT-000045 superada sobre PostgreSQL 18.4 fijado por digest'
