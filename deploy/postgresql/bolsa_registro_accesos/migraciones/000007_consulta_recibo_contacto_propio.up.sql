\set ON_ERROR_STOP on
-- Fuente candidata: consulta del recibo histórico propio; no acredita replay de Guardar.
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000007',0));
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
       OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_recibo_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NULL
       OR EXISTS (SELECT 1 FROM pg_proc WHERE pronamespace='vec_bolsa_registro_accesos'::regnamespace AND proname='registrar_consulta_recibo_contacto_v1') THEN
        RAISE EXCEPTION 'T13/7: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
    FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_owner','vec_contacto_usuario_writer','vec_contacto_usuario_reader','vec_contacto_usuario_migrador'] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolinherit
            AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
           OR has_function_privilege(r,'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
           OR has_function_privilege(r,'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)','EXECUTE')
           OR has_table_privilege(r,'vec_bolsa_registro_accesos.registro_acceso','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
            RAISE EXCEPTION 'T13/7: privilegios incompatibles' USING ERRCODE='55000';
        END IF;
    END LOOP;
END $pre$;

CREATE FUNCTION vec_bolsa_registro_accesos.registrar_consulta_recibo_contacto_v1(
    p_auditoria bytea,p_negocio bytea,p_recurso bytea,p_decision bytea,p_contexto bytea,
    p_decision_ref text,p_consumo_ref text,p_consumo_huella text,p_recibo_original bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET row_security=on
AS $f$
DECLARE a jsonb; d jsonb; b jsonb; entrada jsonb; recibo jsonb; rol text; h text; original jsonb; original_ref text:=''; original_hash text:=''; encontrado boolean;
BEGIN
    IF p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
       OR p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384
       OR p_decision_ref IS NULL OR p_decision_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
       OR p_consumo_huella IS NULL OR p_consumo_huella !~ '^[0-9a-f]{64}$' OR p_consumo_huella=repeat('0',64)
       OR p_consumo_ref IS DISTINCT FROM 'aud_v3_'||substr(p_consumo_huella,1,32) THEN
        RAISE EXCEPTION 'T13/7: material de consumo inválido' USING ERRCODE='22023';
    END IF;
    IF p_recibo_original IS NULL OR octet_length(p_recibo_original)>16384 THEN
        RAISE EXCEPTION 'T13/7: recibo original inválido' USING ERRCODE='22023';
    END IF;
    d:=convert_from(p_decision,'UTF8')::jsonb;
    rol:=CASE WHEN d->>'accion'='vec.contacto_usuario.consultar' THEN 'vec_contacto_usuario_writer' END;
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
        RAISE EXCEPTION 'T13/7: auditoría de contacto denegada' USING ERRCODE='42501';
    END IF;
    b:=vec_autorizacion_atestada_v3.contacto_recibo_material_auditoria_v1(
        d->>'accion',p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
    a:=convert_from(p_auditoria,'UTF8')::jsonb;
    encontrado:=octet_length(p_recibo_original)>0;
    IF encontrado THEN
        original:=convert_from(p_recibo_original,'UTF8')::jsonb;
        IF jsonb_typeof(original) IS DISTINCT FROM 'object'
           OR original->>'id' IS NULL OR original->>'id' !~ '^acc_[0-9a-f]{40}$'
           OR original->>'signature' IS NULL OR original->>'signature' !~ '^[0-9a-f]{64}$'
           OR original->>'integrity_algorithm' IS DISTINCT FROM 'sha256-chain-v1'
           OR original->>'subject_ref' IS DISTINCT FROM b->>'SujetoRef'
           OR original->>'object_version' IS DISTINCT FROM b->>'Version'
           OR original->>'module_id' IS DISTINCT FROM 'vec.module.usuarios'
           OR original->>'purpose' IS DISTINCT FROM 'gestion_contacto_propio'
           OR original->>'action' IS DISTINCT FROM
                (CASE WHEN b->>'Version'='1' THEN 'vec.contacto_usuario.alta' ELSE 'vec.contacto_usuario.actualizar' END)
           OR original->>'result' IS DISTINCT FROM 'permitido'
           OR original->>'authorization_ref' IS NULL OR original->>'authorization_ref'=''
           OR original->>'correlation_ref' IS NOT DISTINCT FROM a->>'correlation_ref'
           OR original#>>'{metadata,consumo_huella_sha256}' IS NULL
           OR original#>>'{metadata,consumo_huella_sha256}' !~ '^[0-9a-f]{64}$'
           OR original#>>'{metadata,consumo_huella_sha256}'=repeat('0',64)
           OR original#>>'{metadata,consumo_ref}' IS DISTINCT FROM
                'aud_v3_'||substr(original#>>'{metadata,consumo_huella_sha256}',1,32) THEN
            RAISE EXCEPTION 'T13/7: recibo histórico desligado' USING ERRCODE='42501';
        END IF;
        original_ref:=original->>'id'; original_hash:=encode(sha256(p_recibo_original),'hex');
    END IF;
    h:=encode(sha256(p_negocio),'hex');
    entrada:=jsonb_build_object(
        'actor_id',a->>'actor_id','actor_profile',a->>'actor_profile','actor_roles',a->'actor_roles',
        'represented_subject_id','','auth_method',a->>'auth_method','auth_assurance',a->>'auth_assurance',
        'authorization_ref',p_decision_ref,'purpose',a->>'purpose','action',a->>'action',
        'module_id',a->>'module_id','subject_ref',a->>'subject_ref','object_version',(a->>'object_version')::bigint,
        'expediente_ref','','document_ref','','rule_ref','','reason','','result','permitido',
        'before_hash','','after_hash',h,'correlation_ref',a->>'correlation_ref',
        'metadata',jsonb_build_object('consumo_ref',p_consumo_ref,'consumo_huella_sha256',p_consumo_huella,
            'material_sha256',h,'contexto_recurso_sha256',encode(sha256(p_recurso),'hex'),
            'recibo_encontrado',CASE WHEN encontrado THEN 'true' ELSE 'false' END,
            'recibo_original_ref',original_ref,'recibo_original_sha256',original_hash),
        'occurred_at',a->>'occurred_at');
    recibo:=vec_bolsa_registro_accesos.registrar_interno_v1(entrada);
    IF recibo->>'id' IS NULL OR recibo->>'id' !~ '^acc_[0-9a-f]{40}$'
       OR recibo->>'signature' IS NULL OR recibo->>'signature' !~ '^[0-9a-f]{64}$'
       OR recibo->>'authorization_ref' IS DISTINCT FROM p_decision_ref
       OR recibo->>'subject_ref' IS DISTINCT FROM a->>'subject_ref'
       OR recibo->>'actor_id' IS DISTINCT FROM a->>'actor_id'
       OR recibo->>'action' IS DISTINCT FROM a->>'action'
       OR recibo->>'after_hash' IS DISTINCT FROM h OR recibo->'metadata' IS DISTINCT FROM entrada->'metadata' THEN
        RAISE EXCEPTION 'T13/7: recibo central divergente' USING ERRCODE='55000';
    END IF;
    RETURN recibo;
EXCEPTION WHEN data_exception THEN
    RAISE EXCEPTION 'T13/7: material inválido' USING ERRCODE='22023';
END $f$;

DO $acl$
DECLARE r record; a record; f regprocedure; propietario oid:='vec_bolsa_accesos_propietario'::regrole; esperado oid;
BEGIN
    FOR r IN SELECT * FROM (VALUES
        ('registrar_consulta_recibo_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,bytea)','vec_contacto_usuario_owner')
    ) AS funciones(firma,rol) LOOP
        f:=('vec_bolsa_registro_accesos.'||r.firma)::regprocedure; esperado:=r.rol::regrole;
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
COMMIT;
