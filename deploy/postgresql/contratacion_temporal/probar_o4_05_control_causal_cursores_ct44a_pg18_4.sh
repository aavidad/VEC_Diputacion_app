#!/usr/bin/env bash
set -Eeuo pipefail
directorio="$(
    cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
    pwd -P
)"
# Reutiliza la línea base real y efímera hasta CT-000043.
# shellcheck disable=SC1091
source "$directorio/probar_o4_05_prueba_resultado_recibo_rrhh_pg18_4.sh"
: "${contenedor:?el runner CT43 debe exponer su contenedor PostgreSQL}"
paso() {
    printf '[O4-05:CT-000044A:PG18.4] %s\n' "$1"
}
familia_ref() {
    printf 'familia:cursor:rrhh:%032x' "$1"
}
sql_resolver() {
    local indice=$1
    local familia
    familia="$(familia_ref "$indice")"
    printf '%s' "
        SELECT (
          vec_contratacion_temporal
          .resolver_estado_cursor_cuadro_rrhh_v1(
            ROW(
              familia.organizacion_ref, familia.clase_ambito,
              familia.ambito_ref
            )::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
            ROW('', '', '', 2,
              pg_catalog.rtrim(pg_catalog.translate(
                pg_catalog.encode(pg_catalog.decode(
                  pg_catalog.repeat(
                    pg_catalog.lpad(
                      pg_catalog.to_hex($indice), 2, '0'
                    ), 32
                  ), 'hex'
                ), 'base64'), '+/', '-_'
              ), E'=\\n')
            )::
              vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
            familia.actor_ref, familia.perfil_ref,
            familia.perfil_version, familia.sesion_ref,
            familia.sesion_huella_sha256
          )
        ).familia_ref
          FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh familia
         WHERE familia.familia_ref = '$familia'"
}
resolver_familia() {
    local indice=$1
    psql_admin --command "
        BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
        SET LOCAL ROLE vec_contratacion_temporal_propietario;
        $(sql_resolver "$indice");
        COMMIT"
}
esperar_senal() {
    local indice=$1
    local _
    for _ in {1..250}; do
        if [[ "$(valor "SELECT pg_catalog.count(*)
              FROM pg_catalog.pg_locks
             WHERE locktype='advisory'
               AND granted
               AND classid=0
               AND objid=$((440000 + indice))")" == 1 ]]; then
            return 0
        fi
        sleep 0.02
    done
    printf 'no apareció la señal causal de la familia %s\n' "$indice" >&2
    return 1
}
retener_familia() {
    local indice=$1
    local segundos=$2
    psql_admin --command "
        BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
        SET LOCAL deadlock_timeout='100ms';
        SET LOCAL ROLE vec_contratacion_temporal_propietario;
        $(sql_resolver "$indice");
        SELECT pg_catalog.pg_advisory_lock($((440000 + indice)));
        SELECT pg_catalog.pg_sleep($segundos);
        COMMIT"
}
revocar_familia() {
    local indice=$1
    local sufijo=$2
    local familia
    familia="$(familia_ref "$indice")"
    psql_admin --command "
        BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
        SET LOCAL ROLE vec_contratacion_temporal_propietario;
        SELECT vec_contratacion_temporal.prueba_revocar_familia_ct44a(
            '$familia', '$sufijo'
        );
        COMMIT"
}
estado_familia() {
    local indice=$1
    local familia
    familia="$(familia_ref "$indice")"
    valor "SELECT causal.revision::text || '|' ||
        (revocacion.familia_ref IS NOT NULL)::text
      FROM vec_contratacion_temporal
           .control_causal_familia_cursor_rrhh causal
      LEFT JOIN vec_contratacion_temporal
           .revocacion_familia_cursor_rrhh revocacion
        USING (familia_ref)
     WHERE causal.familia_ref='$familia'"
}
crear_familia_parametrizada() {
    local indice=$1 autenticacion=$2 sesion=$3 control=$4
    local revision=$5 control_huella=$6
    docker exec --interactive \
        --env VEC_INDICE="$indice" \
        --env VEC_AUTENTICACION="$autenticacion" \
        --env VEC_SESION="$sesion" \
        --env VEC_CONTROL="$control" \
        --env VEC_REVISION="$revision" \
        --env VEC_CONTROL_HUELLA="$control_huella" \
        "$contenedor" psql -XAt \
        --set ON_ERROR_STOP=1 --set VERBOSITY=verbose \
        --username vec_c2d2_registro_runtime --dbname postgres <<'SQL'
\getenv indice VEC_INDICE
\getenv autenticacion VEC_AUTENTICACION
\getenv sesion VEC_SESION
\getenv control VEC_CONTROL
\getenv revision VEC_REVISION
\getenv control_huella VEC_CONTROL_HUELLA
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SELECT vec_contratacion_temporal.crear_familia_cursor_ct44a(
    $1::integer,$2::text,$3::text,$4::text,$5::numeric,$6::text
)
\parse crear_familia
\bind_named crear_familia :indice :autenticacion :sesion :control :revision :control_huella
\g
SET CONSTRAINTS ALL IMMEDIATE;
COMMIT;
SQL
}
paso 'instalación privada y datos sintéticos propios'
psql_admin <<'SQL' >/dev/null
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
\ir /repo/contratacion_temporal/migraciones/000044_componentes/010_tipos_resultado.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/020_guardas_y_contexto.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/030_materializacion_detalle.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/040_materializacion_cuadro.sql
\ir /repo/contratacion_temporal/migraciones/000044_componentes/050_control_causal_y_cursores.sql
COMMIT;
SQL
archivo contratacion_temporal/pruebas_sql/o405_motor_detalle_rrhh.sql
archivo contratacion_temporal/pruebas_sql/o405_motor_cuadro_rrhh.sql
paso 'identidad viva y fachada mínima de prueba con registrador v2'
psql_admin <<'SQL' >/dev/null
SELECT *
  FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
      'opr_eeeeeeeeeeeeeeeeeeeeeeee',
      'vec.identidad.hmac-sha256.v1',
      'idh_eeeeeeeeeeeeeeeeeeeeeeee',
      'clave-hsm-prueba', 1,
      pg_catalog.decode(pg_catalog.repeat('91', 32), 'hex'),
      pg_catalog.decode(pg_catalog.repeat('92', 32), 'hex'),
      false, NULL
  );
