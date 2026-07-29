#!/usr/bin/env bash
set -Eeuo pipefail

directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
patron_cursor_argumento='--env(=|[[:space:]])VEC_''CURSOR[^[:space:]]*='
estado_cursores_argumentos=0
rg -q -- "$patron_cursor_argumento" "${BASH_SOURCE[0]}" ||
    estado_cursores_argumentos=$?
if (( estado_cursores_argumentos != 1 )); then
    printf 'el guion no protege los cursores de los argumentos de Docker\n' >&2
    exit 1
fi
# Línea base real hasta CT-000042 sobre PostgreSQL 18.4 sin red. CT43 y CT43A
# se instalan después sin ejecutar sus baterías completas, ya verdes aparte:
# así la identidad sintética nace inmediatamente antes de las pruebas CT44B.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_canones_resultado_recibo_rrhh_pg18_4.sh"
: "${contenedor:?el ejecutor CT42 debe exponer PostgreSQL}"
paso() {
    printf '[O4-05:CT-000044B:PG18.4] %s\n' "$1"
}
preparar_vector() {
    local caso=$1
    local perfil=$2
    psql_admin --command \
        "SELECT public.preparar_vector_cierre_ct43('$caso','$perfil')" \
        >/dev/null
}
invocar_cuadro() {
    local caso=$1
    local cursor=${2:-}
    local fallo=${3:-}
    local estado=${4:-}
    VEC_CURSOR="$cursor" docker exec --interactive \
        --env VEC_CASO="$caso" \
        --env VEC_CURSOR \
        --env VEC_FALLO="$fallo" \
        --env VEC_ESTADO="$estado" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv caso VEC_CASO
\getenv cursor VEC_CURSOR
\getenv fallo VEC_FALLO
\getenv estado VEC_ESTADO
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SELECT pg_catalog.set_config(
    'vec.prueba_ct44b_fallo', $1, true
)
\parse configurar_fallo
\bind_named configurar_fallo :fallo
\g /dev/null
SELECT pg_catalog.set_config(
    'vec.prueba_ct43_sqlstate', $1, true
)
\parse configurar_estado
\bind_named configurar_estado :estado
\g /dev/null
SELECT vec_contratacion_temporal
       .prueba_invocar_motor_cuadro_ct44b($1, $2)
\parse invocar_motor
\bind_named invocar_motor :caso :cursor
\g
COMMIT;
SQL
}
ajustar_cursor() {
    local caso=$1
    local cursor=$2
    VEC_CURSOR="$cursor" docker exec --interactive \
        --env VEC_CASO="$caso" \
        --env VEC_CURSOR \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username postgres --dbname postgres <<'SQL'
\getenv caso VEC_CASO
\getenv cursor VEC_CURSOR
SELECT public.ajustar_cursor_vector_ct44b($1, $2)
\parse ajustar_cursor
\bind_named ajustar_cursor :caso :cursor
\g /dev/null
SQL
}
preparar_detalle() {
    local caso=$1
    local version=$2
    preparar_vector "$caso" detalle
    psql_admin --command "
        SELECT public.ajustar_version_observada_ct43a(
            '$caso', $version, true
        )" >/dev/null
}
invocar_cuadro_controlado() {
    local caso=$1
    local cursor=$2
    local organizacion=$3
    local devolver_cursor=${4:-false}
    VEC_CURSOR="$cursor" docker exec --interactive \
        --env VEC_CASO="$caso" \
        --env VEC_CURSOR \
        --env VEC_ORGANIZACION="$organizacion" \
        --env VEC_DEVOLVER_CURSOR="$devolver_cursor" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv caso VEC_CASO
\getenv cursor VEC_CURSOR
\getenv organizacion VEC_ORGANIZACION
\getenv devolver_cursor VEC_DEVOLVER_CURSOR
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SELECT vec_contratacion_temporal
       .prueba_invocar_motor_cuadro_controlado_ct44b(
           $1, $2, $3, $4::boolean
       )
\parse invocar_controlado
\bind_named invocar_controlado :caso :cursor :organizacion :devolver_cursor
\g
COMMIT;
SQL
}
invocar_forma_terminal() {
    local variante=$1
    local cursor=${2:-}
    VEC_CURSOR="$cursor" docker exec --interactive \
        --env VEC_VARIANTE="$variante" \
        --env VEC_CURSOR \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv variante VEC_VARIANTE
\getenv cursor VEC_CURSOR
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SELECT vec_contratacion_temporal
       .prueba_forma_terminal_efectos_ct44b($1, $2)
\parse probar_terminal
\bind_named probar_terminal :variante :cursor
\g
COMMIT;
SQL
}
contar_consumos_cursor() {
    local cursor=$1
    VEC_CURSOR="$cursor" docker exec --interactive \
        --env VEC_CURSOR \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 \
        --username postgres --dbname postgres <<'SQL'
\getenv cursor VEC_CURSOR
SELECT pg_catalog.count(*)
  FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
 WHERE token_huella_sha256 = pg_catalog.encode(pg_catalog.sha256(
       pg_catalog.convert_to($1, 'UTF8')
   ), 'hex')
\parse contar_consumos
\bind_named contar_consumos :cursor
\g
SQL
}

