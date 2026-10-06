\set ON_ERROR_STOP on
-- AD207: asientos sin bloqueo común, cola, sellado en orden, verificación y
-- defensas contra escritores antiguos. Ejecutar como superusuario migrador,
-- después de AD207. Todo termina en ROLLBACK; solo se consumen números de la
-- secuencia, que la cadena admite con huecos.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';SET LOCAL statement_timeout='120s';
DO $estructura$
BEGIN
 IF EXISTS(SELECT 1 FROM pg_proc p WHERE p.prosrc ~ 'vec_autorizacion_atestada_v3\.control_cadena_auditoria(_externa)?(\s+\w+)?\s+WHERE\s+(\w+\.)?control_id\s+FOR\s+UPDATE')
 THEN RAISE EXCEPTION 'AD207 prueba: queda un escritor que bloquea el control'; END IF;
 IF (SELECT count(*) FROM pg_proc p WHERE p.prosrc ~ 'reservar_asiento_auditoria(_externa)?_v5\(\) asiento_ad207')<21
 THEN RAISE EXCEPTION 'AD207 prueba: escritores sin reserva'; END IF;
 IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(integer)'::regprocedure)
  IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
   'vec_auditoria_encadenador=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 OR (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(boolean)'::regprocedure)
  IS DISTINCT FROM ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 OR (SELECT rolcanlogin OR rolinherit FROM pg_roles WHERE rolname='vec_auditoria_encadenador')
 THEN RAISE EXCEPTION 'AD207 prueba: permisos inesperados'; END IF;
END $estructura$;

-- La prueba hace de sellador: su latido está al día dentro de la transacción.
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp();
-- Tres asientos por un escritor reescrito: número nuevo, marcador y cola.
SELECT set_config('ad207.a'||g,vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','no_vencido',
 'correlacion_'||md5('ad207'||g||clock_timestamp()::text),
 jsonb_build_object('perfil_tecnico_ref','vec_auditoria_periodica_sellador','configuracion_sha256',repeat('a',64)))->>'secuencia',true)
FROM generate_series(1,3) g;
DO $asientos$
DECLARE v_corte numeric;
BEGIN
 SELECT secuencia INTO STRICT v_corte FROM vec_autorizacion_atestada_v3.control_cadena_auditoria;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
     JOIN vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 p USING (secuencia)
     WHERE a.secuencia IN (current_setting('ad207.a1')::numeric,current_setting('ad207.a2')::numeric,current_setting('ad207.a3')::numeric)
     AND a.secuencia>v_corte AND a.anterior_sha256=repeat('f',64))<>3
 THEN RAISE EXCEPTION 'AD207 prueba: asientos sin marcador o fuera de la cola'; END IF;
END $asientos$;

-- Un escritor antiguo no puede avanzar el control ni colar un asiento enlazado.
DO $antiguo$
BEGIN
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria SET actualizada_en=clock_timestamp();
  RAISE EXCEPTION 'AD207 prueba: control no congelado';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
 BEGIN
  INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
   evento_ref,evento_material_sha256,operador_login,accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref,periodica_detalle)
  SELECT 'aud_v3_per_'||md5('ad207-antiguo'),nextval('vec_autorizacion_atestada_v3.secuencia_auditoria_v5'),a.huella_sha256,encode(sha256('ad207'::bytea),'hex'),
   a.registrada_en,a.tipo_registro,'evento_'||md5('ad207-antiguo'),a.evento_material_sha256,a.operador_login,a.accion,a.modulo_id,a.recurso_ref,
   a.finalidad_ref,a.resultado,a.motivo_ref,a.proceso,a.canal,a.correlacion_ref,a.periodica_detalle
  FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.secuencia=current_setting('ad207.a1')::numeric;
  RAISE EXCEPTION 'AD207 prueba: asiento sin marcador admitido';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.sellar_cadena_auditoria_v5(10);
  RAISE EXCEPTION 'AD207 prueba: sellado fuera de READ COMMITTED';
 EXCEPTION WHEN invalid_transaction_state THEN NULL;
 END;
END $antiguo$;

-- Sellado en orden de número y verificación de extremo a extremo.
SELECT vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(false,50000) AS sellados \gset
DO $sellado$
DECLARE r record;v_ant text;v_pos numeric;n integer:=0;v jsonb;
BEGIN
 IF EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5) THEN
  RAISE EXCEPTION 'AD207 prueba: quedan pendientes tras sellar'; END IF;
 FOR r IN SELECT e.*,a.auditoria_ref,a.tipo_registro,a.huella_sha256,a.registrada_en FROM vec_autorizacion_atestada_v3.eslabon_auditoria_v5 e
  JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING (secuencia)
  WHERE e.secuencia IN (current_setting('ad207.a1')::numeric,current_setting('ad207.a2')::numeric,current_setting('ad207.a3')::numeric)
  ORDER BY e.posicion LOOP
  IF v_pos IS NOT NULL AND (r.posicion<>v_pos+1 OR r.anterior_sha256<>v_ant) THEN
   RAISE EXCEPTION 'AD207 prueba: los tres asientos no quedan seguidos y enlazados'; END IF;
  IF r.eslabon_sha256<>vec_autorizacion_atestada_v3.eslabon_auditoria_v5('interna',r.posicion,r.anterior_sha256,r.secuencia,r.auditoria_ref,r.tipo_registro,r.huella_sha256,r.registrada_en,r.sellado_en)
  THEN RAISE EXCEPTION 'AD207 prueba: eslabón distinto'; END IF;
  v_pos:=r.posicion;v_ant:=r.eslabon_sha256;n:=n+1;
 END LOOP;
 IF n<>3 THEN RAISE EXCEPTION 'AD207 prueba: sellados %',n; END IF;
 v:=vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(false);
 IF v->>'estado'<>'verificada' OR (v->>'pendientes')::int<>0 THEN RAISE EXCEPTION 'AD207 prueba: verificación %',v; END IF;
 BEGIN
  UPDATE vec_autorizacion_atestada_v3.eslabon_auditoria_v5 SET sellado_en=clock_timestamp() WHERE posicion=v_pos;
  RAISE EXCEPTION 'AD207 prueba: eslabón modificable';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN NULL;
 END;
