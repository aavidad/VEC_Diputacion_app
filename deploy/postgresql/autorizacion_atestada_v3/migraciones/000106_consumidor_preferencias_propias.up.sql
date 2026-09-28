\set ON_ERROR_STOP on
-- AD3-106. Dos operaciones nominales de Usuarios 5.08a. Orden: roles
-- Usuarios, AD3-105 y, antes de Usuarios 000001, esta migración. AD3-102/103
-- pueden instalarse antes en su propio orden; esta extensión comprueba su
-- preimagen viva y no altera una instalación histórica. Sin DOWN con historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000106',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_lista_comunicaciones_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_interno' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_ejecutor_externo' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-106: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_reincorporacion_titular_bolsa''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''preferencias_consulta_usuarios''\n               AND p_perfil_mutacion IS DISTINCT FROM ''preferencias_actualizacion_usuarios''\n';
 runtime text:=E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''consulta_reincorporacion_titular_bolsa''\n';
 runtime_nuevo text:=runtime||E'               OR p_perfil_mutacion IS NOT DISTINCT FROM ''preferencias_consulta_usuarios''\n               OR p_perfil_mutacion IS NOT DISTINCT FROM ''preferencias_actualizacion_usuarios''\n';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IN (''preferencias_consulta_usuarios'',''preferencias_actualizacion_usuarios'')\n               AND d #>> ''{vinculo_autenticacion_actor,superficie}'' IS NOT DISTINCT FROM ''interna_corporativa''\n               AND pg_catalog.pg_has_role(session_user,''vec_usuarios_ejecutor_interno'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_usuarios_ejecutor_interno''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_usuarios_ejecutor_interno''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_usuarios_ejecutor_interno'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n           OR (\n               p_perfil_mutacion IN (''preferencias_consulta_usuarios'',''preferencias_actualizacion_usuarios'')\n               AND d #>> ''{vinculo_autenticacion_actor,superficie}'' IS NOT DISTINCT FROM ''externa_personal''\n               AND pg_catalog.pg_has_role(session_user,''vec_usuarios_ejecutor_externo'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_usuarios_ejecutor_externo''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_usuarios_ejecutor_externo''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_usuarios_ejecutor_externo'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_consulta_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.preferencias.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'preferencias_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["catalogo","valores","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'preferencias_actualizacion_usuarios'
 AND ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.interna_corporativa.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
   OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.externa_personal.v1'
       AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.preferencias.actualizar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'usuarios'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'preferencias_persona'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["valores","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.objsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,runtime,''))<>length(runtime)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR strpos(original,'preferencias_consulta_usuarios')<>0
    OR strpos(original,'preferencias_actualizacion_usuarios')<>0
 THEN RAISE EXCEPTION 'AD3-106: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,guarda,guarda_nueva);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(replace(actual,extension||marca,marca),guarda_nueva,guarda),runtime_nuevo,runtime),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.objsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-106: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_contratacion_temporal.comunicaciones_expediente.consultar.v1')=0
 THEN RAISE EXCEPTION 'AD3-106: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH a IN ARRAY ARRAY[
  'vec_usuarios.preferencias.consultar.interna_corporativa.v1',
  'vec_usuarios.preferencias.actualizar.interna_corporativa.v1',
  'vec_usuarios.preferencias.consultar.externa_personal.v1',
  'vec_usuarios.preferencias.actualizar.externa_personal.v1'] LOOP
  IF strpos(d,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-106: audiencia ya registrada' USING ERRCODE='55000'; END IF;
  nueva:=nueva||', '||quote_literal(a)||'::text';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_preferencias_consulta_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-106: material inválido' USING ERRCODE='22023'; END;
 IF NOT ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.interna_corporativa.v1'
          AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
       OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.consultar.externa_personal.v1'
          AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
    OR c->>'operacion' IS DISTINCT FROM 'vec.preferencias.consultar'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'preferencias_persona'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["catalogo","valores","version"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-106: consulta denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'preferencias_consulta_usuarios',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-106: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_preferencias_actualizacion_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-106: material inválido' USING ERRCODE='22023'; END;
 IF NOT ((c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.interna_corporativa.v1'
          AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa')
       OR (c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_usuarios.preferencias.actualizar.externa_personal.v1'
          AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'))
    OR c->>'operacion' IS DISTINCT FROM 'vec.preferencias.actualizar'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'usuarios'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'preferencias_persona'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:usuarios:preferencias-propias:v1'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM '["valores","version"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-106: actualización denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'preferencias_actualizacion_usuarios',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-106: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

DO $acl$ DECLARE nombre text; f regprocedure; x record; BEGIN
 GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_usuarios_propietario;
 FOREACH nombre IN ARRAY ARRAY['consumir_preferencias_consulta_v3_atestada','consumir_preferencias_actualizacion_v3_atestada'] LOOP
  f:=format('vec_autorizacion_atestada_v3.%I(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',nombre)::regprocedure;
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee<>p.proowner LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_usuarios_propietario',f::text);
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
     OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
     OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
     OR NOT has_function_privilege('vec_usuarios_propietario',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD3-106: ACL incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
