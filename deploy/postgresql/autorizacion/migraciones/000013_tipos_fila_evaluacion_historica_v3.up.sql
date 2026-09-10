\set ON_ERROR_STOP on
-- Auth13: tipos fila implícitos sin ACL; no otorga acceso a sus tablas.
-- Sólo elimina la condición de esquema en la excepción del guard de tipos.
-- Avance desde Auth12 exacta; sin reaplicar ni DOWN sobre historia.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:registro_contexto_actor_v3:000005',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:lectura_evaluacion_historica_v3:000012',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:tipos_fila_evaluacion_historica_v3:000013',0));
SET LOCAL ROLE vec_autorizacion_propietario;

DO $auth13$
DECLARE
 propietario oid:='vec_autorizacion_propietario'::regrole;
 f oid:=to_regprocedure('vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)');
 funcion pg_proc%ROWTYPE; ddl text; nuevo text;
 antes jsonb; otras jsonb; deps jsonb; compartidas jsonb; comentario text;
 origen text:=$origen$            -- Sin ACL propia ni acceso a su esquema, no concede acceso a datos.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
              AND NOT pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
$origen$;
 destino text:=$destino$            -- Sin ACL propia, el tipo fila no concede acceso a datos;
            -- el guard anterior rechaza todo acceso a tablas y columnas.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
$destino$;
BEGIN
 IF current_user<>'vec_autorizacion_propietario' OR getdatabaseencoding()<>'UTF8'
  OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_autorizacion' AND nspowner=propietario)
  OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Auth13: propietario incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO STRICT funcion FROM pg_proc WHERE oid=f;
 ddl:=pg_get_functiondef(f);
 IF funcion.proowner<>propietario OR NOT funcion.prosecdef
  OR encode(sha256(convert_to(funcion.prosrc,'UTF8')),'hex') IS DISTINCT FROM '72932131b8bb7bc17cef6cc701870c68023d0f41428da07eca7a9cb89ccdc0ff'
  OR encode(sha256(convert_to(ddl,'UTF8')),'hex') IS DISTINCT FROM 'e08a6d91e7c56609ebdae37f9b1aae0618a2232aa4a4eeed7bcc1add9c5f4baa'
  OR funcion.proacl IS DISTINCT FROM ARRAY[makeaclitem(propietario,propietario,'EXECUTE',false),
   makeaclitem('vec_autorizacion_evaluacion_historica_lector'::regrole,propietario,'EXECUTE',false)]
  OR (SELECT count(*) FROM pg_proc WHERE pronamespace=funcion.pronamespace AND proname=funcion.proname)<>1
 THEN RAISE EXCEPTION 'Auth13: preimagen/contrato/ACL no exactos' USING ERRCODE='55000'; END IF;
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
  RAISE EXCEPTION 'Auth13: sustitución ambigua' USING ERRCODE='55000'; END IF;
 nuevo:=replace(funcion.prosrc,origen,destino);
 ddl:=replace(ddl,funcion.prosrc,nuevo);
 IF encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM 'c59d048bdc170e7e28a3c56362e8abcb1131ccd22e54fc9228f18610392d3785'
  OR encode(sha256(convert_to(ddl,'UTF8')),'hex') IS DISTINCT FROM 'e5dc23519d87547132658cbd74ddd0f70853b0cc62f535bece0f994b01ff68c8' THEN
  RAISE EXCEPTION 'Auth13: postimagen no exacta' USING ERRCODE='55000'; END IF;
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
  RAISE EXCEPTION 'Auth13: postimagen/metadata/dependencias modificadas' USING ERRCODE='55000'; END IF;
END $auth13$;
COMMIT;
