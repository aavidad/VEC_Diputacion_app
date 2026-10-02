\set ON_ERROR_STOP on
-- Ejecutar después de U17 en un clon desechable como DBA.
DO $prueba$
DECLARE
 temas pg_catalog.text[];
 base pg_catalog.jsonb;
 t pg_catalog.text;
 f pg_catalog.regprocedure:='vec_usuarios.valores_validos(jsonb)'::pg_catalog.regprocedure;
BEGIN
 SELECT pg_catalog.array_agg(x->>'codigo' ORDER BY ord)
 INTO temas
 FROM vec_usuarios.catalogo_preferencias c,
  pg_catalog.jsonb_array_elements(c.definicion->'temas') WITH ORDINALITY AS e(x,ord)
 WHERE c.version_ref='usuarios-preferencias-v2';
 IF temas IS DISTINCT FROM ARRAY[
  'sistema','claro','oscuro','diputacion_granada','arena','salvia','lavanda','azul_sereno','noche_suave']
  OR (SELECT pg_catalog.count(*) FROM vec_usuarios.catalogo_preferencias)<>2
  OR (SELECT pg_catalog.array_agg(version_ref ORDER BY secuencia) FROM vec_usuarios.catalogo_publicacion)
    IS DISTINCT FROM ARRAY['usuarios-preferencias-v1','usuarios-preferencias-v2']
  OR EXISTS(SELECT 1 FROM vec_usuarios.catalogo_preferencias c
     WHERE c.huella_sha256<>pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(c.definicion::pg_catalog.text,'UTF8')),'hex'))
 THEN RAISE EXCEPTION 'U17: catálogo/publicación/huella incorrectos'; END IF;
 SELECT definicion->'predeterminados' INTO STRICT base
 FROM vec_usuarios.catalogo_preferencias WHERE version_ref='usuarios-preferencias-v1';
 FOREACH t IN ARRAY temas LOOP
  IF vec_usuarios.valores_validos(pg_catalog.jsonb_set(base,'{tema}',pg_catalog.to_jsonb(t))) IS NOT TRUE
  THEN RAISE EXCEPTION 'U17: tema válido rechazado: %',t; END IF;
 END LOOP;
 IF vec_usuarios.valores_validos(pg_catalog.jsonb_set(base,'{tema}','"no_publicado"'::pg_catalog.jsonb)) IS TRUE
    OR vec_usuarios.valores_validos(base || '{"campo_extra":true}'::pg_catalog.jsonb) IS TRUE
 THEN RAISE EXCEPTION 'U17: valores ajenos aceptados'; END IF;
 IF (SELECT p.proowner<>pg_catalog.to_regrole('vec_usuarios_propietario')
       OR p.proacl IS DISTINCT FROM '{vec_usuarios_propietario=X/vec_usuarios_propietario}'::pg_catalog.aclitem[]
       OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::pg_catalog.text[]
       OR p.prosecdef OR p.provolatile<>'i' FROM pg_catalog.pg_proc p WHERE p.oid=f)
    OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
       CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) a
       WHERE p.oid=f AND a.grantee=0)
    OR pg_catalog.has_table_privilege('vec_usuarios_ejecutor_interno','vec_usuarios.catalogo_preferencias','SELECT')
    OR pg_catalog.has_table_privilege('vec_usuarios_ejecutor_externo','vec_usuarios.catalogo_preferencias','SELECT')
 THEN RAISE EXCEPTION 'U17: ACL o identidad de función alterada'; END IF;
 BEGIN
  UPDATE vec_usuarios.catalogo_preferencias SET definicion='{}'::pg_catalog.jsonb
  WHERE version_ref='usuarios-preferencias-v1';
  RAISE EXCEPTION 'U17: catálogo v1 mutable';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
END $prueba$;
