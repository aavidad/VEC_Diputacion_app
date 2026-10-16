\set ON_ERROR_STOP on
-- F1-37 / CT226. Ejecutar como mantenedor de la copia, con -v ct_login=LOGIN_CT.
-- Sólo lecturas: los catálogos y sus recibos ya instalados no se modifican.
\if :{?ct_login}
\else
\echo 'CT226: falta ct_login nominal para la prueba'
\quit 1
\endif
DO $estructura$
DECLARE f regprocedure; esperado text; p record;
BEGIN
 FOR f,esperado IN SELECT v.f,v.h FROM (VALUES
  ('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)'::regprocedure,'021713a64d1c10265e402e211cd885d5d81f55dc5a21bc96d0475792e2e056bc'),
  ('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text)'::regprocedure,'0ffe98d71d237a5bd056e6a7e59c69ac16adb9db80144fad1ec566ffc9b0dc82')
 ) v(f,h) LOOP
  SELECT x.* INTO STRICT p FROM pg_proc x WHERE x.oid=f;
  IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex') IS DISTINCT FROM esperado
  OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR p.proacl IS DISTINCT FROM ARRAY[
   'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
   'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
  OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 's'
  OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
  THEN RAISE EXCEPTION 'CT226: estructura divergente %',f; END IF;
 END LOOP;
END $estructura$;
-- Material público de la provisión común y de la publicación corporativa.
-- Los GUC siguientes son datos del ensayo; ninguna función de producto los usa.
SELECT set_config('ct226_prueba.base_h',catalogo_huella_sha256,false),
 set_config('ct226_prueba.base_v',version::text,false),
 set_config('ct226_prueba.base_f',catalogo->>'fuente_ref',false),
 set_config('ct226_prueba.base_s',secuencia::text,false)
 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
SELECT set_config('ct226_prueba.actual_h',catalogo_huella_sha256,false),
 set_config('ct226_prueba.actual_v',version::text,false),
 set_config('ct226_prueba.actual_f',catalogo->>'fuente_ref',false)
 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
 WHERE estado='publicado' AND (organizacion_ref='organizacion:desarrollo:dipgra'
  OR secuencia=current_setting('ct226_prueba.base_s')::bigint)
 ORDER BY secuencia DESC LIMIT 1;
SET SESSION AUTHORIZATION :"ct_login";
BEGIN READ ONLY;
DO $ambitos$
DECLARE bh text:=current_setting('ct226_prueba.base_h');
 bv bigint:=current_setting('ct226_prueba.base_v')::bigint;
 bf text:=current_setting('ct226_prueba.base_f');
 ah text:=current_setting('ct226_prueba.actual_h');
 av bigint:=current_setting('ct226_prueba.actual_v')::bigint;
 af text:=current_setting('ct226_prueba.actual_f');
BEGIN
 IF vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:ct226:sin-publicacion',bh,bv,bf) IS DISTINCT FROM true
 OR vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',ah,av,af) IS DISTINCT FROM true
 OR vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(ah,av,af) IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'CT226: provisión común o publicación propia rechazadas'; END IF;
 IF bh=ah THEN RAISE EXCEPTION 'CT226: falta publicación posterior a bootstrap para contrastar organizaciones'; END IF;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:ct226:sin-publicacion',ah,av,af);
  RAISE EXCEPTION 'CT226: publicación ajena admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',bh,bv,bf);
  RAISE EXCEPTION 'CT226: publicación antigua admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(NULL,ah,av,af);
  RAISE EXCEPTION 'CT226: organización ausente admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
END $ambitos$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
