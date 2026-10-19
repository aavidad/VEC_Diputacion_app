\set ON_ERROR_STOP on
-- Tras B137. -v contexto='{...}' aporta nueve campos CT vigentes para bolsa
-- sin política de criterios publicada. No crea datos ni presume ausencia de acto.
\if :{?contexto}
\else
\echo 'Falta -v contexto con CT vigente y bolsa sin criterios'
\quit 3
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
CREATE TEMP TABLE f137_sin_criterios(contexto jsonb) ON COMMIT DROP;
INSERT INTO f137_sin_criterios VALUES(:'contexto'::jsonb);
GRANT SELECT ON f137_sin_criterios TO vec_bolsa_llamamientos_propietario;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $test$ DECLARE ctx jsonb;anterior record;actual record;
BEGIN
 SELECT f.contexto INTO STRICT ctx FROM pg_temp.f137_sin_criterios f;
 IF vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(ctx,true) IS NOT TRUE
 THEN RAISE EXCEPTION 'fixture CT no vigente'; END IF;
 SELECT * INTO STRICT anterior FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida');
 IF anterior.criterios_ofrecimiento IS NOT NULL
 THEN RAISE EXCEPTION 'fixture tiene criterios publicados'; END IF;
 SELECT * INTO STRICT actual FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 IF actual.criterios_ofrecimiento IS NOT NULL OR actual.entradas IS DISTINCT FROM anterior.entradas
 OR EXISTS(SELECT 1 FROM jsonb_array_elements(actual.entradas) e
  WHERE e ? 'comprobacion_ofrecimiento')
 THEN RAISE EXCEPTION 'B137 alteró orden legado o inventó estado sin criterios'; END IF;
END $test$;
ROLLBACK;
