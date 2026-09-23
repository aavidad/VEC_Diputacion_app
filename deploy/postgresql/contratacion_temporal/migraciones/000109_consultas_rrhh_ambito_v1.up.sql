-- CT109: consumo nominal de CA6 y V3 en la misma transaccion de lectura.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $preimagen$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regprocedure('vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(jsonb,text,text,text,text,text)') IS NULL
    OR to_regrole('vec_contratacion_temporal_consultor_rrhh_ambito') IS NULL
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
          FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       IS DISTINCT FROM '6baef6127627ce9d6e6146c9d5425d7463f1a89b10411aac70fa70ed1944fd98'
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')
          FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       IS DISTINCT FROM '3530828669500274e9a46838c0d890aa0a974c2779e3fa38b5b2054c3919706d'
    OR to_regprocedure('vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'CT109: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;

-- La guarda histórica CT44/45 admite ahora la segunda identidad nominal,
-- pero las fachadas antiguas conservan ACL exclusivamente legacy. La nueva
-- identidad solo puede entrar por estas tres envolventes con CA6.
DO $ampliar_guardas_privadas$
DECLARE f regprocedure; original pg_proc%ROWTYPE; actual pg_proc%ROWTYPE;
 cuerpo text; definicion text; esperado integer; huella_pre text; huella_post text;
 viejo constant text:='''vec_contratacion_temporal_consultor_rrhh''';
 nuevo constant text:='(CASE WHEN pg_catalog.pg_has_role(SESSION_USER, ''vec_contratacion_temporal_consultor_rrhh_ambito'', ''MEMBER'') THEN ''vec_contratacion_temporal_consultor_rrhh_ambito'' ELSE ''vec_contratacion_temporal_consultor_rrhh'' END)';
BEGIN
 FOR f,esperado,huella_pre,huella_post IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'::regprocedure,1,'c3e64993bd7bbb7018a03e487b363486d3d501ff1952d994ef7cea732d5afaa8','1ff9a0b40207685e151507380347c3cd3f5656ca30327c932ea827aeb07d05a1'),
  ('vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,3,'d10eb5676125f8e02ca0cf0857b439ce0b240f9f4b48904170396046e970acbd','19c325c76ac29290b71abb7fb3eb13b840ece8336d155b174103774f209af36d'),
  ('vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,3,'7b7d6c4a419262d54ddb7f2a096e5a4d6e1c962bc58e3027546e221717ed1814','982179dcc2783fad81f93c1c96b8598e267507b434e8e3bece6690312382df15')
 ) v(firma,apariciones,pre,post) LOOP
  SELECT * INTO STRICT original FROM pg_proc WHERE oid=f;
  IF original.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR NOT original.prosecdef OR original.provolatile<>'v'
     OR original.proparallel<>'u' OR original.proleakproof
     OR original.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
     OR original.proconfig IS DISTINCT FROM ARRAY[
       'search_path=pg_catalog','row_security=on','TimeZone=UTC',
       'lock_timeout=1s','statement_timeout=4s',
       'idle_in_transaction_session_timeout=6s']::text[]
     OR (length(original.prosrc)-length(replace(original.prosrc,viejo,'')))/length(viejo)<>esperado
     OR encode(sha256(convert_to(original.prosrc,'UTF8')),'hex') IS DISTINCT FROM huella_pre
  THEN RAISE EXCEPTION 'CT109: guarda privada incompatible' USING ERRCODE='55000'; END IF;
  definicion:=pg_get_functiondef(f);
  IF strpos(definicion,original.prosrc)=0 THEN
    RAISE EXCEPTION 'CT109: definicion no univoca' USING ERRCODE='55000'; END IF;
  cuerpo:=replace(original.prosrc,viejo,nuevo);
  EXECUTE replace(definicion,original.prosrc,cuerpo);
  SELECT * INTO STRICT actual FROM pg_proc WHERE oid=f;
  IF actual.prosrc IS DISTINCT FROM cuerpo
    OR encode(sha256(convert_to(actual.prosrc,'UTF8')),'hex') IS DISTINCT FROM huella_post
    OR actual.proacl IS DISTINCT FROM original.proacl
    OR actual.proowner IS DISTINCT FROM original.proowner
  THEN RAISE EXCEPTION 'CT109: ACL o guarda alterada' USING ERRCODE='55000'; END IF;
 END LOOP;
END $ampliar_guardas_privadas$;

CREATE FUNCTION vec_contratacion_temporal.login_consultor_rrhh_ambito_v1()
RETURNS boolean LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog AS $funcion$
 SELECT EXISTS(SELECT 1 FROM pg_roles u WHERE u.rolname=session_user
   AND u.rolcanlogin AND u.rolinherit AND NOT u.rolsuper
   AND NOT u.rolcreatedb AND NOT u.rolcreaterole
   AND NOT u.rolreplication AND NOT u.rolbypassrls)
 AND EXISTS(SELECT 1 FROM pg_roles g
   WHERE g.rolname='vec_contratacion_temporal_consultor_rrhh_ambito'
   AND NOT g.rolcanlogin AND g.rolinherit AND NOT g.rolsuper
   AND NOT g.rolcreatedb AND NOT g.rolcreaterole
   AND NOT g.rolreplication AND NOT g.rolbypassrls)
 AND (SELECT count(*) FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
   WHERE u.rolname=session_user)=1
 AND EXISTS(SELECT 1 FROM pg_auth_members m JOIN pg_roles u ON u.oid=m.member
   JOIN pg_roles g ON g.oid=m.roleid WHERE u.rolname=session_user
   AND g.rolname='vec_contratacion_temporal_consultor_rrhh_ambito'
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.member
   WHERE g.rolname='vec_contratacion_temporal_consultor_rrhh_ambito');
$funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.login_consultor_rrhh_ambito_v1() FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_contratacion_temporal TO vec_contratacion_temporal_consultor_rrhh_ambito;
CREATE FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_cuadro_rrhh_v1,
 p_comprobante jsonb,
 p_capacidad_canonica bytea,p_decision_canonica bytea,
 p_motivo_canonico bytea,p_contexto_actor_canonico bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload_vec_ad_3 bytea,p_sobre_cose_sign_1 bytea,
 p_evidencia_verificacion bytea,p_raiz_publica_spki bytea
) RETURNS TABLE(
 contenido_canonico bytea, cursor_siguiente text, esquema text,
 acceso_ref text, secuencia numeric, anterior_sha256 text,
 huella_sha256 text, vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text, registrada_en timestamptz,
 auditoria_vec_ref text, auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text, contenido_huella_sha256 text,
 resultado_huella_sha256 text, cursor_huella_sha256 text,
 generada_en timestamptz, expediente_ref text,
 version_expediente numeric, total smallint, recibo_sello_sha256 text, total_filtrado numeric, en_tramitacion numeric, con_incidencia numeric, en_llamamiento numeric
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s'
SET idle_in_transaction_session_timeout='6s'
AS $funcion$
DECLARE v_actor jsonb; v_acreditada timestamptz;
BEGIN
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
  'vec_contratacion_temporal:consulta_rrhh_ambito:v1',0));
 IF vec_contratacion_temporal.login_consultor_rrhh_ambito_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_alcance.clase_ambito IS DISTINCT FROM 'organizacion'
    OR p_alcance.ambito_ref IS DISTINCT FROM p_alcance.organizacion_ref
    OR p_comprobante->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref
 THEN RAISE EXCEPTION 'ambito RRHH denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  v_actor:=convert_from(p_contexto_actor_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN
  RAISE EXCEPTION 'contexto RRHH invalido' USING ERRCODE='42501';
 END;
 IF v_actor->>'cuenta_ref' IS DISTINCT FROM p_comprobante->>'cuenta_ref'
    OR v_actor->>'cuenta_version' IS DISTINCT FROM p_comprobante->>'cuenta_version'
    OR v_actor->>'persona_ref' IS DISTINCT FROM p_comprobante->>'persona_ref'
    OR v_actor->>'persona_version' IS DISTINCT FROM p_comprobante->>'persona_version'
    OR v_actor->>'perfil_activo_ref' IS DISTINCT FROM p_comprobante->>'perfil_ref'
    OR v_actor->>'perfil_version' IS DISTINCT FROM p_comprobante->>'perfil_version'
    OR v_actor->>'contexto_actor_ref' IS DISTINCT FROM p_comprobante->>'contexto_ref'
    OR v_actor->>'contexto_version' IS DISTINCT FROM p_comprobante->>'contexto_version'
 THEN RAISE EXCEPTION 'contexto RRHH divergente' USING ERRCODE='42501'; END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
  p_comprobante,p_comprobante->>'cuenta_ref',p_comprobante->>'persona_ref',
  p_comprobante->>'perfil_ref',p_comprobante->>'contexto_ref',
  p_alcance.organizacion_ref);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'ambito RRHH no vigente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT * FROM vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(
  p_alcance,p_consulta,p_capacidad_canonica,p_decision_canonica,
  p_motivo_canonico,p_contexto_actor_canonico,p_persona_version,
  p_perfil_version,p_payload_vec_ad_3,p_sobre_cose_sign_1,
  p_evidencia_verificacion,p_raiz_publica_spki);
 -- Releer el reloj y la generacion tras cualquier espera del motor/V3.
 -- Un vencimiento o revocacion tardia revierte lectura, consumo y auditoria.
 v_acreditada:=vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
  p_comprobante,p_comprobante->>'cuenta_ref',p_comprobante->>'persona_ref',
  p_comprobante->>'perfil_ref',p_comprobante->>'contexto_ref',
  p_alcance.organizacion_ref);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'ambito RRHH vencido' USING ERRCODE='42501';
 END IF;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_consultor_rrhh_ambito;
