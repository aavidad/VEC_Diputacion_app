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
-- Orígenes adicionales: orden 2 con lectura histórica (consulta de motivos) y
-- orden 4 sin contratos (contrato divergente de su origen).
DO $origenes_extra$ DECLARE
 v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_c record;v_politica record;v_lectura record;v_fila record;v_i int;
 v_llamamiento text;v_recibo text;v_origen jsonb;v_ahora timestamptz:=clock_timestamp()-interval '1 second';
 v_conjunto jsonb;
BEGIN
 SELECT * INTO STRICT v_c FROM vec_bolsa_llamamientos.constitucion WHERE bolsa_ref=v_bolsa;
 SELECT * INTO STRICT v_politica FROM vec_bolsa_llamamientos.politica_orden_bolsa
  WHERE bolsa_ref=v_bolsa ORDER BY version DESC LIMIT 1;
 SELECT * INTO STRICT v_lectura
 FROM vec_bolsa_llamamientos.leer_origen_llamamiento_ct_contextual_v1(
  'categoria:rpt:administrativo','interinidad','vacante');
 FOREACH v_i IN ARRAY ARRAY[2,4] LOOP
  SELECT e.participacion_ref,e.orden,vc.candidato_ref INTO STRICT v_fila
   FROM vec_bolsa_llamamientos.constitucion_entrada e
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc
     ON vc.participacion_ref=e.participacion_ref
   WHERE e.instantanea_ref=v_c.instantanea_ref
     AND e.version_instantanea=v_c.version_instantanea AND e.orden=v_i;
  v_llamamiento:='llamamiento:'||encode(sha256(convert_to('b118-fixture:'||v_i,'UTF8')),'hex');
  v_recibo:='recibo:llamamiento:'||encode(sha256(convert_to('b118-fixture-recibo:'||v_i,'UTF8')),'hex');
  INSERT INTO vec_bolsa_llamamientos.llamamiento_emitido(
   llamamiento_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,
   participaciones,configuracion,huella_comando_sha256,huella_finalizacion,
   estado,emitido_en,decision_ref)
  VALUES(v_llamamiento,v_recibo,v_bolsa,'per_'||repeat('a',22),
   'b118fixturellamamiento'||v_i,jsonb_build_array(v_fila.participacion_ref),
   '{}'::jsonb,repeat('a',64),decode(repeat('b',64),'hex'),
   'emision_reservada',v_ahora,'decision:b118:llamamiento:'||v_i);
  v_origen:=jsonb_build_object('organizacion_ref','organizacion:desarrollo:dipgra',
   'unidad_ref','unidad:b118:rrhh','expediente_ref','expediente:b118:fixture:'||v_i,
   'clave_ct','clave:b118:'||v_i,'bolsa_ref',v_bolsa,
   'participacion_ref',v_fila.participacion_ref,'candidato_ref',v_fila.candidato_ref,
   'version_token','b118:'||v_i);
  IF v_i=2 THEN
   v_origen:=v_origen||jsonb_build_object('lectura',jsonb_build_object(
    'corte',v_lectura.corte,'criterios_ofrecimiento',v_lectura.criterios_ofrecimiento,
    'entradas',(SELECT jsonb_agg(x.valor||jsonb_build_object('elegible',
       x.valor->'orden_vigente'<>'null'::jsonb) ORDER BY x.n)
      FROM jsonb_array_elements(v_lectura.entradas) WITH ORDINALITY x(valor,n))));
  END IF;
  INSERT INTO vec_bolsa_llamamientos.origen_llamamiento_ct_v1(
   organizacion_ref,unidad_ref,expediente_ref,version_expediente,clave_ct,necesidad_ref,
   acta_ref,categoria_ref,bolsa_ref,version_bolsa,huella_bolsa_sha256,
   instantanea_ref,version_instantanea,huella_instantanea_sha256,
   politica_ref,version_politica,huella_politica_sha256,
   participacion_ref,candidato_ref,orden_seleccionado,version_token,
   llamamiento_ref,recibo_ref,decision_ref,auditoria_ref,evento_ref,emitido_en,
   origen_canonico,origen_huella_sha256)
  VALUES('organizacion:desarrollo:dipgra','unidad:b118:rrhh','expediente:b118:fixture:'||v_i,1,
   'clave:b118:'||v_i,'necesidad:b118:'||v_i,v_c.acta_ref,v_c.categoria_ref,
   v_bolsa,v_c.version_bolsa,v_c.huella_bolsa_sha256,
   v_c.instantanea_ref,v_c.version_instantanea,v_c.huella_instantanea_sha256,
   v_politica.politica_ref,v_politica.version,repeat('a',64),
   v_fila.participacion_ref,v_fila.candidato_ref,v_fila.orden,'b118:'||v_i,
   v_llamamiento,v_recibo,'decision:b118:origen:'||v_i,
   'auditoria:b118:origen:'||v_i,'evento:bolsa:ct:b118:'||v_i,v_ahora,
   v_origen,encode(sha256(convert_to(v_origen::text,'UTF8')),'hex'));
 END LOOP;
 v_conjunto:=vec_catalogos_configurables.comprobar_conjunto_gestion_rrhh_inscripcion_v1();
 PERFORM set_config('b118_prueba.ambito',v_conjunto->>'ambito_ref',true);
 PERFORM set_config('b118_prueba.llamamiento',
  'llamamiento:'||encode(sha256(convert_to('b118-fixture:2','UTF8')),'hex'),true);
