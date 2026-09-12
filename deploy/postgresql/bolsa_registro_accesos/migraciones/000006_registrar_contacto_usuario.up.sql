\set ON_ERROR_STOP on
-- T13/6: auditoría central nominal de contacto propio VEC. Fuente sin instalación.
-- El owner contacto llama después de AD3-35 nuevo, en la misma TX de estado/outbox.
-- Consulta: confirmar esta TX ANTES de descifrar/invocar callback; revalidar al abrir.
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000006',0));
DO $pre$
DECLARE r text;
BEGIN
    IF current_user<>'vec_bolsa_accesos_propietario' OR getdatabaseencoding()<>'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_bolsa_registro_accesos' AND nspowner='vec_bolsa_accesos_propietario'::regrole)
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=to_regprocedure('vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)')
           AND proowner='vec_bolsa_accesos_propietario'::regrole AND NOT prosecdef AND provolatile='v'
           AND proconfig=ARRAY['search_path=pg_catalog']
           AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')='d5a61411e6f547eddb2f5cfc386366484929a17b1e68d568b1c1caa891b280d9')
       OR NOT EXISTS (SELECT 1 FROM pg_class WHERE oid=to_regclass('vec_bolsa_registro_accesos.registro_acceso')
           AND relowner='vec_bolsa_accesos_propietario'::regrole AND relrowsecurity AND relforcerowsecurity)
       OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_bolsa_registro_accesos'::regnamespace AND proname='registrar_contacto_usuario_v1') THEN
        RAISE EXCEPTION 'T13/6: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
    FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_owner','vec_contacto_usuario_writer','vec_contacto_usuario_reader','vec_contacto_usuario_migrador'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolinherit
            AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
           OR has_function_privilege(r,'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
           OR has_function_privilege(r,'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)','EXECUTE')
           OR has_table_privilege(r,'vec_bolsa_registro_accesos.registro_acceso','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
            RAISE EXCEPTION 'T13/6: privilegios incompatibles' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;

CREATE FUNCTION vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(
    p_auditoria bytea,p_negocio bytea,p_recurso bytea,p_decision bytea,p_contexto bytea,
    p_decision_ref text,p_consumo_ref text,p_consumo_huella text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET row_security=on
AS $f$
DECLARE a jsonb; d jsonb; b jsonb; entrada jsonb; recibo jsonb; rol text; h text;
BEGIN
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
       OR p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384
       OR p_decision_ref IS NULL OR p_decision_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_consumo_huella IS NULL OR p_consumo_huella !~ '^[0-9a-f]{64}$' OR p_consumo_huella=repeat('0',64)
       OR p_consumo_ref IS DISTINCT FROM 'aud_v3_'||substr(p_consumo_huella,1,32) THEN
        RAISE EXCEPTION 'T13/6: material de consumo inválido' USING ERRCODE='22023';
    END IF;
    d:=convert_from(p_decision,'UTF8')::jsonb;
    rol:=CASE d->>'accion' WHEN 'vec.contacto_usuario.alta' THEN 'vec_contacto_usuario_writer'
        WHEN 'vec.contacto_usuario.actualizar' THEN 'vec_contacto_usuario_writer'
        WHEN 'vec.contacto_usuario.consultar' THEN 'vec_contacto_usuario_reader' END;
    IF rol IS NULL OR current_user<>'vec_bolsa_accesos_propietario' OR session_user=current_user
       OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
       OR current_setting('TimeZone')<>'UTC'
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper
            AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
       OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
       OR NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole AND roleid=rol::regrole
            AND NOT admin_option AND inherit_option AND NOT set_option)
       OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member=rol::regrole)
       OR d->>'decision_ref' IS DISTINCT FROM p_decision_ref THEN
        RAISE EXCEPTION 'T13/6: auditoría de contacto denegada' USING ERRCODE='42501';
    END IF;
    b:=vec_autorizacion_atestada_v3.contacto_material_auditoria_v1(
        d->>'accion',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    a:=convert_from(p_auditoria,'UTF8')::jsonb;
    h:=encode(sha256(p_negocio),'hex');
    entrada:=jsonb_build_object(
        'actor_id',a->>'actor_id','actor_profile',a->>'actor_profile','actor_roles',a->'actor_roles',
        'represented_subject_id','','auth_method',a->>'auth_method','auth_assurance',a->>'auth_assurance',
        'authorization_ref',p_decision_ref,'purpose',a->>'purpose','action',a->>'action',
        'module_id',a->>'module_id','subject_ref',a->>'subject_ref','object_version',(a->>'object_version')::bigint,
        'expediente_ref','','document_ref','','rule_ref','','reason','','result','permitido',
        'before_hash','','after_hash',h,'correlation_ref',a->>'correlation_ref',
        'metadata',jsonb_build_object('consumo_ref',p_consumo_ref,'consumo_huella_sha256',p_consumo_huella,
            'material_sha256',h,'contexto_recurso_sha256',encode(sha256(p_recurso),'hex')),
        'occurred_at',a->>'occurred_at');
    recibo:=vec_bolsa_registro_accesos.registrar_interno_v1(entrada);
    IF recibo->>'id' IS NULL OR recibo->>'id' !~ '^acc_[0-9a-f]{40}$'
       OR recibo->>'signature' IS NULL OR recibo->>'signature' !~ '^[0-9a-f]{64}$'
       OR recibo->>'authorization_ref' IS DISTINCT FROM p_decision_ref
       OR recibo->>'subject_ref' IS DISTINCT FROM a->>'subject_ref'
       OR recibo->>'actor_id' IS DISTINCT FROM a->>'actor_id'
       OR recibo->>'action' IS DISTINCT FROM a->>'action'
       OR recibo->>'after_hash' IS DISTINCT FROM h OR recibo->'metadata' IS DISTINCT FROM entrada->'metadata' THEN
        RAISE EXCEPTION 'T13/6: recibo central divergente' USING ERRCODE='55000';
    END IF;
    RETURN recibo;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'T13/6: material inválido' USING ERRCODE='22023';
END $f$;

DO $acl$
DECLARE f regprocedure:='vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text)'::regprocedure; a record;
BEGIN
    EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f);
    FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f,pg_get_userbyid(a.grantee));
    END LOOP;
END $acl$;
GRANT USAGE ON SCHEMA vec_bolsa_registro_accesos TO vec_contacto_usuario_owner;
GRANT EXECUTE ON FUNCTION vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text) TO vec_contacto_usuario_owner;
DO $post$
DECLARE propietario oid:='vec_bolsa_accesos_propietario'::regrole; contacto oid:='vec_contacto_usuario_owner'::regrole;
    f oid:='vec_bolsa_registro_accesos.registrar_contacto_usuario_v1(bytea,bytea,bytea,bytea,bytea,text,text,text)'::regprocedure;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario AND prosecdef AND provolatile='v'
            AND pronargdefaults=0 AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s','row_security=on'])
       OR NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT x.grantee)=2
            AND bool_and(x.grantee IN (propietario,contacto) AND x.grantor=propietario AND x.privilege_type='EXECUTE' AND NOT x.is_grantable)
            FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'T13/6: ACL final divergente' USING ERRCODE='55000';
    END IF;
END $post$;
COMMIT;