END $sellado$;

-- La verificación completa tarda: el latido se renueva como haría el sellador.
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp();
-- Un asiento que sale de la cola sin sellar se detecta.
SELECT vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','no_vencido',
 'correlacion_'||md5('ad207-huerfano'||clock_timestamp()::text),
 jsonb_build_object('perfil_tecnico_ref','vec_auditoria_periodica_sellador','configuracion_sha256',repeat('a',64)))->>'secuencia' AS huerfano \gset
DELETE FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_v5 WHERE secuencia=:huerfano;
DO $huerfano$
DECLARE v jsonb:=vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(false);
BEGIN
 IF v->>'estado'<>'rechazada' OR (v->>'sin_sellar_fuera_de_cola')::int<1 THEN
  RAISE EXCEPTION 'AD207 prueba: asiento sin sellar no detectado %',v; END IF;
END $huerfano$;
-- La verificación completa tarda: el latido se renueva como haría el sellador.
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp();
-- Cadena externa: el escritor real necesita una decisión firmada de candidato.
-- Aquí los padres sintéticos entran sin sus claves ajenas (replica) y el
-- asiento por la reserva externa, con el disparador y el sellado reales.
SET LOCAL session_replication_role=replica;
INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3_externa
SELECT 'decision:ad207:'||g,encode(sha256(('d'||g)::bytea),'hex'),convert_to(repeat('d',256),'UTF8'),convert_to(repeat('m',32),'UTF8'),
 convert_to(repeat('c',64),'UTF8'),'\x00','\x00','\x00','\x00',convert_to(repeat('k',512),'UTF8'),encode(sha256(('k'||g)::bytea),'hex'),
 'efecto:ad207:'||g,encode(sha256(('e'||g)::bytea),'hex'),clock_timestamp() FROM generate_series(1,2) g;
INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3_externa
SELECT 'decision:ad207:'||g,encode(sha256(('d'||g)::bytea),'hex'),'nonce:ad207:'||g,'efecto:ad207:'||g,encode(sha256(('e'||g)::bytea),'hex'),
 encode(sha256(('c'||g)::bytea),'hex'),clock_timestamp() FROM generate_series(1,2) g;
SET LOCAL session_replication_role=origin;
DO $asientos_externos$
DECLARE r record;
BEGIN
 FOR g IN 1..2 LOOP
  SELECT * INTO STRICT r FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_externa_v5();
  INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3_externa
  VALUES ('aud_v3_ext_ad207_'||g,r.secuencia_previa+1,'decision:ad207:'||g,'efecto:ad207:'||g,encode(sha256(('e'||g)::bytea),'hex'),
   r.anterior_sha256,encode(sha256(('h'||g)::bytea),'hex'),clock_timestamp());
 END LOOP;
END $asientos_externos$;
DO $externa$
DECLARE v jsonb;n integer;
BEGIN
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.pendiente_sellado_auditoria_externa_v5)<2 THEN
  RAISE EXCEPTION 'AD207 prueba: la cadena externa no encola'; END IF;
 n:=vec_autorizacion_atestada_v3.sellar_tramo_auditoria_v5(true,50000);
 v:=vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(true);
 IF n<2 OR v->>'estado'<>'verificada' OR (v->>'sellados')::int<2 OR (v->>'pendientes')::int<>0 THEN
  RAISE EXCEPTION 'AD207 prueba: cadena externa %',v; END IF;
END $externa$;

-- Plazo máximo: con el latido caducado no se confirma ningún asiento.
UPDATE vec_autorizacion_atestada_v3.sellado_auditoria_v5 SET latido=clock_timestamp()-interval '1 hour';
DO $plazo$
BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_operacion_periodica_v1('capturar_sello_periodico_v1','permitido','no_vencido',
   'correlacion_'||md5('ad207-plazo'||clock_timestamp()::text),
   jsonb_build_object('perfil_tecnico_ref','vec_auditoria_periodica_sellador','configuracion_sha256',repeat('a',64)));
  RAISE EXCEPTION 'AD207 prueba: asiento admitido con el sellado detenido';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN
  IF SQLERRM<>'sellado VEC-AD-3 detenido: asiento rechazado' THEN RAISE; END IF;
 END;
END $plazo$;
SELECT 'AD207 prueba: correcta' AS resultado;
ROLLBACK;
