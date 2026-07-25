#!/usr/bin/env bash
# shellcheck disable=SC2154

# Se carga desde el runner O2-05 una vez instaladas 000013, 000014 y 000015.
# Requiere sus helpers `archivo`, `sql`, `valor`, `preparar`, `invocar` y el
# contenedor efímero. No crea contenedores, redes ni procesos persistentes.

preparar_reserva_variante_o3_04() {
    local operacion="$1"
    local version="$2"
    local actor="$3"
    local perfil="$4"
    local artefacto="$5"
    local ambito_tail="$6"
    local huella_tail="$7"

    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_ct_o205_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT resultado
  FROM vec_contratacion_temporal.preparar_operacion_analisis_v1(
    jsonb_build_object(
      'esquema',
        'vec.contratacion-temporal.preparar-operacion-analisis.v1',
      'operacion','${operacion}',
      'organizacion_ref','organizacion:dipgra',
      'expediente_ref','expediente:ct:o205:rc_coste',
      'version_expediente',${version},
      'actor_ref','${actor}',
      'perfil_ref','${perfil}',
      'artefacto_ref','${artefacto}',
      'artefacto_huella_sha256',repeat('${ambito_tail}',64),
      'sellos_hmac',jsonb_build_object(
        'activo',jsonb_build_object(
          'generacion',2,
          'ambito_hmac',
            'hmac-sha256:vec.contratacion-temporal.analisis.ambito-idempotencia/v2:' ||
            repeat('${ambito_tail}',64),
          'huella_peticion_hmac',
            'hmac-sha256:vec.contratacion-temporal.analisis.huella-semantica/v2:' ||
            repeat('${huella_tail}',64)
        ),
        'retenidos',jsonb_build_array(jsonb_build_object(
          'generacion',1,
          'ambito_hmac',
            'hmac-sha256:vec.contratacion-temporal.analisis.ambito-idempotencia/v1:' ||
            repeat('${ambito_tail}',64),
          'huella_peticion_hmac',
            'hmac-sha256:vec.contratacion-temporal.analisis.huella-semantica/v1:' ||
            repeat('${huella_tail}',64)
        ))
      )
    )
  );
COMMIT;
SQL
}

invocar_variante_o3_04() {
    local caso="$1"
    docker exec --interactive "${contenedor}" \
        psql -XAtq --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_ct_o205_runtime --dbname postgres <<SQL
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_json::text
  FROM public.invocar_confirmacion_o3_variante('${caso}');
COMMIT;
SQL
}

esperar_rechazo_variante_o3_04() {
    local descripcion="$1"
    local caso="$2"
    local salida

    if salida="$(invocar_variante_o3_04 "${caso}" 2>&1)"; then
        printf 'Se esperaba rechazo O3-04 (%s):\n%s\n' \
            "${descripcion}" "${salida}" >&2
        return 1
    fi
}

limpiar_pruebas_variantes_o3_04() {
    sql postgres \
        'DROP FUNCTION IF EXISTS public.mutar_vector_o3_variante(text,text,text); DROP FUNCTION IF EXISTS public.invocar_confirmacion_o3_variante(text); DROP FUNCTION IF EXISTS public.construir_vector_confirmacion_o3_variante(text,text); DROP TABLE IF EXISTS public.vectores_confirmacion_analisis_o3_variantes' \
        >/dev/null
}