SQL
identidad_ct44a="$(
    psql_admin --no-align --tuples-only --field-separator='|' <<'SQL'
SELECT autenticacion_ref, sesion_ref, control_sesion_ref,
       control_sesion_revision_texto, control_sesion_huella_sha256
  FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
      'opr_ffffffffffffffffffffffff',
      'vec.identidad.hmac-sha256.v1',
      'idh_eeeeeeeeeeeeeeeeeeeeeeee',
      'clave-hsm-prueba', 1,
      pg_catalog.decode(pg_catalog.repeat('93', 32), 'hex'),
      pg_catalog.decode(pg_catalog.repeat('94', 32), 'hex'),
      pg_catalog.decode(pg_catalog.repeat('92', 32), 'hex'),
      pg_catalog.decode(pg_catalog.repeat('91', 32), 'hex'),
      NULL, false, 'interna_corporativa', 'kerberos_ad', 'alto',
      pg_catalog.repeat('a', 64),
      pg_catalog.date_trunc(
          'microseconds', pg_catalog.clock_timestamp() - interval '2 seconds'
      ),
      pg_catalog.date_trunc(
          'microseconds', pg_catalog.clock_timestamp() - interval '1 second'
      ),
      pg_catalog.date_trunc(
          'microseconds', pg_catalog.clock_timestamp() + interval '4 minutes'
      ),
      'pga_eeeeeeeeeeeeeeeeeeeeeeee', pg_catalog.repeat('e', 64)
  );
SQL
)"
IFS='|' read -r autenticacion_ct44a sesion_ct44a control_ct44a \
    revision_ct44a control_huella_ct44a <<<"$identidad_ct44a"
if [[ -z $autenticacion_ct44a || -z $sesion_ct44a ||
    -z $control_ct44a || -z $revision_ct44a ||
    $control_huella_ct44a == *'|'* || -z $control_huella_ct44a ]]; then
    printf 'la identidad sintética CT44A no quedó disponible\n' >&2
    exit 1
