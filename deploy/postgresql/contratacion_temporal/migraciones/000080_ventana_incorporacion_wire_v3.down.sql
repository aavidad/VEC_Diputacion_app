\set ON_ERROR_STOP on
-- CT80 DOWN: decisión V3 fija6 / capacidad RFC3339Nano, nunca recanoniza autoridad.
-- Sólo sustituye el bucle del helper. Conserva OID, firma, ACL, invoker y llamadores.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000075:registro:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000076:lector:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000079:cotejo-personal:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000080:ventana-wire:v3',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
LOCK TABLE vec_contratacion_temporal.incorporacion_registro_v2,
 vec_contratacion_temporal.incorporacion_auditoria_v2,
 vec_contratacion_temporal.incorporacion_outbox_v2 IN ACCESS EXCLUSIVE MODE;
DO $ct80$
DECLARE
 f oid:=to_regprocedure('vec_contratacion_temporal.incorporacion75_ventana(jsonb,jsonb,timestamptz)');
 autoridad oid:=to_regprocedure('vec_contratacion_temporal.incorporacion75_autoridad(jsonb,bytea[],numeric,numeric,timestamptz)');
 escritor oid:=to_regprocedure('vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)');
 codec oid:=to_regprocedure('vec_contratacion_temporal.instante_incorporacion_go_v2(text,boolean)');
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 p pg_proc%ROWTYPE;
 antes jsonb; despues jsonb; otras jsonb; deps jsonb; ddl text; nuevo text;
 origen text:=$origen$ FOREACH s IN ARRAY ARRAY[c->>'emitida_en',c->>'expira_en',c->>'configuracion_expira_en',c->>'raiz_valida_hasta'] LOOP
  IF vec_contratacion_temporal.instante_incorporacion_go_v2(s,false) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT75: ventana inválida' USING ERRCODE='42501'; END IF;
 END LOOP;
 FOREACH s IN ARRAY ARRAY[d->>'emitida_en',d->>'valida_hasta'] LOOP
  IF vec_contratacion_temporal.instante_incorporacion_go_v2(s,true) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT75: ventana inválida' USING ERRCODE='42501'; END IF;
 END LOOP;
$origen$;
 destino text:=$destino$ FOREACH s IN ARRAY ARRAY[c->>'emitida_en',c->>'expira_en',d->>'emitida_en',d->>'valida_hasta',c->>'configuracion_expira_en',c->>'raiz_valida_hasta'] LOOP
  IF vec_contratacion_temporal.instante_incorporacion_go_v2(s,false) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT75: ventana inválida' USING ERRCODE='42501'; END IF;
 END LOOP;
