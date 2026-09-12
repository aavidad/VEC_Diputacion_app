\set ON_ERROR_STOP on
-- CT79: cotejo wire Personal11 desde resultado CT12 confirmado; NO muta originales.
-- CT70 valida primero las 12 claves y motivo_rechazo exactamente cero (no null).
-- Se proyecta SOLO el resultado esperado, nunca el JSON recibido/material/recibo.
-- Igualdad cerrada de los once campos, resto del registro y permisos intactos.
-- Sin auxiliares persistentes/TEMP, roles nuevos, grants ni recanonización.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
-- Primero Personal, después CT: mismo orden que CT75/Personal5/AD329.
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000075:registro:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000076:lector:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000079:cotejo-personal:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
LOCK TABLE vec_contratacion_temporal.incorporacion_registro_v2,
 vec_contratacion_temporal.incorporacion_auditoria_v2,
 vec_contratacion_temporal.incorporacion_outbox_v2 IN ACCESS EXCLUSIVE MODE;
DO $ct79$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)');
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 ejecutor oid:='vec_contratacion_temporal_ejecutor'::regrole;
 personal_owner oid:='vec_personal_propietario'::regrole;
 personal_executor oid:='vec_personal_ejecutor'::regrole;
 fachada oid:=to_regprocedure('vec_personal.acreditar_alta_ejercicio_v2(text,text,text,bigint,text,text,text,text,text,text,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea)');
 p pg_proc%ROWTYPE; q pg_proc%ROWTYPE;
 antes jsonb; despues jsonb; otras jsonb; deps jsonb; ddl text; nuevo text;
 origen text:=$origen$'resultado',resultado_personal,'material_canonico'$origen$;
 destino text:=$destino$'resultado',(resultado_personal - 'motivo_rechazo'),'material_canonico'$destino$;
