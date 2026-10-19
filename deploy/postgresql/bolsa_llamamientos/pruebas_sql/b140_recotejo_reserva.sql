\set ON_ERROR_STOP on
-- F1-10: B140, AD297 y CT240 en una copia desechable con la lista instalada.
-- Se ejecuta como superusuario local. Prepara datos ficticios dentro de la
-- transacción (expediente CT, criterios publicados y una persona pendiente)
-- y lo revierte todo. No simula V3: el registro real del acto queda para el
-- recorrido; aquí se insertan actos como hace la prueba de B137.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='60s';
\ir b140_preparar_caso.sql
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;

-- 3. Cotejos.
DO $test$ DECLARE
 ctx jsonb;b record;sin_contexto record;origen jsonb;oferta jsonb;entradas jsonb;pendiente jsonb;
 seleccionado bigint;po record;canon jsonb;ref text;estado jsonb;capturada jsonb;z jsonb;
 control bigint;
BEGIN
 SELECT valor INTO STRICT ctx FROM pg_temp.b140_caso WHERE nombre='ctx';
 SELECT * INTO STRICT sin_contexto FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida');
 SELECT * INTO STRICT b FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 IF b.criterios_ofrecimiento IS NULL OR EXISTS(
  SELECT 1 FROM jsonb_array_elements(sin_contexto.entradas) x WHERE x ? 'comprobacion_ofrecimiento')
 THEN RAISE EXCEPTION 'B140: sin criterios o la lectura sin contexto publica estado'; END IF;
 -- Ajuste B137 (1): el estado solo acompaña a la persona pendiente.
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(b.entradas) x
  WHERE (x->>'razon'='pendiente_comprobacion_rrhh') IS DISTINCT FROM (x ? 'comprobacion_ofrecimiento'))
 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(b.entradas) x
  WHERE x->>'razon'='pendiente_comprobacion_rrhh')
 THEN RAISE EXCEPTION 'B140: estado fuera de la persona pendiente o sin pendiente'; END IF;
 IF (SELECT jsonb_agg(x.valor-'comprobacion_ofrecimiento' ORDER BY x.n)
     FROM jsonb_array_elements(b.entradas) WITH ORDINALITY x(valor,n)) IS DISTINCT FROM sin_contexto.entradas
 THEN RAISE EXCEPTION 'B140: la proyección cambió orden, razón o elegibilidad'; END IF;
 SELECT min((x->>'orden_acta')::bigint) INTO seleccionado
 FROM jsonb_array_elements(b.entradas) x WHERE x->>'orden_vigente'='1';
 SELECT x INTO pendiente FROM jsonb_array_elements(b.entradas) x
 WHERE x->>'razon'='pendiente_comprobacion_rrhh' ORDER BY (x->>'orden_acta')::bigint LIMIT 1;
 IF seleccionado IS NULL OR (pendiente->>'orden_acta')::bigint<=seleccionado
 OR pendiente->'comprobacion_ofrecimiento'<>'{"version_esperada":0}'::jsonb
 THEN RAISE EXCEPTION 'B140: caso sin elegible anterior a la pendiente: % %',seleccionado,pendiente; END IF;
 -- La lectura llega a la reserva tras el adaptador Go: candidato vacío en lugar de nulo.
 SELECT jsonb_agg(jsonb_set(x.valor,'{candidato_ref}',to_jsonb(coalesce(x.valor->>'candidato_ref','')))
  ORDER BY x.n) INTO entradas FROM jsonb_array_elements(b.entradas) WITH ORDINALITY x(valor,n);
 origen:=jsonb_build_object(
  'organizacion_ref',ctx->>'organizacion_ref','expediente_ref',ctx->>'expediente_ref',
  'necesidad_ref',ctx->>'necesidad_ref','unidad_ref',ctx->>'unidad_ref',
  'categoria_ref',ctx->>'categoria_ref','version_expediente',ctx->'version_expediente',
  'acta_ref',b.acta_ref,'bolsa_ref',b.bolsa_ref,'version_bolsa',b.version_bolsa,
  'version_instantanea',b.version_instantanea,'version_politica',b.version_politica,
  'criterios_ofrecimiento',b.criterios_ofrecimiento,'orden_seleccionado',seleccionado,
  'lectura',jsonb_build_object('entradas',entradas,'criterios_ofrecimiento',b.criterios_ofrecimiento));
 -- Reserva y reintento idéntico.
 IF vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(origen) IS NOT TRUE
 OR vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(origen) IS NOT TRUE
 THEN RAISE EXCEPTION 'B140: reserva o reintento idéntico denegados'; END IF;
 SELECT g.version INTO control FROM vec_bolsa_llamamientos.version_comprobacion_reserva_v1 g
 WHERE g.organizacion_ref=ctx->>'organizacion_ref' AND g.expediente_ref=ctx->>'expediente_ref'
   AND g.necesidad_ref=ctx->>'necesidad_ref' AND g.participacion_ref=pendiente->>'participacion_ref';
 IF control IS DISTINCT FROM 0 THEN RAISE EXCEPTION 'B140: control de versión ausente'; END IF;
 -- Oferta CT220: sin persona seleccionada; CT240 añade los criterios en raíz.
 oferta:=(origen-'orden_seleccionado'-'criterios_ofrecimiento')||jsonb_build_object(
  'criterios_ofrecimiento',origen#>'{lectura,criterios_ofrecimiento}');
 IF vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(oferta) IS NOT TRUE
 THEN RAISE EXCEPTION 'B140: oferta sin selección denegada'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(origen-'orden_seleccionado'-'criterios_ofrecimiento');
  RAISE EXCEPTION 'B140: oferta sin criterios en raíz admitida';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(
   jsonb_set(origen,'{orden_seleccionado}',to_jsonb(seleccionado::text)));
  RAISE EXCEPTION 'B140: orden textual admitido';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 -- Detención: elegir a la pendiente o a alguien posterior no se admite.
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(
   jsonb_set(origen,'{orden_seleccionado}',pendiente->'orden_acta'));
  RAISE EXCEPTION 'B140: selección a partir de la pendiente admitida';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 -- Contexto cambiado entre lectura y reserva.
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(
   jsonb_set(origen,'{version_politica}',to_jsonb(b.version_politica+1)));
  RAISE EXCEPTION 'B140: política distinta admitida';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(
   jsonb_set(origen,'{version_expediente}',to_jsonb((ctx->>'version_expediente')::bigint+1)));
  RAISE EXCEPTION 'B140: versión de expediente distinta admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 -- Acto registrado (v1) y vigente: la lectura nueva coteja aunque Go escriba la fecha con Z.
 SELECT version,huella_sha256 INTO STRICT po FROM vec_bolsa_llamamientos.politica_ofertas_version
 WHERE bolsa_ref=b.bolsa_ref ORDER BY version DESC LIMIT 1;
 canon:=jsonb_build_object('ambito_ref',ctx->>'ambito_ref','bolsa_ref',b.bolsa_ref,
  'categoria_ref',ctx->>'categoria_ref','causa_ofrecida',ctx->>'causa_ofrecida',
  'expediente_ref',ctx->>'expediente_ref','huella_politica_ofertas_sha256',po.huella_sha256,
  'modalidad_ofrecida',ctx->>'modalidad_ofrecida','necesidad_ref',ctx->>'necesidad_ref',
  'orden_acta',(pendiente->>'orden_acta')::bigint,'organizacion_ref',ctx->>'organizacion_ref',
  'participacion_ref',pendiente->>'participacion_ref','unidad_ref',ctx->>'unidad_ref',
  'version_bolsa',b.version_bolsa,'version_expediente',(ctx->>'version_expediente')::bigint,
  'version_instantanea',b.version_instantanea,'version_politica_ofertas',po.version);
 ref:='comprobacion:'||encode(sha256(convert_to('b140:1:'||(pendiente->>'participacion_ref'),'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1(
  comprobacion_ref,organizacion_ref,expediente_ref,necesidad_ref,bolsa_ref,participacion_ref,
  version,version_esperada,contexto,politica,hechos,resultado,material_sha256,actor_ref,
  perfil_ref,clave_idempotencia,recibo_ref,evento_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(ref,ctx->>'organizacion_ref',ctx->>'expediente_ref',ctx->>'necesidad_ref',
  b.bolsa_ref,pendiente->>'participacion_ref',1,0,canon,'{}','{}',
  '{"resultado":"pendiente","motivo_clave":"antecedentes_incompletos"}',repeat('a',64),
  'per_fixture','prf_fixture','b140:fixture:1','recibo:'||ref,'evento:'||ref,
  'decision:b140:1','auditoria:b140:1','2026-10-19 10:00:00.1234+00');
 UPDATE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 g SET version=1
 WHERE g.organizacion_ref=ctx->>'organizacion_ref' AND g.expediente_ref=ctx->>'expediente_ref'
   AND g.necesidad_ref=ctx->>'necesidad_ref' AND g.participacion_ref=pendiente->>'participacion_ref'
   AND g.version=0;
 IF NOT FOUND THEN RAISE EXCEPTION 'B140: control no actualizado'; END IF;
 -- La captura anterior ya no vale: el acto cambió después de la lectura.
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(origen);
  RAISE EXCEPTION 'B140: acto posterior a la lectura admitido';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 SELECT * INTO STRICT b FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 SELECT x->'comprobacion_ofrecimiento' INTO STRICT estado FROM jsonb_array_elements(b.entradas) x
 WHERE x->>'participacion_ref'=pendiente->>'participacion_ref';
 IF estado->>'version_esperada'<>'1' OR estado->>'vigente'<>'true'
 OR estado->>'registrada_en' !~ '\+00:00$'
 THEN RAISE EXCEPTION 'B140: acto vigente mal proyectado: %',estado; END IF;
 SELECT jsonb_agg(CASE WHEN x.valor ? 'comprobacion_ofrecimiento' THEN
   jsonb_set(x.valor,'{comprobacion_ofrecimiento,registrada_en}','"2026-10-19T10:00:00.1234Z"')
  ELSE x.valor END || jsonb_build_object('candidato_ref',coalesce(x.valor->>'candidato_ref',''))
  ORDER BY x.n) INTO entradas FROM jsonb_array_elements(b.entradas) WITH ORDINALITY x(valor,n);
 capturada:=jsonb_set(origen,'{lectura,entradas}',entradas);
 IF vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(capturada) IS NOT TRUE
 THEN RAISE EXCEPTION 'B140: la misma fecha con Z se tomó por cambio'; END IF;
 z:=jsonb_set(capturada,'{lectura,entradas}',(SELECT jsonb_agg(CASE WHEN x.valor ? 'comprobacion_ofrecimiento'
   THEN jsonb_set(x.valor,'{comprobacion_ofrecimiento,registrada_en}','"2026-10-19T10:00:00.1235Z"')
   ELSE x.valor END ORDER BY x.n) FROM jsonb_array_elements(entradas) WITH ORDINALITY x(valor,n)));
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(z);
  RAISE EXCEPTION 'B140: instante distinto admitido';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 -- Ajuste B137 (2): una rectificación guardada con otra bolsa no se oculta.
 ref:='comprobacion:'||encode(sha256(convert_to('b140:2:'||(pendiente->>'participacion_ref'),'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.comprobacion_ofrecimiento_v1(
  comprobacion_ref,organizacion_ref,expediente_ref,necesidad_ref,bolsa_ref,participacion_ref,
  version,version_esperada,contexto,politica,hechos,resultado,material_sha256,actor_ref,
  perfil_ref,clave_idempotencia,recibo_ref,evento_ref,decision_ref,auditoria_ref,registrada_en)
 VALUES(ref,ctx->>'organizacion_ref',ctx->>'expediente_ref',ctx->>'necesidad_ref',
  'bolsa:otra',pendiente->>'participacion_ref',2,1,jsonb_set(canon,'{bolsa_ref}','"bolsa:otra"'),
  '{}','{}','{"resultado":"no_aplicable","motivo_clave":"sin_supuesto_articulo_9"}',repeat('b',64),
  'per_fixture','prf_fixture','b140:fixture:2','recibo:'||ref,'evento:'||ref,
  'decision:b140:2','auditoria:b140:2',clock_timestamp());
 -- Rectificación confirmada sin pasar por el control: la reserva exige reintentar.
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(capturada);
  RAISE EXCEPTION 'B140: rectificación concurrente admitida';
 EXCEPTION WHEN serialization_failure THEN NULL; END;
 UPDATE vec_bolsa_llamamientos.version_comprobacion_reserva_v1 g SET version=2
 WHERE g.organizacion_ref=ctx->>'organizacion_ref' AND g.expediente_ref=ctx->>'expediente_ref'
   AND g.necesidad_ref=ctx->>'necesidad_ref' AND g.participacion_ref=pendiente->>'participacion_ref';
 SELECT * INTO STRICT b FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  ctx->>'categoria_ref',ctx->>'modalidad_ofrecida',ctx->>'causa_ofrecida',ctx);
 SELECT x->'comprobacion_ofrecimiento' INTO STRICT estado FROM jsonb_array_elements(b.entradas) x
 WHERE x->>'participacion_ref'=pendiente->>'participacion_ref';
 IF estado->>'version_esperada'<>'2' OR estado->>'vigente'<>'false'
 THEN RAISE EXCEPTION 'B140: último acto de otra bolsa oculto: %',estado; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(capturada);
  RAISE EXCEPTION 'B140: rectificación posterior a la lectura admitida';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL; END;
 INSERT INTO pg_temp.b140_caso VALUES('origen',origen);
END $test$;
RESET ROLE;

-- 4. Permisos: solo el propietario de CT ejecuta el cotejo; nadie más lee el control.
DO $permisos$ DECLARE origen jsonb; BEGIN
 SELECT valor INTO STRICT origen FROM pg_temp.b140_caso WHERE nombre='origen';
 IF has_function_privilege('vec_contratacion_temporal_propietario',
   'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)','EXECUTE')
 OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)','EXECUTE')
 OR has_function_privilege('public',
   'vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(jsonb)','EXECUTE')
 OR has_table_privilege('vec_contratacion_temporal_propietario',
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1','SELECT')
 OR has_table_privilege('vec_bolsa_llamamientos_ejecutor',
   'vec_bolsa_llamamientos.version_comprobacion_reserva_v1','SELECT')
 THEN RAISE EXCEPTION 'B140: concesiones inesperadas'; END IF;
 EXECUTE 'SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor';
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(origen);
  RAISE EXCEPTION 'B140: ejecutor de Bolsa admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 EXECUTE 'SET LOCAL ROLE vec_contratacion_temporal_ejecutor';
 BEGIN
  PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1(origen);
  RAISE EXCEPTION 'B140: ejecutor de CT admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 EXECUTE 'RESET ROLE';
END $permisos$;
SELECT 'b140_recotejo_reserva: 0 fallos' AS resultado;
ROLLBACK;

-- 5. Fuera de SERIALIZABLE el cotejo no se ejecuta.
BEGIN ISOLATION LEVEL READ COMMITTED;
DO $aislamiento$ BEGIN
 PERFORM vec_bolsa_llamamientos.cotejar_comprobacion_reserva_ct_v1('{}'::jsonb);
 RAISE EXCEPTION 'B140: cotejo fuera de SERIALIZABLE admitido';
EXCEPTION WHEN invalid_parameter_value THEN NULL; END $aislamiento$;
ROLLBACK;