END $origenes_extra$;

-- Un contrato que diverge de su origen queda en cuarentena sin atascar el
-- relevo; su repetición sigue en cuarentena y la persona queda pendiente.
DO $contrato_divergente$ DECLARE
 v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_o record;v_evento jsonb;v_resultado record;v_ahora timestamptz:=clock_timestamp()-interval '1 second';
 v_criterios jsonb;v_origen_ref constant text:='ref:b118:fixture:divergente';
BEGIN
 SELECT * INTO STRICT v_o FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
  WHERE expediente_ref='expediente:b118:fixture:4';
 SELECT po.politica->'criterios_ofrecimiento' INTO STRICT v_criterios
  FROM vec_bolsa_llamamientos.politica_ofertas_version po
  WHERE po.bolsa_ref=v_bolsa ORDER BY po.version DESC LIMIT 1;
 IF vec_bolsa_llamamientos.motivo_relacion_oferta_v1(v_o.participacion_ref,'disponible',false,
   'interinidad','vacante',clock_timestamp(),v_criterios) IS DISTINCT FROM 'libre' THEN
  RAISE EXCEPTION 'B118 prueba: la posición 4 debía estar libre antes del contrato'; END IF;
 v_evento:=jsonb_build_object(
  'categoria_ref','categoria:rpt:otra','causa_clave','necesidad_temporal',
  'esquema','vec.contratacion-temporal.contrato-bolsa.v1',
  'evento_ref','evento:ct:contrato-bolsa:'||encode(sha256(convert_to(
    'incorporacion'||chr(31)||v_origen_ref,'UTF8')),'hex'),
  'expediente_ref',v_o.expediente_ref,'fin_previsto','null'::jsonb,
  'inicio',to_char((v_ahora-interval '1 day') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'llamamiento_ref',v_o.llamamiento_ref,'modalidad_clave','acumulacion_tareas',
  'ocurrido_en',to_char(v_ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'organizacion_ref',v_o.organizacion_ref,'origen_ref',v_origen_ref,'tipo','incorporacion');
 SELECT * INTO STRICT v_resultado FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
  v_evento,encode(sha256(convert_to(v_evento::text,'UTF8')),'hex'),v_ahora,90);
 IF v_resultado.en_cuarentena IS NOT TRUE OR v_resultado.participacion_ref IS NOT NULL
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion_cuarentena q
   WHERE q.evento_ref=v_evento->>'evento_ref')
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contrato_participacion cp
   WHERE cp.evento_ref=v_evento->>'evento_ref' AND cp.participacion_ref IS NULL AND cp.bolsa_ref IS NULL)
 THEN RAISE EXCEPTION 'B118 prueba: contrato divergente sin cuarentena'; END IF;
 SELECT * INTO STRICT v_resultado FROM vec_bolsa_llamamientos.registrar_contrato_participacion_v1(
  v_evento,encode(sha256(convert_to(v_evento::text,'UTF8')),'hex'),v_ahora,90);
 IF v_resultado.en_cuarentena IS NOT TRUE OR v_resultado.reutilizado THEN
  RAISE EXCEPTION 'B118 prueba: la repetición salió de la cuarentena'; END IF;
 IF vec_bolsa_llamamientos.motivo_relacion_oferta_v1(v_o.participacion_ref,'disponible',false,
   'interinidad','vacante',clock_timestamp(),v_criterios) IS DISTINCT FROM 'pendiente_comprobacion_rrhh' THEN
  RAISE EXCEPTION 'B118 prueba: la cuarentena divergente no dejó pendiente a la persona'; END IF;