ejecutar_pruebas_variantes_o3_04() {
    local requerido
    local cadena_antes
    local cadena_despues
    local recibo_registro
    local replay_registro
    local recibo_rectificacion
    local replay_rectificacion

    for requerido in archivo sql valor preparar invocar; do
        if ! declare -F "${requerido}" >/dev/null; then
            printf 'Falta helper del runner para O3-04: %s\n' \
                "${requerido}" >&2
            return 64
        fi
    done
    if [[ -z "${contenedor:-}" ]]; then
        printf 'Falta contenedor efímero para O3-04\n' >&2
        return 64
    fi
    if [[ "$(valor \
        "SELECT (
           to_regprocedure(
             'vec_contratacion_temporal.confirmar_operacion_analisis_v3(jsonb)'
           ) IS NOT NULL
           AND to_regprocedure(
             'public.instante_go_analisis_o3(timestamp with time zone)'
           ) IS NOT NULL
         )::text")" != 'true' ]]; then
        printf 'O3-04 requiere V3 y el helper temporal del fixture base\n' >&2
        return 65
    fi

    # Alta O2 aislada: la historia comienza exactamente en v1.
    preparar rc_coste
    invocar rc_coste >/dev/null
    [[ "$(valor \
        "SELECT (
           (SELECT version
              FROM vec_contratacion_temporal.expediente_integral_actual
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=1
           AND
           (SELECT count(*)
              FROM vec_contratacion_temporal.expediente_version_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=1
         )::text")" == 'true' ]]

    archivo postgres \
        deploy/postgresql/contratacion_temporal/pruebas_sql/confirmacion_analisis_o3_variantes.sql \
        >/dev/null
    sql postgres \
        'GRANT USAGE ON SCHEMA public TO vec_ct_o205_runtime; GRANT EXECUTE ON FUNCTION public.invocar_confirmacion_o3_variante(text) TO vec_ct_o205_runtime' \
        >/dev/null

    cadena_antes="$(valor \
        "SELECT secuencia_auditoria::text || ':' ||
                secuencia_outbox::text
           FROM vec_contratacion_temporal
                .control_cadenas_expediente_integral
          WHERE control_id")"

    preparar_reserva_variante_o3_04 \
        registrar 1 \
        per_sintetica_bbbbbbbbbbbbbbbbbbbbbbbb \
        prf_sintetico_cccccccccccccccccccccccc \
        artefacto:analisis-rc-coste c d >/dev/null
    sql postgres \
        "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
         SELECT public.construir_vector_confirmacion_o3_variante(
           'registrar_rc_coste','registrar'
         );
         COMMIT" >/dev/null
    recibo_registro="$(invocar_variante_o3_04 registrar_rc_coste)"
    replay_registro="$(invocar_variante_o3_04 registrar_rc_coste)"
    [[ "${recibo_registro}" == "${replay_registro}" ]]
    [[ "${recibo_registro}" == *'"version_resultante": 2'* ]]
    [[ "$(valor \
        "SELECT (
           (SELECT version
              FROM vec_contratacion_temporal.expediente_integral_actual
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=2
           AND
           (SELECT agregado_json #>> '{analisis,validacion_rc,resultado}'
              FROM vec_contratacion_temporal.expediente_version_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste'
               AND version=2)='validada'
           AND
           (SELECT (agregado_json #>>
                      '{analisis,validacion_rc,importe,centimos}')::numeric
              FROM vec_contratacion_temporal.expediente_version_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste'
               AND version=2)=5000000
           AND
           (SELECT (agregado_json #>>
                      '{analisis,coste_previsto,centimos}')::numeric
              FROM vec_contratacion_temporal.expediente_version_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste'
               AND version=2)=4000000
         )::text")" == 'true' ]]

    preparar_reserva_variante_o3_04 \
        rectificar 2 \
        per_o3b_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb \
        prf_o3b_cccccccccccccccccccccccccccc \
        artefacto:rectificacion-rc-coste e f >/dev/null
    sql postgres \
        "BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
         SELECT public.construir_vector_confirmacion_o3_variante(
           'rectificar_rc_coste','rectificar'
         );
         COMMIT" >/dev/null

    # Ambas alteraciones conservan contrato y transición. Solo rompen el
    # compromiso exacto previamente autorizado por VEC.
    for requerido in unidad motivo; do
        sql postgres \
            "SELECT public.mutar_vector_o3_variante(
               'rectificar_rc_coste','ataque_${requerido}','${requerido}'
             )" >/dev/null
        [[ "$(valor \
            "SELECT (
               vec_contratacion_temporal
                 .entrada_confirmacion_analisis_valida_v2(v.operacion)
               AND vec_contratacion_temporal
                 .transicion_confirmacion_analisis_valida_v2(
                   v.operacion, actual.agregado_json
                 )
               AND vec_contratacion_temporal
                 .huella_contexto_recurso_analisis_v2(v.operacion)
                   IS DISTINCT FROM
                 v.operacion #>>
                   '{autorizacion,contexto_recurso_huella_sha256}'
             )::text
               FROM public.vectores_confirmacion_analisis_o3_variantes v
               CROSS JOIN LATERAL (
                 SELECT version.agregado_json
                   FROM vec_contratacion_temporal
                        .expediente_integral_actual puntero
                   JOIN vec_contratacion_temporal
                        .expediente_version_integral version
                     USING (expediente_ref,version)
                  WHERE puntero.expediente_ref=
                        'expediente:ct:o205:rc_coste'
               ) actual
              WHERE v.caso='ataque_${requerido}'")" == 'true' ]]
        esperar_rechazo_variante_o3_04 \
            "vínculo VEC de ${requerido}" "ataque_${requerido}"
        [[ "$(valor \
            "SELECT version
               FROM vec_contratacion_temporal.expediente_integral_actual
              WHERE expediente_ref='expediente:ct:o205:rc_coste'")" == '2' ]]
    done

    recibo_rectificacion="$(
        invocar_variante_o3_04 rectificar_rc_coste
    )"
    replay_rectificacion="$(
        invocar_variante_o3_04 rectificar_rc_coste
    )"
    [[ "${recibo_rectificacion}" == "${replay_rectificacion}" ]]
    [[ "${recibo_rectificacion}" == *'"version_resultante": 3'* ]]
    cadena_despues="$(valor \
        "SELECT secuencia_auditoria::text || ':' ||
                secuencia_outbox::text
           FROM vec_contratacion_temporal
                .control_cadenas_expediente_integral
          WHERE control_id")"

    [[ "$(valor \
        "WITH cadenas AS (
           SELECT split_part('${cadena_antes}',':',1)::bigint AS aud_antes,
                  split_part('${cadena_antes}',':',2)::bigint AS out_antes,
                  split_part('${cadena_despues}',':',1)::bigint AS aud_despues,
                  split_part('${cadena_despues}',':',2)::bigint AS out_despues
         )
         SELECT (
           (SELECT version
              FROM vec_contratacion_temporal.expediente_integral_actual
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=3
           AND
           (SELECT array_agg(version ORDER BY version)
              FROM vec_contratacion_temporal.expediente_version_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste')
               =ARRAY[1,2,3]::numeric[]
           AND
           (SELECT (agregado_json #>>
                      '{analisis,coste_previsto,centimos}')::numeric
              FROM vec_contratacion_temporal.expediente_version_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste'
               AND version=3)=3900000
           AND
           (SELECT count(*)
              FROM vec_contratacion_temporal.actuacion_expediente_integral
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=2
           AND
           (SELECT count(*)
              FROM vec_contratacion_temporal.consumo_fuentes_analisis
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=2
           AND
           (SELECT count(*)
              FROM vec_contratacion_temporal.consumo_decision_analisis d
              JOIN vec_contratacion_temporal.reserva_operacion_analisis r
                USING (ambito_raiz_hmac)
             WHERE r.expediente_ref='expediente:ct:o205:rc_coste')=2
           AND
           (SELECT count(*)
              FROM vec_contratacion_temporal.reserva_operacion_analisis
             WHERE expediente_ref='expediente:ct:o205:rc_coste')=2
           AND
           (SELECT count(*)
              FROM vec_contratacion_temporal.confirmacion_operacion_analisis c
              JOIN vec_contratacion_temporal.reserva_operacion_analisis r
                USING (ambito_raiz_hmac)
             WHERE r.expediente_ref='expediente:ct:o205:rc_coste')=2
           AND cadenas.aud_despues=cadenas.aud_antes+2
           AND cadenas.out_despues=cadenas.out_antes+2
         )::text
           FROM cadenas")" == 'true' ]]

    limpiar_pruebas_variantes_o3_04
}