fi
psql_admin <<'SQL' >/dev/null
CREATE FUNCTION vec_contratacion_temporal.crear_familia_cursor_ct44a(
    p_indice integer, p_autenticacion text, p_sesion text,
    p_control text, p_revision numeric, p_control_huella text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_resultado jsonb;
    v_acceso record;
    v_familia text;
    v_token text;
    v_token_huella text;
    v_filtros_huella text;
    v_expediente text;
    v_prueba bytea;
    v_valor text;
    v_corte numeric(20, 0);
BEGIN
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR SESSION_USER <> 'vec_c2d2_registro_runtime'
       OR p_indice NOT BETWEEN 101 AND 105 THEN
        RAISE EXCEPTION USING ERRCODE='42501',
            MESSAGE='datos sintéticos causales CT44A rechazados';
    END IF;
    v_familia := 'familia:cursor:rrhh:'
        || pg_catalog.lpad(pg_catalog.to_hex(p_indice), 32, '0');
    v_token := pg_catalog.rtrim(pg_catalog.translate(
        pg_catalog.encode(pg_catalog.decode(
            pg_catalog.repeat(
                pg_catalog.lpad(pg_catalog.to_hex(p_indice), 2, '0'), 32
            ), 'hex'
        ), 'base64'), '+/', '-_'
    ), E'=\n');
    v_token_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_token, 'UTF8')
    ), 'hex');
    v_filtros_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.canon_familia_cuadro_rrhh_v1(
            ROW('', '', '', 2, v_token)::
                vec_contratacion_temporal.consulta_cuadro_rrhh_v1
        )
    ), 'hex');
    v_expediente := 'expediente:ct44a:' || p_indice::text;
    SELECT pg_catalog.min(corte_global)
      INTO STRICT v_corte
      FROM vec_contratacion_temporal.publicacion_version_rrhh;
    IF v_corte IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='42501',
            MESSAGE='datos sintéticos causales CT44A rechazados';
    END IF;
    v_resultado :=
        vec_contratacion_temporal.registrar_acceso_rrhh_interno_v2(
            pg_catalog.jsonb_build_object(
                'registro', pg_catalog.jsonb_build_object(
                    'accion', 'contratacion_temporal.cuadro.consultar',
                    'actor_ref', 'actor:rrhh:ct44a',
                    'ambito_ref', 'organizacion:rrhh:ct44a',
                    'audiencia',
                    'vec_contratacion_temporal.'
                    || 'consultar_cuadro_rrhh_atestado.v1',
                    'auditoria_vec_huella_sha256',
                    pg_catalog.encode(pg_catalog.sha256(
                        pg_catalog.convert_to(
                            'ct44a:auditoria:' || p_indice::text, 'UTF8'
                        )
                    ), 'hex'),
                    'auditoria_vec_ref',
                    'auditoria:vec:ct44a:' || p_indice::text,
                    'capacidad_huella_sha256',
                    pg_catalog.encode(pg_catalog.sha256(
                        pg_catalog.convert_to(
                            'ct44a:capacidad:' || p_indice::text, 'UTF8'
                        )
                    ), 'hex'),
                    'consulta_huella_sha256',
                    pg_catalog.encode(pg_catalog.sha256(
                        pg_catalog.convert_to(
                            'ct44a:consulta:' || p_indice::text, 'UTF8'
                        )
                    ), 'hex'),
                    'consumo_vec_huella_sha256',
                    pg_catalog.encode(pg_catalog.sha256(
                        pg_catalog.convert_to(
                            'ct44a:consumo:' || p_indice::text, 'UTF8'
                        )
                    ), 'hex'),
                    'correlacion_ref',
                    'correlacion:rrhh:ct44a:' || p_indice::text,
                    'decision_huella_sha256',
                    pg_catalog.encode(pg_catalog.sha256(
                        pg_catalog.convert_to(
                            'ct44a:decision:' || p_indice::text, 'UTF8'
                        )
                    ), 'hex'),
                    'decision_ref',
                    'decision:rrhh:ct44a:' || p_indice::text,
                    'dominio_huella_consulta',
                    'vec.contratacion_temporal.consulta_rrhh.cuadro.v1',
                    'expediente_ref', NULL,
                    'finalidad',
                    'gestion_operativa_contratacion_temporal',
                    'modulo_id', 'contratacion_temporal',
                    'organizacion_ref', 'organizacion:rrhh:ct44a',
                    'perfil_id', 'perfil:rrhh:ct44a',
                    'perfil_version', 1,
                    'recurso_ref', 'organizacion:rrhh:ct44a',
                    'recurso_tipo',
                    'cuadro_rrhh_contratacion_temporal',
                    'resultado_generico', 'entregado',
                    'resultado_huella_sha256',
                    pg_catalog.encode(pg_catalog.sha256(
                        pg_catalog.convert_to(
                            'ct44a:resultado:' || p_indice::text, 'UTF8'
                        )
                    ), 'hex'),
                    'sesion_huella_sha256', p_control_huella,
                    'sesion_id', p_sesion,
                    'tipo_consulta', 'cuadro',
                    'total', 1,
                    'version_expediente', NULL
                ),
                'alcance', pg_catalog.jsonb_build_object(
                    'clase_ambito', 'organizacion',
                    'familia_ref', v_familia
                ),
                'identidad', pg_catalog.jsonb_build_object(
                    'actor_ref', 'actor:rrhh:ct44a',
                    'autenticacion_huella_sha256',
                    pg_catalog.repeat('a', 64),
                    'autenticacion_ref', p_autenticacion,
                    'control_sesion_huella_sha256',
                    p_control_huella,
                    'control_sesion_ref', p_control,
                    'control_sesion_revision', p_revision,
                    'organizacion_ref', 'organizacion:rrhh:ct44a',
                    'perfil_ref', 'perfil:rrhh:ct44a',
                    'perfil_version', 1,
                    'sesion_ref', p_sesion
                )
            )
        );
    SELECT * INTO STRICT v_acceso
      FROM vec_contratacion_temporal.registro_acceso_rrhh
     WHERE acceso_ref = v_resultado ->> 'acceso_ref';
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-FAMILIA-CURSOR-CUADRO-RRHH-V1'
        || pg_catalog.chr(10), 'UTF8'
    );
    FOREACH v_valor IN ARRAY ARRAY[
        v_familia, v_acceso.organizacion_ref, 'organizacion',
        v_acceso.ambito_ref, v_acceso.actor_ref, v_acceso.perfil_id,
        v_acceso.perfil_version::text, v_acceso.sesion_id,
        v_acceso.sesion_huella_sha256,
        'vec.contratacion_temporal.filtros_rrhh.cuadro.v1',
        v_filtros_huella, '2', v_corte::text,
        vec_contratacion_temporal.instante_utc_v1(
            v_acceso.registrada_en
        ),
        vec_contratacion_temporal.instante_utc_v1(
            v_acceso.registrada_en + interval '4 minutes'
        ),
        v_acceso.acceso_ref
    ]::text[] LOOP
        v_prueba := v_prueba
            || vec_contratacion_temporal.encuadrar_texto_v1(v_valor);
    END LOOP;
    INSERT INTO vec_contratacion_temporal.familia_cursor_cuadro_rrhh
    VALUES (
        v_familia, v_acceso.organizacion_ref, 'organizacion',
        v_acceso.ambito_ref, v_acceso.actor_ref, v_acceso.perfil_id,
        v_acceso.perfil_version, v_acceso.sesion_id,
        v_acceso.sesion_huella_sha256,
        'vec.contratacion_temporal.filtros_rrhh.cuadro.v1',
        v_filtros_huella, 2, v_corte, v_acceso.registrada_en,
        v_acceso.registrada_en + interval '4 minutes',
        v_acceso.acceso_ref, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-CURSOR-CUADRO-RRHH-V1' || pg_catalog.chr(10), 'UTF8'
    );
    FOREACH v_valor IN ARRAY ARRAY[
        v_token_huella, v_familia, '', '2', '',
        vec_contratacion_temporal.instante_utc_v1(
            v_acceso.registrada_en
        ),
        v_expediente,
        vec_contratacion_temporal.instante_utc_v1(
            v_acceso.registrada_en
        ),
        vec_contratacion_temporal.instante_utc_v1(
            v_acceso.registrada_en + interval '4 minutes'
        ),
        vec_contratacion_temporal.instante_utc_v1(
            v_acceso.registrada_en
        ),
        v_acceso.acceso_ref
    ]::text[] LOOP
        v_prueba := v_prueba
            || vec_contratacion_temporal.encuadrar_texto_v1(v_valor);
    END LOOP;
    INSERT INTO vec_contratacion_temporal.cursor_cuadro_rrhh (
        token_huella_sha256, familia_ref, padre_token_huella_sha256,
        pagina, padre_emitida_en, ultimo_actualizado_en,
        ultimo_expediente_ref, familia_creada_en,
        familia_valida_hasta, emitida_en, acceso_emision_ref,
        prueba_canonica, prueba_huella_sha256
    ) VALUES (
        v_token_huella, v_familia, NULL, 2, NULL,
        v_acceso.registrada_en, v_expediente, v_acceso.registrada_en,
        v_acceso.registrada_en + interval '4 minutes',
        v_acceso.registrada_en, v_acceso.acceso_ref, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
    INSERT INTO
    vec_contratacion_temporal.control_causal_familia_cursor_rrhh
    VALUES (
        v_familia, v_acceso.registrada_en, 0, v_acceso.registrada_en
    );
END
$funcion$;
ALTER FUNCTION vec_contratacion_temporal.crear_familia_cursor_ct44a(
    integer, text, text, text, numeric, text
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.crear_familia_cursor_ct44a(
    integer, text, text, text, numeric, text
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.crear_familia_cursor_ct44a(
    integer, text, text, text, numeric, text
) TO vec_contratacion_temporal_consultor_rrhh;
CREATE FUNCTION vec_contratacion_temporal.prueba_revocar_familia_ct44a(
    p_familia text, p_sufijo text
)
RETURNS void
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_familia record;
    v_instante timestamptz(6);
    v_prueba bytea;
    v_decision text := 'decision:revocacion:ct44a:' || p_sufijo;
    v_auditoria text := 'auditoria:revocacion:ct44a:' || p_sufijo;
    v_motivo text := 'motivo:revocacion:ct44a:' || p_sufijo;
    v_decision_huella text;
    v_auditoria_huella text;
    v_motivo_huella text;
BEGIN
    SELECT * INTO STRICT v_familia
      FROM vec_contratacion_temporal.familia_cursor_cuadro_rrhh
     WHERE familia_ref = p_familia;
    v_instante := pg_catalog.date_trunc(
        'microseconds', pg_catalog.clock_timestamp()
    );
    v_decision_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_decision, 'UTF8')
    ), 'hex');
    v_auditoria_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_auditoria, 'UTF8')
    ), 'hex');
    v_motivo_huella := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(v_motivo, 'UTF8')
    ), 'hex');
    v_prueba := pg_catalog.convert_to(
        'VEC-CT-REVOCACION-FAMILIA-CURSOR-RRHH-V1'
        || pg_catalog.chr(10), 'UTF8'
    )
    || vec_contratacion_temporal.encuadrar_texto_v1(p_familia)
    || vec_contratacion_temporal.encuadrar_texto_v1(
        vec_contratacion_temporal.instante_utc_v1(v_familia.creada_en)
    )
    || vec_contratacion_temporal.encuadrar_texto_v1(v_decision)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_decision_huella)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_huella)
    || vec_contratacion_temporal.encuadrar_texto_v1(v_motivo)
    || vec_contratacion_temporal.encuadrar_texto_v1('1')
    || vec_contratacion_temporal.encuadrar_texto_v1(v_motivo_huella)
    || vec_contratacion_temporal.encuadrar_texto_v1(
        vec_contratacion_temporal.instante_utc_v1(v_instante)
    );
    INSERT INTO vec_contratacion_temporal.revocacion_familia_cursor_rrhh
    VALUES (
        p_familia, v_familia.creada_en, v_decision,
        v_decision_huella, v_auditoria, v_auditoria_huella,
        v_motivo, 1, v_motivo_huella, v_instante, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex')
    );