contar_apariciones_tokens() {
    local cursor_a=$1
    local cursor_b=$2
    VEC_CURSOR_A="$cursor_a" VEC_CURSOR_B="$cursor_b" \
        docker exec --interactive \
        --env VEC_CURSOR_A --env VEC_CURSOR_B \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 \
        --username postgres --dbname postgres <<'SQL'
\getenv cursor_a VEC_CURSOR_A
\getenv cursor_b VEC_CURSOR_B
SELECT pg_catalog.count(*)
  FROM (
      SELECT pg_catalog.to_jsonb(familia)::text AS dato
        FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
      UNION ALL
      SELECT pg_catalog.to_jsonb(cursor)::text
        FROM vec_contratacion_temporal.cursor_cuadro_rrhh cursor
      UNION ALL
      SELECT pg_catalog.to_jsonb(consumo)::text
        FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh consumo
      UNION ALL
      SELECT pg_catalog.to_jsonb(prueba)::text
        FROM vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 prueba
  ) material
 WHERE pg_catalog.strpos(material.dato, $1) > 0
    OR pg_catalog.strpos(material.dato, $2) > 0
\parse buscar_tokens
\bind_named buscar_tokens :cursor_a :cursor_b
\g
SQL
}

estado_cursores() {
    valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.familia_cursor_cuadro_rrhh),
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.control_causal_familia_cursor_rrhh),
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.cursor_cuadro_rrhh),
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.consumo_cursor_cuadro_rrhh)
    )"
}

