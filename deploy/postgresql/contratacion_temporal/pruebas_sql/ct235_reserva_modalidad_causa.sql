\set ON_ERROR_STOP on
-- F1-11: reserva de oferta con modalidad y causa sobre un expediente real.
-- Ejecutar como superusuario en una copia desechable de HZ37 con CT235.
-- Todo se revierte. Dentro de la transacción, y solo en ella: el expediente
-- fiscalizado elegido pasa a pedir dos personas (variante plural del caso 4.0)
-- y el consumo de autorización se sustituye por un doble, porque el material
-- firmado V3 se comprueba en el recorrido y no aquí.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.consumir_etapa_seleccion_oferta_ct220(
 p_etapa text,p_intencion jsonb,p_comando jsonb,p_oferta jsonb,p_material text,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,auditoria_ref text,consumo_huella_sha256 text)
LANGUAGE sql AS $doble$ SELECT 'decision:doble-f111','auditoria:doble-f111',repeat('d',64) $doble$;

DO $test$
DECLARE x record;i jsonb;c jsonb;r jsonb;clave uuid;otra uuid;n integer;
 legado jsonb;cl text;
BEGIN
 SELECT a.expediente_ref,a.version::bigint AS version,e.organizacion_ref,v.agregado_json AS g
 INTO x
 FROM vec_contratacion_temporal.expediente_integral_actual a
 JOIN vec_contratacion_temporal.expediente_alta e USING(expediente_ref)
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=a.expediente_ref AND v.version=a.version
 WHERE v.agregado_json->>'fase_actual'='fiscalizacion'
   AND v.agregado_json->>'estado_actual'='en_curso'
   AND v.agregado_json#>>'{via_cobertura,via_clave}'='bolsa_vigente'
   AND v.agregado_json#>>'{via_cobertura,decision_gobernada,via_elegida}'='bolsa_vigente'
   AND v.agregado_json#>>'{asignacion,unidad_ref}' IS NOT NULL
   AND v.agregado_json#>>'{analisis,modalidad_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'
   AND v.agregado_json#>>'{analisis,causa_clave}' ~ '^[a-z][a-z0-9._-]{1,79}$'
   AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.vinculo_emision_bolsa_ct_v1 y
     WHERE y.expediente_ref=a.expediente_ref)
   AND NOT EXISTS(SELECT 1 FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 y
     WHERE y.solicitud_json->>'expediente_ref'=a.expediente_ref)
 ORDER BY a.expediente_ref LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'F1-11: la copia no tiene un expediente fiscalizado apto'; END IF;

 -- Variante plural, solo en esta transacción.
 SET LOCAL session_replication_role=replica;
 UPDATE vec_contratacion_temporal.expediente_version_integral
 SET agregado_json=jsonb_set(agregado_json,'{solicitud,numero_personas}','2'::jsonb),
     agregado_json_huella_sha256=encode(sha256(convert_to(
      jsonb_set(agregado_json,'{solicitud,numero_personas}','2'::jsonb)::text,'UTF8')),'hex')
 WHERE expediente_ref=x.expediente_ref AND version=x.version;
 SET LOCAL session_replication_role=origin;

 clave:='22222222-2222-4222-8222-222222222222';
 i:=jsonb_build_object('organizacion_ref',x.organizacion_ref,'expediente_ref',x.expediente_ref,
  'version_expediente',x.version,'clave_idempotencia',clave::text,
  'actor_ref','per_f111pruebamodalidadcausa','perfil_ref','perfil:prueba:f111',
  'categoria_ref',x.g#>>'{analisis,categoria_ref}','centro_ref',x.g#>>'{solicitud,centro_ref}');
 i:=i||jsonb_build_object('huella_intencion',
  vec_contratacion_temporal.huella_intencion_seleccion_oferta_ct220(i));
 IF vec_contratacion_temporal.intencion_seleccion_oferta_valida_ct220(i,clave,i->>'huella_intencion') IS NOT TRUE THEN
  RAISE EXCEPTION 'F1-11: intención de prueba inválida'; END IF;
 cl:='ct-oferta:'||encode(sha256(convert_to(x.organizacion_ref||chr(31)||x.expediente_ref||chr(31)||
  x.version::text||chr(31)||clave::text,'UTF8')),'hex');
 c:=jsonb_build_object('organizacion_ref',x.organizacion_ref,'expediente_ref',x.expediente_ref,
  'version_expediente',x.version,'actor_ref',i->>'actor_ref','bolsa_ref','bolsa:prueba-f111',
  'necesidad_ref','necesidad:prueba-f111','numero_plazas',2,'numero_elegibles',2,
  'clave_bolsa',cl,'inicio_plazo','publicacion',
  'datos',jsonb_build_object('categoria','Categoría de prueba','centro','Centro de prueba',
   'fecha_inicio','2026-11-02','descripcion','Cobertura temporal de prueba'),
  'origen_ct',jsonb_build_object('organizacion_ref',x.organizacion_ref,
   'expediente_ref',x.expediente_ref,'version_expediente',x.version,'clave_ct',clave::text,
   'bolsa_ref','bolsa:prueba-f111','necesidad_ref','necesidad:prueba-f111',
   'categoria_ref',i->>'categoria_ref','unidad_ref',x.g#>>'{asignacion,unidad_ref}',
   'lectura',jsonb_build_object('corte','2026-10-19T08:00:00.000000Z','entradas','[]'::jsonb)));
 legado:=c;

 -- 1. Una reserva nueva sin el par se rechaza antes de escribir.
 BEGIN
  PERFORM vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,c,'material',
   NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'F1-11: reserva nueva sin modalidad admitida';
 EXCEPTION WHEN SQLSTATE '22023' THEN
  IF SQLERRM<>'CT235: la nueva reserva exige modalidad y causa' THEN RAISE; END IF;
 END;

 -- 2. Un par distinto del análisis vigente (CT221) se deniega.
 c:=jsonb_set(c,'{datos,modalidad_clave}',to_jsonb(x.g#>>'{analisis,modalidad_clave}'));
 c:=jsonb_set(c,'{datos,causa_clave}',to_jsonb('otra_causa_f111'::text));
 BEGIN
  PERFORM vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,c,'material',
   NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'F1-11: causa distinta del análisis admitida';
 EXCEPTION WHEN SQLSTATE '42501' THEN
  IF SQLERRM<>'CT235: modalidad o causa distinta del análisis vigente' THEN RAISE; END IF;
 END;
 SELECT count(*) INTO n FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE solicitud_json->>'expediente_ref'=x.expediente_ref;
 IF n<>0 THEN RAISE EXCEPTION 'F1-11: un rechazo dejó una reserva'; END IF;

 -- 3. Una operación anterior a CT235, sin el par, se recupera tal cual.
 BEGIN
  INSERT INTO vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6(
   clave_idempotencia,huella_semantica,solicitud_json,reserva_ref,tipo_seleccion,comando_oferta_json)
  VALUES(clave,i->>'huella_intencion',i,'reserva:seleccion-oferta:v1:'||repeat('a',64),'oferta',legado);
  r:=vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,legado,'material',
   NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  IF r->>'situacion' IS DISTINCT FROM 'ocupada' THEN
   RAISE EXCEPTION 'F1-11: la operación anterior no se recuperó: %',r; END IF;
  RAISE SQLSTATE 'P0F11';
 EXCEPTION WHEN SQLSTATE 'P0F11' THEN NULL;
 END;

 -- 4. El par del análisis reserva una sola vez y queda congelado en el comando.
 c:=jsonb_set(c,'{datos,causa_clave}',to_jsonb(x.g#>>'{analisis,causa_clave}'));
 r:=vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,c,'material',
  NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
 IF r->>'situacion' IS DISTINCT FROM 'reservada' OR r->'comando' IS DISTINCT FROM c THEN
  RAISE EXCEPTION 'F1-11: reserva con el par del análisis no creada: %',r; END IF;

 -- 5. El reintento con la misma clave no duplica; con otra causa, colisiona.
 r:=vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,c,'material',
  NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
 IF r->>'situacion' IS DISTINCT FROM 'ocupada' THEN
  RAISE EXCEPTION 'F1-11: reintento inesperado: %',r; END IF;
 r:=vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,
  jsonb_set(c,'{datos,causa_clave}',to_jsonb('otra_causa_f111'::text)),'material',
  NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
 IF r->>'situacion' IS DISTINCT FROM 'colision' THEN
  RAISE EXCEPTION 'F1-11: cambio de causa tras reservar admitido: %',r; END IF;
 SELECT count(*) INTO n FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
 WHERE solicitud_json->>'expediente_ref'=x.expediente_ref
   AND comando_oferta_json#>>'{datos,modalidad_clave}'=x.g#>>'{analisis,modalidad_clave}'
   AND comando_oferta_json#>>'{datos,causa_clave}'=x.g#>>'{analisis,causa_clave}';
 IF n<>1 THEN RAISE EXCEPTION 'F1-11: reservas esperadas 1, hay %',n; END IF;

 -- 6. Otra clave para el mismo expediente no abre una segunda oferta.
 otra:='33333333-3333-4333-8333-333333333333';
 i:=jsonb_set(i-'huella_intencion','{clave_idempotencia}',to_jsonb(otra::text));
 i:=i||jsonb_build_object('huella_intencion',
  vec_contratacion_temporal.huella_intencion_seleccion_oferta_ct220(i));
 c:=jsonb_set(jsonb_set(c,'{origen_ct,clave_ct}',to_jsonb(otra::text)),'{clave_bolsa}',
  to_jsonb('ct-oferta:'||encode(sha256(convert_to(x.organizacion_ref||chr(31)||x.expediente_ref||chr(31)||
  x.version::text||chr(31)||otra::text,'UTF8')),'hex')));
 BEGIN
  PERFORM vec_contratacion_temporal.reservar_seleccion_oferta_ct220(i,c,'material',
   NULL,NULL,NULL,NULL,1,1,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'F1-11: segunda oferta para el mismo expediente admitida';
 EXCEPTION WHEN unique_violation THEN NULL;
 END;

 -- 7. Permisos: el ayudante no es ejecutable fuera del propietario y PUBLIC no reserva.
 IF has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.comando_seleccion_oferta_valido_ct220(jsonb,jsonb)','EXECUTE')
  OR has_function_privilege('public',
   'vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
  OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',
   'vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
  RAISE EXCEPTION 'F1-11: permisos de la reserva distintos de lo esperado'; END IF;
 RAISE NOTICE 'F1-11: expediente % v% con %/%; siete comprobaciones superadas',
  x.expediente_ref,x.version,x.g#>>'{analisis,modalidad_clave}',x.g#>>'{analisis,causa_clave}';
END $test$;
ROLLBACK;
