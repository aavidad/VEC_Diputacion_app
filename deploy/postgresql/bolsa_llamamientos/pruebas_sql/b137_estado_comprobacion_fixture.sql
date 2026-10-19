\set ON_ERROR_STOP on
-- Tras B137 y AD293, en copia desechable. El operador aporta -v contexto='{...}'
-- con los nueve campos CT vigentes y una bolsa constituida con política publicada.
-- Esta prueba inserta actos ficticios como propietario y revierte todo; no simula V3.
\if :{?contexto}
\else
\echo 'Falta -v contexto con los nueve campos de CT vigente'
\quit 3
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
CREATE TEMP TABLE f137_fixture(contexto jsonb) ON COMMIT DROP;
INSERT INTO f137_fixture VALUES(:'contexto'::jsonb);
GRANT SELECT ON f137_fixture TO vec_bolsa_llamamientos_propietario;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $test$ DECLARE
 ctx jsonb;actual jsonb;obsoleto jsonb;antes record;despues record;po record;
 fila jsonb;estado jsonb;part text;canon jsonb;ref1 text;ref2 text;k text;
 instante timestamptz(6):=date_trunc('microseconds',clock_timestamp());
BEGIN
 SELECT f.contexto INTO STRICT ctx FROM pg_temp.f137_fixture f;
 IF vec_contratacion_temporal.cotejar_comprobacion_bolsa_v1(ctx,true) IS NOT TRUE
 THEN RAISE EXCEPTION 'fixture CT no vigente'; END IF;
 SELECT * INTO STRICT antes FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida');
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(antes.entradas) e WHERE e ? 'comprobacion_ofrecimiento')
 THEN RAISE EXCEPTION 'la llamada antigua publicó estado'; END IF;
 SELECT * INTO STRICT despues FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 IF despues.criterios_ofrecimiento IS NULL
 THEN RAISE EXCEPTION 'fixture sin criterios; usar b137_sin_criterios_fixture.sql'; END IF;
 IF (SELECT jsonb_agg(e.valor-'comprobacion_ofrecimiento' ORDER BY e.n)
     FROM jsonb_array_elements(despues.entradas) WITH ORDINALITY e(valor,n)) IS DISTINCT FROM antes.entradas
 THEN RAISE EXCEPTION 'cambió elegibilidad, razón, orden o eventos'; END IF;
 SELECT e.valor INTO STRICT fila FROM jsonb_array_elements(despues.entradas) e(valor)
 WHERE e.valor->'comprobacion_ofrecimiento'='{"version_esperada":0}'::jsonb
 ORDER BY (e.valor->>'orden_acta')::bigint LIMIT 1;
 part:=fila->>'participacion_ref';
 SELECT p.version,p.huella_sha256 INTO STRICT po FROM vec_bolsa_llamamientos.politica_ofertas_version p
 WHERE p.bolsa_ref=despues.bolsa_ref ORDER BY p.version DESC LIMIT 1;
 canon:=jsonb_build_object('ambito_ref',ctx->>'ambito_ref','bolsa_ref',despues.bolsa_ref,
  'categoria_ref',ctx->>'categoria_ref','causa_ofrecida',ctx->>'causa_ofrecida',
  'expediente_ref',ctx->>'expediente_ref','huella_politica_ofertas_sha256',po.huella_sha256,
  'modalidad_ofrecida',ctx->>'modalidad_ofrecida','necesidad_ref',ctx->>'necesidad_ref',
  'orden_acta',(fila->>'orden_acta')::bigint,'organizacion_ref',ctx->>'organizacion_ref',
  'participacion_ref',part,'unidad_ref',ctx->>'unidad_ref','version_bolsa',despues.version_bolsa,
  'version_expediente',(ctx->>'version_expediente')::bigint,
  'version_instantanea',despues.version_instantanea,'version_politica_ofertas',po.version);
 ref1:='comprobacion:'||encode(sha256(convert_to('b137:1:'||part,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1(
  comprobacion_ref,organizacion_ref,expediente_ref,necesidad_ref,bolsa_ref,participacion_ref,
  version,version_esperada,contexto,politica,hechos,resultado,material_sha256,actor_ref,
  perfil_ref,clave_idempotencia,recibo_ref,evento_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(ref1,ctx->>'organizacion_ref',ctx->>'expediente_ref',ctx->>'necesidad_ref',despues.bolsa_ref,part,
  1,0,canon,'{}','{}',jsonb_build_object('resultado','pendiente','motivo_clave','antecedentes_incompletos'),
  repeat('a',64),'per_fixture','prf_fixture','b137:fixture:1','recibo:'||ref1,'evento:'||ref1,
  'decision:b137:1','auditoria:b137:1',instante);
 SELECT * INTO STRICT despues FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 SELECT e.valor->'comprobacion_ofrecimiento' INTO STRICT estado
 FROM jsonb_array_elements(despues.entradas) e(valor) WHERE e.valor->>'participacion_ref'=part;
 IF estado->>'comprobacion_ref'<>ref1 OR estado->>'resultado'<>'pendiente'
 OR estado->>'version_esperada'<>'1' OR estado->>'vigente'<>'true'
 THEN RAISE EXCEPTION 'acto inicial omitido: %',estado; END IF;
 ref2:='comprobacion:'||encode(sha256(convert_to('b137:2:'||part,'UTF8')),'hex');
 obsoleto:=jsonb_set(canon,'{version_politica_ofertas}',to_jsonb(po.version+1));
 INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1(
  comprobacion_ref,organizacion_ref,expediente_ref,necesidad_ref,bolsa_ref,participacion_ref,
  version,version_esperada,contexto,politica,hechos,resultado,material_sha256,actor_ref,
  perfil_ref,clave_idempotencia,recibo_ref,evento_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(ref2,ctx->>'organizacion_ref',ctx->>'expediente_ref',ctx->>'necesidad_ref',despues.bolsa_ref,part,
  2,1,obsoleto,'{}','{}',jsonb_build_object('resultado','no_aplicable','motivo_clave','sin_supuesto_articulo_9'),
  repeat('b',64),'per_fixture','prf_fixture','b137:fixture:2','recibo:'||ref2,'evento:'||ref2,
  'decision:b137:2','auditoria:b137:2',instante);
 SELECT * INTO STRICT despues FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 SELECT e.valor->'comprobacion_ofrecimiento' INTO STRICT estado
 FROM jsonb_array_elements(despues.entradas) e(valor) WHERE e.valor->>'participacion_ref'=part;
 IF estado->>'comprobacion_ref'<>ref2 OR estado->>'resultado'<>'no_aplicable'
 OR estado->>'version_esperada'<>'2' OR estado->>'vigente'<>'false'
 OR estado->>'recibo_ref'<>'recibo:'||ref2
 OR (SELECT count(*) FROM vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1 a
     WHERE a.organizacion_ref=ctx->>'organizacion_ref' AND a.expediente_ref=ctx->>'expediente_ref'
       AND a.necesidad_ref=ctx->>'necesidad_ref' AND a.participacion_ref=part)<>2
 OR estado ? 'hechos' OR estado ? 'actor_ref' OR estado ? 'documento_ref'
 THEN RAISE EXCEPTION 'rectificación, historia o minimización incorrecta: %',estado; END IF;
 FOREACH k IN ARRAY ARRAY['expediente_ref','organizacion_ref','necesidad_ref','ambito_ref'] LOOP
  actual:=jsonb_set(ctx,ARRAY[k],to_jsonb(k||':ajeno'));
  BEGIN
   PERFORM 1 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
    ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',actual);
   RAISE EXCEPTION 'se admitió contexto ajeno: %',k;
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 END LOOP;
END $test$;
ROLLBACK;
