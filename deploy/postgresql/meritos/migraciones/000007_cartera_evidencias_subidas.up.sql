\set ON_ERROR_STOP on
-- RUM7: la cartera propia proyecta las referencias de evidencias ya guardadas
-- en la versión del hecho. No añade escritura ni altera la historia.
BEGIN;
SET LOCAL ROLE vec_meritos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000007',0));
DO $pre$
DECLARE f oid:='vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb)'::regprocedure;
 s text; marca text;
BEGIN
 IF current_user<>'vec_meritos_propietario' OR f IS NULL
 THEN RAISE EXCEPTION 'RUM7: preimagen ausente' USING ERRCODE='55000'; END IF;
 s:=pg_get_functiondef(f);
 IF encode(sha256(convert_to(s,'UTF8')),'hex') IS DISTINCT FROM
  'fc93ab0cb8527a368d4e953a9280952af39f20c92d8aeba3a4976645c168be4b' OR
  strpos(s,'registrar_lectura_cartera_meritos_v1')=0 OR
  strpos(s,'vec_meritos.persona_ref')=0
 THEN RAISE EXCEPTION 'RUM7: cuerpo instalado divergente' USING ERRCODE='55000'; END IF;
 marca:=$m$'retirada',CASE WHEN p.retirada_estado IS NULL THEN NULL ELSE
    jsonb_build_object('recibo_ref',p.recibo_ref,'fecha',to_char(p.retirada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   ORDER BY$m$;
 IF (length(s)-length(replace(s,marca,'')))/length(marca)<>1 OR
  strpos(s,$q$p_orden->'campos_permitidos' ? 'evidencias'$q$)<>0
 THEN RAISE EXCEPTION 'RUM7: proyección incompatible' USING ERRCODE='55000'; END IF;
 s:=replace(s,marca,$n$'retirada',CASE WHEN p.retirada_estado IS NULL THEN NULL ELSE
    jsonb_build_object('recibo_ref',p.recibo_ref,'fecha',to_char(p.retirada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   || CASE WHEN p_orden->'campos_permitidos' ? 'evidencias'
    THEN jsonb_build_object('evidencias',coalesce(nullif(p.hecho->'evidencias','null'::jsonb),'[]'::jsonb))
    ELSE '{}'::jsonb END ORDER BY$n$);
 EXECUTE s;
END $pre$;
COMMIT;
