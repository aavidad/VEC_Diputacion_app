\set ON_ERROR_STOP on
-- Copia desechable tras CT221/B118. Fixture sintético directo, sin probar V3.
-- La transacción completa se revierte: FKs, RLS y disparadores siguen activos.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
DO $fixture$ DECLARE
 v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_c record;v_politica record;v_fila record;v_ahora timestamptz:=clock_timestamp()-interval '2 seconds';
 v_criterios jsonb;v_politica_ofertas jsonb;v_llamamiento text;v_recibo text;
 v_expediente text;v_origen jsonb;v_evento jsonb;v_origen_ref text;v_evento_ref text;
 v_resultado record;v_i int;
BEGIN
 SELECT * INTO STRICT v_c FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref=v_bolsa;
 SELECT * INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_orden_bolsa
  WHERE bolsa_ref=v_bolsa ORDER BY version DESC LIMIT 1;
 v_criterios:='{"fuente_normativa":"BOP Granada 9/2026, art. 9.4", "mejora": {
  "habilitada":true,"modalidades_destino":["interinidad"],
  "causas_destino":["vacante"],"maximo_vacante_meses":36,
  "reduccion_por_modalidades_previas":["acumulacion_tareas"],
  "condiciones_actuales":[{"modalidad":"acumulacion_tareas"}]}}'::jsonb;
 v_politica_ofertas:=jsonb_build_object('criterios_ofrecimiento',v_criterios);
 INSERT INTO vec_bolsa_llamamientos.politica_ofertas_version(
  bolsa_ref,version,politica,huella_sha256,ejemplo,actor_ref,clave_idempotencia,
  version_esperada,recibo_ref,publicada_en,decision_ref,auditoria_ref)
 VALUES(v_bolsa,1,v_politica_ofertas,
  encode(sha256(convert_to(v_politica_ofertas::text,'UTF8')),'hex'),true,
  'per_'||repeat('a',22),'b118fixturepolitica',0,
  'recibo:politica-ofertas:'||repeat('a',64),v_ahora,
  'decision:b118:politica','auditoria:b118:politica');
 -- La primera relación tiene inicio fiable. La tercera no lo tiene: ambas
 -- deben quedar pendientes; la segunda no puede adelantarse a la primera.
 FOREACH v_i IN ARRAY ARRAY[1,3] LOOP
  SELECT e.participacion_ref,e.orden,vc.candidato_ref INTO STRICT v_fila
   FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc
     ON vc.participacion_ref=e.participacion_ref
   WHERE e.instantanea_ref=v_c.instantanea_ref
     AND e.version_instantanea=v_c.version_instantanea AND e.orden=v_i;
  v_llamamiento:='llamamiento:'||encode(sha256(convert_to('b118-fixture:'||v_i,'UTF8')),'hex');
  v_recibo:='recibo:llamamiento:'||encode(sha256(convert_to('b118-fixture-recibo:'||v_i,'UTF8')),'hex');
  v_expediente:='expediente:b118:fixture:'||v_i;
  v_origen_ref:='ref:b118:fixture:'||v_i;
  v_evento_ref:='evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
    'incorporacion'||chr(31)||v_origen_ref,'UTF8')),'hex');
  INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
   llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,
   participaciones,configuracion,huella_comando_sha256,huella_finalizacion,
   estado,emitido_en,decision_ref)
  VALUES(v_llamamiento,v_recibo,v_bolsa,'per_'||repeat('a',22),
   'b118fixturellamamiento'||v_i,jsonb_build_array(v_fila.participacion_ref),
   '{}'::jsonb,repeat('a',64),decode(repeat('b',64),'hex'),
   'emision_reservada',v_ahora,'decision:b118:llamamiento:'||v_i);
  v_origen:=jsonb_build_object('organizacion_ref','organizacion:desarrollo:dipgra',
   'unidad_ref','unidad:b118:rrhh','expediente_ref',v_expediente,
   'clave_ct','clave:b118:'||v_i,'bolsa_ref',v_bolsa,
   'participacion_ref',v_fila.participacion_ref,'candidato_ref',v_fila.candidato_ref,
   'version_token','b118:'||v_i);
  INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
   organizacion_ref,unidad_ref,expediente_ref,version_expediente,clave_ct,necesidad_ref,
   acta_ref,categoria_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
   instantanea_ref,version_instantanea,huella_instantanea_sha256,
   politica_ref,version_politica,huella_politica_sha256,
   participacion_ref,candidato_ref,orden_seleccionado,version_token,
   llamamiento_ref,recibo_ref,decision_ref,auditoria_ref,evento_ref,emitido_en,
   origen_canonico,origen_huella_sha256)
  VALUES('organizacion:desarrollo:dipgra','unidad:b118:rrhh',v_expediente,1,
   'clave:b118:'||v_i,'necesidad:b118:'||v_i,v_c.acta_ref,v_c.categoria_ref,
   v_bolsa,v_c.version_bolsa,v_c.huella_bolsa_sha256,
   v_c.instantanea_ref,v_c.version_instantanea,v_c.huella_instantanea_sha256,
   v_politica.politica_ref,v_politica.version,repeat('a',64),
   v_fila.participacion_ref,v_fila.candidato_ref,v_fila.orden,'b118:'||v_i,
   v_llamamiento,v_recibo,'decision:b118:origen:'||v_i,
   'auditoria:b118:origen:'||v_i,'evento:bolsa:ct:b118:'||v_i,v_ahora,
   v_origen,encode(sha256(convert_to(v_origen::text,'UTF8')),'hex'));
  v_evento:=jsonb_build_object(
   'categoria_ref',v_c.categoria_ref,'causa_clave','necesidad_temporal',
   'esquema','vec.contratacion-temporal.contrato-bolsa.v1',
   'evento_ref',v_evento_ref,'expediente_ref',v_expediente,
   'fin_previsto',CASE WHEN v_i=1 THEN to_jsonb(to_char(
     (v_ahora+interval '6 months') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
     ELSE 'null'::jsonb END,
   'inicio',CASE WHEN v_i=1 THEN to_jsonb(to_char(
     (v_ahora-interval '1 day') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))
     ELSE 'null'::jsonb END,
   'llamamiento_ref',v_llamamiento,'modalidad_clave','acumulacion_tareas',
   'ocurrido_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'organizacion_ref','organizacion:desarrollo:dipgra',
   'origen_ref',v_origen_ref,'tipo','incorporacion');
  SELECT * INTO STRICT v_resultado FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
   v_evento,encode(sha256(convert_to(v_evento::text,'UTF8')),'hex'),v_ahora,v_i);
  IF v_resultado.participacion_ref IS DISTINCT FROM v_fila.participacion_ref
   OR v_resultado.en_cuarentena THEN
   RAISE EXCEPTION 'B118: B13 no enlazó el origen B110'; END IF;
  INSERT INTO vec_bolsa_llamamientos.situacion_participacion(
   participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,
   registrada_en,clave_idempotencia,recibo_ref)
  VALUES(v_fila.participacion_ref,'trabajando',v_ahora,NULL,NULL,
   'Relación ficticia del ensayo','sistema:b118',v_ahora,
   'b118:trabajando:'||v_i,'recibo:b118:trabajando:'||v_i);
 END LOOP;
END $fixture$;

DO $verificar$ DECLARE v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_primero record;v_segundo record;v_tercero record;v_lectura record;v_entrada jsonb;
BEGIN
 SELECT * INTO STRICT v_primero FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
  v_bolsa,'interinidad','vacante',statement_timestamp()) WHERE orden_acta=1;
 SELECT * INTO STRICT v_segundo FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
  v_bolsa,'interinidad','vacante',statement_timestamp()) WHERE orden_acta=2;
 SELECT * INTO STRICT v_tercero FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
  v_bolsa,'interinidad','vacante',statement_timestamp()) WHERE orden_acta=3;
 IF v_primero.razon IS DISTINCT FROM 'pendiente_comprobacion_rrhh'
 OR v_primero.orden_vigente IS NOT NULL
 OR jsonb_array_length(v_primero.eventos_relacion)<>1
 OR v_segundo.orden_vigente IS NOT NULL
 OR v_segundo.razon IS DISTINCT FROM 'pendiente_comprobacion_previa'
 OR v_tercero.razon IS DISTINCT FROM 'pendiente_comprobacion_rrhh'
 THEN RAISE EXCEPTION 'B118: incertidumbre adelantó otra posición'; END IF;
 SELECT * INTO STRICT v_lectura
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  'categoria:rpt:administrativo','interinidad','vacante');
 SELECT valor INTO STRICT v_entrada FROM jsonb_array_elements(v_lectura.entradas) j(valor)
  WHERE j.valor->>'orden_acta'='1';
 IF v_lectura.criterios_ofrecimiento->>'version_politica_ofertas' IS DISTINCT FROM '1'
 OR v_lectura.criterios_ofrecimiento->>'causa_ofrecida' IS DISTINCT FROM 'vacante'
 OR jsonb_array_length(v_entrada->'eventos_relacion')<>1
 THEN RAISE EXCEPTION 'B118: snapshot o eventos de origen incompletos'; END IF;
