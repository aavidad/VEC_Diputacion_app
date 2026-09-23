\set ON_ERROR_STOP on
-- T13/8: auditoría central de preparar/cancelar/consultar intención propia.
-- Ningún campo contiene correo, HMAC o sobre; sólo se invoca desde Contacto3.
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contacto_usuario_v1:dependencias:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_registro_accesos:migracion:000008',0));
DO $pre$
DECLARE r text;
BEGIN
 IF current_user<>'vec_bolsa_accesos_propietario' OR getdatabaseencoding()<>'UTF8'
    OR to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_version_contacto_v1(bytea,bytea,bytea,bytea,bytea,text,text,text,boolean,numeric)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(text,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text)') IS NOT NULL THEN
    RAISE EXCEPTION 'T13/8: preimagen o dependencia incompatibles' USING ERRCODE='55000';
 END IF;
 FOREACH r IN ARRAY ARRAY['vec_contacto_usuario_owner','vec_contacto_usuario_writer',
      'vec_contacto_usuario_reader','vec_contacto_usuario_migrador'] LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=r AND NOT rolcanlogin AND NOT rolinherit
        AND NOT rolsuper AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
       OR has_function_privilege(r,'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
       OR has_table_privilege(r,'vec_bolsa_registro_accesos.registro_acceso','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
       RAISE EXCEPTION 'T13/8: rol o ACL de contacto incompatibles' USING ERRCODE='55000';
    END IF;
 END LOOP;
END $pre$;

CREATE FUNCTION vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(
    p_accion text,p_auditoria bytea,p_negocio bytea,p_recurso bytea,p_decision bytea,p_contexto bytea,
    p_decision_ref text,p_consumo_ref text,p_consumo_huella text,p_estado text,p_operacion_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s' SET row_security=on
AS $f$
DECLARE a jsonb; d jsonb; b jsonb; entrada jsonb; recibo jsonb; h text;
BEGIN
 IF p_accion IS NULL OR p_accion NOT IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
                    'vec.contacto_usuario.operacion.listar','vec.contacto_usuario.operacion.detalle')
    OR p_estado IS NULL OR p_estado NOT IN ('preparada','confirmada','cancelada','consulta','encontrada','ausente','conflicto')
    OR p_operacion_ref IS NULL OR (p_operacion_ref<>'' AND p_operacion_ref !~ '^opr_[A-Za-z0-9_-]{22,128}$')
    OR p_decision_ref IS NULL OR p_decision_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_consumo_huella IS NULL OR p_consumo_huella !~ '^[0-9a-f]{64}$' OR p_consumo_huella=repeat('0',64)
    OR p_consumo_ref IS DISTINCT FROM 'aud_v3_'||substr(p_consumo_huella,1,32)
    OR p_negocio IS NULL OR octet_length(p_negocio) NOT BETWEEN 2 AND 65536
    OR p_auditoria IS NULL OR octet_length(p_auditoria) NOT BETWEEN 2 AND 16384 THEN
    RAISE EXCEPTION 'T13/8: material de operación inválido' USING ERRCODE='22023';
 END IF;
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF current_user<>'vec_bolsa_accesos_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'decision_ref' IS DISTINCT FROM p_decision_ref
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin AND NOT rolsuper
        AND NOT rolbypassrls AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication)
    OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole
        AND roleid='vec_contacto_usuario_writer'::regrole AND NOT admin_option AND inherit_option AND NOT set_option)
    OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_contacto_usuario_writer'::regrole) THEN
    RAISE EXCEPTION 'T13/8: auditoría de operación denegada' USING ERRCODE='42501';
 END IF;
 b:=vec_autorizacion_atestada_v3.contacto_operacion_material_auditoria_v1(
    p_accion,p_negocio,p_recurso,p_auditoria,p_decision,p_contexto);
 IF b->>'SujetoRef' IS NULL
    OR (p_accion IN ('vec.contacto_usuario.operacion.preparar','vec.contacto_usuario.operacion.cancelar',
                    'vec.contacto_usuario.operacion.detalle') AND b->>'OperacionRef' IS DISTINCT FROM p_operacion_ref)
    OR (p_accion='vec.contacto_usuario.operacion.listar' AND b->>'DespuesDe' IS DISTINCT FROM p_operacion_ref)
    OR (p_accion='vec.contacto_usuario.operacion.listar' AND p_estado<>'consulta')
    OR (p_accion='vec.contacto_usuario.operacion.preparar' AND p_estado NOT IN ('preparada','confirmada','conflicto'))
    OR (p_accion='vec.contacto_usuario.operacion.cancelar' AND p_estado NOT IN ('cancelada','conflicto','ausente'))
    OR (p_accion='vec.contacto_usuario.operacion.detalle' AND p_estado NOT IN ('encontrada','ausente')) THEN
    RAISE EXCEPTION 'T13/8: resultado ajeno al material' USING ERRCODE='42501';
 END IF;
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
        'material_sha256',h,'contexto_recurso_sha256',encode(sha256(p_recurso),'hex'),
        'operacion_ref',p_operacion_ref,'estado_operacion',p_estado),
    'occurred_at',a->>'occurred_at');
 recibo:=vec_bolsa_registro_accesos.registrar_interno_v1(entrada);
 IF recibo->>'id' IS NULL OR recibo->>'id' !~ '^acc_[0-9a-f]{40}$'
    OR recibo->>'signature' IS NULL OR recibo->>'signature' !~ '^[0-9a-f]{64}$'
    OR recibo->>'authorization_ref' IS DISTINCT FROM p_decision_ref
    OR recibo->>'subject_ref' IS DISTINCT FROM b->>'SujetoRef'
    OR recibo->>'action' IS DISTINCT FROM p_accion OR recibo->>'after_hash' IS DISTINCT FROM h
    OR recibo->'metadata' IS DISTINCT FROM entrada->'metadata' THEN
    RAISE EXCEPTION 'T13/8: recibo central divergente' USING ERRCODE='55000';
 END IF;
 RETURN recibo;
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'T13/8: material inválido' USING ERRCODE='22023';
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text) TO vec_contacto_usuario_owner;
DO $post$
DECLARE f oid:='vec_bolsa_registro_accesos.registrar_operacion_contacto_v1(text,bytea,bytea,bytea,bytea,bytea,text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner='vec_bolsa_accesos_propietario'::regrole
     AND prosecdef AND provolatile='v' AND pronargdefaults=0
     AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s','row_security=on'])
    OR has_function_privilege('vec_contacto_usuario_writer',f,'EXECUTE')
    OR has_function_privilege('vec_contacto_usuario_reader',f,'EXECUTE')
    OR NOT has_function_privilege('vec_contacto_usuario_owner',f,'EXECUTE') THEN
    RAISE EXCEPTION 'T13/8: ACL final divergente' USING ERRCODE='42501';
 END IF;
END $post$;
-- La guarda permanece hasta disponer de AD3 post-CT51, prueba PG18 y E10.
DO $incompleta$ BEGIN RAISE EXCEPTION 'T13/8 WIP: falta AD3 post-CT51 y validación' USING ERRCODE='55000'; END $incompleta$;
COMMIT;