END
$funcion$;
ALTER FUNCTION vec_contratacion_temporal.prueba_revocar_familia_ct44a(
    text, text
)
OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.prueba_revocar_familia_ct44a(text, text)
FROM PUBLIC;

CREATE FUNCTION
vec_contratacion_temporal.probar_contexto_invalido_ct44a()
RETURNS text
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_codigo text;
    v_mensaje text;
BEGIN
    PERFORM
        vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
            ROW(
                'organizacion:rrhh:ct44a', 'organizacion',
                'organizacion:rrhh:ct44a'
            )::vec_contratacion_temporal.alcance_consulta_rrhh_v1,
            NULL::vec_contratacion_temporal
                .material_autorizacion_consulta_rrhh_v3
        );
    RAISE EXCEPTION 'contexto inválido CT44A aceptado';
EXCEPTION WHEN OTHERS THEN
    GET STACKED DIAGNOSTICS
        v_codigo = RETURNED_SQLSTATE,
        v_mensaje = MESSAGE_TEXT;
    RETURN v_codigo || '|' || v_mensaje;
END
$funcion$;
ALTER FUNCTION
vec_contratacion_temporal.probar_contexto_invalido_ct44a()
OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.probar_contexto_invalido_ct44a()
FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.probar_contexto_invalido_ct44a()
TO vec_contratacion_temporal_consultor_rrhh;
SQL
if [[ "$(psql_runtime --command \
    'SELECT vec_contratacion_temporal.probar_contexto_invalido_ct44a()')" \
    != '42501|motor de consultas RRHH rechazado' ]]; then
    printf 'el contexto inválido CT44A no produjo 42501 fijo\n' >&2
    exit 1