END $verificar$;

DO $cuarentena_tardia$ DECLARE
 v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_lectura record;v_cp record;v_part text;v_malo jsonb;v_resultado record;
 v_orden record;v_corte timestamptz;v_criterios jsonb;
BEGIN
 SELECT * INTO STRICT v_lectura
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  'categoria:rpt:administrativo','interinidad','vacante');
 v_corte:=v_lectura.corte;
 v_criterios:=v_lectura.criterios_ofrecimiento->'configuracion';
 SELECT cp.* INTO STRICT v_cp FROM vec_bolsa_llamamientos.contrato_participacion cp
  WHERE cp.expediente_ref='expediente:b118:fixture:1';
 SELECT o.participacion_ref INTO STRICT v_part
  FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1 o
  WHERE o.expediente_ref='expediente:b118:fixture:1';
 v_malo:=jsonb_set(v_cp.evento,'{causa_clave}','"otra_causa"'::jsonb);
 SELECT * INTO STRICT v_resultado FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
  v_malo,encode(sha256(convert_to(v_malo::text,'UTF8')),'hex'),
  v_cp.origen_creada_en,v_cp.origen_posicion);
 IF v_resultado.en_cuarentena IS NOT TRUE THEN
  RAISE EXCEPTION 'B118: divergencia tardía no quedó en cuarentena'; END IF;
 SELECT * INTO STRICT v_orden FROM vec_bolsa_llamamientos.leer_orden_contextual_oferta_v1(
  v_bolsa,'interinidad','vacante',v_corte) WHERE orden_acta=1;
 IF v_orden.razon IS DISTINCT FROM 'pendiente_comprobacion_rrhh'
 OR jsonb_array_length(v_orden.eventos_relacion)<>1
 OR vec_bolsa_llamamientos.motivo_relacion_oferta_v1(
   v_part,'trabajando',false,'interinidad','vacante',v_corte,v_criterios)
    IS DISTINCT FROM 'pendiente_comprobacion_rrhh' THEN
  RAISE EXCEPTION 'B118: la cuarentena tardía reescribió el corte histórico'; END IF;
END $cuarentena_tardia$;
ROLLBACK;
