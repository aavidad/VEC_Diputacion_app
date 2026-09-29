\set ON_ERROR_STOP on
-- AD3-110: tres operaciones nominales de la ficha propia de Aspirantes, solo
-- en la superficie externa. Orden: roles Aspirantes, esta migración y
-- después Aspirantes 000001. No depende de Usuarios (AD3-106/107/108): sus
-- anclajes existen una sola vez en el núcleo con o sin ellas, y las
-- migraciones de Usuarios siguen encontrando los suyos después de esta.
-- Sin DOWN con historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000110',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$ BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_aspirantes_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_aspirantes_ejecutor_externo' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-110: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Parche acotado del núcleo en cuatro puntos: exclusión del camino genérico
-- de Contratación, guarda técnica de sesión antes del parseo, ligadura
-- nominal de las tres operaciones y guarda de superficie tras el parseo.
DO $nucleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_reincorporacion_titular_bolsa''\n';
 excl_nuevo text:=excl||E'               AND p_perfil_mutacion IS DISTINCT FROM ''aspirantes_ficha_consultar''\n               AND p_perfil_mutacion IS DISTINCT FROM ''aspirantes_ficha_alta''\n               AND p_perfil_mutacion IS DISTINCT FROM ''aspirantes_ficha_rectificar''\n';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IN (''aspirantes_ficha_consultar'',''aspirantes_ficha_alta'',''aspirantes_ficha_rectificar'')\n               AND pg_catalog.pg_has_role(session_user,''vec_aspirantes_ejecutor_externo'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_aspirantes_ejecutor_externo''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_aspirantes_ejecutor_externo''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_aspirantes_ejecutor_externo'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 post_marca text:=E'    v_huella_capacidad := pg_catalog.encode(';
 post_guard text:=$post$    IF p_perfil_mutacion IN ('aspirantes_ficha_consultar','aspirantes_ficha_alta','aspirantes_ficha_rectificar')
       AND NOT (
         d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
         AND pg_catalog.pg_has_role(session_user,'vec_aspirantes_ejecutor_externo','MEMBER')
         AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid='vec_aspirantes_ejecutor_externo'::regrole AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
         AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
       ) THEN
        RAISE EXCEPTION USING ERRCODE='42501',MESSAGE='consumo Aspirantes rechazado';
    END IF;
$post$;
 extension text:='';
 i integer; a text;
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR length(original)-length(replace(original,post_marca,''))<>length(post_marca)
    OR strpos(original,'aspirantes_ficha_')<>0
 THEN RAISE EXCEPTION 'AD3-110: núcleo incompatible' USING ERRCODE='55000'; END IF;
 FOR i IN 1..3 LOOP
  a:=(ARRAY['consultar','alta','rectificar'])[i];
  extension:=extension||format(E'           OR (\n p_perfil_mutacion IS NOT DISTINCT FROM %L\n AND c->>''audiencia_consumo'' IS NOT DISTINCT FROM %L\n AND d #>> ''{vinculo_autenticacion_actor,superficie}'' IS NOT DISTINCT FROM ''externa_personal''\n AND c->>''operacion'' IS NOT DISTINCT FROM %L\n AND d->>''accion'' IS NOT DISTINCT FROM c->>''operacion''\n AND d->>''modulo_id'' IS NOT DISTINCT FROM ''aspirantes''\n AND d->>''tipo_recurso'' IS NOT DISTINCT FROM ''ficha_aspirante_propia''\n AND d->>''finalidad'' IS NOT DISTINCT FROM ''finalidad:aspirantes:ficha-propia:v1''\n AND d->>''recurso_ref'' IS NOT DISTINCT FROM c->>''efecto_ref''\n AND d->>''contexto_recurso_huella_sha256'' IS NOT DISTINCT FROM c->>''huella_efecto_sha256''\n AND d->''campos_permitidos'' IS NOT DISTINCT FROM %L::jsonb\n AND d->''obligaciones'' IS NOT DISTINCT FROM ''[]''::jsonb)\n',
   'aspirantes_ficha_'||a,'vec_aspirantes.ficha.'||a||'.externa_personal.v1','vec.aspirantes.ficha.'||a,
   CASE a WHEN 'rectificar' THEN '["codigo_postal","domicilio","movil","telefono","version"]'
    ELSE '["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]' END);
 END LOOP;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,guarda,guarda_nueva);
 nuevo:=replace(nuevo,marca,extension||marca);
 nuevo:=replace(nuevo,post_marca,post_guard||post_marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(replace(actual,post_guard||post_marca,post_marca),extension||marca,marca),guarda_nueva,guarda),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-110: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; a text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
 THEN RAISE EXCEPTION 'AD3-110: audiencias incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH a IN ARRAY ARRAY[
  'vec_aspirantes.ficha.consultar.externa_personal.v1',
  'vec_aspirantes.ficha.alta.externa_personal.v1',
  'vec_aspirantes.ficha.rectificar.externa_personal.v1'] LOOP
  IF strpos(d,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-110: audiencia ya registrada' USING ERRCODE='55000'; END IF;
  nueva:=nueva||', '||quote_literal(a)||'::text';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

-- Consumidor nominal: coteja la ligadura antes de delegar en el núcleo y
-- exige un consumo nuevo. Solo lo ejecuta el propietario de Aspirantes.
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(
 p_accion text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; segmento text; campos jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-110: material inválido' USING ERRCODE='22023'; END;
 SELECT q.segmento,q.campos INTO segmento,campos FROM (VALUES
 ('vec.aspirantes.ficha.consultar','consultar','["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb),
 ('vec.aspirantes.ficha.alta','alta','["apellidos","codigo_postal","documento","domicilio","movil","nombre","telefono","version"]'::jsonb),
 ('vec.aspirantes.ficha.rectificar','rectificar','["codigo_postal","domicilio","movil","telefono","version"]'::jsonb)
 ) q(accion,segmento,campos) WHERE q.accion=p_accion;
 IF segmento IS NULL
    OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d #>> '{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'externa_personal'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_aspirantes.ficha.'||segmento||'.externa_personal.v1'
    OR c->>'operacion' IS DISTINCT FROM p_accion OR d->>'accion' IS DISTINCT FROM p_accion
    OR d->>'modulo_id' IS DISTINCT FROM 'aspirantes' OR d->>'tipo_recurso' IS DISTINCT FROM 'ficha_aspirante_propia'
    OR d->>'finalidad' IS DISTINCT FROM 'finalidad:aspirantes:ficha-propia:v1' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-110: ficha denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'aspirantes_ficha_'||segmento,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-110: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_aspirantes_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_aspirantes_propietario;
DO $acl$ DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_aspirantes_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR NOT has_function_privilege('vec_aspirantes_propietario',f,'EXECUTE')
    OR has_function_privilege('vec_aspirantes_ejecutor_externo',f,'EXECUTE')
    OR NOT has_schema_privilege('vec_aspirantes_propietario','vec_autorizacion_atestada_v3','USAGE')
    OR has_schema_privilege('vec_aspirantes_ejecutor_externo','vec_autorizacion_atestada_v3','USAGE')
    OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
      WHERE p.oid=f AND (a.grantee=0 OR a.grantee NOT IN (p.proowner,'vec_aspirantes_propietario'::regrole)))
 THEN RAISE EXCEPTION 'AD3-110: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