fi
for indice_ct44a in 101 102 103 104 105; do
    crear_familia_parametrizada \
        "$indice_ct44a" "$autenticacion_ct44a" "$sesion_ct44a" \
        "$control_ct44a" "$revision_ct44a" "$control_huella_ct44a" \
        >/dev/null
done
psql_admin <<'SQL' >/dev/null
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
\ir /repo/contratacion_temporal/pruebas_sql/o405_control_causal_cursores_ct44a.sql
COMMIT;
SQL
paso 'catálogo, ACL, propietario, configuración y dependencia'
archivo contratacion_temporal/pruebas_sql/o405_catalogo_control_causal_ct44a.sql
paso 'revocación confirmada antes deniega sin oráculo'
revocar_familia 101 primero >/dev/null
[[ "$(estado_familia 101)" == '1|true' ]]
esperar_fallo 'revocación primero CT44A' 42501 \
    'resolución de cursor RRHH rechazada' resolver_familia 101
paso 'motor primero ordena la revocación después del COMMIT'
salida_motor="$(mktemp "${TMPDIR:-/tmp}/vec-ct44a-motor.XXXXXX")"
temporales+=("$salida_motor")
retener_familia 102 0.6 >"$salida_motor" 2>&1 &
pid_motor=$!
esperar_senal 102
inicio_ns="$(date +%s%N)"
revocar_familia 102 segundo >/dev/null
fin_ns="$(date +%s%N)"
wait "$pid_motor"
if (( fin_ns - inicio_ns < 250000000 )); then
    printf 'la revocación no esperó al motor de su familia\n' >&2
    exit 1
