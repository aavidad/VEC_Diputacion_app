\set ON_ERROR_STOP on
-- U17: seis temas nuevos. Requiere U15 y U16 instaladas; conserva la v1,
-- las preferencias guardadas y la identidad/ACL de valores_validos(jsonb).
-- No hay DOWN: una preferencia v2 puede aparecer en historia inmutable.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('usuarios_vec:migracion:000017',0));
DO $pre$
DECLARE
 f pg_catalog.regprocedure:=pg_catalog.to_regprocedure('vec_usuarios.valores_validos(jsonb)');
 u16 pg_catalog.regprocedure:=pg_catalog.to_regprocedure('vec_usuarios_correos_externo.reservar_aviso_externo_v1(text)');
 p pg_catalog.pg_proc%ROWTYPE;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
    OR pg_catalog.current_setting('transaction_isolation')<>'read committed'
    OR f IS NULL OR u16 IS NULL
    OR pg_catalog.to_regclass('vec_usuarios.catalogo_preferencias') IS NULL
    OR pg_catalog.to_regclass('vec_usuarios.catalogo_publicacion') IS NULL THEN
  RAISE EXCEPTION 'U17: migrador o dependencias U15/U16 ausentes' USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT p FROM pg_catalog.pg_proc WHERE oid=f;
 IF p.proowner IS DISTINCT FROM pg_catalog.to_regrole('vec_usuarios_propietario')
    OR p.prosecdef OR p.provolatile<>'i'
    OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::pg_catalog.text[]
    OR p.proacl IS DISTINCT FROM '{vec_usuarios_propietario=X/vec_usuarios_propietario}'::pg_catalog.aclitem[]
    OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc,'UTF8')),'hex')
       <> 'f774be4478db9d2bfaccb7e554db7f37b22848733c3ba98644cd912cf6a92c90'
    OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p16 WHERE p16.oid=u16
       AND p16.proowner=pg_catalog.to_regrole('vec_usuarios_correos_externo_propietario')
       AND p16.prosecdef AND p16.provolatile='v'
       AND p16.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']::pg_catalog.text[]
       AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p16.prosrc,'UTF8')),'hex')
          ='eb1a20bda0ed7ded62ce80e1f39484d69faa42017484d270a8521377730c8ce8') THEN
  RAISE EXCEPTION 'U17: preimagen U15/U16 incompatible' USING ERRCODE='55000';
 END IF;
 IF (SELECT pg_catalog.count(*) FROM vec_usuarios.catalogo_preferencias)<>1
    OR (SELECT pg_catalog.count(*) FROM vec_usuarios.catalogo_publicacion)<>1
    OR NOT EXISTS(SELECT 1 FROM vec_usuarios.catalogo_preferencias c
      JOIN vec_usuarios.catalogo_publicacion b ON b.version_ref=c.version_ref
      WHERE b.secuencia=1 AND c.version_ref='usuarios-preferencias-v1'
        AND c.huella_sha256='adf70f782b223c29e6eb84b231015c1407b456fd38ab642910386f868aeb62d7'
        AND c.huella_sha256=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(c.definicion::pg_catalog.text,'UTF8')),'hex')
        AND c.definicion->'temas'='[{"codigo":"sistema","nombre_key":"ui.usuarios.preferencias.tema.sistema"},{"codigo":"claro","nombre_key":"ui.usuarios.preferencias.tema.claro"},{"codigo":"oscuro","nombre_key":"ui.usuarios.preferencias.tema.oscuro"}]'::pg_catalog.jsonb) THEN
  RAISE EXCEPTION 'U17: catálogo v1 o secuencia de publicación incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

SET LOCAL ROLE vec_usuarios_propietario;
CREATE OR REPLACE FUNCTION vec_usuarios.valores_validos(v jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RETURN jsonb_typeof(v)='object'
  AND ARRAY(SELECT jsonb_object_keys(v) ORDER BY 1) IS NOT DISTINCT FROM
   ARRAY['alto_contraste','aviso_correo_plazos','aviso_correo_tareas','filas','idioma','inicio','tamano_texto','tema']
  AND v->>'idioma' IN ('navegador','es','en')
  AND v->>'tamano_texto' IN ('normal','grande','muy_grande')
  AND jsonb_typeof(v->'alto_contraste')='boolean'
  AND v->>'tema' IN ('sistema','claro','oscuro','diputacion_granada','arena','salvia','lavanda','azul_sereno','noche_suave')
  AND v->>'inicio' IN ('cuadro','peticiones','bolsas')
  AND jsonb_typeof(v->'filas')='number' AND v->>'filas' IN ('20','50','100')
  AND jsonb_typeof(v->'aviso_correo_tareas')='boolean'
  AND jsonb_typeof(v->'aviso_correo_plazos')='boolean';
END $f$;

INSERT INTO vec_usuarios.catalogo_preferencias(version_ref,definicion,huella_sha256,publicado_en)
SELECT 'usuarios-preferencias-v2',d,
 pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(d::pg_catalog.text,'UTF8')),'hex'),
 pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp())
FROM (
 SELECT pg_catalog.jsonb_set(
  pg_catalog.jsonb_set(c.definicion,'{version_ref}','"usuarios-preferencias-v2"'::pg_catalog.jsonb),
  '{temas}', '[
   {"codigo":"sistema","nombre_key":"ui.usuarios.preferencias.tema.sistema"},
   {"codigo":"claro","nombre_key":"ui.usuarios.preferencias.tema.claro"},
   {"codigo":"oscuro","nombre_key":"ui.usuarios.preferencias.tema.oscuro"},
   {"codigo":"diputacion_granada","nombre_key":"ui.usuarios.preferencias.tema.diputacion_granada"},
   {"codigo":"arena","nombre_key":"ui.usuarios.preferencias.tema.arena"},
   {"codigo":"salvia","nombre_key":"ui.usuarios.preferencias.tema.salvia"},
   {"codigo":"lavanda","nombre_key":"ui.usuarios.preferencias.tema.lavanda"},
   {"codigo":"azul_sereno","nombre_key":"ui.usuarios.preferencias.tema.azul_sereno"},
   {"codigo":"noche_suave","nombre_key":"ui.usuarios.preferencias.tema.noche_suave"}
  ]'::pg_catalog.jsonb) AS d
 FROM vec_usuarios.catalogo_preferencias c WHERE c.version_ref='usuarios-preferencias-v1'
) q;
INSERT INTO vec_usuarios.catalogo_publicacion(secuencia,version_ref,publicada_en)
VALUES(2,'usuarios-preferencias-v2',pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp()));
COMMIT;
