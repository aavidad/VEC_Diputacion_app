\set ON_ERROR_STOP on
-- AD3-000038: reserva confirmada por integrador; borrador privado no instalado.
-- Fuente: AD3-10 (fiscalización), extensiones nominales AD3-20/31.
-- CT confirma subsanación y consume esta autorización en la misma transacción.
-- CT coteja ámbitos, material, retorno, política y motivo contra su estado real.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_contratacion_temporal.dependencias.subsanacion_reparos.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000038', 0));

DO $precondicion$
DECLARE
    propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    ct oid := 'vec_contratacion_temporal_propietario'::regrole;
BEGIN
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR getdatabaseencoding() <> 'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace
                      WHERE nspname='vec_autorizacion_atestada_v3'
                        AND nspowner=propietario)
       OR EXISTS (SELECT 1 FROM pg_roles WHERE oid IN (propietario,ct)
                   AND (rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb
                        OR rolcreaterole OR rolreplication OR rolbypassrls))
       OR NOT has_schema_privilege(ct,'vec_autorizacion_atestada_v3','USAGE')
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc
                   WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace
                     AND proname='registrar_y_consumir_subsanacion_reparos_v3_atestada') THEN
        RAISE EXCEPTION 'estado incompatible para consumo de subsanación VEC-AD-3'
            USING ERRCODE='55000';
    END IF;
END $precondicion$;

-- Se amplía únicamente la lista cerrada del núcleo vigente. Las comprobaciones
-- de identidad viva, HMAC, motivo, vigencia y auditoría permanecen en ese núcleo.
-- No se reemplaza por una versión antigua ni se modifican fachadas anteriores.
DO $ampliar$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    definicion text;
    esperada text;
    metadata jsonb;
    dependencias jsonb;
    marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'subsanacion_reparos_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_contratacion_temporal.confirmar_alta_atestada.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND d ->> 'accion' IS NOT DISTINCT FROM
                   'contratacion_temporal.subsanacion_reparos.registrar'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM
                   'subsanacion_reparo_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM
                   'gestionar_contratacion_temporal'
           )
$perfil$;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc'
      INTO STRICT definicion,metadata FROM pg_proc p
     WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef AND p.provolatile='v'
       AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
    IF length(definicion)-length(replace(definicion,marca,''))<>length(marca)
       OR strpos(definicion,'subsanacion_reparos_ct')<>0
       OR strpos(definicion,'contratacion_temporal.subsanacion_reparos.registrar')<>0
       OR strpos(definicion,'p_perfil_mutacion IS NOT DISTINCT FROM ''fiscalizacion''')=0
       OR strpos(definicion,'contratacion_temporal.fiscalizacion.registrar')=0
       OR strpos(definicion,'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva')=0 THEN
        RAISE EXCEPTION 'núcleo incompatible para subsanación VEC-AD-3'
            USING ERRCODE='55000';
    END IF;
    SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
                            d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      INTO dependencias FROM pg_depend d
     WHERE d.classid='pg_proc'::regclass AND d.objid=f;
    esperada := replace(definicion,marca,extension||marca);
    EXECUTE esperada;
    IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f)
           IS DISTINCT FROM metadata
       OR pg_get_functiondef(f) IS DISTINCT FROM esperada
       OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
                                   d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
           IS DISTINCT FROM dependencias THEN
        RAISE EXCEPTION 'subsanación alteró metadata o dependencias del núcleo VEC-AD-3'
            USING ERRCODE='55000';
    END IF;
END $ampliar$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    consumida_en timestamptz,
    consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    c jsonb;
    d jsonb;
    v_consumo record;
BEGIN
    IF vec_autorizacion_atestada_v3.capacidad_cruda_prevalida(p_capacidad_canonica) IS NOT TRUE
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(p_decision_canonica) NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR p_contexto_actor_canonico IS NULL
       OR p_persona_version IS NULL OR p_perfil_version IS NULL
       OR p_payload_vec_ad_3 IS NULL OR p_sobre_cose_sign1 IS NULL
       OR p_evidencia_verificacion IS NULL OR p_raiz_publica_spki IS NULL THEN
        RAISE EXCEPTION 'subsanación VEC-AD-3 inválida' USING ERRCODE='22023';
    END IF;
    BEGIN
        c := pg_catalog.convert_from(p_capacidad_canonica,'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica,'UTF8')::jsonb;
    EXCEPTION WHEN data_exception OR invalid_text_representation
                   OR character_not_in_repertoire OR untranslatable_character THEN
        RAISE EXCEPTION 'subsanación VEC-AD-3 inválida' USING ERRCODE='22023';
    END;
    IF c ->> 'audiencia_consumo' IS DISTINCT FROM
           'vec_contratacion_temporal.confirmar_alta_atestada.v1'
       OR c ->> 'operacion' IS DISTINCT FROM
           'contratacion_temporal.subsanacion_reparos.registrar'
       OR d ->> 'accion' IS DISTINCT FROM
           'contratacion_temporal.subsanacion_reparos.registrar'
       OR d ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR d ->> 'tipo_recurso' IS DISTINCT FROM 'subsanacion_reparo_contratacion_temporal'
       OR d ->> 'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
       OR d ->> 'recurso_ref' IS DISTINCT FROM c ->> 'efecto_ref'
       OR d ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM
           c ->> 'huella_efecto_sha256' THEN
        RAISE EXCEPTION 'subsanación VEC-AD-3 rechazada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
          'subsanacion_reparos_ct',p_capacidad_canonica,p_decision_canonica,
          p_motivo_canonico,p_contexto_actor_canonico,p_persona_version,p_perfil_version,
          p_payload_vec_ad_3,p_sobre_cose_sign1,p_evidencia_verificacion,p_raiz_publica_spki);
    -- El replay semántico confirmado pertenece al recibo CT; una escritura
    -- nueva nunca puede reutilizar una autorización consumida anteriormente.
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'subsanación requiere consumo nuevo' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END $funcion$;

ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC;
-- Cerrar también ACL predeterminadas ajenas sobre esta función nueva.
DO $acl_nueva$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    a record;
BEGIN
    FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,
             LATERAL aclexplode(p.proacl) x
              WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',
                       f::regprocedure,pg_get_userbyid(a.grantee));
    END LOOP;
END $acl_nueva$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_contratacion_temporal_propietario;

DO $poscondicion$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_subsanacion_reparos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    ct oid := 'vec_contratacion_temporal_propietario'::regrole;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
                    AND p.proowner=propietario AND p.prosecdef AND p.provolatile='v'
                    AND p.pronargdefaults=0
                    AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
       OR NOT coalesce((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
                        AND bool_and(a.grantee IN (propietario,ct)
                                     AND a.grantor=propietario
                                     AND a.privilege_type='EXECUTE'
                                     AND NOT a.is_grantable)
                          FROM pg_proc p CROSS JOIN LATERAL aclexplode(
                              coalesce(p.proacl,acldefault('f',p.proowner))) a
                         WHERE p.oid=f),false)
       OR has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
       OR has_function_privilege('vec_autorizacion_atestada_v3_consumidor',f,'EXECUTE')
       OR has_function_privilege('vec_autorizacion_atestada_v3_emisor',f,'EXECUTE') THEN
        RAISE EXCEPTION 'autoridad incompatible para subsanación VEC-AD-3'
            USING ERRCODE='55000';
    END IF;
END $poscondicion$;
COMMIT;