fi
[[ "$(estado_familia 102)" == '1|true' ]]
paso 'rollback de revocación no deja rechazo falso'
familia_103="$(familia_ref 103)"
psql_admin --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL ROLE vec_contratacion_temporal_propietario;
    SELECT vec_contratacion_temporal.prueba_revocar_familia_ct44a(
        '$familia_103', 'rollback'
    );
    ROLLBACK" >/dev/null
[[ "$(estado_familia 103)" == '0|false' ]]
resolver_familia 103 >/dev/null
paso 'familias distintas no comparten cerrojo causal'
salida_paralela="$(mktemp "${TMPDIR:-/tmp}/vec-ct44a-paralela.XXXXXX")"
temporales+=("$salida_paralela")
retener_familia 104 1.0 >"$salida_paralela" 2>&1 &
pid_paralelo=$!
esperar_senal 104
resolver_familia 105 >/dev/null
if ! kill -0 "$pid_paralelo" 2>/dev/null; then
    printf 'la prueba paralela no conservó el primer cerrojo\n' >&2
    exit 1
fi
wait "$pid_paralelo"
paso '55P03 real y 40001 causal se conservan exactamente'
salida_bloqueo="$(mktemp "${TMPDIR:-/tmp}/vec-ct44a-bloqueo.XXXXXX")"
temporales+=("$salida_bloqueo")
retener_familia 103 1.5 >"$salida_bloqueo" 2>&1 &
pid_bloqueo=$!
esperar_senal 103
esperar_fallo 'lock_timeout causal CT44A' 55P03 \
    'canceling statement due to lock timeout' resolver_familia 103
wait "$pid_bloqueo"
salida_serializable="$(
    mktemp "${TMPDIR:-/tmp}/vec-ct44a-serializable.XXXXXX"
)"
temporales+=("$salida_serializable")
psql_admin --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL ROLE vec_contratacion_temporal_propietario;
    SELECT revision
     FROM vec_contratacion_temporal
           .control_causal_familia_cursor_rrhh
     WHERE familia_ref='$(familia_ref 105)';
    SELECT pg_catalog.pg_advisory_lock(440105);
    SELECT pg_catalog.pg_sleep(0.5);
    $(sql_resolver 105);
    COMMIT" >"$salida_serializable" 2>&1 &