$destino$;
BEGIN
 IF getdatabaseencoding()<>'UTF8' OR current_user<>'vec_contratacion_temporal_propietario'
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
 OR f IS NULL OR autoridad IS NULL OR escritor IS NULL OR codec IS NULL THEN
  RAISE EXCEPTION 'CT80: propietario o dependencia requerida' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
 IF octet_length(convert_to(p.prosrc,'UTF8'))<>1135
 OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM '9474153d1023ca3617e551398d3862191b91ae56aa818dc9fbf090cda7aba059'
 OR p.proowner<>propietario OR p.pronamespace<>'vec_contratacion_temporal'::regnamespace
 OR p.proname<>'incorporacion75_ventana' OR p.prorettype<>'timestamptz'::regtype
 OR p.proretset OR p.prokind<>'f' OR p.prolang<>(SELECT oid FROM pg_language WHERE lanname='plpgsql')
 OR p.prosecdef OR p.provolatile<>'i' OR p.proparallel<>'s' OR p.proisstrict OR p.proleakproof
 OR p.prosupport<>0 OR p.procost<>100 OR p.prorows<>0 OR p.provariadic<>0 OR p.pronargs<>3
 OR p.pronargdefaults<>0 OR p.proargdefaults IS NOT NULL OR p.proallargtypes IS NOT NULL
 OR p.protrftypes IS NOT NULL OR p.proargmodes IS NOT NULL OR p.probin IS NOT NULL OR p.prosqlbody IS NOT NULL
 OR p.proargnames IS DISTINCT FROM ARRAY['c','d','ahora']::text[]
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog']::text[]
 OR obj_description(f,'pg_proc') IS DISTINCT FROM 'CT80:ventana-wire-v3:fija6-nano'
 OR (SELECT count(*) FROM pg_proc WHERE pronamespace=p.pronamespace AND proname=p.proname)<>1 THEN
  RAISE EXCEPTION 'CT80: definición previa alterada' USING ERRCODE='55000'; END IF;
 IF (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(coalesce(p.proacl,acldefault('f',propietario))) a)
 IS DISTINCT FROM (SELECT array_agg(a::text ORDER BY a::text) FROM unnest(ARRAY[
  makeaclitem(propietario,propietario,'EXECUTE',false)]) a) THEN
  RAISE EXCEPTION 'CT80: ACL privada alterada' USING ERRCODE='42501'; END IF;
 -- Dos llamadores directos legítimos, no se rechazan por depender del helper.
 -- PL/pgSQL no registra todas sus referencias de cuerpo en pg_depend.
 IF (SELECT count(*) FROM pg_proc z WHERE z.oid IN (autoridad,escritor,codec)
  AND z.proowner=propietario AND z.pronamespace=p.pronamespace
  AND z.prokind='f' AND z.prolang=p.prolang
  AND (SELECT count(*) FROM pg_proc zz WHERE zz.pronamespace=z.pronamespace AND zz.proname=z.proname)=1
  AND ((z.oid=autoridad AND NOT z.prosecdef AND z.provolatile='i' AND z.proparallel='s'
   AND encode(sha256(convert_to(z.prosrc,'UTF8')),'hex')='a77ab1845387b0c23339519d64b33f6486b7865933e322ca3e90af2a4f4acee3'
   AND obj_description(z.oid,'pg_proc')='CT75:registro-incorporacion-v2')
  OR (z.oid=escritor AND z.prosecdef AND z.provolatile='v' AND z.proparallel='u'
   AND encode(sha256(convert_to(z.prosrc,'UTF8')),'hex')='b4c9c6a4eb59a42974fc0cf085b4a77ca7e1427880a257177761ed9d13072a39'
   AND obj_description(z.oid,'pg_proc')='CT79:cotejo-personal11:sobre-CT76')
  OR (z.oid=codec AND NOT z.prosecdef AND z.proisstrict AND z.provolatile='i' AND z.proparallel='s'
   AND encode(sha256(convert_to(z.prosrc,'UTF8')),'hex')='bcb1c3abf96b342925bfb2f0fd2835d554ff697bcbcb412315f459fc80d9702d')))<>3 THEN
  RAISE EXCEPTION 'CT80: codec o llamadores alterados' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_proc'::regclass AND d.refobjid=f
  AND NOT(d.classid='pg_proc'::regclass AND d.objid IN (f,autoridad,escritor)))
 OR EXISTS (SELECT 1 FROM pg_proc z WHERE z.oid NOT IN (f,autoridad,escritor)
  AND strpos(z.prosrc,'incorporacion75_ventana')>0) THEN
  RAISE EXCEPTION 'CT80: dependencia llamadora inesperada' USING ERRCODE='55000'; END IF;
 -- RLS cerrada: un propietario no puede interpretar historia oculta como ausencia.
 IF (SELECT count(*) FROM pg_class t WHERE t.relnamespace=p.pronamespace
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
  RAISE EXCEPTION 'CT80: inventario o RLS de historia alterados' USING ERRCODE='55000'; END IF;
 -- Volver a la forma defectuosa puede impedir validar evidencia original fija6.
 -- Guarda conservadora, bajo lock: NO borrar historia para habilitar la retirada.
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2)
 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_auditoria_v2)
 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_outbox_v2) THEN
  RAISE EXCEPTION 'CT80: historia CT conservada' USING ERRCODE='55000'; END IF;
 antes:=to_jsonb(p)-'prosrc';
 SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
 INTO otras FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f;
 -- Conserva dependencias entrantes y salientes, incluidas las legítimas.
 SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
 INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
 OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
 ddl:=pg_get_functiondef(f);
 IF (length(p.prosrc)-length(replace(p.prosrc,origen,'')))/length(origen)<>1
 OR (length(ddl)-length(replace(ddl,origen,'')))/length(origen)<>1
 OR strpos(p.prosrc,destino)>0 THEN RAISE EXCEPTION 'CT80: sustitución no unívoca' USING ERRCODE='55000'; END IF;
 nuevo:=replace(ddl,origen,destino);
 EXECUTE nuevo;
 SELECT to_jsonb(z)-'prosrc' INTO STRICT despues FROM pg_proc z WHERE oid=f;
 IF despues IS DISTINCT FROM antes OR pg_get_functiondef(f) IS DISTINCT FROM nuevo
 OR (SELECT octet_length(convert_to(prosrc,'UTF8')) FROM pg_proc WHERE oid=f)<>931
 OR (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f) IS DISTINCT FROM '1fbddde59729e8cf05f469b7b25e322ea79e7eceae34c3bedb47387b9cd2c21f'
 OR (SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
  FROM pg_proc z WHERE z.pronamespace=p.pronamespace AND z.oid<>f) IS DISTINCT FROM otras
 OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
  FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
   OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'CT80: postimagen no exacta' USING ERRCODE='55000'; END IF;
 EXECUTE format('COMMENT ON FUNCTION %s IS %L',f::regprocedure,'CT75:registro-incorporacion-v2');
 IF obj_description(f,'pg_proc') IS DISTINCT FROM 'CT75:registro-incorporacion-v2' THEN
  RAISE EXCEPTION 'CT80: comentario versionado incompatible' USING ERRCODE='55000'; END IF;
END $ct80$;
COMMIT;