END $contrato_divergente$;

-- Consulta de motivos: la unidad y el ámbito de la orden deben ser los del
-- llamamiento y del conjunto de RRHH. Otra unidad u otro ámbito se deniegan
-- sin datos; la propia pasa la frontera y llega al acuse nominal de AD276.
RESET ROLE;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $exclusiones_unidad$ DECLARE
 v_bolsa constant text:='bolsa:administrativo:algicjlleggibhgagmjmoipmakdijibk';
 v_llamamiento text:=current_setting('b118_prueba.llamamiento');
 v_orden jsonb;v_caso jsonb;v_fallo text;v_estado text;v_resultado jsonb;
BEGIN
 v_orden:=jsonb_build_object('accion','bolsa.rrhh.exclusiones_llamamiento.consultar',
  'recurso_ref',v_llamamiento,'bolsa_ref',v_bolsa,
  'finalidad_ref','seguimiento_contacto_rrhh','resultado','obtenida',
  'motivo_ref','lectura_exclusiones_llamamiento_correcta',
  'correlacion_ref','correlacion:b118:prueba',
  'unidad_ref','unidad:b118:rrhh','ambito_ref',current_setting('b118_prueba.ambito'));
 FOR v_caso IN SELECT * FROM jsonb_array_elements(jsonb_build_array(
   jsonb_build_object('unidad_ref','unidad:b118:ajena'),
   jsonb_build_object('ambito_ref','ambito:b118:ajeno'),
   jsonb_build_object('unidad_ref',NULL))) LOOP
  v_fallo:='sin_error';v_resultado:=NULL;
  BEGIN
   v_resultado:=vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(
    v_bolsa,v_llamamiento,0,10,'\x00'::bytea,'\x00'::bytea,
    jsonb_strip_nulls(v_orden||v_caso));
  EXCEPTION WHEN OTHERS THEN
   GET STACKED DIAGNOSTICS v_fallo=MESSAGE_TEXT,v_estado=RETURNED_SQLSTATE;
  END;
  IF v_fallo IS DISTINCT FROM 'B118: exclusiones no autorizadas'
  OR v_estado IS DISTINCT FROM '42501' OR v_resultado IS NOT NULL THEN
   RAISE EXCEPTION 'B118 prueba: unidad o ámbito ajenos no denegados (%: %)',v_caso,v_fallo; END IF;
 END LOOP;
 v_fallo:='sin_error';
 BEGIN
  PERFORM vec_bolsa_llamamientos.consultar_exclusiones_ofrecimiento_rrhh_v1(
   v_bolsa,v_llamamiento,0,10,'\x00'::bytea,'\x00'::bytea,v_orden);
 EXCEPTION WHEN OTHERS THEN
  GET STACKED DIAGNOSTICS v_fallo=MESSAGE_TEXT,v_estado=RETURNED_SQLSTATE;
 END;
 IF v_fallo LIKE 'B118:%' OR v_fallo='sin_error' THEN
  RAISE EXCEPTION 'B118 prueba: la unidad propia no llegó al acuse nominal (%)',v_fallo; END IF;
 RAISE NOTICE 'B118 prueba: unidad propia llega al acuse nominal; rechazo del acuse sintético: % (%)',v_fallo,v_estado;
END $exclusiones_unidad$;
ROLLBACK;
\echo B118_PRUEBA_FIXTURE_OK
