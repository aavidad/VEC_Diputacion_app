\set ON_ERROR_STOP on
SET SESSION AUTHORIZATION vec_o404e_tcb;
DO $limites$
BEGIN
  BEGIN
    PERFORM vec_o404e_concedida.preparar('cero',0);
    RAISE EXCEPTION 'C1=0 aceptado';
  EXCEPTION WHEN OTHERS THEN
    IF SQLERRM='C1=0 aceptado' THEN RAISE; END IF;
  END;
  BEGIN
    PERFORM vec_o404e_concedida.preparar('demasiados',513);
    RAISE EXCEPTION 'C1=513 aceptado';
  EXCEPTION WHEN OTHERS THEN
    IF SQLERRM='C1=513 aceptado' THEN RAISE; END IF;
  END;
END
$limites$;
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.preparar('uno',1) AS carga \gset
SELECT recibo_json AS recibo_uno FROM vec_contratacion_temporal
 .confirmar_operacion_decision_cobertura_o404e_v1(:'carga'::jsonb) \gset
SELECT vec_o404e_concedida.guardar_recibo(
  'uno',:'recibo_uno'::jsonb) AS recibo_guardado \gset
SELECT (:'recibo_uno'::jsonb->>'aplicada')::boolean AS aplicada \gset
\if :recibo_guardado
\else
  SELECT 1/0;
\endif
\if :aplicada
\else
  \warn 'golden concedido no aplicado'
  SELECT 1/0;
\endif
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT recibo_json AS recibo_replay FROM vec_contratacion_temporal
 .confirmar_operacion_decision_cobertura_o404e_v1(
   vec_o404e_concedida.carga('uno')) \gset
SELECT :'recibo_uno'::jsonb=:'recibo_replay'::jsonb AS replay_exacto \gset
\if :replay_exacto
\else
  \warn 'replay concedido no fue JSON exacto'
  SELECT 1/0;
\endif
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $mutacion_sellada$
BEGIN
  BEGIN
    PERFORM * FROM vec_contratacion_temporal
      .confirmar_operacion_decision_cobertura_o404e_v1(
        pg_catalog.jsonb_set(
          vec_o404e_concedida.carga('uno'),
          '{concesion,motivo_funcional,clave_i18n}',
          pg_catalog.to_jsonb('mutado'::text)));
    RAISE EXCEPTION 'replay con carga mutada aceptado';
  EXCEPTION WHEN OTHERS THEN
    IF SQLERRM='replay con carga mutada aceptado' THEN RAISE; END IF;
  END;
END
$mutacion_sellada$;
COMMIT;

BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
SELECT vec_o404e_concedida.preparar('maximo',512);
COMMIT;
RESET SESSION AUTHORIZATION;

DO $transiciones$
DECLARE
  c jsonb;
  a jsonb;
  s jsonb;
  p jsonb;
  d jsonb;
  u jsonb;
  x jsonb;
  r jsonb;
  y jsonb;
  q jsonb;
  motivo jsonb;
  efecto timestamptz(6);
  h text;