invocar_detalle() {
    local caso=$1
    local version=$2
    docker exec --interactive \
        --env VEC_CASO="$caso" \
        --env VEC_VERSION="$version" \
        "$contenedor" psql -XqAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv caso VEC_CASO
\getenv version VEC_VERSION
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL TimeZone='UTC';
SELECT vec_contratacion_temporal
       .prueba_invocar_motor_detalle_ct44b($1, $2::numeric)
\parse invocar_detalle
\bind_named invocar_detalle :caso :version
\g
COMMIT;
SQL
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

paso 'instalación directa CT43, CT43A e identidad/VEC sintéticas frescas'
for ruta in \
    autorizacion_atestada_v3/migraciones/000003_consumidor_consulta_cuadro_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000004_consumidor_consulta_detalle_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000005_revalidacion_final_consultas_rrhh_v3.up.sql \
    autorizacion_atestada_v3/migraciones/000006_prueba_consumo_consultas_rrhh_v3.up.sql
do
    archivo "$ruta"
done
(
    cd /tmp
    psql_admin --file \
        /repo/contratacion_temporal/migraciones/000043_prueba_resultado_recibo_rrhh.up.sql \
        >/dev/null
)
archivo \
    contratacion_temporal/migraciones/000043a_detalle_version_actual.up.sql
psql_admin --command \
    'GRANT EXECUTE ON FUNCTION public.gen_random_bytes(integer) TO vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo autorizacion_atestada_v3/pruebas_sql/preparar_entorno_o2_05.sql
archivo autorizacion_atestada_v3/pruebas_sql/consultas_rrhh_v3.sql
psql_admin --command \
    'REVOKE EXECUTE ON FUNCTION public.gen_random_bytes(integer) FROM vec_autorizacion_atestada_v3_propietario' \
    >/dev/null
archivo \
    autorizacion_atestada_v3/pruebas_sql/revalidacion_final_consultas_rrhh_v3.sql
archivo \
    autorizacion_atestada_v3/pruebas_sql/prueba_consumo_consultas_rrhh_v3.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_prueba_resultado_recibo_rrhh_datos_sinteticos.sql
archivo \
    contratacion_temporal/pruebas_sql/o405_corrector_detalle_version_actual.sql
psql_admin <<'SQL' >/dev/null
ALTER DATABASE postgres SET log_statement = 'none';
ALTER DATABASE postgres SET log_parameter_max_length = 0;
ALTER DATABASE postgres SET log_parameter_max_length_on_error = 0;
SQL
paso 'instalación privada de componentes CT44 y corpus comprometido'
psql_admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
\ir /repo/contratacion_temporal/migraciones/000044_componentes/010_tipos_resultado.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/020_guardas_y_contexto.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/030_materializacion_detalle.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/040_materializacion_cuadro.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/050_control_causal_y_cursores.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/055_efectos_cursor.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/060_motor_atomico_y_efectos.sql
COMMIT;
\ir /repo/contratacion_temporal/pruebas_sql/o405_motor_atomico_consultas_ct44b_datos_sinteticos.sql
\ir /repo/contratacion_temporal/pruebas_sql/o405_motor_atomico_consultas_ct44b_adversarial.sql
SQL
paso 'catálogo privado, límites y revalidación nominal encapsulada'
psql_admin <<'SQL' >/dev/null
DO $prueba$
DECLARE
    v_firma regprocedure;
    v_funcion pg_catalog.pg_proc%ROWTYPE;
    v_consumidor pg_catalog.pg_proc%ROWTYPE;
    v_interno pg_catalog.pg_proc%ROWTYPE;
    v_identidad pg_catalog.pg_proc%ROWTYPE;
BEGIN
    FOREACH v_firma IN ARRAY ARRAY[
        'vec_contratacion_temporal.motor_consultar_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'::regprocedure,
        'vec_contratacion_temporal.motor_consultar_detalle_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'::regprocedure,
        'vec_contratacion_temporal.aplicar_efectos_cursor_cuadro_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,vec_contratacion_temporal.estado_cursor_entrada_cuadro_rrhh_v1,vec_contratacion_temporal.salida_cursor_cuadro_rrhh_v1,vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3,bytea,vec_contratacion_temporal.resultado_cierre_prueba_rrhh_v2)'::regprocedure
    ] LOOP
        SELECT * INTO STRICT v_funcion FROM pg_catalog.pg_proc
         WHERE oid = v_firma;
        IF v_funcion.proowner <>
               'vec_contratacion_temporal_propietario'::regrole
           OR NOT v_funcion.prosecdef
           OR v_funcion.provolatile <> 'v'
           OR v_funcion.proparallel <> 'u'
           OR v_funcion.proconfig <> ARRAY[
               'search_path=pg_catalog', 'row_security=on',
               'TimeZone=UTC', 'lock_timeout=1s',
               'statement_timeout=4s',
               'idle_in_transaction_session_timeout=6s'
           ]::text[]
           OR pg_catalog.has_function_privilege(
               'vec_c2d2_registro_runtime', v_firma, 'EXECUTE'
           )
           OR (
               SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(
                   acl.grantor, acl.grantee, acl.privilege_type,
                   acl.is_grantable
               ))
                 FROM pg_catalog.aclexplode(v_funcion.proacl) acl
           ) IS DISTINCT FROM pg_catalog.jsonb_build_array(
               pg_catalog.jsonb_build_array(
                   v_funcion.proowner, v_funcion.proowner,
                   'EXECUTE', false
               )
           ) THEN
            RAISE EXCEPTION 'catálogo CT44B inseguro';
        END IF;
        IF v_funcion.proname = 'motor_consultar_cuadro_rrhh_v1'
           AND (
               pg_catalog.strpos(
                   v_funcion.prosrc,
                   'consumir_autorizacion_motor_consultas_rrhh_v1'
               ) >= pg_catalog.strpos(
                   v_funcion.prosrc, 'v_contexto_huella :='
               )
               OR pg_catalog.strpos(
                   v_funcion.prosrc, 'v_contexto_huella :='
               ) >= pg_catalog.strpos(
                   v_funcion.prosrc,
                   'resolver_estado_cursor_cuadro_rrhh_v1'
               )
               OR pg_catalog.strpos(v_funcion.prosrc,
                   'resolver_estado_cursor_cuadro_rrhh_v1') >=
                  pg_catalog.strpos(v_funcion.prosrc,
                   'cerrar_prueba_resultado_recibo_rrhh_v2')
               OR pg_catalog.strpos(v_funcion.prosrc,
                   'cerrar_prueba_resultado_recibo_rrhh_v2') >=
                  pg_catalog.strpos(v_funcion.prosrc,
                   'aplicar_efectos_cursor_cuadro_rrhh_v1')
           ) THEN
            RAISE EXCEPTION 'orden prelectura de cuadro CT44B alterado';
        END IF;
        IF v_funcion.proname = 'motor_consultar_detalle_rrhh_v1'
           AND (
               pg_catalog.strpos(
                   v_funcion.prosrc,
                   'consumir_autorizacion_motor_consultas_rrhh_v1'
               ) >= pg_catalog.strpos(
                   v_funcion.prosrc, 'v_contexto_huella :='
               )
               OR pg_catalog.strpos(
                   v_funcion.prosrc, 'v_contexto_huella :='
               ) >= pg_catalog.strpos(
                   v_funcion.prosrc,
                   'SELECT control.ultimo_corte'
               )
        ) THEN
            RAISE EXCEPTION 'orden prelectura de detalle CT44B alterado';
        END IF;
        IF v_funcion.proname =
               'aplicar_efectos_cursor_cuadro_rrhh_v1'
           AND (
               pg_catalog.strpos(
                   v_funcion.prosrc,
                   'p_estado.es_continuacion IS DISTINCT FROM'
               ) = 0
               OR pg_catalog.strpos(
                   v_funcion.prosrc, '(p_consulta.cursor <> '''')'
               ) = 0
               OR pg_catalog.strpos(
                   pg_catalog.regexp_replace(
                       v_funcion.prosrc, '[[:space:]]+', ' ', 'g'
                   ),
                   'p_estado.token_presentado_huella_sha256 '
                   || 'IS DISTINCT FROM pg_catalog.encode( '
                   || 'pg_catalog.sha256(pg_catalog.convert_to( '
                   || 'p_consulta.cursor, ''UTF8'''
               ) = 0
           ) THEN
            RAISE EXCEPTION 'ligadura consulta/estado CT44B alterada';
        END IF;
    END LOOP;
    SELECT * INTO STRICT v_consumidor
      FROM pg_catalog.pg_proc
     WHERE oid =
       'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    SELECT * INTO STRICT v_interno
      FROM pg_catalog.pg_proc
     WHERE oid =
       'vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    SELECT * INTO STRICT v_identidad
      FROM pg_catalog.pg_proc
     WHERE oid =
       'vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1(text,text)'::regprocedure;
    IF pg_catalog.strpos(
           v_consumidor.prosrc, 'consumir_consulta_rrhh_v3_interna'
       ) = 0
       OR pg_catalog.strpos(
           v_interno.prosrc,
           'revalidar_decision_contexto_actor_v3_viva'
       ) = 0
       OR pg_catalog.strpos(
           v_identidad.prosrc,
           'WHEN ''interna_corporativa'' THEN interval ''15 minutes'''
       ) = 0 THEN
        RAISE EXCEPTION 'consumidor VEC sin revalidación viva';
    END IF;
    IF pg_catalog.current_setting(
           'log_parameter_max_length_on_error'
       ) <> '0'
       OR pg_catalog.current_setting(
           'log_parameter_max_length'
       ) <> '0'
       OR pg_catalog.current_setting('log_statement') <> 'none' THEN
        RAISE EXCEPTION 'parámetros de error podrían exponer cursores';
    END IF;
