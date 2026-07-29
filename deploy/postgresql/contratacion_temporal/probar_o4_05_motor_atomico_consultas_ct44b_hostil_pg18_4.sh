#!/usr/bin/env bash

# Batería focal cargada por el ejecutor CT44B. Se mantiene separada para que
# cada ejecutor respete el límite DEC-051 sin reducir las aserciones hostiles.

contiene_secreto() {
    local secreto=$1
    shift
    [[ -n $secreto ]] || return 2
    VEC_SECRETO_BUSCADO="$secreto" awk '
        index($0, ENVIRON["VEC_SECRETO_BUSCADO"]) { encontrado=1; exit }
        END { exit encontrado ? 0 : 1 }
    ' "$@"
}

invocar_detalle_controlado_ct44b() {
    local caso=$1
    local expediente=$2
    local version=$3
    # `contenedor` lo proporciona el ejecutor principal inmediatamente antes.
    # shellcheck disable=SC2154
    docker exec --interactive \
        --env VEC_CASO="$caso" \
        --env VEC_EXPEDIENTE="$expediente" \
        --env VEC_VERSION="$version" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv caso VEC_CASO
\getenv expediente VEC_EXPEDIENTE
\getenv version VEC_VERSION
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SELECT vec_contratacion_temporal
       .prueba_invocar_motor_detalle_controlado_ct44b($1, $2, $3::numeric)
\parse invocar_detalle_controlado
\bind_named invocar_detalle_controlado :caso :expediente :version
\g
COMMIT;
SQL
}

probar_matriz_hostil_ct44b() {
    local cursor_pagina_1=$1
    local variante caso_base cursores_antes_local efectos_antes_local

    paso 'mutaciones directas de consulta, familia, keyset, corte y estado'
    psql_admin <<'SQL' >/dev/null
CREATE FUNCTION public.bloquear_dml_055_ct44b()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE='57014',
        MESSAGE='el rechazo hostil CT44B alcanzó DML';
END
$funcion$;
REVOKE ALL ON FUNCTION public.bloquear_dml_055_ct44b() FROM PUBLIC;
CREATE TRIGGER bloquear_familia_055_ct44b
BEFORE INSERT ON vec_contratacion_temporal.familia_cursor_cuadro_rrhh
FOR EACH ROW EXECUTE FUNCTION public.bloquear_dml_055_ct44b();
CREATE TRIGGER bloquear_control_055_ct44b
BEFORE INSERT ON
    vec_contratacion_temporal.control_causal_familia_cursor_rrhh
FOR EACH ROW EXECUTE FUNCTION public.bloquear_dml_055_ct44b();
CREATE TRIGGER bloquear_cursor_055_ct44b
BEFORE INSERT ON vec_contratacion_temporal.cursor_cuadro_rrhh
FOR EACH ROW EXECUTE FUNCTION public.bloquear_dml_055_ct44b();
CREATE TRIGGER bloquear_consumo_055_ct44b
BEFORE INSERT ON vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
FOR EACH ROW EXECUTE FUNCTION public.bloquear_dml_055_ct44b();
SQL
    for variante in \
        consulta_ajena limite_ajeno corte_ajeno \
        familia_ajena keyset_ajeno estado_ajeno
    do
        caso_base=ct44b_inicial_final
        if [[ $variante == familia_ajena ||
              $variante == keyset_ajeno ||
              $variante == estado_ajeno ]]
        then
            caso_base=ct44b_pagina_1
        fi
        cursores_antes_local="$(estado_cursores)"
        efectos_antes_local="$(efectos_decision "$caso_base")"
        esperar_fallo "mutación directa $variante CT44B" 42501 \
            'efectos de cursor RRHH rechazados' \
            invocar_forma_terminal "$variante" "$cursor_pagina_1"
        [[ "$(estado_cursores)" == "$cursores_antes_local" ]]
        [[ "$(efectos_decision "$caso_base")" == "$efectos_antes_local" ]]
    done
    psql_admin <<'SQL' >/dev/null
DROP TRIGGER bloquear_familia_055_ct44b
ON vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
DROP TRIGGER bloquear_control_055_ct44b
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh;
DROP TRIGGER bloquear_cursor_055_ct44b
ON vec_contratacion_temporal.cursor_cuadro_rrhh;
DROP TRIGGER bloquear_consumo_055_ct44b
ON vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
DROP FUNCTION public.bloquear_dml_055_ct44b();
SQL

    paso 'un detalle autorizado no permite consultar otro expediente'
    cursores_antes_local="$(estado_cursores)"
    preparar_detalle ct44b_detalle_cruzado 0
    esperar_fallo 'detalle cruzado prelectura CT44B' 42501 \
        'consulta de detalle RRHH rechazada' \
        invocar_detalle_controlado_ct44b ct44b_detalle_cruzado \
            expediente:rrhh:ajeno 0
    [[ "$(efectos_decision ct44b_detalle_cruzado)" == '0|0|0|0' ]]
    [[ "$(estado_cursores)" == "$cursores_antes_local" ]]
}
