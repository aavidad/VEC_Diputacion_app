\set ON_ERROR_STOP on
-- Fuente candidata: consulta del recibo histórico propio; no acredita replay de Guardar.
BEGIN;
SET LOCAL ROLE vec_contacto_usuario_owner;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:migracion:2',0));
DO $pre$
BEGIN
    IF getdatabaseencoding()<>'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contacto_usuario_v1' AND nspowner=current_user::regrole)
       OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_contacto_usuario_v1.versiones')
            AND relowner=current_user::regrole AND relrowsecurity AND relforcerowsecurity)
       OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_autorizacion_atestada_v3.revalidar_recibo_contacto_usuario_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_recibo_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,bytea)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_contacto_usuario_v1'::regnamespace
            AND proname='consultar_recibo_contacto_v1') THEN
        RAISE EXCEPTION 'contacto recibo: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
END $pre$;

-- Reutiliza lectura RLS de versiones: no exige que la versión siga en actual.
-- El sujeto debe coincidir con la persona autenticada antes de fijar contexto RLS.
CREATE FUNCTION vec_contacto_usuario_v1.consultar_recibo_contacto_v1(
    p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload_v3 bytea,p_sobre_cose bytea,
    p_evidencia bytea,p_raiz bytea,p_negocio bytea,p_recurso bytea,p_auditoria bytea)
RETURNS TABLE(encontrado boolean,sujeto_ref text,version numeric,recibo_original bytea,
    consumo_original_ref text,consumo_original_huella_sha256 text,auditoria_consulta bytea,
    consumo_consulta_ref text,consumo_consulta_huella_sha256 text)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s'
AS $f$
DECLARE v_consumo record; v_revalidacion record; v_original record; b jsonb; o jsonb;
    v_encontrado boolean; v_original_bytes bytea:=''::bytea; v_original_ref text:=''; v_original_hash text:='';
    v_consulta bytea; v_sujeto text; v_version numeric;
BEGIN
    IF p_accion IS DISTINCT FROM 'vec.contacto_usuario.consultar'
       OR current_user<>'vec_contacto_usuario_owner'
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
       OR NOT pg_has_role(session_user,'vec_contacto_usuario_writer','MEMBER') THEN
        RAISE EXCEPTION 'contacto recibo: consulta denegada' USING ERRCODE='42501';
    END IF;
    SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_recibo_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    IF v_consumo.consumo_nuevo IS DISTINCT FROM true THEN
        RAISE EXCEPTION 'contacto recibo: autorización nueva requerida' USING ERRCODE='P1102';
    END IF;
    b:=convert_from(p_negocio,'UTF8')::jsonb;
    v_sujeto:=b->>'SujetoRef'; v_version:=(b->>'Version')::numeric;
    IF v_sujeto IS NULL OR v_sujeto IS DISTINCT FROM convert_from(p_contexto,'UTF8')::jsonb->>'persona_ref' THEN
        RAISE EXCEPTION 'contacto recibo: sujeto ajeno' USING ERRCODE='42501';
    END IF;
    PERFORM set_config('vec.contacto.sujeto_ref',v_sujeto,true),set_config('vec.contacto.accion',p_accion,true),
        set_config('vec.contacto.version',v_version::text,true),set_config('vec.contacto.version_anterior',v_version::text,true);
    SELECT v.auditoria_central,v.decision_ref,v.consumo_ref,v.consumo_huella_sha256,
        encode(sha256(v.negocio),'hex') AS material_sha256,
        encode(sha256(v.contexto_recurso),'hex') AS contexto_recurso_sha256
        INTO v_original FROM vec_contacto_usuario_v1.versiones v
        WHERE v.sujeto_ref=v_sujeto AND v.version=v_version;
    v_encontrado:=FOUND;
    IF v_encontrado THEN
        o:=convert_from(v_original.auditoria_central,'UTF8')::jsonb;
        IF o->>'authorization_ref' IS DISTINCT FROM v_original.decision_ref
           OR o->>'subject_ref' IS DISTINCT FROM v_sujeto OR (o->>'object_version')::numeric IS DISTINCT FROM v_version
           OR o->>'after_hash' IS DISTINCT FROM v_original.material_sha256
           OR o->'metadata' IS DISTINCT FROM jsonb_build_object('consumo_ref',v_original.consumo_ref,
                'consumo_huella_sha256',v_original.consumo_huella_sha256,'material_sha256',v_original.material_sha256,
                'contexto_recurso_sha256',v_original.contexto_recurso_sha256) THEN
            RAISE EXCEPTION 'contacto recibo: evidencia histórica divergente' USING ERRCODE='55000';
        END IF;
        v_original_bytes:=v_original.auditoria_central;
        v_original_ref:=v_original.consumo_ref; v_original_hash:=v_original.consumo_huella_sha256;
    END IF;
    -- La ausencia también se audita. No afirma que una transacción pendiente no confirme después.
    v_consulta:=convert_to(vec_bolsa_registro_accesos.registrar_consulta_recibo_contacto_v1(
        p_auditoria,p_negocio,p_recurso,p_decision,p_contexto,v_consumo.decision_ref,
        v_consumo.auditoria_ref,v_consumo.consumo_huella_sha256,v_original_bytes)::text,'UTF8');
    -- T13 puede esperar: revalidar el mismo consumo después de todas las esperas.
    SELECT * INTO STRICT v_revalidacion FROM vec_autorizacion_atestada_v3.revalidar_recibo_contacto_usuario_v3_atestada(
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload_v3,p_sobre_cose,p_evidencia,p_raiz,p_negocio,p_recurso,p_auditoria);
    IF v_revalidacion.decision_ref IS DISTINCT FROM v_consumo.decision_ref
       OR v_revalidacion.consumo_huella_sha256 IS DISTINCT FROM v_consumo.consumo_huella_sha256
       OR v_revalidacion.revalidada_en IS NULL THEN
        RAISE EXCEPTION 'contacto recibo: revalidación divergente' USING ERRCODE='42501';
    END IF;
    RETURN QUERY SELECT v_encontrado,v_sujeto,v_version,v_original_bytes,v_original_ref,v_original_hash,
        v_consulta,v_consumo.auditoria_ref::text,v_consumo.consumo_huella_sha256::text;