END
$prueba$;
SQL
paso 'regresión de materializadores y control causal previo'
archivo contratacion_temporal/pruebas_sql/o405_motor_detalle_rrhh.sql
archivo contratacion_temporal/pruebas_sql/o405_motor_cuadro_rrhh.sql
paso 'primera página terminal sin familia ni cursor'
preparar_vector ct44b_inicial_final cuadro
salida="$(invocar_cuadro ct44b_inicial_final)"
[[ -z $salida ]]
[[ "$(efectos_decision ct44b_inicial_final)" == '1|1|1|1' ]]
[[ "$(valor "SELECT pg_catalog.count(*)
      FROM vec_contratacion_temporal.alcance_acceso_rrhh alcance
      JOIN vec_contratacion_temporal.registro_acceso_rrhh acceso
        USING (acceso_ref)
     WHERE acceso.decision_ref=
           'decision:consulta-rrhh:ct44b_inicial_final'
       AND alcance.familia_ref IS NOT NULL")" == 0 ]]
paso 'cruces de consulta y alcance se rechazan después del consumo y antes de leer'
cursores_antes="$(estado_cursores)"
preparar_vector ct44b_cruce_consulta cuadro
cursor_ajeno="$(printf 'A%.0s' {1..43})"
esperar_fallo 'consulta cruzada prelectura CT44B' 42501 \
    'consulta de cuadro RRHH rechazada' \
    invocar_cuadro ct44b_cruce_consulta "$cursor_ajeno"
