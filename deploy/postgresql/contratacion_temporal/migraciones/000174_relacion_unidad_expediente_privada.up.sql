\set ON_ERROR_STOP on
-- CT174 preparatoria: predicado privado de relación expediente/unidad.
-- Reserva CT174 fuera de Git antes de este archivo. No instala CT172 ni K.
-- El caller definitivo debe consumir V3/auditoría comunes antes de invocarlo.
-- Este helper no autoriza, no acredita documento/expediente y no consume V3.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000174',0));

DO $pre$
DECLARE nombre text;
BEGIN
 IF current_user<>'vec_contratacion_temporal_propietario' THEN
  RAISE EXCEPTION 'ct174_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['expediente_integral_actual','expediente_version_integral',
  'reserva_asignacion','terminal_asignacion','actuacion_expediente_integral','outbox_expediente_integral'] LOOP
  IF to_regclass('vec_contratacion_temporal.'||nombre) IS NULL THEN
   RAISE EXCEPTION 'ct174_preimagen_incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF to_regprocedure('vec_contratacion_temporal.instante_utc_v1(timestamp with time zone)') IS NULL
  OR to_regprocedure('vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(text,text,text,numeric)') IS NOT NULL THEN
  RAISE EXCEPTION 'ct174_preimagen_incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(
 p_organizacion_ref text,p_expediente_ref text,p_unidad_ref_esperada text,p_version_expediente_solicitada numeric
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY INVOKER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s'
AS $f$
DECLARE
 actual vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 solicitada vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 origen vec_contratacion_temporal.expediente_version_integral%ROWTYPE;
 reserva vec_contratacion_temporal.reserva_asignacion%ROWTYPE;
 terminal vec_contratacion_temporal.terminal_asignacion%ROWTYPE;
 actuacion vec_contratacion_temporal.actuacion_expediente_integral%ROWTYPE;
 evento vec_contratacion_temporal.outbox_expediente_integral%ROWTYPE;
 version_actual numeric; version_origen numeric; secuencia_origen numeric;
 asignacion jsonb; vinculo jsonb; carga_evento jsonb;