CREATE FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(
 p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
 p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
 p_comprobante jsonb,
 p_capacidad_canonica bytea,p_decision_canonica bytea,
 p_motivo_canonico bytea,p_contexto_actor_canonico bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload_vec_ad_3 bytea,p_sobre_cose_sign_1 bytea,
 p_evidencia_verificacion bytea,p_raiz_publica_spki bytea
) RETURNS TABLE(
 contenido_canonico bytea, esquema text,
 acceso_ref text, secuencia numeric, anterior_sha256 text,
 huella_sha256 text, vinculo_identidad_huella_sha256 text,
 alcance_huella_sha256 text, registrada_en timestamptz,
 auditoria_vec_ref text, auditoria_vec_huella_sha256 text,
 consumo_vec_huella_sha256 text, contenido_huella_sha256 text,
 resultado_huella_sha256 text, cursor_huella_sha256 text,
 generada_en timestamptz, expediente_ref text,
 version_expediente numeric, total smallint, recibo_sello_sha256 text
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET timezone='UTC'
SET lock_timeout='1s' SET statement_timeout='4s'
SET idle_in_transaction_session_timeout='6s'
AS $funcion$
DECLARE v_actor jsonb; v_acreditada timestamptz;
BEGIN
 PERFORM pg_advisory_xact_lock_shared(hashtextextended(
  'vec_contratacion_temporal:consulta_rrhh_ambito:v1',0));
 IF vec_contratacion_temporal.login_consultor_rrhh_ambito_v1() IS NOT TRUE
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_alcance.clase_ambito IS DISTINCT FROM 'organizacion'
    OR p_alcance.ambito_ref IS DISTINCT FROM p_alcance.organizacion_ref
    OR p_comprobante->>'organizacion_ref' IS DISTINCT FROM p_alcance.organizacion_ref
 THEN RAISE EXCEPTION 'ambito RRHH denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  v_actor:=convert_from(p_contexto_actor_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN
  RAISE EXCEPTION 'contexto RRHH invalido' USING ERRCODE='42501';
 END;
 IF v_actor->>'cuenta_ref' IS DISTINCT FROM p_comprobante->>'cuenta_ref'
    OR v_actor->>'cuenta_version' IS DISTINCT FROM p_comprobante->>'cuenta_version'
    OR v_actor->>'persona_ref' IS DISTINCT FROM p_comprobante->>'persona_ref'
    OR v_actor->>'persona_version' IS DISTINCT FROM p_comprobante->>'persona_version'
    OR v_actor->>'perfil_activo_ref' IS DISTINCT FROM p_comprobante->>'perfil_ref'
    OR v_actor->>'perfil_version' IS DISTINCT FROM p_comprobante->>'perfil_version'
    OR v_actor->>'contexto_actor_ref' IS DISTINCT FROM p_comprobante->>'contexto_ref'
    OR v_actor->>'contexto_version' IS DISTINCT FROM p_comprobante->>'contexto_version'
 THEN RAISE EXCEPTION 'contexto RRHH divergente' USING ERRCODE='42501'; END IF;
 v_acreditada:=vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
  p_comprobante,p_comprobante->>'cuenta_ref',p_comprobante->>'persona_ref',
  p_comprobante->>'perfil_ref',p_comprobante->>'contexto_ref',
  p_alcance.organizacion_ref);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'ambito RRHH no vigente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT * FROM vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(
  p_alcance,p_consulta,p_capacidad_canonica,p_decision_canonica,
  p_motivo_canonico,p_contexto_actor_canonico,p_persona_version,
  p_perfil_version,p_payload_vec_ad_3,p_sobre_cose_sign_1,
  p_evidencia_verificacion,p_raiz_publica_spki);
 -- Releer el reloj y la generacion tras cualquier espera del motor/V3.
 -- Un vencimiento o revocacion tardia revierte lectura, consumo y auditoria.
 v_acreditada:=vec_contexto_actor_v1.acreditar_ambito_rrhh_v1(
  p_comprobante,p_comprobante->>'cuenta_ref',p_comprobante->>'persona_ref',
  p_comprobante->>'perfil_ref',p_comprobante->>'contexto_ref',
  p_alcance.organizacion_ref);
 IF v_acreditada IS NULL THEN
  RAISE EXCEPTION 'ambito RRHH vencido' USING ERRCODE='42501';
 END IF;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_consultor_rrhh_ambito;
DO $postimagen_acl$
DECLARE f regprocedure; nuevo oid:='vec_contratacion_temporal_consultor_rrhh_ambito'::regrole;
 viejo oid:='vec_contratacion_temporal_consultor_rrhh'::regrole;
BEGIN
 FOR f IN SELECT unnest(ARRAY[
  'vec_contratacion_temporal.consultar_cuadro_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_detalle_rrhh_ambito_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
 ]) LOOP
  IF NOT has_function_privilege(nuevo,f,'EXECUTE')
   OR has_function_privilege(viejo,f,'EXECUTE')
  THEN RAISE EXCEPTION 'CT109: ACL nueva incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOR f IN SELECT unnest(ARRAY[
  'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_cuadro_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_preparacion_resolucion_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,text,date,date)'::regprocedure
 ]) LOOP
  IF has_function_privilege(nuevo,f,'EXECUTE')
     OR NOT has_function_privilege(viejo,f,'EXECUTE')
  THEN RAISE EXCEPTION 'CT109: entrada antigua accesible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $postimagen_acl$;
COMMIT;