[[ "$(efectos_decision ct44b_cruce_consulta)" == '0|0|0|0' ]]
preparar_vector ct44b_cruce_alcance cuadro
esperar_fallo 'alcance cruzado prelectura CT44B' 42501 \
    'consulta de cuadro RRHH rechazada' \
    invocar_cuadro_controlado ct44b_cruce_alcance '' \
        organizacion:ajena false
[[ "$(efectos_decision ct44b_cruce_alcance)" == '0|0|0|0' ]]
[[ "$(estado_cursores)" == "$cursores_antes" ]]
paso 'formas terminales hostiles no aceptan nulo, digest ni cierre paginado'
for variante in \
    cursor_nulo cursor_siguiente_no_vacio cursor_no_vacio \
    cierre_terminal_intercambiable
do
    esperar_fallo "forma terminal $variante CT44B" 42501 \
        'efectos de cursor RRHH rechazados' \
        invocar_forma_terminal "$variante"
done
[[ "$(estado_cursores)" == '0|0|0|0' ]]
paso 'primera con página dos, continuación con hijo y final'
psql_admin --command 'SELECT public.ampliar_corpus_cuadro_ct44b()' \
    >/dev/null
tokens_claros=("$cursor_ajeno")
preparar_vector ct44b_pagina_1 cuadro
token_1="$(invocar_cuadro ct44b_pagina_1)"
[[ $token_1 =~ ^[A-Za-z0-9_-]{43}$ ]]
tokens_claros+=("$token_1")
# Matriz directa 055 y cruce de detalle, separada para mantener DEC-051.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_motor_atomico_consultas_ct44b_hostil_pg18_4.sh"
if ! declare -F contiene_secreto >/dev/null; then
    printf 'la búsqueda segura de secretos CT44B no está disponible\n' >&2
    exit 1
fi
probar_matriz_hostil_ct44b "$token_1"
preparar_vector ct44b_pagina_2 cuadro
ajustar_cursor ct44b_pagina_2 "$token_1"
token_2="$(invocar_cuadro ct44b_pagina_2 "$token_1")"
[[ $token_2 =~ ^[A-Za-z0-9_-]{43}$ ]]
tokens_claros+=("$token_2")
preparar_vector ct44b_pagina_3 cuadro
ajustar_cursor ct44b_pagina_3 "$token_2"
salida="$(invocar_cuadro ct44b_pagina_3 "$token_2")"
[[ -z $salida ]]
[[ "$(valor "SELECT pg_catalog.concat_ws('|',
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.familia_cursor_cuadro_rrhh),
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.control_causal_familia_cursor_rrhh),
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.cursor_cuadro_rrhh),
        (SELECT pg_catalog.count(*) FROM
          vec_contratacion_temporal.consumo_cursor_cuadro_rrhh)
	    )")" == '1|1|2|2' ]]
paso 'un cursor consumido rechaza una capacidad VEC nueva sin duplicar efectos'
preparar_vector ct44b_replay_cursor cuadro
ajustar_cursor ct44b_replay_cursor "$token_2"
esperar_fallo 'replay de cursor con VEC fresca CT44B' 42501 \
    'consulta de cuadro RRHH rechazada' \
    invocar_cuadro ct44b_replay_cursor "$token_2"
[[ "$(efectos_decision ct44b_replay_cursor)" == '0|0|0|0' ]]
[[ "$(contar_consumos_cursor "$token_2")" == 1 ]]
paso 'detalle de versión actual y versión exacta sin cursores'
psql_admin <<'SQL' >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $precomprobacion$
DECLARE
    v_material
        vec_contratacion_temporal.materializacion_detalle_rrhh_v1;
    v_corte numeric(20, 0);
