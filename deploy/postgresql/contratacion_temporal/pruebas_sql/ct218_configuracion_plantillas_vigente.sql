\set ON_ERROR_STOP on
-- CT218: ensayo sólo de lectura en el clon sintético, tras CT218 y V4.
-- Usa un LOGIN de ejecución existente. No fabrica actos, roles ni historia.
BEGIN READ ONLY;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
SET LOCAL lock_timeout='2s';
DO $guarda$
DECLARE b record; p record;
BEGIN
 IF pg_catalog.current_setting('vec.ensayo_clon',true) IS DISTINCT FROM 'si'
    OR NOT (SELECT rolsuper FROM pg_catalog.pg_roles WHERE rolname=current_user) THEN
  RAISE EXCEPTION 'CT218: prueba sólo en clon con lectura superusuario'; END IF;
 SELECT * INTO b FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1;
 SELECT * INTO p FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
  WHERE estado='publicado' ORDER BY secuencia DESC LIMIT 1;
 IF b.origen IS DISTINCT FROM 'bootstrap' OR b.organizacion_ref IS NOT NULL
    OR p.secuencia=b.secuencia OR p.origen IS DISTINCT FROM 'publicacion'
    OR p.version<>4 OR p.organizacion_ref IS NULL
    OR p.catalogo->>'fuente_ref' IS NULL THEN
  RAISE EXCEPTION 'CT218: falta provisión inicial y publicación V4 real'; END IF;
END $guarda$;

-- La instantánea de prueba contiene únicamente referencias y huellas públicas.
-- set_config(..., true) dura esta transacción y evita dar SELECT sobre historia
-- al LOGIN que probará la fachada.
WITH b AS (
 SELECT * FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1 ORDER BY secuencia LIMIT 1
), p AS (
 SELECT * FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1
 WHERE estado='publicado' ORDER BY secuencia DESC LIMIT 1
)
SELECT pg_catalog.set_config('vec.ct218.muestra',pg_catalog.jsonb_build_object(
 'huella',p.catalogo_huella_sha256,'version',p.version,'fuente',p.catalogo->>'fuente_ref',
 'base_huella',b.catalogo_huella_sha256,'base_version',b.version,
 'base_fuente',b.provision_fuente_ref)::text,true) FROM b,p;

-- Se elige un LOGIN auténtico del clon que hereda el grupo ejecutor y no
-- pertenece a los grupos propietarios o migradores prohibidos por CT131.
SELECT r.rolname AS ct218_login FROM pg_catalog.pg_roles r
 WHERE r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolbypassrls
   AND pg_catalog.pg_has_role(r.rolname,'vec_contratacion_temporal_ejecutor','MEMBER')
   AND NOT pg_catalog.pg_has_role(r.rolname,'vec_contratacion_temporal_propietario','MEMBER')
   AND NOT pg_catalog.pg_has_role(r.rolname,'vec_contratacion_temporal_migrador','MEMBER')
 ORDER BY r.rolname LIMIT 1 \gset
\if :{?ct218_login}
\else
\echo CT218: falta LOGIN ejecutor apto en el clon
\quit 3
\endif

SET SESSION AUTHORIZATION :'ct218_login';
DO $prueba$
DECLARE e jsonb; huella text; fuente text; version bigint; casos integer:=0;
BEGIN
 e:=pg_catalog.current_setting('vec.ct218.muestra')::jsonb;
 huella:=e->>'huella'; fuente:=e->>'fuente'; version:=(e->>'version')::bigint;
 IF vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(huella,version,fuente) IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'CT218: publicación V4 real rechazada'; END IF;
 casos:=casos+1;

 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(
   e->>'base_huella',(e->>'base_version')::bigint,e->>'base_fuente');
  RAISE EXCEPTION 'CT218: el fichero inicial suplantó la publicación V4';
 EXCEPTION WHEN SQLSTATE '55000' THEN casos:=casos+1;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(
   CASE WHEN huella=pg_catalog.repeat('0',64) THEN pg_catalog.repeat('1',64)
        ELSE pg_catalog.repeat('0',64) END,version,fuente);
  RAISE EXCEPTION 'CT218: huella ajena aceptada';
 EXCEPTION WHEN SQLSTATE '55000' THEN casos:=casos+1;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(huella,version-1,fuente);
  RAISE EXCEPTION 'CT218: versión anterior aceptada';
 EXCEPTION WHEN SQLSTATE '55000' THEN casos:=casos+1;
 END;
 BEGIN
  PERFORM vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(huella,version,'fuente:ct218:ajena');
  RAISE EXCEPTION 'CT218: fuente ajena aceptada';
 EXCEPTION WHEN SQLSTATE '55000' THEN casos:=casos+1;
 END;
 IF casos<>5 THEN RAISE EXCEPTION 'CT218: comprobaciones incompletas'; END IF;
 RAISE NOTICE 'CT218-PRUEBA-OK: % casos de solo lectura',casos;
END $prueba$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
