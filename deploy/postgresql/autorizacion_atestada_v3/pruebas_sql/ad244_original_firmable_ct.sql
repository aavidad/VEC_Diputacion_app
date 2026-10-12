\set ON_ERROR_STOP on
-- PG18 sobre copia autorizada con AD69 + AD244 y publicación Documentos real.
-- El material se obtiene de las tablas gobernadas sin imprimirlo. No crea
-- claves, raíz, configuración, identidad, decisiones o capacidad ficticias.
-- El LOGIN técnico de contraste solo existe dentro de esta transacción.
BEGIN;
SET LOCAL search_path = pg_catalog, pg_temp;
DO $login$
BEGIN
 IF pg_catalog.to_regrole('vec_interno_preflight_v3_desarrollo') IS NULL THEN
  CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE
    NOREPLICATION NOBYPASSRLS INHERIT;
  GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo
    WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
 END IF;
END $login$;
WITH c AS (
 SELECT c.* FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
 JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c ON c.revision=p.configuracion_revision
 ORDER BY p.orden DESC LIMIT 1
), r AS (
 SELECT r.* FROM c JOIN vec_autorizacion_atestada_v3.configuracion_raiz cr ON cr.configuracion_revision=c.revision
 JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r ON (r.clave_id,r.version)=(cr.raiz_clave_id,cr.raiz_version)
), k AS (
 SELECT k.* FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
 JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k ON (k.clave_id,k.version)=(p.clave_id,p.version)
 WHERE k.audiencia_consumo='vec_documentos.operacion.v1' ORDER BY p.orden DESC LIMIT 1
)
SELECT jsonb_build_object(
 'configuracion',jsonb_build_object('revision',c.revision,'secuencia',c.secuencia,'huella_configuracion_sha256',c.huella_configuracion_sha256),
 'raiz',jsonb_build_object('clave_id',r.clave_id,'version',r.version,'huella_spki_sha256',r.huella_spki_sha256,'audiencia_despliegue',r.audiencia_despliegue,'suite',r.suite),
 'claves',jsonb_build_array(jsonb_build_object('audiencia_consumo',k.audiencia_consumo,'clave_id',k.clave_id,'version',k.version,'revision_gobierno',k.revision_gobierno,'huella_gobierno_sha256',k.huella_gobierno_sha256,'huella_secreto_sha256',k.huella_secreto_sha256,'emisor_id',k.emisor_id))
) AS material FROM c CROSS JOIN r CROSS JOIN k
\gset

SELECT pg_catalog.set_config('vec_ensayo_ad244.material', :'material', true) AS material_fijado
\gset
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
DO $prueba$
DECLARE
 m jsonb := pg_catalog.current_setting('vec_ensayo_ad244.material')::jsonb;
 resultado jsonb;
BEGIN
 IF NOT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct_firma_r5',m) THEN
  RAISE EXCEPTION 'AD244 clave=material_documentos esperado=true observado=false';
 END IF;
 resultado := vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct_firma_r5',m);
 IF resultado->>'revision' IS DISTINCT FROM m->'configuracion'->>'revision' THEN
  RAISE EXCEPTION 'AD244 clave=revision esperado=% observado=%',m->'configuracion'->>'revision',resultado->>'revision';
 END IF;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('consumidor_libre',m);
  RAISE EXCEPTION 'AD244 consumidor desconocido aceptado';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct_firma_r5',
    pg_catalog.jsonb_set(m,'{claves,0,audiencia_consumo}','"vec_personal.alta_ejercicio.v1"'::jsonb));
  RAISE EXCEPTION 'AD244 audiencia ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('ct_firma_r5',
    pg_catalog.jsonb_set(m,'{claves,0,huella_secreto_sha256}',pg_catalog.to_jsonb(pg_catalog.repeat('0',64))));
  RAISE EXCEPTION 'AD244 huella alterada aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct',m);
  RAISE EXCEPTION 'AD244 material Documentos aceptado como CT';
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
 RAISE NOTICE 'AD244 material publicado y rechazos nominales: OK';
END $prueba$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