BEGIN
    SELECT ultimo_corte INTO STRICT v_corte
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    v_material :=
        vec_contratacion_temporal.materializar_detalle_rrhh_v1(
            ROW(
                'organizacion:diputacion-granada', 'organizacion',
                'organizacion:diputacion-granada'
            ),
            ROW('expediente:rrhh:minimizado', 0::numeric),
            v_corte
        );
    IF ((v_material.detalle).resumen).version <> 1 THEN
        RAISE EXCEPTION 'corpus de detalle CT44B incoherente';
    END IF;
END
$precomprobacion$;
ROLLBACK;
SQL
preparar_detalle ct44b_detalle_actual 0
acceso_actual="$(invocar_detalle ct44b_detalle_actual 0)"
[[ $acceso_actual =~ ^acceso:rrhh:[0-9a-f]{32}$ ]]
preparar_detalle ct44b_detalle_exacta 1
acceso_exacto="$(invocar_detalle ct44b_detalle_exacta 1)"
[[ $acceso_exacto =~ ^acceso:rrhh:[0-9a-f]{32}$ ]]
[[ "$(valor "SELECT pg_catalog.count(*)
      FROM vec_contratacion_temporal.alcance_acceso_rrhh alcance
     WHERE alcance.acceso_ref IN (
         '$acceso_actual', '$acceso_exacto'
     )")" == 0 ]]
paso 'repetición de capacidad falla cerrada sin duplicar efectos'
esperar_fallo 'replay VEC CT44B' 42501 \
    'consulta de cuadro RRHH rechazada' \
    invocar_cuadro ct44b_inicial_final
[[ "$(efectos_decision ct44b_inicial_final)" == '1|1|1|1' ]]

paso 'fallos posteriores a CT43A y entre consumo padre e hijo revierten todo'
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION public.forzar_fallo_efectos_ct44b()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    v_fallo text := pg_catalog.current_setting(
        'vec.prueba_ct44b_fallo', true
    );
    v_codigo text;
BEGIN
    IF v_fallo = 'despues_ct43a'
       AND TG_TABLE_NAME = 'familia_cursor_cuadro_rrhh' THEN
        RAISE EXCEPTION USING ERRCODE='23514',
            MESSAGE='fallo sintético posterior a CT43A';
    END IF;
    IF v_fallo = 'entre_consumo_e_hijo'
       AND TG_TABLE_NAME = 'cursor_cuadro_rrhh' THEN
        RAISE EXCEPTION USING ERRCODE='23514',
            MESSAGE='fallo sintético entre consumo e hijo';
    END IF;
    IF v_fallo LIKE 'estado_familia:%'
       AND TG_TABLE_NAME = 'familia_cursor_cuadro_rrhh' THEN
        v_codigo := pg_catalog.split_part(v_fallo, ':', 2);
    ELSIF v_fallo LIKE 'estado_consumo:%'
       AND TG_TABLE_NAME = 'consumo_cursor_cuadro_rrhh' THEN
        v_codigo := pg_catalog.split_part(v_fallo, ':', 2);
    ELSIF v_fallo LIKE 'estado_cursor:%'
       AND TG_TABLE_NAME = 'cursor_cuadro_rrhh' THEN
        v_codigo := pg_catalog.split_part(v_fallo, ':', 2);
    END IF;
    IF v_codigo IS NOT NULL THEN
        IF v_codigo NOT IN ('40001', '40P01', '55P03', '57014') THEN
            RAISE EXCEPTION 'SQLSTATE sintético CT44B inválido';
        END IF;
        RAISE EXCEPTION USING ERRCODE=v_codigo,
            MESSAGE='estado transitorio sintético CT44B';
    END IF;
    RETURN NEW;
END
$funcion$;
REVOKE ALL ON FUNCTION public.forzar_fallo_efectos_ct44b()
FROM PUBLIC;
CREATE TRIGGER forzar_fallo_familia_ct44b
BEFORE INSERT ON vec_contratacion_temporal.familia_cursor_cuadro_rrhh
FOR EACH ROW EXECUTE FUNCTION public.forzar_fallo_efectos_ct44b();
CREATE TRIGGER forzar_fallo_hijo_ct44b
BEFORE INSERT ON vec_contratacion_temporal.cursor_cuadro_rrhh
FOR EACH ROW EXECUTE FUNCTION public.forzar_fallo_efectos_ct44b();
CREATE TRIGGER forzar_fallo_consumo_ct44b
BEFORE INSERT ON vec_contratacion_temporal.consumo_cursor_cuadro_rrhh
FOR EACH ROW EXECUTE FUNCTION public.forzar_fallo_efectos_ct44b();
SQL
preparar_vector ct44b_fallo_cierre cuadro
esperar_fallo 'rollback posterior a CT43A' 42501 \
    'consulta de cuadro RRHH rechazada' \
    invocar_cuadro ct44b_fallo_cierre '' despues_ct43a