END $f$;
DO $acl$
DECLARE r record; a record; f regprocedure; propietario oid:='vec_contacto_usuario_owner'::regrole; esperado oid;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','vec_contacto_usuario_writer')
    ) AS funciones(firma,rol) LOOP
        f:=('vec_contacto_usuario_v1.'||r.firma)::regprocedure; esperado:=r.rol::regrole;
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
        FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
            WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
            EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
        END LOOP;
        IF esperado<>propietario THEN EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO %I',f,r.rol); END IF;
        IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
           OR NOT COALESCE((SELECT count(*)=CASE WHEN esperado=propietario THEN 1 ELSE 2 END
                AND count(DISTINCT x.grantee)=count(*)
                AND bool_and(x.grantee IN (propietario,esperado) AND x.grantor=propietario
                    AND x.privilege_type='EXECUTE' AND NOT x.is_grantable)
                FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
            RAISE EXCEPTION 'contacto recibo: ACL divergente' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $acl$;
DO $post$
DECLARE r text; t text; privilegio text;
BEGIN
    FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_writer','vec_contacto_usuario_reader'] LOOP
        IF has_schema_privilege(r,'vec_contacto_usuario_v1','CREATE')
           OR has_database_privilege(r,current_database(),'CREATE') THEN
            RAISE EXCEPTION 'contacto recibo: CREATE excesivo' USING ERRCODE='42501';
        END IF;
        FOREACH t IN ARRAY ARRAY['versiones','actual','outbox'] LOOP
            FOREACH privilegio IN ARRAY ARRAY['SELECT','INSERT','UPDATE','DELETE','TRUNCATE','REFERENCES','TRIGGER'] LOOP
                IF has_table_privilege(r,format('vec_contacto_usuario_v1.%I',t),privilegio) THEN
                    RAISE EXCEPTION 'contacto recibo: acceso directo a tablas' USING ERRCODE='42501';
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;
    IF has_function_privilege('vec_contacto_usuario_reader','vec_contacto_usuario_v1.consultar_recibo_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE')
       OR has_function_privilege('vec_contacto_usuario_writer','vec_contacto_usuario_v1.consultar_contacto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','EXECUTE') THEN
        RAISE EXCEPTION 'contacto recibo: permisos cruzados' USING ERRCODE='42501';
    END IF;
END $post$;
COMMIT;
