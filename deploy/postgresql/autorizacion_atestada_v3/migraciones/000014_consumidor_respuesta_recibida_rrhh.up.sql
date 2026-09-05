\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000014', 0));
-- Declaración de RRHH, no verificación del correo ni aceptación terminal.
-- Extender el único núcleo conserva las ramas nominales de 000011–000013.
DO $ampliar$
DECLARE
    v_def text; v_acl aclitem[];
    v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'respuesta_recibida_rrhh'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.llamamiento.respuesta.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'respuesta_recibida_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$perfil$;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_reanudacion_seleccion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
        RAISE EXCEPTION 'estado incompatible para respuesta recibida' USING ERRCODE = '55000';
    END IF;
    SELECT pg_get_functiondef(p.oid), p.proacl INTO STRICT v_def, v_acl
      FROM pg_proc p
     WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_marca,'')) <> length(v_marca)
       OR strpos(v_def, 'p_perfil_mutacion IS NOT DISTINCT FROM ''respuesta_recibida_rrhh''') <> 0 THEN
        RAISE EXCEPTION 'núcleo incompatible para respuesta recibida' USING ERRCODE = '55000';
    END IF;
    EXECUTE replace(v_def, v_marca, v_extension || v_marca);
    IF (SELECT proacl FROM pg_proc WHERE oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'extensión alteró permisos del núcleo' USING ERRCODE = '55000';
    END IF;
END
$ampliar$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s'
AS $funcion$
DECLARE v_consumo record;
BEGIN
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'respuesta_recibida_rrhh',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    -- Ni un replay del justificante admite una capacidad ya consumida.
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'respuesta recibida requiere consumo nuevo' USING ERRCODE = 'P0563';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC, vec_autorizacion_atestada_v3_consumidor, vec_autorizacion_atestada_v3_emisor,
    vec_contratacion_temporal_ejecutor, vec_contratacion_temporal_migrador,
    vec_bolsa_llamamientos_propietario, vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_respuesta_recibida_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_propietario;

-- Bloque extraíble literalmente para instalaciones que ya tienen 000014.
-- Ejecutarlo en transacción con SET LOCAL ROLE del propietario V3; no bajar
-- ni reinstalar tablas/consumidores. Repetirlo falla por marca incompatible.
DO $fechas$
DECLARE
    v_def text; v_acl aclitem[];
    v_caso record; v_divergente boolean;
    v_fecha_textual text := $textual$d ->> 'valida_hasta' <> c ->> 'decision_valida_hasta'$textual$;
    v_fecha_instante text := $instante$(d ->> 'valida_hasta')::timestamptz <> (c ->> 'decision_valida_hasta')::timestamptz$instante$;
BEGIN
    SELECT pg_get_functiondef(p.oid), p.proacl INTO STRICT v_def, v_acl
      FROM pg_proc p
     WHERE p.oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_fecha_textual,'')) <> length(v_fecha_textual)
       OR strpos(v_def,v_fecha_instante) <> 0 THEN
        RAISE EXCEPTION 'núcleo incompatible para ligadura temporal VEC-AD-3' USING ERRCODE = '55000';
    END IF;
    -- La decisión usa seis decimales; la capacidad usa RFC3339Nano sin
    -- ceros finales. Comparar el instante, sin cambiar bytes firmados, hashes,
    -- MAC, permisos, revalidación ni la exigencia posterior de consumo nuevo.
    -- Regresión focal del fragmento que se instalará; no consulta ni escribe
    -- datos de negocio. Un microsegundo distinto sigue siendo divergencia.
    FOR v_caso IN
        SELECT * FROM (VALUES
            ('2026-09-05T18:00:00.123450Z','2026-09-05T18:00:00.12345Z',false),
            ('2026-09-05T18:00:00.000000Z','2026-09-05T18:00:00Z',false),
            ('2026-09-05T18:00:00.123450Z','2026-09-05T18:00:00.123451Z',true)
        ) AS casos(decision_hasta,capacidad_hasta,divergente)
    LOOP
        EXECUTE 'SELECT ' || v_fecha_instante ||
            ' FROM (SELECT $1::jsonb AS d,$2::jsonb AS c) AS material'
            INTO v_divergente
            USING jsonb_build_object('valida_hasta',v_caso.decision_hasta),
                  jsonb_build_object('decision_valida_hasta',v_caso.capacidad_hasta);
        IF v_divergente IS DISTINCT FROM v_caso.divergente THEN
            RAISE EXCEPTION 'regresión de ligadura temporal VEC-AD-3' USING ERRCODE = '55000';
        END IF;
    END LOOP;
    EXECUTE replace(v_def,v_fecha_textual,v_fecha_instante);
    IF (SELECT proacl FROM pg_proc WHERE oid = 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'ligadura temporal alteró permisos del núcleo' USING ERRCODE = '55000';
    END IF;
END
$fechas$;
COMMIT;