[[ "$(efectos_decision ct44b_fallo_cierre)" == '0|0|0|0' ]]

paso 'familia de control para el rollback entre consumo e hijo'
preparar_vector ct44b_padre_fallo cuadro
token_fallo="$(invocar_cuadro ct44b_padre_fallo)"
tokens_claros+=("$token_fallo")
paso 'inyección entre el consumo del padre y el alta del hijo'
preparar_vector ct44b_hijo_fallo cuadro
ajustar_cursor ct44b_hijo_fallo "$token_fallo"
esperar_fallo 'rollback entre consumo padre e hijo' 42501 \
    'consulta de cuadro RRHH rechazada' \
    invocar_cuadro ct44b_hijo_fallo "$token_fallo" \
        entre_consumo_e_hijo
[[ "$(efectos_decision ct44b_hijo_fallo)" == '0|0|0|0' ]]
[[ "$(contar_consumos_cursor "$token_fallo")" == 0 ]]

paso 'cuatro SQLSTATE transitorios exactos y rollback total'
psql_admin --command \
    'CREATE TRIGGER forzar_sqlstate_ct44b BEFORE INSERT ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2 FOR EACH ROW EXECUTE FUNCTION public.forzar_sqlstate_prueba_ct43()' \
    >/dev/null
for estado in 40001 40P01 55P03 57014; do
    caso="ct44b_estado_${estado,,}"
    preparar_vector "$caso" cuadro
    esperar_fallo "SQLSTATE $estado CT44B" "$estado" \
        'estado transitorio sintético CT43' \
        invocar_cuadro "$caso" '' '' "$estado"
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
done
psql_admin --command \
    'DROP TRIGGER forzar_sqlstate_ct44b ON vec_contratacion_temporal.prueba_resultado_recibo_rrhh_v2' \
    >/dev/null

paso 'los cuatro SQLSTATE nacen también dentro de los efectos 055'
for estado in 40001 40P01; do
    caso="ct44b_055_familia_${estado,,}"
    preparar_vector "$caso" cuadro
    cursores_antes="$(estado_cursores)"
    esperar_fallo "055 familia $estado CT44B" "$estado" \
        'estado transitorio sintético CT44B' \
        invocar_cuadro "$caso" '' "estado_familia:$estado"
    [[ "$(efectos_decision "$caso")" == '0|0|0|0' ]]
    [[ "$(estado_cursores)" == "$cursores_antes" ]]
done

preparar_vector ct44b_055_padre_consumo cuadro
token_estado_consumo="$(invocar_cuadro ct44b_055_padre_consumo)"
tokens_claros+=("$token_estado_consumo")
preparar_vector ct44b_055_consumo_55p03 cuadro
ajustar_cursor ct44b_055_consumo_55p03 "$token_estado_consumo"
cursores_antes="$(estado_cursores)"
esperar_fallo '055 consumo 55P03 CT44B' 55P03 \
    'estado transitorio sintético CT44B' \
    invocar_cuadro ct44b_055_consumo_55p03 \
        "$token_estado_consumo" 'estado_consumo:55P03'
[[ "$(efectos_decision ct44b_055_consumo_55p03)" == '0|0|0|0' ]]
[[ "$(estado_cursores)" == "$cursores_antes" ]]
[[ "$(contar_consumos_cursor "$token_estado_consumo")" == 0 ]]

preparar_vector ct44b_055_padre_cursor cuadro
token_estado_cursor="$(invocar_cuadro ct44b_055_padre_cursor)"
tokens_claros+=("$token_estado_cursor")
preparar_vector ct44b_055_cursor_57014 cuadro
ajustar_cursor ct44b_055_cursor_57014 "$token_estado_cursor"
cursores_antes="$(estado_cursores)"
esperar_fallo '055 cursor 57014 CT44B' 57014 \
    'estado transitorio sintético CT44B' \
    invocar_cuadro ct44b_055_cursor_57014 \
        "$token_estado_cursor" 'estado_cursor:57014'
