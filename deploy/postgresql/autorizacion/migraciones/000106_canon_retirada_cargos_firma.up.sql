\set ON_ERROR_STOP on
-- AUT106 / F1-118: el censo Go serializa el instante cero de un rol publicado.
-- La publicación de cargos lo omite; el cotejo normaliza solo esa ausencia.
-- No modifica documentos, huellas, concesiones ni controles ya publicados.
BEGIN;
SET LOCAL ROLE vec_autorizacion_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion:migracion:000106',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(jsonb)');
BEGIN
 IF current_user<>'vec_autorizacion_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_is_in_recovery()
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_propietario'::regrole
   AND NOT p.prosecdef AND p.provolatile='i' AND p.proisstrict AND p.proparallel='s'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')=
       'cfd9ff89b51069fc0f54e5702803ee62f6d9043e11bc55cc477dd64ab9053cfd')
 THEN RAISE EXCEPTION 'AUT106: normalizador instalado incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(p_documento jsonb)
 RETURNS jsonb
 LANGUAGE sql
 IMMUTABLE PARALLEL SAFE STRICT
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
 SELECT CASE WHEN p_documento->>'estado'='publicada' AND NOT (p_documento ? 'retirada_en')
  THEN pg_catalog.jsonb_set(n.rol,'{retirada_en}','"0001-01-01T00:00:00Z"'::jsonb)
  ELSE n.rol END
 FROM (SELECT pg_catalog.jsonb_set(p_documento,'{concesiones}',
  (SELECT pg_catalog.jsonb_agg(
   CASE WHEN gc ? 'obligaciones' AND gc->'obligaciones' IN ('[]'::jsonb,'null'::jsonb)
    THEN gc-'obligaciones' ELSE gc END ORDER BY n)
   FROM pg_catalog.jsonb_array_elements(p_documento->'concesiones') WITH ORDINALITY AS x(g,n)
   CROSS JOIN LATERAL (SELECT CASE WHEN g ? 'campos_permitidos'
    AND g->'campos_permitidos' IN ('[]'::jsonb,'null'::jsonb)
    THEN g-'campos_permitidos' ELSE g END AS gc) normalizada)) AS rol) n
$function$;

DO $post$
DECLARE f oid:=to_regprocedure('vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(jsonb)');
 n jsonb; base jsonb:='{"estado":"publicada","concesiones":[{"accion":"prueba","campos_permitidos":[],"obligaciones":[]}]}'::jsonb;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_propietario'::regrole
   AND NOT p.prosecdef AND p.provolatile='i' AND p.proisstrict AND p.proparallel='s'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp']::text[]
   AND p.proacl=ARRAY['vec_autorizacion_propietario=X/vec_autorizacion_propietario']::aclitem[]
   AND encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')=
       '53b0c7cfc9b513ca54bebd3c6bcd4b7545f3609f1b32112c829a3e4cc39a2c5d')
 THEN RAISE EXCEPTION 'AUT106: metadatos o ACL divergentes' USING ERRCODE='55000'; END IF;
 n:=vec_autorizacion.normalizar_rol_catalogo_acciones_admin_v1(base);
 IF n IS DISTINCT FROM '{"estado":"publicada","retirada_en":"0001-01-01T00:00:00Z","concesiones":[{"accion":"prueba"}]}'::jsonb
 THEN RAISE EXCEPTION 'AUT106: normalización publicada incompatible' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
