\set ON_ERROR_STOP on
-- El script PG18 inserta la migración en la marca, dentro del mismo ROLLBACK.
-- Requiere al menos un snapshot externo aprobado ya existente en el clon.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE prueba_ctx17_canon_antes AS
SELECT s.provision_ref,s.version,b.representacion,b.manifiesto
FROM vec_contexto_actor_v1.contexto_externo_versiones s
CROSS JOIN LATERAL vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(s.snapshot,'2026-09-30T00:00:00Z'::timestamptz) b;
CREATE TEMP TABLE prueba_ctx17_copias_antes AS
SELECT p.oid,pg_catalog.to_jsonb(p) AS metadatos FROM pg_catalog.pg_proc p
WHERE p.oid IN(
 'vec_contexto_actor_v1.exigir_runtime_contexto_actor_v1()'::regprocedure,
 'vec_contexto_actor_v1.acreditar_uso_registro_contexto_interno_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'::regprocedure);
CREATE TEMP TABLE prueba_ctx17_datos_antes AS
SELECT to_jsonb(s) AS datos FROM vec_contexto_actor_v1.contexto_externo_versiones s;
CREATE TEMP TABLE prueba_ctx17_recibos_antes AS
SELECT to_jsonb(r) AS datos FROM vec_contexto_actor_v1.registros_contexto_externo_v2 r;
-- CTX17-APLICAR-AQUI
DO $conservacion$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_temp.prueba_ctx17_canon_antes)
    OR EXISTS(SELECT 1 FROM pg_temp.prueba_ctx17_canon_antes a
        JOIN vec_contexto_actor_v1.contexto_externo_versiones s USING(provision_ref,version)
        CROSS JOIN LATERAL vec_contexto_actor_v1.canon_snapshot_contexto_externo_v2(s.snapshot,'2026-09-30T00:00:00Z'::timestamptz) b
        WHERE a.representacion IS DISTINCT FROM b.representacion OR a.manifiesto IS DISTINCT FROM b.manifiesto)
    OR EXISTS(SELECT 1 FROM pg_temp.prueba_ctx17_copias_antes a JOIN pg_catalog.pg_proc p USING(oid)
        WHERE a.metadatos IS DISTINCT FROM pg_catalog.to_jsonb(p))
    OR EXISTS(SELECT 1 FROM (SELECT datos FROM pg_temp.prueba_ctx17_datos_antes EXCEPT ALL SELECT to_jsonb(s) FROM vec_contexto_actor_v1.contexto_externo_versiones s) cambio)
    OR EXISTS(SELECT 1 FROM (SELECT to_jsonb(s) FROM vec_contexto_actor_v1.contexto_externo_versiones s EXCEPT ALL SELECT datos FROM pg_temp.prueba_ctx17_datos_antes) cambio)
    OR EXISTS(SELECT 1 FROM (SELECT datos FROM pg_temp.prueba_ctx17_recibos_antes EXCEPT ALL SELECT to_jsonb(r) FROM vec_contexto_actor_v1.registros_contexto_externo_v2 r) cambio)
    OR EXISTS(SELECT 1 FROM (SELECT to_jsonb(r) FROM vec_contexto_actor_v1.registros_contexto_externo_v2 r EXCEPT ALL SELECT datos FROM pg_temp.prueba_ctx17_recibos_antes) cambio)
 THEN RAISE EXCEPTION 'CTX17: divergencia de canon/historia/recibos/circuito interno'; END IF;
END $conservacion$;
ROLLBACK;
\echo CTX17-CANON-HISTORIA-COPIAS-OK