[[ "$(efectos_decision ct44b_055_cursor_57014)" == '0|0|0|0' ]]
[[ "$(estado_cursores)" == "$cursores_antes" ]]
[[ "$(contar_consumos_cursor "$token_estado_cursor")" == 0 ]]

paso 'dos continuaciones concurrentes del mismo cursor dejan un ganador'
preparar_vector ct44b_concurrencia_padre cuadro
token_concurrente="$(invocar_cuadro ct44b_concurrencia_padre)"
tokens_claros+=("$token_concurrente")
for caso in ct44b_concurrencia_a ct44b_concurrencia_b; do
    preparar_vector "$caso" cuadro
    ajustar_cursor "$caso" "$token_concurrente"
done
familias_antes="$(valor \
    'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh')"
cursores_antes="$(valor \
    'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.cursor_cuadro_rrhh')"
consumos_antes="$(valor \
    'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh')"
salida_concurrente_a="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44b-concurrente-a.XXXXXX"
)"
salida_concurrente_b="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44b-concurrente-b.XXXXXX"
)"
temporales+=("$salida_concurrente_a" "$salida_concurrente_b")
invocar_cuadro_controlado ct44b_concurrencia_a "$token_concurrente" \
    organizacion:diputacion-granada false \
    >"$salida_concurrente_a" 2>&1 &
pid_concurrente_a=$!
invocar_cuadro_controlado ct44b_concurrencia_b "$token_concurrente" \
    organizacion:diputacion-granada false \
    >"$salida_concurrente_b" 2>&1 &
pid_concurrente_b=$!
estado_concurrente_a=0
estado_concurrente_b=0
wait "$pid_concurrente_a" || estado_concurrente_a=$?
wait "$pid_concurrente_b" || estado_concurrente_b=$?
if (( (estado_concurrente_a == 0) + (estado_concurrente_b == 0) != 1 )); then
    sed -n '1,20p' "$salida_concurrente_a" >&2
    sed -n '1,20p' "$salida_concurrente_b" >&2
    exit 1
fi
[[ "$(contar_consumos_cursor "$token_concurrente")" == 1 ]]
[[ "$(valor \
    'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh')" \
    == "$familias_antes" ]]
[[ "$(valor \
    'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.cursor_cuadro_rrhh')" \
    == "$((cursores_antes + 1))" ]]
[[ "$(valor \
    'SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.consumo_cursor_cuadro_rrhh')" \
    == "$((consumos_antes + 1))" ]]
if (( estado_concurrente_a == 0 )); then
    [[ "$(efectos_decision ct44b_concurrencia_a)" == '1|1|1|1' ]]
    [[ "$(efectos_decision ct44b_concurrencia_b)" == '0|0|0|0' ]]
else
    [[ "$(efectos_decision ct44b_concurrencia_a)" == '0|0|0|0' ]]
    [[ "$(efectos_decision ct44b_concurrencia_b)" == '1|1|1|1' ]]
fi
if contiene_secreto "$token_concurrente" \
    "$salida_concurrente_a" "$salida_concurrente_b"; then
    printf 'el cursor concurrente apareció en una salida temporal\n' >&2
    exit 1
fi

paso 'el token claro no aparece en tablas ni registros del servidor'
for token_claro in "${tokens_claros[@]}"; do
    [[ "$(contar_apariciones_tokens "$token_claro" "$token_claro")" == 0 ]]
    if docker logs "$contenedor" 2>&1 |
        contiene_secreto "$token_claro"; then
        printf 'un token claro apareció en el registro del servidor\n' >&2
        exit 1
    fi
done
unset token_claro token_1 token_2 token_fallo token_estado_consumo
unset token_estado_cursor token_concurrente salida
unset tokens_claros

paso 'retirada de disparadores focales y cierre verde'
psql_admin <<'SQL' >/dev/null
DROP TRIGGER forzar_fallo_familia_ct44b
ON vec_contratacion_temporal.familia_cursor_cuadro_rrhh;
DROP TRIGGER forzar_fallo_hijo_ct44b
ON vec_contratacion_temporal.cursor_cuadro_rrhh;
DROP TRIGGER forzar_fallo_consumo_ct44b
ON vec_contratacion_temporal.consumo_cursor_cuadro_rrhh;
DROP FUNCTION public.forzar_fallo_efectos_ct44b();
SQL

paso 'motor atómico de consultas CT-000044B superado'
