\set ON_ERROR_STOP on
-- CT85: evita la colisión del alias pg_policy p con el estado PL/pgSQL p.
-- Sólo los dos alias del guard RLS; conserva la variable de estado y el contrato.
-- Avance desde la preimagen exacta: no reaplicar CT81 ni DOWN sobre historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000081:localizador:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000085:alias-politica-localizador',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

DO $ct85$
DECLARE
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 f oid:=to_regprocedure('vec_contratacion_temporal.localizar_incorporacion_original_v2(text,text,text)');
 funcion pg_proc%ROWTYPE; ddl text; nuevo text;
 antes jsonb; otras jsonb; deps jsonb; compartidas jsonb; comentario text;
 origen text:=$origen$  AND (SELECT count(*) FROM pg_policy p WHERE p.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy p WHERE p.polrelid=c.oid AND p.polname='propietario'
   AND p.polcmd='*' AND p.polpermissive AND p.polroles=ARRAY['vec_contratacion_temporal_propietario'::regrole::oid]
   AND pg_get_expr(p.polqual,p.polrelid)='true' AND pg_get_expr(p.polwithcheck,p.polrelid)='true'))$origen$;
 destino text:=$destino$  AND (SELECT count(*) FROM pg_policy localizador_politica WHERE localizador_politica.polrelid=c.oid)=1
  AND EXISTS(SELECT 1 FROM pg_policy localizador_politica WHERE localizador_politica.polrelid=c.oid AND localizador_politica.polname='propietario'
   AND localizador_politica.polcmd='*' AND localizador_politica.polpermissive AND localizador_politica.polroles=ARRAY['vec_contratacion_temporal_propietario'::regrole::oid]
   AND pg_get_expr(localizador_politica.polqual,localizador_politica.polrelid)='true' AND pg_get_expr(localizador_politica.polwithcheck,localizador_politica.polrelid)='true'))$destino$;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' OR getdatabaseencoding()<>'UTF8'
  OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
  OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT85: propietario incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT funcion FROM pg_proc WHERE oid=f;
 ddl:=pg_get_functiondef(f);
 IF funcion.proowner<>propietario OR NOT funcion.prosecdef
  OR encode(sha256(convert_to(funcion.prosrc,'UTF8')),'hex') IS DISTINCT FROM 'c6444520d67ab4871f0a921e1ea282a1a2d0e7c89b3eea2f3d4689480e3bd05b'
  OR encode(sha256(convert_to(ddl,'UTF8')),'hex') IS DISTINCT FROM 'bc3becae1537225b809fb90b824c5707bb5ec8f9ffb27264bfd5c8ea84d7156b'
  OR funcion.proacl IS DISTINCT FROM ARRAY[makeaclitem(propietario,propietario,'EXECUTE',false),
   makeaclitem('vec_contratacion_temporal_localizador_incorporacion'::regrole,propietario,'EXECUTE',false)]
  OR (SELECT count(*) FROM pg_proc WHERE pronamespace=funcion.pronamespace AND proname=funcion.proname)<>1
 THEN RAISE EXCEPTION 'CT85: preimagen/contrato/ACL no exactos' USING ERRCODE='55000'; END IF;
 antes:=to_jsonb(funcion)-'prosrc'; comentario:=obj_description(f,'pg_proc');
 SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
  INTO otras FROM pg_proc z WHERE z.pronamespace=funcion.pronamespace AND z.oid<>f;
 SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
  INTO deps FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
   OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
 SELECT jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
  INTO compartidas FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF (length(funcion.prosrc)-length(replace(funcion.prosrc,origen,'')))/length(origen)<>1
  OR (length(ddl)-length(replace(ddl,funcion.prosrc,'')))/length(funcion.prosrc)<>1 THEN
  RAISE EXCEPTION 'CT85: sustitución ambigua' USING ERRCODE='55000'; END IF;
 nuevo:=replace(funcion.prosrc,origen,destino);
 ddl:=replace(ddl,funcion.prosrc,nuevo);
 IF encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM '42351df1e5e324fa265427713c818417082b7fbe931b9cf9333fac200e4c781e'
  OR encode(sha256(convert_to(ddl,'UTF8')),'hex') IS DISTINCT FROM '21d8742e8ff9ceacd71aed55d95c96a322e50f64e1459cffcf8febd3c874c7a5' THEN
  RAISE EXCEPTION 'CT85: postimagen no exacta' USING ERRCODE='55000'; END IF;
 EXECUTE ddl;
 IF (SELECT to_jsonb(z)-'prosrc' FROM pg_proc z WHERE oid=f) IS DISTINCT FROM antes
  OR pg_get_functiondef(f) IS DISTINCT FROM ddl OR obj_description(f,'pg_proc') IS DISTINCT FROM comentario
  OR (SELECT prosrc FROM pg_proc WHERE oid=f) IS DISTINCT FROM nuevo
  OR (SELECT jsonb_agg(jsonb_build_object('funcion',to_jsonb(z),'comentario',obj_description(z.oid,'pg_proc')) ORDER BY z.oid)
      FROM pg_proc z WHERE z.pronamespace=funcion.pronamespace AND z.oid<>f) IS DISTINCT FROM otras
  OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
      FROM pg_depend d WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
       OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM deps
  OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype)
      FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
       AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM compartidas THEN
  RAISE EXCEPTION 'CT85: postimagen/metadata/dependencias modificadas' USING ERRCODE='55000'; END IF;
END $ct85$;
COMMIT;
