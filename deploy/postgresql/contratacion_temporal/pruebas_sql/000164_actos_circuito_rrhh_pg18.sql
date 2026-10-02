\set ON_ERROR_STOP on
-- Ensayo focal en clon desechable PG18: no sustituye la autorización V3.
-- Usa solo el materializador real y las guardas CT; no cambia fachadas/ACL.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';
DO $prueba$
DECLARE
 e jsonb; bytes bytea; v jsonb; previo jsonb; siguiente jsonb; sin_circuito jsonb;
 v_nuevo jsonb:='{"definicion_ref":"flujo:ct:rrhh:20261002","version":2,"huella_sha256":"dbc9eccf1d677b2ee0a1a3c8e3344417d356aee9ec89fe32f6069013a7855cff"}';
 v_total bigint; v_total_despues bigint; v_detectado boolean;
 v_fuente record; v_version integer;
BEGIN
 SELECT count(*) INTO v_total FROM vec_contratacion_temporal.expediente_version_integral;
 SELECT convert_from(alta_canonica,'UTF8')::jsonb INTO STRICT e
  FROM vec_contratacion_temporal.expediente_alta_version ORDER BY expediente_ref LIMIT 1;
 e:=e||jsonb_build_object('expediente_ref','expediente:ct164:maria-castro','reserva_ref','reserva:ct164:maria-castro',
  'numero_visible','2026/CT164','organizacion_ref','organizacion:ct164','actor_ref','persona:maria-castro',
  'perfil_ref','perfil:ct164:rrhh','recibo_ref','recibo:ct164:alta','flujo',v_nuevo,
  'fase_actual','solicitud','estado_actual','en_curso');
 e:=jsonb_set(e,'{actuacion,actor_ref}','"persona:maria-castro"');
 e:=jsonb_set(e,'{actuacion,recibo_ref}','"recibo:ct164:alta"');
 e:=jsonb_set(e,'{actuacion,fase_origen}','"solicitud"');
 e:=jsonb_set(e,'{actuacion,fase_destino}','"solicitud"');
 INSERT INTO vec_contratacion_temporal.identidad_reserva_alta
  VALUES('hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||repeat('e',64),
   'reserva:ct164:maria-castro','expediente:ct164:maria-castro','2026/CT164','recibo:ct164:alta',
   'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:'||repeat('d',64),
   'organizacion:ct164','persona:maria-castro','perfil:ct164:rrhh',timestamptz '2026-10-02T12:00:00Z');
 -- Confirmación FK diferida: este fixture siempre termina en ROLLBACK.
 INSERT INTO vec_contratacion_temporal.expediente_alta
  VALUES('expediente:ct164:maria-castro','reserva:ct164:maria-castro','2026/CT164',
   'organizacion:ct164','persona:maria-castro','perfil:ct164:rrhh','decision:ct164:alta',
   'efecto:ct164:alta',repeat('e',64),timestamptz '2026-10-02T12:00:00Z','cnf_ct_'||repeat('e',32));
 bytes:=vec_contratacion_temporal.reconstruir_efecto_alta_v2(e);
 IF bytes IS NULL THEN RAISE EXCEPTION 'fixture CT164: alta no canónica'; END IF;
 PERFORM vec_contratacion_temporal.materializar_version_inicial_v1(
  'expediente:ct164:maria-castro',1,bytes,v_nuevo->>'definicion_ref',2,v_nuevo->>'huella_sha256',
  'solicitud','en_curso',timestamptz '2026-10-02T12:00:00Z');
 SELECT agregado_json INTO STRICT v FROM vec_contratacion_temporal.expediente_version_integral
  WHERE expediente_ref='expediente:ct164:maria-castro' AND version=1;
 IF v->'circuito' IS DISTINCT FROM jsonb_build_object('definicion',v_nuevo,'estado_actual','solicitud','hitos','[]'::jsonb)
  OR vec_contratacion_temporal.circuito_agregado_valido_ct164(v) IS NOT TRUE
 THEN RAISE EXCEPTION 'CT164: alta sin circuito vacío ligado'; END IF;
 sin_circuito:=v-'circuito';
 IF vec_contratacion_temporal.circuito_agregado_valido_ct164(sin_circuito) IS NOT FALSE
 THEN RAISE EXCEPTION 'CT164: omisión de circuito admitida'; END IF;
 IF vec_contratacion_temporal.circuito_agregado_valido_ct164(jsonb_set(v,'{flujo,huella_sha256}',to_jsonb(repeat('f',64)))) IS NOT FALSE
 THEN RAISE EXCEPTION 'CT164: terna divergente admitida'; END IF;
 -- Ausencia de fuentes: no basta adjuntar circuito, referencias o huellas.
 FOR v_version IN 2..6 LOOP
  previo:=v||jsonb_build_object('version',v_version-1);
  siguiente:=v||jsonb_build_object('version',v_version);
  IF vec_contratacion_temporal.circuito_acto_admitido_ct164(previo,siguiente) IS NOT FALSE
  THEN RAISE EXCEPTION 'CT164: acto sin fuente admitido v%',v_version; END IF;
  v_detectado:=false;
  BEGIN
   PERFORM vec_contratacion_temporal.circuito_proyectar_acto_ct164(previo,siguiente,siguiente);
  EXCEPTION WHEN insufficient_privilege THEN v_detectado:=true; END;
  IF NOT v_detectado THEN RAISE EXCEPTION 'CT164: proyección no denegada v%',v_version; END IF;
 END LOOP;
 -- La tabla real tampoco admite v2 con circuito omitido o historia sin fuente.
 FOR v_version IN 0..1 LOOP
  v_detectado:=false;
  BEGIN
   INSERT INTO vec_contratacion_temporal.expediente_version_integral (
    expediente_ref,version,agregado_json,agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,
    flujo_ref,flujo_version,flujo_huella_sha256,fase_clave,estado,origen_version,operacion_ref,registrada_en)
   SELECT expediente_ref,2,
    CASE WHEN v_version=0 THEN (agregado_json-'circuito')||jsonb_build_object('version',2)
         ELSE agregado_json||jsonb_build_object('version',2) END,
    agregado_json_huella_sha256,prueba_canonica,prueba_huella_sha256,flujo_ref,flujo_version,
    flujo_huella_sha256,fase_clave,estado,'analisis_o3',operacion_ref,registrada_en
   FROM vec_contratacion_temporal.expediente_version_integral
    WHERE expediente_ref='expediente:ct164:maria-castro' AND version=1;
  EXCEPTION WHEN insufficient_privilege THEN v_detectado:=true; END;
  IF NOT v_detectado THEN RAISE EXCEPTION 'CT164: INSERT v2 sin fuente no denegado'; END IF;
 END LOOP;
 -- El camino antiguo conserva identidad JSON y la semántica del validador.
 FOR v_fuente IN SELECT agregado_json FROM vec_contratacion_temporal.expediente_version_integral
  WHERE expediente_ref<>'expediente:ct164:maria-castro' LOOP
  IF vec_contratacion_temporal.circuito_agregado_valido_ct164(v_fuente.agregado_json) IS NOT TRUE
   OR vec_contratacion_temporal.circuito_proyectar_acto_ct164(v_fuente.agregado_json,v_fuente.agregado_json,v_fuente.agregado_json)
      IS DISTINCT FROM v_fuente.agregado_json
  THEN RAISE EXCEPTION 'CT164: legacy alterado'; END IF;
 END LOOP;
 SELECT count(*) INTO v_total_despues FROM vec_contratacion_temporal.expediente_version_integral;
 IF v_total_despues<>v_total+1 THEN RAISE EXCEPTION 'CT164: efecto inesperado'; END IF;
 RAISE NOTICE 'CT164 focal OK: materializador real v1 vacío, omisión/terna divergente denegadas, v2-v6 sin fuente denegadas, INSERT v2 denegado, legacy idéntico';
END $prueba$;
ROLLBACK;
