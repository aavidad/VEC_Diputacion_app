\set ON_ERROR_STOP on
-- Ejecutar como login miembro de vec_contratacion_temporal_ejecutor, sobre un
-- expediente ficticio existente, con -v organizacion_ref=... -v expediente_ref=...
\if :{?organizacion_ref}
\else
\quit 2
\endif
\if :{?expediente_ref}
\else
\quit 2
\endif
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='10s';
SELECT set_config('vec.ct242.prueba.organizacion_ref', :'organizacion_ref', true);
SELECT set_config('vec.ct242.prueba.expediente_ref', :'expediente_ref', true);
DO $prueba$
DECLARE org text:=current_setting('vec.ct242.prueba.organizacion_ref');
 exp text:=current_setting('vec.ct242.prueba.expediente_ref');
 estado jsonb; ajena jsonb; acreditada jsonb;
BEGIN
 estado:=vec_contratacion_temporal.consultar_cese_cierre_expediente_v1(org,exp);
 IF estado->>'expediente_ref' IS DISTINCT FROM exp
    OR (estado->>'version_expediente')::bigint NOT BETWEEN 1 AND 9007199254740991
    OR estado->>'fase_clave' IS NULL OR estado->>'estado_clave' IS NULL
    OR estado->>'numero_expediente' IS NULL
    OR estado->>'numero_expediente' ~ '^[0-9]{4}/CT-[0-9a-f]{12,}$'
    OR estado ? 'actor_ref' OR estado ? 'perfil_ref' THEN
  RAISE EXCEPTION 'CT242: lectura exacta o proyección inválida'; END IF;
 -- Con propuesta pendiente, la consulta de CT124 dice quién la hizo.
 acreditada:=vec_contratacion_temporal.consultar_incorporacion_acreditada_v1(org,exp);
 IF jsonb_typeof(acreditada->'no_incorporacion_propuesta')='object'
    AND coalesce(acreditada->'no_incorporacion_propuesta'->>'propuesta_actor_ref','')='' THEN
  RAISE EXCEPTION 'CT242: propuesta sin quién la hizo'; END IF;
 BEGIN
  ajena:=vec_contratacion_temporal.consultar_cese_cierre_expediente_v1('organizacion:ct242:ajena',exp);
  RAISE EXCEPTION 'CT242: otra organización obtuvo lectura: %',ajena;
 EXCEPTION WHEN insufficient_privilege THEN NULL;
 END;
END $prueba$;
ROLLBACK;