BEGIN
  SELECT carga->'concesion' INTO STRICT c
    FROM vec_o404e_concedida.cargas WHERE caso='uno';
  a:=c->'agregado_anterior';
  s:=c->'agregado_siguiente';
  p:=c->'propuesta';
  motivo:=c->'motivo_funcional';
  efecto:=(c->>'efecto_en')::timestamptz;
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,p,motivo,efecto) IS NOT TRUE THEN
    RAISE EXCEPTION 'control inicial O4-04E inválido';
  END IF;

  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       pg_catalog.jsonb_set(a,'{analisis,categoria_ref}',
         pg_catalog.to_jsonb('categoria:otra'::text)),
       pg_catalog.jsonb_set(s,'{analisis,categoria_ref}',
         pg_catalog.to_jsonb('categoria:otra'::text)),
       p,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'categoría análisis/propuesta desligada';
  END IF;

  d:=pg_catalog.jsonb_set(
    s#>'{decisiones_cobertura,0}','{preparacion_evidencias_ref}',
    pg_catalog.to_jsonb('preparacion:otra'::text));
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  x:=pg_catalog.jsonb_set(s,'{decisiones_cobertura,0}',d);
  x:=pg_catalog.jsonb_set(x,'{via_cobertura,decision_gobernada}',d);
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,x,p,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'preparación propuesta/decisión desligada';
  END IF;

  -- Construye una rectificación canónica válida sobre el resultado inicial.
  a:=s;
  u:=a#>'{decisiones_cobertura,0}';
  p:=pg_catalog.jsonb_set(p,'{version_expediente}','3'::jsonb);
  p:=pg_catalog.jsonb_set(p,'{via_propuesta}',
    pg_catalog.to_jsonb('via_recta'::text));
  p:=pg_catalog.jsonb_set(p,'{evaluaciones,0,via_clave}',
    pg_catalog.to_jsonb('via_recta'::text));
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(p)),'hex');
  p:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    p,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('propuesta-cobertura:sha256:'||h));
  efecto:=(u->>'decidida_en')::timestamptz+interval '1 microsecond';
  motivo:=pg_catalog.jsonb_build_object(
    'referencia_catalogo',pg_catalog.jsonb_build_object(
      'catalogo_id','motivos_v3','catalogo_version',1,
      'catalogo_huella_sha256',pg_catalog.repeat('9',64),
      'entrada_clave','motivo_44444444444444444444444444444444'),
    'clave_i18n','motivo_rectificacion');
  x:=pg_catalog.jsonb_build_object(
    'secuencia',4,'version_expediente',4,
    'accion_clave','contratacion_temporal.cobertura.rectificar',
    'actor_ref','actor_rectificador:o404e',
    'unidad_ref',u#>>'{actuacion,unidad_ref}',
    'realizada_en',
      vec_contratacion_temporal.texto_instante_utc_go_v2(efecto::text),
    'fase_origen',a->>'fase_actual','fase_destino',a->>'fase_actual',
    'estado_origen',a->>'estado_actual','estado_destino',a->>'estado_actual',
    'recibo_ref','recibo_rectificacion:o404e:uno');
  d:=u||pg_catalog.jsonb_build_object(
    'tipo','rectificacion','version_expediente_origen',3,
    'version_expediente',4,'actor_ref','actor_rectificador:o404e',
    'propuesta_ref',p->>'referencia',
    'propuesta_huella_sha256',p->>'huella_sha256',
    'preparacion_evidencias_ref',p->>'preparacion_evidencias_ref',
    'preparacion_evidencias_huella_sha256',
      p->>'preparacion_evidencias_huella_sha256',
    'analisis_ref',p->>'analisis_ref',
    'analisis_huella_sha256',p->>'analisis_huella_sha256',
    'catalogo',p->'catalogo','politica',p->'politica',
    'via_elegida','via_recta','via_recomendada','via_recta',
    'motivo',motivo,'predecesora_ref',u->>'referencia',
    'predecesora_huella_sha256',u->>'huella_sha256',
    'decidida_en',
      vec_contratacion_temporal.texto_instante_utc_go_v2(efecto::text),
    'actuacion',x);
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  s:=a||pg_catalog.jsonb_build_object(
    'version',4,'actualizado_en',
      vec_contratacion_temporal.texto_instante_utc_go_v2(efecto::text),
    'via_cobertura',pg_catalog.jsonb_build_object(
      'via_clave','via_recta','decision_gobernada',d),
    'decisiones_cobertura',
      a->'decisiones_cobertura'||pg_catalog.jsonb_build_array(d),
    'actuaciones',a->'actuaciones'||pg_catalog.jsonb_build_array(x));
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,p,motivo,efecto) IS NOT TRUE THEN
    RAISE EXCEPTION 'control rectificación O4-04E inválido';
  END IF;
  r:=d;
  y:=s;

  d:=pg_catalog.jsonb_set(r,'{predecesora_huella_sha256}',
    pg_catalog.to_jsonb(pg_catalog.repeat('8',64)));
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  s:=pg_catalog.jsonb_set(s,'{decisiones_cobertura,1}',d);
  s:=pg_catalog.jsonb_set(s,'{via_cobertura,decision_gobernada}',d);
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,p,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'predecesora obsoleta aceptada';
  END IF;

  d:=pg_catalog.jsonb_set(r,'{actor_ref}',u->'actor_ref');
  d:=pg_catalog.jsonb_set(
    d,'{actuacion,actor_ref}',u#>'{actuacion,actor_ref}');
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  s:=pg_catalog.jsonb_set(y,'{decisiones_cobertura,1}',d);
  s:=pg_catalog.jsonb_set(s,'{via_cobertura,decision_gobernada}',d);
  s:=pg_catalog.jsonb_set(s,'{actuaciones,3,actor_ref}',u->'actor_ref');
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,p,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'mismo actor rectificador aceptado';
  END IF;

  q:=pg_catalog.jsonb_set(p,'{via_propuesta}',
    pg_catalog.to_jsonb(u->>'via_elegida'));
  q:=pg_catalog.jsonb_set(q,'{evaluaciones,0,via_clave}',
    pg_catalog.to_jsonb(u->>'via_elegida'));
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_propuesta_cobertura_v1(q)),'hex');
  q:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    q,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('propuesta-cobertura:sha256:'||h));
  d:=r||pg_catalog.jsonb_build_object(
    'propuesta_ref',q->>'referencia',
    'propuesta_huella_sha256',q->>'huella_sha256',
    'via_elegida',u->>'via_elegida',
    'via_recomendada',u->>'via_elegida');
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  s:=pg_catalog.jsonb_set(y,'{decisiones_cobertura,1}',d);
  s:=pg_catalog.jsonb_set(s,'{via_cobertura,decision_gobernada}',d);
  s:=pg_catalog.jsonb_set(s,'{via_cobertura,via_clave}',
    pg_catalog.to_jsonb(u->>'via_elegida'));
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,q,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'rectificación sin cambio de vía aceptada';
  END IF;

  d:=pg_catalog.jsonb_set(r,'{decidida_en}',u->'decidida_en');
  d:=pg_catalog.jsonb_set(
    d,'{actuacion,realizada_en}',u->'decidida_en');
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  s:=pg_catalog.jsonb_set(y,'{decisiones_cobertura,1}',d);
  s:=pg_catalog.jsonb_set(s,'{via_cobertura,decision_gobernada}',d);
  s:=pg_catalog.jsonb_set(
    s,'{actuaciones,3,realizada_en}',u->'decidida_en');
  s:=pg_catalog.jsonb_set(s,'{actualizado_en}',u->'decidida_en');
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,p,motivo,(u->>'decidida_en')::timestamptz) IS TRUE THEN
    RAISE EXCEPTION 'instante rectificador no posterior aceptado';
  END IF;

  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       pg_catalog.jsonb_set(a,'{asignacion}',
         pg_catalog.jsonb_build_object('unidad_ref','unidad:ocupada')),
       pg_catalog.jsonb_set(y,'{asignacion}',
         pg_catalog.jsonb_build_object('unidad_ref','unidad:ocupada')),
       p,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'rectificación con asignación aceptada';
  END IF;

  d:=pg_catalog.jsonb_set(
    r,'{actuacion,fase_destino}',
    pg_catalog.to_jsonb('fase_avanzada'::text));
  h:=pg_catalog.encode(pg_catalog.sha256(
    vec_contratacion_temporal.o404e_material_decision_cobertura_v1(d)),'hex');
  d:=pg_catalog.jsonb_set(pg_catalog.jsonb_set(
    d,'{huella_sha256}',pg_catalog.to_jsonb(h)),
    '{referencia}',pg_catalog.to_jsonb('decision-cobertura:sha256:'||h));
  s:=pg_catalog.jsonb_set(y,'{decisiones_cobertura,1}',d);
  s:=pg_catalog.jsonb_set(s,'{via_cobertura,decision_gobernada}',d);
  s:=pg_catalog.jsonb_set(s,'{actuaciones,3,fase_destino}',
    pg_catalog.to_jsonb('fase_avanzada'::text));
  s:=pg_catalog.jsonb_set(s,'{fase_actual}',
    pg_catalog.to_jsonb('fase_avanzada'::text));
  IF vec_contratacion_temporal.o404e_transicion_dominio_ligada_v1(
       a,s,p,motivo,efecto) IS TRUE THEN
    RAISE EXCEPTION 'rectificación con avance de fase aceptada';
  END IF;
END
$transiciones$;
