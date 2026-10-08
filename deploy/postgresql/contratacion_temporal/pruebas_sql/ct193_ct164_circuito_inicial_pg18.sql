\set ON_ERROR_STOP on
-- Prueba de la materialización inicial CT164 sobre una copia desechable.
-- No crea decisión atestada ni confirma un alta V3; todo se revierte.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;

DO $prueba$
DECLARE
 v_ref text := 'expediente:ct193-r5-ensayo';
 v_reserva text := 'reserva:ct193-r5-ensayo';
 v_numero text := '2026/CT193R5';
 v_flujo jsonb := '{"definicion_ref":"flujo:ct:rrhh:20261002","version":2,"huella_sha256":"1721c3a66576b21163b590602589f1627095bd6b6bfa37c79862af62775146e2"}'::jsonb;
 v_efecto jsonb;
 v_mal jsonb;
 v_raw bytea;
 v_agregado jsonb;
 v_esperado jsonb;
 v_rechazado boolean := false;
 v_fecha timestamptz := pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp());
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_alta WHERE expediente_ref=v_ref)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.identidad_reserva_alta WHERE reserva_ref=v_reserva)
 THEN RAISE EXCEPTION 'CT193: referencia de prueba ya ocupada'; END IF;

 SELECT pg_catalog.convert_from(alta_canonica,'UTF8')::jsonb INTO STRICT v_efecto
   FROM vec_contratacion_temporal.expediente_alta_version
  WHERE version=1 AND pg_catalog.convert_from(alta_canonica,'UTF8')::jsonb->>'esquema'
        ='vec.contratacion-temporal.efecto-alta.v2'
  ORDER BY expediente_ref LIMIT 1;
 v_efecto := v_efecto || pg_catalog.jsonb_build_object(
   'expediente_ref',v_ref,'reserva_ref',v_reserva,'numero_visible',v_numero,
   'recibo_ref','recibo:ct193-r5-ensayo','flujo',v_flujo);
 v_raw := vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_efecto);
 IF v_raw IS NULL THEN RAISE EXCEPTION 'CT193: material de prueba no reconstruible'; END IF;

 INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
   ambito_hmac,reserva_ref,expediente_ref,numero_visible,recibo_ref,
   huella_peticion_hmac,organizacion_ref,actor_ref,perfil_ref,creada_en)
 VALUES (
   'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||pg_catalog.repeat('e',64),
   v_reserva,v_ref,v_numero,'recibo:reserva-ct193-r5-ensayo',
   'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:'||pg_catalog.repeat('e',64),
   v_efecto->>'organizacion_ref',v_efecto->>'actor_ref',v_efecto->>'perfil_ref',v_fecha);
 INSERT INTO vec_contratacion_temporal.expediente_alta (
   expediente_ref,reserva_ref,numero_visible,organizacion_ref,actor_ref,
   perfil_ref,decision_ref,efecto_ref,huella_efecto_sha256,creada_en,
   confirmacion_ref)
 VALUES (v_ref,v_reserva,v_numero,v_efecto->>'organizacion_ref',
   v_efecto->>'actor_ref',v_efecto->>'perfil_ref',
   'decision:ct193-r5-ensayo','efecto:ct193-r5-ensayo',
   pg_catalog.encode(pg_catalog.sha256(v_raw),'hex'),v_fecha,
   'cnf_ct_'||pg_catalog.repeat('e',32));

 v_mal := pg_catalog.jsonb_set(v_efecto,'{flujo,huella_sha256}',
   pg_catalog.to_jsonb(pg_catalog.repeat('0',64)));
 BEGIN
   PERFORM vec_contratacion_temporal.materializar_version_inicial_v1(
     v_ref,1,vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_mal),
     'flujo:ct:rrhh:20261002',2,pg_catalog.repeat('0',64),
     v_efecto->>'fase_actual',v_efecto->>'estado_actual',v_fecha);
 EXCEPTION WHEN SQLSTATE '42501' THEN v_rechazado:=true;
 END;
 IF NOT v_rechazado THEN RAISE EXCEPTION 'CT193: flujo R5 divergente admitido'; END IF;

 PERFORM vec_contratacion_temporal.materializar_version_inicial_v1(
   v_ref,1,v_raw,'flujo:ct:rrhh:20261002',2,
   '1721c3a66576b21163b590602589f1627095bd6b6bfa37c79862af62775146e2',
   v_efecto->>'fase_actual',v_efecto->>'estado_actual',v_fecha);
 SELECT agregado_json INTO STRICT v_agregado
   FROM vec_contratacion_temporal.expediente_version_integral
  WHERE expediente_ref=v_ref AND version=1;
 v_esperado := pg_catalog.jsonb_build_object(
   'definicion',v_flujo,'estado_actual','solicitud','hitos','[]'::jsonb);
 IF v_agregado->'circuito' IS DISTINCT FROM v_esperado
    OR vec_contratacion_temporal.circuito_agregado_valido_ct164(v_agregado) IS NOT TRUE
    OR vec_contratacion_temporal.circuito_agregado_valido_ct164(
      pg_catalog.jsonb_set(v_agregado,'{circuito,estado_actual}','"credito"'::jsonb)) IS NOT FALSE
 THEN RAISE EXCEPTION 'CT193: circuito inicial CT164 divergente'; END IF;
END
$prueba$;
ROLLBACK;