BEGIN
 -- INVOKER: el ejecutor no adquiere privilegios del propietario al llamarlo.
 -- Sólo una función CT SECURITY DEFINER ya autorizada debe llegar aquí.
 IF current_user<>'vec_contratacion_temporal_propietario'
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR current_setting('TimeZone')<>'UTC' OR pg_is_in_recovery() THEN
  RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501'; END IF;
 IF p_organizacion_ref IS NULL OR octet_length(p_organizacion_ref) NOT BETWEEN 3 AND 160
  OR p_expediente_ref IS NULL OR octet_length(p_expediente_ref) NOT BETWEEN 3 AND 160
  OR p_unidad_ref_esperada IS NULL OR octet_length(p_unidad_ref_esperada) NOT BETWEEN 3 AND 160
  OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR p_expediente_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR p_unidad_ref_esperada !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
  OR p_version_expediente_solicitada IS NULL
  OR p_version_expediente_solicitada NOT BETWEEN 1 AND 9007199254740991
  OR p_version_expediente_solicitada<>trunc(p_version_expediente_solicitada) THEN
  RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501'; END IF;

 -- FOR SHARE impide UPDATE del puntero, incluida su version no-key, hasta
 -- terminar la Tx exterior. FOR KEY SHARE no protege ese cambio.
 SELECT a.version INTO STRICT version_actual
 FROM vec_contratacion_temporal.expediente_integral_actual a
 WHERE a.expediente_ref=p_expediente_ref FOR SHARE;
 IF p_version_expediente_solicitada>version_actual THEN
  RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501'; END IF;
 SELECT v.* INTO STRICT actual FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_expediente_ref AND v.version=version_actual;
 SELECT v.* INTO STRICT solicitada FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_expediente_ref AND v.version=p_version_expediente_solicitada;
 asignacion:=actual.agregado_json->'asignacion';
 vinculo:=asignacion->'actuacion_registro';
 IF actual.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref
  OR solicitada.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref
  OR jsonb_typeof(asignacion) IS DISTINCT FROM 'object'
  OR jsonb_typeof(vinculo) IS DISTINCT FROM 'object'
  OR asignacion->>'unidad_ref' IS DISTINCT FROM p_unidad_ref_esperada
  OR solicitada.agregado_json#>>'{asignacion,unidad_ref}' IS DISTINCT FROM p_unidad_ref_esperada
  OR solicitada.agregado_json->'asignacion' IS DISTINCT FROM asignacion
  OR vinculo->>'unidad_asignada_ref' IS DISTINCT FROM p_unidad_ref_esperada THEN
  RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501'; END IF;
 version_origen:=(vinculo->>'version_expediente')::numeric;
 secuencia_origen:=(vinculo->>'secuencia')::numeric;
 IF version_origen IS NULL OR version_origen NOT BETWEEN 1 AND p_version_expediente_solicitada
  OR version_origen<>trunc(version_origen) OR secuencia_origen IS DISTINCT FROM version_origen
  OR vinculo->>'accion_clave' IS DISTINCT FROM 'contratacion_temporal.unidad.asignar' THEN
  RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501'; END IF;
 SELECT v.* INTO STRICT origen FROM vec_contratacion_temporal.expediente_version_integral v
 WHERE v.expediente_ref=p_expediente_ref AND v.version=version_origen;
 SELECT r.* INTO STRICT reserva FROM vec_contratacion_temporal.reserva_asignacion r
 WHERE r.expediente_ref=p_expediente_ref AND r.organizacion_ref=p_organizacion_ref
  AND r.reserva_ref=origen.operacion_ref AND r.estado='confirmada';
 SELECT t.* INTO STRICT terminal FROM vec_contratacion_temporal.terminal_asignacion t
 WHERE t.ambito_hmac=reserva.ambito_hmac;
 SELECT a.* INTO STRICT actuacion FROM vec_contratacion_temporal.actuacion_expediente_integral a
 WHERE a.expediente_ref=p_expediente_ref AND a.secuencia=secuencia_origen;
 SELECT e.* INTO STRICT evento FROM vec_contratacion_temporal.outbox_expediente_integral e
 WHERE e.evento_ref=reserva.evento_ref;
 carga_evento:=convert_from(evento.payload_canonico,'UTF8')::jsonb;
 IF origen.origen_version IS DISTINCT FROM 'asignacion_o5'
  OR origen.agregado_json->'asignacion' IS DISTINCT FROM asignacion
  OR origen.agregado_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref
  OR origen.agregado_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(origen.agregado_json::text,'UTF8')),'hex')
  OR origen.prueba_huella_sha256 IS DISTINCT FROM encode(sha256(origen.prueba_canonica),'hex')
  OR origen.prueba_huella_sha256 !~ '^[0-9a-f]{64}$'
  OR reserva.operacion IS DISTINCT FROM 'asignar'
  OR reserva.version_expediente+1 IS DISTINCT FROM version_origen
  OR reserva.unidad_ref IS DISTINCT FROM p_unidad_ref_esperada
  OR reserva.responsable_ref IS DISTINCT FROM asignacion->>'responsable_ref'
  OR reserva.notificacion_ref IS DISTINCT FROM asignacion->>'notificacion_ref'
  OR reserva.recibo_ref IS DISTINCT FROM vinculo->>'recibo_ref'
  OR reserva.responsable_ref IS DISTINCT FROM vinculo->>'responsable_asignado_ref'
  OR reserva.notificacion_ref IS DISTINCT FROM vinculo->>'notificacion_ref'
  OR reserva.confirmada_en IS NULL OR NOT isfinite(reserva.confirmada_en)
  OR reserva.confirmada_en IS DISTINCT FROM date_trunc('microseconds',reserva.confirmada_en)
  OR terminal.confirmada_en IS DISTINCT FROM reserva.confirmada_en
  OR origen.registrada_en IS DISTINCT FROM reserva.confirmada_en
  OR terminal.recibo_json->>'expediente_ref' IS DISTINCT FROM p_expediente_ref
  OR terminal.recibo_json->>'organizacion_ref' IS DISTINCT FROM p_organizacion_ref
  OR terminal.recibo_json->>'unidad_ref' IS DISTINCT FROM p_unidad_ref_esperada
  OR terminal.recibo_json->>'recibo_ref' IS DISTINCT FROM reserva.recibo_ref
  OR (terminal.recibo_json->>'version_resultante')::numeric IS DISTINCT FROM version_origen
  OR (terminal.recibo_json->>'confirmada_en')::timestamptz IS DISTINCT FROM reserva.confirmada_en
  OR terminal.terminal_json#>>'{referencias,ReservaRef}' IS DISTINCT FROM reserva.reserva_ref
  OR actuacion.version_expediente IS DISTINCT FROM version_origen
  OR actuacion.operacion_ref IS DISTINCT FROM origen.operacion_ref
  OR actuacion.recibo_ref IS DISTINCT FROM reserva.recibo_ref
  OR actuacion.registrada_en IS DISTINCT FROM reserva.confirmada_en
  OR (actuacion.actuacion_json->>'secuencia')::numeric IS DISTINCT FROM secuencia_origen
  OR (actuacion.actuacion_json->>'version_expediente')::numeric IS DISTINCT FROM version_origen
  OR actuacion.actuacion_json->>'accion_clave' IS DISTINCT FROM 'contratacion_temporal.unidad.asignar'
  OR actuacion.actuacion_json->>'recibo_ref' IS DISTINCT FROM reserva.recibo_ref
  OR actuacion.actuacion_json_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(actuacion.actuacion_json::text,'UTF8')),'hex')
  OR evento.expediente_ref IS DISTINCT FROM p_expediente_ref
  OR evento.version_expediente IS DISTINCT FROM version_origen
  OR evento.operacion_ref IS DISTINCT FROM origen.operacion_ref
  OR evento.tipo_evento IS DISTINCT FROM 'contratacion_temporal.asignacion_confirmada'
  OR evento.registrada_en IS DISTINCT FROM reserva.confirmada_en
  OR evento.payload_huella_sha256 IS DISTINCT FROM encode(sha256(evento.payload_canonico),'hex')
  OR carga_evento->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.asignacion-confirmada.v1'
  OR carga_evento->>'expediente_ref' IS DISTINCT FROM p_expediente_ref
  OR carga_evento->>'unidad_ref' IS DISTINCT FROM p_unidad_ref_esperada
  OR carga_evento->>'recibo_ref' IS DISTINCT FROM reserva.recibo_ref
  OR carga_evento->>'responsable_ref' IS DISTINCT FROM reserva.responsable_ref
  OR (carga_evento->>'version_resultante')::numeric IS DISTINCT FROM version_origen THEN
  RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501'; END IF;
 -- Huella de prueba del snapshot origen ya persistida, no hash inventado del
 -- par. Fecha de confirmación durable, nunca una hora de esta consulta.
 RETURN jsonb_build_object(
  'esquema','vec.contratacion-temporal.relacion-unidad-expediente.v1',
  'organizacion_ref',p_organizacion_ref,'expediente_ref',p_expediente_ref,
  'unidad_ref_esperada',p_unidad_ref_esperada,'version_expediente_solicitada',p_version_expediente_solicitada::bigint,
  'unidad_ref',asignacion->>'unidad_ref',
  'unidad_snapshot_solicitado_ref',solicitada.agregado_json#>>'{asignacion,unidad_ref}',
  'version_expediente_actual',version_actual::bigint,'version_origen_vinculo',version_origen::bigint,
  'operacion_origen_ref',origen.operacion_ref,'reserva_asignacion_ref',reserva.reserva_ref,
  'recibo_asignacion_ref',reserva.recibo_ref,
  'prueba_snapshot_origen_huella_sha256',origen.prueba_huella_sha256,
  'tipo_evento_origen',evento.tipo_evento,'evento_asignacion_ref',evento.evento_ref,
  'evento_payload_huella_sha256',evento.payload_huella_sha256,
  'asignacion_confirmada_en',vec_contratacion_temporal.instante_utc_v1(reserva.confirmada_en));
EXCEPTION
 WHEN SQLSTATE '40001' OR SQLSTATE '40P01' OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
 WHEN OTHERS THEN RAISE EXCEPTION 'ct_relacion_unidad_no_disponible' USING ERRCODE='42501';
END $f$;

REVOKE ALL ON FUNCTION vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(text,text,text,numeric) FROM PUBLIC;
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(text,text,text,numeric)'::regprocedure;
 a record; owner_id oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 -- Retirar también destinatarios de ACL por defecto. No se concede a K,
 -- LOGIN, ejecutores, consultores ni cualquier otro rol exterior.
 FOR a IN SELECT x.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
  WHERE p.oid=f AND x.grantee<>owner_id LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
   CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=owner_id
  AND NOT p.prosecdef AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
  AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','lock_timeout=2s'])
  OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f)<>1
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
   AND (x.grantee<>owner_id OR x.grantor<>owner_id OR x.privilege_type<>'EXECUTE' OR x.is_grantable)) THEN
  RAISE EXCEPTION 'ct174_acl_incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