pid_serializable=$!
esperar_senal 105
revocar_familia 105 serializable >/dev/null
estado_serializable=0
wait "$pid_serializable" || estado_serializable=$?
if (( estado_serializable == 0 )) \
   || ! rg -Fq 'ERROR:  40001:' "$salida_serializable"; then
    sed -n '1,40p' "$salida_serializable" >&2
    exit 1
fi
paso '57014 sintético atraviesa el ayudante sin normalización'
psql_admin <<'SQL'
CREATE FUNCTION vec_contratacion_temporal.forzar_cancelacion_ct44a()
RETURNS boolean
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    RAISE EXCEPTION USING ERRCODE='57014',
        MESSAGE='cancelación sintética CT44A';
END
$funcion$;
REVOKE ALL ON FUNCTION
vec_contratacion_temporal.forzar_cancelacion_ct44a() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION
vec_contratacion_temporal.forzar_cancelacion_ct44a()
    TO vec_contratacion_temporal_propietario;
ALTER POLICY propietario_total
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh
USING (vec_contratacion_temporal.forzar_cancelacion_ct44a())
WITH CHECK (vec_contratacion_temporal.forzar_cancelacion_ct44a());
SQL
esperar_fallo 'cancelación causal CT44A' 57014 \
    'cancelación sintética CT44A' resolver_familia 103
psql_admin <<'SQL'
ALTER POLICY propietario_total
ON vec_contratacion_temporal.control_causal_familia_cursor_rrhh
USING (true) WITH CHECK (true);
DROP FUNCTION vec_contratacion_temporal.forzar_cancelacion_ct44a();
SQL
paso '40P01 real entre dos órdenes conserva su estado'
salida_a="$(mktemp "${TMPDIR:-/tmp}/vec-ct44a-dead-a.XXXXXX")"
salida_b="$(mktemp "${TMPDIR:-/tmp}/vec-ct44a-dead-b.XXXXXX")"
temporales+=("$salida_a" "$salida_b")
psql_admin --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL deadlock_timeout='100ms';
    SET LOCAL ROLE vec_contratacion_temporal_propietario;
    $(sql_resolver 103);
    SELECT pg_catalog.pg_advisory_lock(440103);
    SELECT pg_catalog.pg_sleep(0.5);
    $(sql_resolver 104);
    COMMIT" >"$salida_a" 2>&1 &
pid_a=$!
psql_admin --command "
    BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE;
    SET LOCAL deadlock_timeout='100ms';
    SET LOCAL ROLE vec_contratacion_temporal_propietario;
    $(sql_resolver 104);
    SELECT pg_catalog.pg_advisory_lock(440104);
    SELECT pg_catalog.pg_sleep(0.5);
    $(sql_resolver 103);
    COMMIT" >"$salida_b" 2>&1 &
pid_b=$!
estado_a=0
estado_b=0
esperar_senal 103
esperar_senal 104
wait "$pid_a" || estado_a=$?
wait "$pid_b" || estado_b=$?
if (( (estado_a == 0) + (estado_b == 0) != 1 )) \
   || ! { rg -Fq 'ERROR:  40P01:' "$salida_a" \
          || rg -Fq 'ERROR:  40P01:' "$salida_b"; }; then
    sed -n '1,30p' "$salida_a" >&2
    sed -n '1,30p' "$salida_b" >&2
    exit 1
fi
paso 'token claro ausente y ayudantes de prueba retirados'
for indice_ct44a in 101 102 103 104 105; do
    token_ct44a="$(valor "SELECT pg_catalog.rtrim(pg_catalog.translate(
        pg_catalog.encode(pg_catalog.decode(pg_catalog.repeat(
            pg_catalog.lpad(
                pg_catalog.to_hex($indice_ct44a), 2, '0'
            ), 32
        ), 'hex'), 'base64'), '+/', '-_'
    ), E'=\\n')")"
    if docker logs "$contenedor" 2>&1 | rg -Fq -- "$token_ct44a"; then
        printf 'un token sintético apareció en logs PostgreSQL\n' >&2
        exit 1
    fi
done
psql_admin <<'SQL' >/dev/null
DROP FUNCTION vec_contratacion_temporal.prueba_revocar_familia_ct44a(
    text,text
);
DROP FUNCTION vec_contratacion_temporal.crear_familia_cursor_ct44a(
    integer,text,text,text,numeric,text
);
DROP FUNCTION
vec_contratacion_temporal.probar_contexto_invalido_ct44a();
SQL
paso 'control causal y cursores CT-000044A superados'
