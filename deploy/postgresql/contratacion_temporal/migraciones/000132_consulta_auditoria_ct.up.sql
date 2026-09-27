\set ON_ERROR_STOP on
-- CT132: proyección mínima de la historia propia de Contratación temporal.
-- AD3-91 debe estar instalado antes. No se consulta ninguna tabla de otro módulo;
-- el consumo nominal V3 se realiza antes de leer la historia y queda unido a
-- esta lectura en la transacción SERIALIZABLE del llamante.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:migracion:000132', 0));

DO $pre$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.actuacion_expediente_integral') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.actuacion_alta') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = 'vec_contratacion_temporal_consultor_rrhh'
                      AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls) THEN
        RAISE EXCEPTION 'CT132: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
    p_fuente text, p_expediente_ref text, p_actor_ref text,
    p_desde timestamptz, p_hasta timestamptz, p_limite integer,
    p_antes_en timestamptz, p_antes_fuente text, p_antes_id text,
    p_finalidad_ref text, p_motivo_ref text,
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
)
RETURNS TABLE (
    id text, fuente text, modulo_id text, accion text, actor_ref text,
    resultado text, expediente_ref text, recibo_ref text,
    antes_sha256 text, despues_sha256 text, motivo text,
    ocurrido_en timestamptz, antes jsonb, despues jsonb,
    datos_disponibles boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $f$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_contexto jsonb;
    v_filtro_sha text;
    v_recurso_sha text;
    v_consumo record;
BEGIN
    SELECT * INTO v_login FROM pg_catalog.pg_roles WHERE rolname = SESSION_USER;
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL OR NOT v_login.rolcanlogin OR NOT v_login.rolinherit
       OR v_login.rolsuper OR v_login.rolcreatedb OR v_login.rolcreaterole
       OR v_login.rolreplication OR v_login.rolbypassrls
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = v_login.oid) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
             JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
            WHERE m.member = v_login.oid
              AND r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid = v_login.oid)
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR p_fuente IS DISTINCT FROM 'ct'
       OR p_expediente_ref IS NULL OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) > 160
       OR p_actor_ref ~ '[\r\n]'
       OR p_desde IS NULL OR p_hasta IS NULL OR p_desde >= p_hasta
       OR p_hasta - p_desde > interval '31 days'
       OR p_desde <> pg_catalog.date_trunc('microseconds', p_desde)
       OR p_hasta <> pg_catalog.date_trunc('microseconds', p_hasta)
       OR p_limite IS NULL OR p_limite NOT BETWEEN 1 AND 100
       OR (p_antes_en IS NULL AND (p_antes_fuente <> '' OR p_antes_id <> ''))
       OR (p_antes_en IS NOT NULL AND (p_antes_fuente = '' OR p_antes_id = ''))
       OR p_antes_fuente IS NULL OR p_antes_id IS NULL
       OR pg_catalog.octet_length(p_antes_fuente) > 64
       OR pg_catalog.octet_length(p_antes_id) > 512
       OR p_antes_fuente NOT IN ('', 'ct', 'bolsa')
       OR p_antes_fuente ~ '[\r\n]' OR p_antes_id ~ '[\r\n]'
       OR (p_antes_en IS NOT NULL AND
           (p_antes_en <> pg_catalog.date_trunc('microseconds', p_antes_en)
            OR p_antes_en < p_desde OR p_antes_en >= p_hasta))
       OR p_finalidad_ref IS NULL OR p_finalidad_ref = ''
       OR pg_catalog.octet_length(p_finalidad_ref) > 160 OR p_finalidad_ref ~ '[\r\n]'
       OR p_motivo_ref IS NULL OR p_motivo_ref = ''
       OR pg_catalog.octet_length(p_motivo_ref) > 160 OR p_motivo_ref ~ '[\r\n]'
       OR p_capacidad IS NULL OR pg_catalog.octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
       OR p_decision IS NULL OR pg_catalog.octet_length(p_decision) NOT BETWEEN 1 AND 524288
       OR p_motivo IS NULL OR pg_catalog.octet_length(p_motivo) NOT BETWEEN 1 AND 65536
       OR p_contexto IS NULL OR pg_catalog.octet_length(p_contexto) NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
       OR p_perfil_version IS NULL OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload IS NULL OR pg_catalog.octet_length(p_payload) NOT BETWEEN 1 AND 1048576
       OR p_sobre IS NULL OR pg_catalog.octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
       OR p_evidencia IS NULL OR pg_catalog.octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
       OR p_raiz IS NULL OR pg_catalog.octet_length(p_raiz) <> 44 THEN
        RAISE EXCEPTION 'CT132: consulta denegada' USING ERRCODE = '42501';
    END IF;

    -- Mismo canon UTF8 que HuellaFiltro del puerto común. Ningún componente
    -- admite salto de línea, por lo que la concatenación es inequívoca.
    v_filtro_sha := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        pg_catalog.array_to_string(ARRAY[
            'vec.auditoria.filtro.v1', p_fuente, p_expediente_ref, p_actor_ref,
            pg_catalog.to_char(p_desde AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            pg_catalog.to_char(p_hasta AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
            p_limite::text,
            CASE WHEN p_antes_en IS NULL THEN '' ELSE
                pg_catalog.to_char(p_antes_en AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END,
            p_antes_fuente, p_antes_id, p_finalidad_ref, p_motivo_ref
        ], E'\n'), 'UTF8')), 'hex');
    -- Go encoding/json no separa propiedades con espacios; el hash debe usar
    -- exactamente los bytes de contextoRecursoAutorizacionCanonico.
    v_recurso_sha := pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
        '{"ambitos":{"expediente_ref":"' || p_expediente_ref
        || '","fuente":"ct"},"atributos":{"filtro_sha256":"'
        || v_filtro_sha || '"}}', 'UTF8')), 'hex');
    v_capacidad := pg_catalog.convert_from(p_capacidad, 'UTF8')::jsonb;
    v_decision := pg_catalog.convert_from(p_decision, 'UTF8')::jsonb;
    v_contexto := pg_catalog.convert_from(p_contexto, 'UTF8')::jsonb;
    IF v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM 'vec_auditoria.consulta_rrhh.v1'
       OR v_capacidad ->> 'operacion' IS DISTINCT FROM 'vec.auditoria.consultar'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM p_expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM v_recurso_sha
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(p_decision), 'hex')
       OR v_decision ->> 'accion' IS DISTINCT FROM 'vec.auditoria.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM 'auditoria'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM 'historial_auditoria'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM p_expediente_ref
       OR v_decision ->> 'finalidad' IS DISTINCT FROM p_finalidad_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
       OR v_decision ->> 'principal_id' IS DISTINCT FROM v_contexto ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM v_contexto ->> 'perfil_activo_ref'
       OR v_contexto ->> 'persona_version' IS DISTINCT FROM p_persona_version::text
       OR v_contexto ->> 'perfil_version' IS DISTINCT FROM p_perfil_version::text
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM
          '["accion","actor_ref","antes","antes_sha256","datos_disponibles","despues","despues_sha256","expediente_ref","fuente","id","modulo_id","motivo","ocurrido_en","recibo_ref","resultado"]'::jsonb
       OR v_decision -> 'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
        RAISE EXCEPTION 'CT132: consulta denegada' USING ERRCODE = '42501';
    END IF;

    SELECT * INTO STRICT v_consumo FROM
      vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_auditoria_ct_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE
       OR v_consumo.efecto_ref IS DISTINCT FROM p_expediente_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_recurso_sha
       OR v_consumo.decision_ref IS DISTINCT FROM v_decision ->> 'decision_ref'
       OR v_consumo.auditoria_ref IS NULL THEN
        RAISE EXCEPTION 'CT132: consulta denegada' USING ERRCODE = '42501';
    END IF;

    RETURN QUERY
      SELECT 'ct:v:' || pg_catalog.lpad(v.version::text, 16, '0'),
             'ct'::text, 'contratacion_temporal'::text, v.origen_version,
             CASE WHEN v.version = 1 THEN COALESCE(al.actor_ref, '')
                  ELSE COALESCE(a.actuacion_json ->> 'actor_ref', '') END,
             v.estado, v.expediente_ref,
             CASE WHEN v.version = 1 THEN COALESCE(al.recibo_ref, '')
                  ELSE COALESCE(a.recibo_ref, '') END,
             COALESCE(prev.agregado_json_huella_sha256, ''),
             v.agregado_json_huella_sha256,
             CASE WHEN a.actuacion_json ->> 'motivo_clave' ~ '^[a-z][a-z0-9_.-]{0,79}$'
                  THEN a.actuacion_json ->> 'motivo_clave'
                  WHEN a.actuacion_json ->> 'causa_clave' ~ '^[a-z][a-z0-9_.-]{0,79}$'
                  THEN a.actuacion_json ->> 'causa_clave'
                  ELSE '' END,
             v.registrada_en,
             CASE WHEN prev.version IS NULL THEN NULL::jsonb
                  ELSE pg_catalog.jsonb_build_object('fase', prev.fase_clave, 'estado', prev.estado) END,
             pg_catalog.jsonb_build_object('fase', v.fase_clave, 'estado', v.estado),
             true
        FROM vec_contratacion_temporal.expediente_version_integral v
        LEFT JOIN vec_contratacion_temporal.expediente_version_integral prev
          ON prev.expediente_ref = v.expediente_ref AND prev.version = v.version - 1
        LEFT JOIN vec_contratacion_temporal.actuacion_expediente_integral a
          ON a.expediente_ref = v.expediente_ref AND a.version_expediente = v.version
        LEFT JOIN vec_contratacion_temporal.actuacion_alta al
          ON al.expediente_ref = v.expediente_ref AND v.version = 1
       WHERE v.expediente_ref = p_expediente_ref
         AND v.registrada_en >= p_desde AND v.registrada_en < p_hasta
         AND (p_actor_ref = '' OR
              (CASE WHEN v.version = 1 THEN al.actor_ref
                    ELSE a.actuacion_json ->> 'actor_ref' END) = p_actor_ref)
         AND (p_antes_en IS NULL OR
              (v.registrada_en, 'ct'::text,
               'ct:v:' || pg_catalog.lpad(v.version::text, 16, '0'))
              < (p_antes_en, p_antes_fuente, p_antes_id))
       ORDER BY v.registrada_en DESC,
                ('ct:v:' || pg_catalog.lpad(v.version::text, 16, '0')) DESC
       LIMIT p_limite + 1;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01' OR SQLSTATE '55P03'
      OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION 'CT132: consulta denegada' USING ERRCODE = '42501';
END $f$;

ALTER FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
    text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
    text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(
    text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_consultor_rrhh;
COMMIT;
