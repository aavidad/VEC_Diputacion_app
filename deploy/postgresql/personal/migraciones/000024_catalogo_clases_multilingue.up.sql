\set ON_ERROR_STOP on
-- Amplía la validación de etiquetas sin reescribir la colección existente.
-- Las claves de idioma siguen un subconjunto acotado de BCP 47; ninguna
-- lista de idiomas queda compilada en la función de producción.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000024',0));
DO $funcion$
DECLARE f oid:=to_regprocedure('vec_personal.validar_version_clases_ocupacion_plan_ct()');
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean; volatilidad "char";
 preimagen_sha text:='b92b7aac4d7b92698a1167f053dd76216e057893f16be97ba17d9143a8130d06';
 marca text:=$marca$     OR ARRAY(SELECT jsonb_object_keys(opcion->'etiquetas') ORDER BY 1) IS DISTINCT FROM ARRAY['en','es']
     OR jsonb_typeof(opcion->'etiquetas'->'es') IS DISTINCT FROM 'string'
     OR jsonb_typeof(opcion->'etiquetas'->'en') IS DISTINCT FROM 'string'
     OR length(opcion->'etiquetas'->>'es') NOT BETWEEN 1 AND 80
     OR length(opcion->'etiquetas'->>'en') NOT BETWEEN 1 AND 80$marca$;
 sustitucion text:=$nueva$     OR opcion->'etiquetas'='{}'::jsonb
     OR EXISTS (SELECT 1 FROM jsonb_each(opcion->'etiquetas') AS e(idioma,etiqueta)
       WHERE e.idioma !~ '^[A-Za-z]{2,8}(-[A-Za-z0-9]{1,8}){0,7}$'
          OR jsonb_typeof(e.etiqueta) IS DISTINCT FROM 'string'
          OR length(e.etiqueta #>> '{}') NOT BETWEEN 1 AND 80)$nueva$;
BEGIN
 IF current_user<>'vec_personal_propietario' OR f IS NULL
    OR to_regclass('vec_personal.clases_ocupacion_plan_ct_catalogo') IS NULL THEN
  RAISE EXCEPTION 'Personal24: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef,p.provolatile
 INTO STRICT original,meta,acl,propietario,config,definidora,volatilidad FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_personal_propietario'::regrole OR definidora OR volatilidad<>'v'
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog']
    OR (SELECT encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM preimagen_sha
    OR length(original)-length(replace(original,marca,''))<>length(marca) THEN
  RAISE EXCEPTION 'Personal24: función anterior divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,sustitucion);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,sustitucion,marca) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT provolatile FROM pg_proc WHERE oid=f) IS DISTINCT FROM volatilidad
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
  RAISE EXCEPTION 'Personal24: función alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $funcion$;
COMMIT;
