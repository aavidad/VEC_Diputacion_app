\set ON_ERROR_STOP on
-- Bolsa 000030: solicitudes y respuestas del portal del candidato. Datos
-- sintéticos dentro de una transacción que se revierte. El núcleo se ejerce
-- sin material V3 (funciones internas); el material real se prueba en Go.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE b text; p text; e record; cand text := 'can_portal_sintetico_0000000001'; otro text := 'can_portal_sintetico_0000000002';
 base timestamptz := clock_timestamp() + interval '1 hour'; r record; j jsonb; efectivos text[] := ARRAY['contactado'];
 sol text := 'solicitud-portal:'||repeat('a',64); sol2 text := 'solicitud-portal:'||repeat('b',64);
 lla text := 'llamamiento:'||repeat('c',64);
BEGIN
 SELECT c.bolsa_ref, en.participacion_ref, c.acta_ref, en.instantanea_ref, en.version_instantanea INTO STRICT e
   FROM vec_bolsa_llamamientos.constitucion c JOIN vec_bolsa_llamamientos.constitucion_entrada en USING (instantanea_ref, version_instantanea) LIMIT 1;
 b := e.bolsa_ref; p := e.participacion_ref;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.vinculo_candidato v WHERE v.participacion_ref = p) THEN
  INSERT INTO vec_bolsa_llamamientos.vinculo_candidato(participacion_ref, candidato_ref, acta_ref, instantanea_ref, version_instantanea, registrada_en)
  VALUES (p, cand, e.acta_ref, e.instantanea_ref, e.version_instantanea, base - interval '1 day');
 ELSE
  SELECT v.candidato_ref INTO STRICT cand FROM vec_bolsa_llamamientos.vinculo_candidato v WHERE v.participacion_ref = p;
 END IF;

 -- La participación se resuelve por candidato y bolsa; otro candidato no.
 IF vec_bolsa_llamamientos.participacion_portal_candidato_v1(cand, b) <> p THEN RAISE EXCEPTION 'B30: participación'; END IF;
 BEGIN PERFORM vec_bolsa_llamamientos.participacion_portal_candidato_v1(otro, b); RAISE EXCEPTION 'B30: participación ajena aceptada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;

 -- Pausa: la situación actual debe admitirla y el fin no pasa del máximo.
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(sol, 'recibo:'||sol, cand, b, p, 'pausa', base + interval '13 months',
    base + interval '12 months', ARRAY['disponible'], 'vec.bolsa.reglas:1:b18.pausa_voluntaria', 'clave-portal-1', base, 'decision:1');
  RAISE EXCEPTION 'B30: pausa por encima del máximo aceptada';
 EXCEPTION WHEN invalid_parameter_value THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(sol, 'recibo:'||sol, cand, b, p, 'pausa', base + interval '1 month',
    base + interval '12 months', ARRAY['no_disponible'], 'vec.bolsa.reglas:1:b18.pausa_voluntaria', 'clave-portal-1', base, 'decision:1');
  RAISE EXCEPTION 'B30: pausa desde situación no admitida';
 EXCEPTION WHEN SQLSTATE 'VBP03' THEN NULL; END;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(sol, 'recibo:'||sol, cand, b, p, 'pausa', base + interval '1 month',
    base + interval '12 months', ARRAY['disponible'], 'vec.bolsa.reglas:1:b18.pausa_voluntaria', 'clave-portal-1', base, 'decision:1');
 IF r.reutilizada OR r.solicitud_ref <> sol THEN RAISE EXCEPTION 'B30: solicitud'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(sol2, 'recibo:'||sol2, cand, b, p, 'reactivacion', NULL, NULL,
    ARRAY['disponible','no_disponible'], 'vec.bolsa.reglas:1:b29.portal_candidato', 'clave-portal-2', base + interval '1 second', 'decision:2');
  RAISE EXCEPTION 'B30: segunda solicitud pendiente aceptada';
 EXCEPTION WHEN SQLSTATE 'VBP02' THEN NULL; END;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(base + interval '1 minute') a WHERE a.tipo = 'solicitud_portal' AND a.referencia = sol) <> 1 THEN
  RAISE EXCEPTION 'B30: la solicitud no llega a la bandeja de RRHH'; END IF;

 -- RRHH valida con la operación B8 que cita la solicitud: deja de estar pendiente.
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion(participacion_ref, situacion, desde, motivo, actor, registrada_en, clave_idempotencia, recibo_ref)
 VALUES (p, 'no_disponible', base + interval '2 second', 'Pausa voluntaria pedida en el portal', 'persona:rrhh-b30', base + interval '2 second', 'b30:pausa', 'recibo:b30:pausa');
 INSERT INTO vec_bolsa_llamamientos.operacion_situacion_participacion(participacion_ref, desde, operacion, justificante_tipo, justificante_ref, justificante_sha256, actor, validador, validada_en, registrada_en, clave_idempotencia)
 VALUES (p, base + interval '2 second', 'pausar', 'solicitud_candidato', sol, repeat('d',64), 'persona:rrhh-b30', 'persona:rrhh-b30', base + interval '2 second', base + interval '2 second', 'b30:pausa');
 IF vec_bolsa_llamamientos.solicitud_portal_pendiente_v1(sol) THEN RAISE EXCEPTION 'B30: la validación de RRHH no cierra la solicitud'; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_solicitud_portal_interna_v1(sol2, 'recibo:'||sol2, cand, b, p, 'reactivacion', NULL, NULL,
    ARRAY['no_disponible','disponible_desde'], 'vec.bolsa.reglas:1:b29.portal_candidato', 'clave-portal-2', base + interval '3 second', 'decision:2');
 IF (SELECT situacion_previa FROM vec_bolsa_llamamientos.solicitud_portal_candidato WHERE solicitud_ref = sol2) <> 'no_disponible' THEN RAISE EXCEPTION 'B30: situación previa'; END IF;

 -- Respuesta: llamamiento emitido y contacto efectivo; el plazo lo trae el servidor.
 INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(llamamiento_ref, recibo_ref, bolsa_ref, actor_ref, clave_idempotencia, participaciones, configuracion,
   huella_comando_sha256, huella_finalizacion, estado, emitido_en, decision_ref)
 VALUES (lla, 'recibo:llamamiento:'||repeat('c',64), b, 'per_'||repeat('x',22), 'clave-llamamiento-b30', jsonb_build_array(p), '{}'::jsonb,
   repeat('e',64), decode(repeat('ab',32),'hex'), 'emision_reservada', base + interval '10 second', 'decision:llamamiento:b30');
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(p, base + interval '20 second', efectivos)) THEN
  RAISE EXCEPTION 'B30: llamamiento abierto sin contacto efectivo'; END IF;
 INSERT INTO vec_bolsa_llamamientos.contacto_participacion(contacto_ref, bolsa_ref, participacion_ref, llamamiento_ref, canal, instante, actor, resultado, anotacion, clave_idempotencia, recibo_ref)
 VALUES ('contacto:b30', b, p, NULL, 'telefono', base + interval '30 second', 'per_'||repeat('x',22), 'contactado', 'Contacto sintético B30.', 'clave-contacto-b30', 'recibo:contacto:b30');
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.llamamiento_abierto_portal_v1(p, base + interval '1 minute', efectivos);
 IF r.llamamiento_ref <> lla OR r.contacto_en <> base + interval '30 second' THEN RAISE EXCEPTION 'B30: llamamiento abierto'; END IF;
 -- La lectura exige la marca de consumo propio de este candidato en la
 -- transacción (aquí la anota el propietario, sin material).
 BEGIN PERFORM vec_bolsa_llamamientos.leer_portal_candidato_v1(cand, base + interval '1 minute', efectivos); RAISE EXCEPTION 'B30: lectura sin marca';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_v1('consulta', otro, 'mi-bolsa');
 BEGIN PERFORM vec_bolsa_llamamientos.leer_portal_candidato_v1(cand, base + interval '1 minute', efectivos); RAISE EXCEPTION 'B30: lectura con marca ajena';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 PERFORM vec_bolsa_llamamientos.anotar_consumo_candidato_v1('consulta', cand, 'mi-bolsa');
 j := vec_bolsa_llamamientos.leer_portal_candidato_v1(cand, base + interval '1 minute', efectivos);
 IF jsonb_array_length(j) <> 1 OR j->0->>'bolsa' <> b OR j->0->'llamamiento_abierto' IS NULL OR j->0->'solicitud_pendiente'->>'tipo' <> 'reactivacion' OR strpos(j::text, p) <> 0 THEN
  RAISE EXCEPTION 'B30: lectura del portal %', j; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1('respuesta-portal:'||repeat('1',64), 'recibo:respuesta-portal:'||repeat('1',64), cand, b, p, 'acepta',
    NULL, NULL, NULL, 'firme', base + interval '31 second', base + interval '1 day', efectivos, 'vec.bolsa.reglas:1:b05.plazo_respuesta', 'clave-respuesta-1', base + interval '1 minute', 'decision:3');
  RAISE EXCEPTION 'B30: contacto distinto aceptado';
 EXCEPTION WHEN SQLSTATE 'VBP04' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1('respuesta-portal:'||repeat('1',64), 'recibo:respuesta-portal:'||repeat('1',64), cand, b, p, 'acepta',
    NULL, NULL, NULL, 'firme', base + interval '30 second', base + interval '50 second', efectivos, 'vec.bolsa.reglas:1:b05.plazo_respuesta', 'clave-respuesta-1', base + interval '1 minute', 'decision:3');
  RAISE EXCEPTION 'B30: respuesta fuera de plazo aceptada';
 EXCEPTION WHEN SQLSTATE 'VBP05' THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1('respuesta-portal:'||repeat('1',64), 'recibo:respuesta-portal:'||repeat('1',64), cand, b, p, 'renuncia_justificada',
    NULL, NULL, NULL, 'firme', base + interval '30 second', base + interval '1 day', efectivos, 'vec.bolsa.reglas:1:b05.plazo_respuesta', 'clave-respuesta-1', base + interval '1 minute', 'decision:3');
  RAISE EXCEPTION 'B30: renuncia justificada sin causa ni justificante';
 EXCEPTION WHEN check_violation THEN NULL; END;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1('respuesta-portal:'||repeat('1',64), 'recibo:respuesta-portal:'||repeat('1',64), cand, b, p, 'renuncia_justificada',
    'enfermedad', 'justificante:b30', repeat('f',64), 'firme', base + interval '30 second', base + interval '1 day', efectivos, 'vec.bolsa.reglas:1:b05.plazo_respuesta', 'clave-respuesta-1', base + interval '1 minute', 'decision:3');
 IF r.reutilizada OR r.modo <> 'firme' THEN RAISE EXCEPTION 'B30: respuesta'; END IF;
 BEGIN
  PERFORM vec_bolsa_llamamientos.registrar_respuesta_portal_interna_v1('respuesta-portal:'||repeat('2',64), 'recibo:respuesta-portal:'||repeat('2',64), cand, b, p, 'acepta',
    NULL, NULL, NULL, 'firme', base + interval '30 second', base + interval '1 day', efectivos, 'vec.bolsa.reglas:1:b05.plazo_respuesta', 'clave-respuesta-2', base + interval '2 minute', 'decision:4');
  RAISE EXCEPTION 'B30: segunda respuesta al mismo llamamiento';
 EXCEPTION WHEN SQLSTATE 'VBP04' THEN NULL; END;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.consultar_avisos_portal_rrhh_v1(base + interval '3 minute') a WHERE a.tipo = 'respuesta_portal' AND a.detalle->>'causa' = 'enfermedad') <> 1 THEN
  RAISE EXCEPTION 'B30: la respuesta no llega a la bandeja de RRHH'; END IF;
 BEGIN
  UPDATE vec_bolsa_llamamientos.respuesta_portal_llamamiento SET modo = 'propuesta_rrhh';
  RAISE EXCEPTION 'B30: respuesta mutable';
 EXCEPTION WHEN others THEN IF SQLERRM = 'B30: respuesta mutable' THEN RAISE; END IF; END;
 -- Respuesta en dos pasos (como el servidor): preparar consume la decisión y
 -- deja la marca «responder»; con ella se lee el portal y responder usa esa
 -- misma decisión (aquí, el replay de la respuesta ya registrada) y la borra.
 PERFORM set_config('vec_bolsa_llamamientos.marca_consumo', '', true);
 PERFORM vec_bolsa_llamamientos.preparar_respuesta_portal_v1(cand, b,
   convert_to(jsonb_build_object('efecto_ref','mi-bolsa:'||cand,'operacion','bolsa.participaciones_propias.responder_llamamiento')::text,'UTF8'),
   convert_to(jsonb_build_object('recurso_ref','mi-bolsa:'||cand,'accion','bolsa.participaciones_propias.responder_llamamiento')::text,'UTF8'),
   NULL, convert_to(jsonb_build_object('vinculos', jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',cand)))::text,'UTF8'),
   1, 1, NULL, NULL, NULL, NULL);
 j := vec_bolsa_llamamientos.leer_portal_candidato_v1(cand, base + interval '3 minute', efectivos);
 IF j->0->'ultima_respuesta'->>'respuesta' <> 'renuncia_justificada' THEN RAISE EXCEPTION 'B30: lectura tras preparar %', j; END IF;
 SELECT * INTO STRICT r FROM vec_bolsa_llamamientos.responder_llamamiento_portal_v1('respuesta-portal:'||repeat('1',64), 'recibo:respuesta-portal:'||repeat('1',64),
   cand, b, 'renuncia_justificada', 'enfermedad', 'justificante:b30', repeat('f',64), 'firme', NULL, NULL, efectivos,
   'vec.bolsa.reglas:1:b05.plazo_respuesta', 'clave-respuesta-1', base + interval '3 minute',
   convert_to(jsonb_build_object('efecto_ref','mi-bolsa:'||cand,'operacion','bolsa.participaciones_propias.responder_llamamiento')::text,'UTF8'),
   convert_to(jsonb_build_object('recurso_ref','mi-bolsa:'||cand,'accion','bolsa.participaciones_propias.responder_llamamiento')::text,'UTF8'),
   NULL, convert_to(jsonb_build_object('vinculos', jsonb_build_array(jsonb_build_object('tipo','candidato','estado','activo','referencia',cand)))::text,'UTF8'),
   1, 1, NULL, NULL, NULL, NULL);
 IF NOT r.reutilizada OR r.recibo_ref <> 'recibo:respuesta-portal:'||repeat('1',64) THEN RAISE EXCEPTION 'B30: replay en dos pasos %', r; END IF;
 IF current_setting('vec_bolsa_llamamientos.marca_consumo', true) <> '' THEN RAISE EXCEPTION 'B30: la respuesta no borra la marca'; END IF;
 BEGIN PERFORM vec_bolsa_llamamientos.leer_portal_candidato_v1(cand, base + interval '3 minute', efectivos); RAISE EXCEPTION 'B30: marca reutilizable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $prueba$;

SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
DO $acl$
BEGIN
 BEGIN PERFORM 1 FROM vec_bolsa_llamamientos.solicitud_portal_candidato; RAISE EXCEPTION 'B30: lectura directa de tabla';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN PERFORM vec_bolsa_llamamientos.solicitud_portal_pendiente_v1('x'); RAISE EXCEPTION 'B30: función interna alcanzable';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_bolsa_llamamientos.solicitar_portal_candidato_v1('solicitud-portal:'||repeat('9',64), 'recibo:solicitud-portal:'||repeat('9',64),
    'can_portal_sintetico_0000000001', 'bolsa:x', 'pausa', clock_timestamp() + interval '1 month', clock_timestamp() + interval '1 year', ARRAY['disponible'],
    'regla', 'clave-sin-material', clock_timestamp(), convert_to('{}','UTF8'), convert_to('{}','UTF8'), NULL, convert_to('{}','UTF8'), 1, 1, NULL, NULL, NULL, NULL);
  RAISE EXCEPTION 'B30: solicitud sin material V3';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $acl$;
ROLLBACK;
