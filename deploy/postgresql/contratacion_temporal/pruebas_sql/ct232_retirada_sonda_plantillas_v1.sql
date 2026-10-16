\set ON_ERROR_STOP on
-- CT232: lectura de la estructura final y de la publicación existente.
\if :{?ct_login}
\else
\echo 'CT232: falta ct_login nominal para la prueba'
\quit 1
\endif
DO $estructura$
DECLARE p record;
BEGIN
 IF to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'CT232: V1 sigue instalada'; END IF;
 SELECT x.* INTO p FROM pg_proc x
  WHERE x.oid=to_regprocedure('vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(text,text,bigint,text)');
 IF p.oid IS NULL OR encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')
      IS DISTINCT FROM '0ffe98d71d237a5bd056e6a7e59c69ac16adb9db80144fad1ec566ffc9b0dc82'
    OR p.proowner IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
    OR p.proacl IS DISTINCT FROM ARRAY[
      'vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario',
      'vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario']::aclitem[]
    OR p.prosecdef IS DISTINCT FROM true OR p.provolatile IS DISTINCT FROM 's'
    OR p.proconfig IS DISTINCT FROM ARRAY[
      'search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
 THEN RAISE EXCEPTION 'CT232: V2 divergente'; END IF;
END $estructura$;
SELECT set_config('ct232_prueba.base_h',catalogo_huella_sha256,false),
 set_config('ct232_prueba.base_v',version::text,false),
 set_config('ct232_prueba.base_f',catalogo->>'fuente_ref',false),
 set_config('ct232_prueba.base_s',secuencia::text,false)
 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
SELECT set_config('ct232_prueba.actual_h',catalogo_huella_sha256,false),
 set_config('ct232_prueba.actual_v',version::text,false),
 set_config('ct232_prueba.actual_f',catalogo->>'fuente_ref',false)
 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
 WHERE estado='publicado' AND (organizacion_ref='organizacion:desarrollo:dipgra'
  OR secuencia=current_setting('ct232_prueba.base_s')::bigint)
 ORDER BY secuencia DESC LIMIT 1;
SET SESSION AUTHORIZATION :"ct_login";
BEGIN READ ONLY;
DO $casos$
DECLARE bh text:=current_setting('ct232_prueba.base_h');
 bv bigint:=current_setting('ct232_prueba.base_v')::bigint;
 bf text:=current_setting('ct232_prueba.base_f');
 ah text:=current_setting('ct232_prueba.actual_h');
 av bigint:=current_setting('ct232_prueba.actual_v')::bigint;
 af text:=current_setting('ct232_prueba.actual_f');
BEGIN
 IF vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:ct232:sin-publicacion',bh,bv,bf) IS DISTINCT FROM true
    OR vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',ah,av,af) IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'CT232: provisión o publicación vigente rechazadas'; END IF;
 IF bh=ah OR av<2 THEN
  RAISE EXCEPTION 'CT232: falta publicación posterior para contrastar'; END IF;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:ct232:sin-publicacion',ah,av,af);
  RAISE EXCEPTION 'CT232: publicación ajena admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',bh,bv,bf);
  RAISE EXCEPTION 'CT232: publicación antigua admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',repeat('0',64),av,af);
  RAISE EXCEPTION 'CT232: huella ajena admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',ah,av-1,af);
  RAISE EXCEPTION 'CT232: versión anterior admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2('organizacion:desarrollo:dipgra',ah,av,'fuente:ct232:ajena');
  RAISE EXCEPTION 'CT232: fuente ajena admitida';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v2(NULL,ah,av,af);
  RAISE EXCEPTION 'CT232: organización ausente admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL;
 END;
 RAISE NOTICE 'CT232-PRUEBA-OK: V1 ausente, V2 intacta, dos positivos y seis rechazos';
END $casos$;
ROLLBACK;
RESET SESSION AUTHORIZATION;