BEGIN
 IF getdatabaseencoding()<>'UTF8' OR current_user<>'vec_contratacion_temporal_propietario'
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
 OR f IS NULL THEN RAISE EXCEPTION 'CT79: propietario o firma incompatibles' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 -- Preimagen completa, no tolera deriva, overload ni defaults ocultos.
 IF octet_length(convert_to(p.prosrc,'UTF8'))<>23377
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '6b376e406f16c60b73d19c9854d7ab142fd452c2c74f2e8b88b64c327cbaef25'
 OR p.proowner<>propietario OR p.pronamespace<>'vec_contratacion_temporal'::regnamespace
 OR p.proname<>'registrar_incorporacion_ejercicio_v2' OR p.prorettype<>'jsonb'::regtype
 OR p.proretset OR p.prokind<>'f' OR p.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
 OR NOT p.prosecdef OR p.provolatile<>'v' OR p.proparallel<>'u' OR p.proisstrict OR p.proleakproof
 OR p.prosupport<>0 OR p.procost<>100 OR p.prorows<>0 OR p.provariadic<>0 OR p.pronargs<>23
 OR p.pronargdefaults<>0 OR p.proargdefaults IS NOT NULL OR p.proallargtypes IS NOT NULL
 OR p.protrftypes IS NOT NULL OR p.proargmodes IS NOT NULL OR p.probin IS NOT NULL OR p.prosqlbody IS NOT NULL
 OR p.proargnames IS DISTINCT FROM ARRAY['material','seguimiento_ref','ct_capacidad','ct_decision','ct_motivo','ct_contexto','ct_persona_version','ct_perfil_version','ct_payload','ct_sobre','ct_evidencia','ct_raiz','lector_capacidad','lector_decision','lector_motivo','lector_contexto','lector_persona_version','lector_perfil_version','lector_payload','lector_sobre','lector_evidencia','lector_raiz','evidencia_orden']::text[]
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s','statement_timeout=5s']::text[]
 OR obj_description(f,'pg_proc') IS DISTINCT FROM 'CT76:lector-incorporacion-v2:sobre-CT75'
 OR (SELECT count(*) FROM pg_proc WHERE pronamespace=p.pronamespace AND proname=p.proname)<>1 THEN
  RAISE EXCEPTION 'CT79: definición previa alterada' USING ERRCODE='55000'; END IF;
 IF (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(coalesce(p.proacl,acldefault('f',propietario))) a)
 IS DISTINCT FROM (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(ARRAY[
  makeaclitem(propietario,propietario,'EXECUTE',false),makeaclitem(ejecutor,propietario,'EXECUTE',false)]) a) THEN
  RAISE EXCEPTION 'CT79: ACL previa alterada' USING ERRCODE='42501'; END IF;
 -- Personal5 real, SD nominal20. No inventa su hash mientras lo publica su propietario.
 -- El cuerpo y el permiso V2 se validan/revisan en Personal5/AD329; aquí frontera y ACL.
 IF fachada IS NULL OR NOT has_schema_privilege(current_user,'vec_personal','USAGE')
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_personal' AND nspowner=personal_owner)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=personal_owner AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT has_function_privilege(current_user,fachada,'EXECUTE') THEN
  RAISE EXCEPTION 'CT79: fachada propietaria V2 requerida' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT q FROM pg_proc WHERE oid=fachada;
 IF q.proowner<>personal_owner OR q.pronamespace<>'vec_personal'::regnamespace OR q.prorettype<>'jsonb'::regtype
 OR q.proretset OR q.prokind<>'f' OR q.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
 OR NOT q.prosecdef OR q.provolatile<>'v' OR q.proparallel<>'u' OR q.proisstrict OR q.proleakproof
 OR q.prosupport<>0 OR q.procost<>100 OR q.prorows<>0 OR q.provariadic<>0 OR q.pronargs<>20
 OR q.pronargdefaults<>0 OR q.proargdefaults IS NOT NULL OR q.proallargtypes IS NOT NULL
 OR q.protrftypes IS NOT NULL OR q.proargmodes IS NOT NULL OR q.probin IS NOT NULL OR q.prosqlbody IS NOT NULL
 OR q.proargnames IS DISTINCT FROM ARRAY['p_organizacion_ref','p_solicitud_ref','p_expediente_ref','p_version_expediente','p_resultado_ref','p_recibo_ref','p_relacion_ref','p_ocupacion_ref','p_material_sha256','p_unidad_ref','p_capacidad','p_decision','p_motivo','p_contexto','p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz']::text[]
 OR q.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s']::text[]
 OR obj_description(fachada,'pg_proc') IS DISTINCT FROM 'Personal000005:lector_incorporacion:v2:ambitos_org_unidad'
 OR (SELECT count(*) FROM pg_proc WHERE pronamespace=q.pronamespace AND proname=q.proname)<>1
 OR (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(coalesce(q.proacl,acldefault('f',personal_owner))) a)
 IS DISTINCT FROM (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(ARRAY[
  makeaclitem(personal_owner,personal_owner,'EXECUTE',false),makeaclitem(personal_executor,personal_owner,'EXECUTE',false),
  makeaclitem(propietario,personal_owner,'EXECUTE',false)]) a) THEN
  RAISE EXCEPTION 'CT79: firma, metadata o ACL Personal V2 incompatibles' USING ERRCODE='55000'; END IF;
 -- Inventario/RLS cerrado antes de leer historia: ninguna policy puede ocultarla.
 IF (SELECT count(*) FROM pg_class t WHERE t.relnamespace='vec_contratacion_temporal'::regnamespace
  AND t.relname IN ('incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2')
  AND t.relkind='r' AND t.relpersistence='p' AND NOT t.relispartition AND t.relowner=propietario
  AND t.relrowsecurity AND t.relforcerowsecurity
  AND obj_description(t.oid,'pg_class')='CT75:registro-incorporacion-v2:inmutable'
  AND NOT EXISTS (SELECT 1 FROM pg_inherits WHERE inhparent=t.oid OR inhrelid=t.oid)
  AND (SELECT count(*) FROM pg_policy WHERE polrelid=t.oid)=1
  AND EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t.oid AND polname='propietario' AND polcmd='*'
   AND polpermissive AND polroles=ARRAY[propietario] AND pg_get_expr(polqual,polrelid)='true'
   AND pg_get_expr(polwithcheck,polrelid)='true')
  AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(t.relacl,acldefault('r',t.relowner))) a
   WHERE a.grantee<>propietario OR a.grantor<>propietario OR a.is_grantable)
  AND NOT EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid=t.oid AND attacl IS NOT NULL))<>3 THEN
  RAISE EXCEPTION 'CT79: inventario, ACL o RLS de historia alterados' USING ERRCODE='55000'; END IF;
 -- Dependientes normales (refclassid evita colisiones OID) y PL/pgSQL sin auto-track.
 IF EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=f
  AND NOT(d.classid='pg_proc'::regclass AND d.objid=f))
 OR EXISTS (SELECT 1 FROM pg_proc z WHERE z.oid<>f AND strpos(z.prosrc,'registrar_incorporacion_ejercicio_v2')>0) THEN
  RAISE EXCEPTION 'CT79: dependencia llamadora conservada' USING ERRCODE='55000'; END IF;
 antes:=to_jsonb(p)-'prosrc';
 SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
 INTO otras FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f;
 SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 ddl:=pg_get_functiondef(f);
 IF (length(p.prosrc)-length(replace(p.prosrc,origen,'')))/length(origen)<>1
 OR (length(ddl)-length(replace(ddl,origen,'')))/length(origen)<>1
 OR strpos(p.prosrc,destino)>0 THEN RAISE EXCEPTION 'CT79: sustitución no unívoca' USING ERRCODE='55000'; END IF;
 nuevo:=replace(ddl,origen,destino);
 -- CREATE OR REPLACE desde definición catalogada, nunca DROP ni cuerpo abreviado.
 EXECUTE nuevo;
 SELECT to_jsonb(z)-'prosrc' INTO STRICT despues FROM pg_proc z WHERE oid=f;
 IF despues IS DISTINCT FROM antes OR pg_get_functiondef(f) IS DISTINCT FROM nuevo
 OR (SELECT octet_length(convert_to(prosrc,'UTF8')) FROM pg_proc WHERE oid=f)<>23398
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'b4c9c6a4eb59a42974fc0cf085b4a77ca7e1427880a257177761ed9d13072a39'
 OR (SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
  FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f) IS DISTINCT FROM otras
 OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
  FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT79: postimagen no exacta' USING ERRCODE='55000'; END IF;
 EXECUTE format('COMMENT ON FUNCTION %s IS %L',f::regprocedure,'CT79:cotejo-personal11:sobre-CT76');
 IF obj_description(f,'pg_proc') IS DISTINCT FROM 'CT79:cotejo-personal11:sobre-CT76' THEN
  RAISE EXCEPTION 'CT79: comentario versionado incompatible' USING ERRCODE='55000'; END IF;
END $ct79$;
COMMIT;
