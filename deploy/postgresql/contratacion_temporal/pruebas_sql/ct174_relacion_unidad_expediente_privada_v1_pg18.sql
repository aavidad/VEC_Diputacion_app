\set ON_ERROR_STOP on
-- Preparada, no ejecutada. Sólo clon privado PG18 post27, no principal.
-- Invocación directa por owner en esta prueba es estructural: no simula V3,
-- firma, caller K ni el futuro efecto autorizado CT172.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $prueba$
DECLARE
 f regprocedure:='vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(text,text,text,numeric)'::regprocedure;
 muestra record; datos jsonb; historica jsonb; rechazos integer:=0;
 version_prueba numeric; unidad_prueba text; org_prueba text;
BEGIN
 IF (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM false
  OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
    WHERE p.oid=f AND (a.grantee<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'ct174_prueba_acl'; END IF;
 SELECT r.organizacion_ref,r.expediente_ref,r.unidad_ref,r.reserva_ref,r.recibo_ref,r.evento_ref,
   r.confirmada_en,v.version AS origen_version,a.version AS actual_version,
   v.prueba_huella_sha256,e.payload_huella_sha256
 INTO STRICT muestra
 FROM vec_contratacion_temporal.reserva_asignacion r
 JOIN vec_contratacion_temporal.terminal_asignacion t USING(ambito_hmac)
 JOIN vec_contratacion_temporal.expediente_version_integral v
   ON v.expediente_ref=r.expediente_ref AND v.operacion_ref=r.reserva_ref
 JOIN vec_contratacion_temporal.expediente_integral_actual a ON a.expediente_ref=r.expediente_ref
 JOIN vec_contratacion_temporal.expediente_version_integral actual
   ON actual.expediente_ref=a.expediente_ref AND actual.version=a.version
 JOIN vec_contratacion_temporal.outbox_expediente_integral e ON e.evento_ref=r.evento_ref
 WHERE r.estado='confirmada' AND t.confirmada_en=r.confirmada_en
   AND actual.agregado_json->'asignacion'=v.agregado_json->'asignacion'
   AND a.version>v.version
 ORDER BY r.expediente_ref LIMIT 1;
 datos:=vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(
   muestra.organizacion_ref,muestra.expediente_ref,muestra.unidad_ref,muestra.actual_version);
 historica:=vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(
   muestra.organizacion_ref,muestra.expediente_ref,muestra.unidad_ref,muestra.origen_version);
 IF datos->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.relacion-unidad-expediente.v1'
  OR datos->>'operacion_origen_ref' IS DISTINCT FROM muestra.reserva_ref
  OR datos->>'reserva_asignacion_ref' IS DISTINCT FROM muestra.reserva_ref
  OR datos->>'recibo_asignacion_ref' IS DISTINCT FROM muestra.recibo_ref
  OR datos->>'evento_asignacion_ref' IS DISTINCT FROM muestra.evento_ref
  OR datos->>'tipo_evento_origen' IS DISTINCT FROM 'contratacion_temporal.asignacion_confirmada'
  OR datos->>'prueba_snapshot_origen_huella_sha256' IS DISTINCT FROM muestra.prueba_huella_sha256
  OR datos->>'evento_payload_huella_sha256' IS DISTINCT FROM muestra.payload_huella_sha256
  OR (datos->>'asignacion_confirmada_en')::timestamptz IS DISTINCT FROM muestra.confirmada_en
  OR datos->>'unidad_ref' IS DISTINCT FROM muestra.unidad_ref
  OR historica->>'unidad_snapshot_solicitado_ref' IS DISTINCT FROM muestra.unidad_ref
  OR (historica->>'version_expediente_solicitada')::numeric IS DISTINCT FROM muestra.origen_version
  OR (historica->>'version_expediente_actual')::numeric IS DISTINCT FROM muestra.actual_version
  OR datos->>'asignacion_confirmada_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
  OR (datos->'version_expediente_actual')::text !~ '^[0-9]+$'
  OR (datos->'version_expediente_solicitada')::text !~ '^[0-9]+$'
  OR (datos->'version_origen_vinculo')::text !~ '^[0-9]+$'
  OR (historica-'version_expediente_solicitada') IS DISTINCT FROM (datos-'version_expediente_solicitada') THEN
  RAISE EXCEPTION 'ct174_prueba_procedencia_historica'; END IF;
 -- Versión previa sin asignación, futura, unidad divergente y organización
 -- divergente deben denegar igual; ningún campo del caller crea el vínculo.
 FOR version_prueba,unidad_prueba,org_prueba IN SELECT * FROM (VALUES
  (muestra.origen_version-1,muestra.unidad_ref,muestra.organizacion_ref),
  (muestra.actual_version+1,muestra.unidad_ref,muestra.organizacion_ref),
  (muestra.actual_version,'unidad:ct174:divergente',muestra.organizacion_ref),
  (muestra.actual_version,muestra.unidad_ref,'organizacion:ct174:divergente')) AS x(v,u,o) LOOP
  BEGIN
   PERFORM vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(
    org_prueba,muestra.expediente_ref,unidad_prueba,version_prueba);
  EXCEPTION WHEN SQLSTATE '42501' THEN rechazos:=rechazos+1;
  END;
 END LOOP;
 IF rechazos<>4 THEN RAISE EXCEPTION 'ct174_prueba_denegacion'; END IF;
 -- El bloqueo real del puntero también debe pasar la prueba de dos sesiones
 -- preparada en el protocolo: UPDATE no-key espera hasta ROLLBACK del lector.
END $prueba$;
ROLLBACK;

-- Protocolo concurrente pendiente, dos sesiones del MISMO clon privado.
-- Elegir la misma muestra confirmada de arriba (org/exp/unidad/version).
-- A: BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
--    SET LOCAL ROLE vec_contratacion_temporal_propietario;
--    SET LOCAL timezone='UTC';
--    SELECT vec_contratacion_temporal.leer_revalidar_relacion_unidad_expediente_ct_v1(
--      :'org',:'exp',:'unidad',:'version'::numeric);
--    Mantener A abierta mientras B completa su intento.
-- B: BEGIN; SET LOCAL ROLE vec_contratacion_temporal_propietario;
--    SET LOCAL lock_timeout='100ms';
--    UPDATE vec_contratacion_temporal.expediente_integral_actual
--      SET version=version WHERE expediente_ref=:'exp';
--    Esperado 55P03: incluso UPDATE no-key/no-op queda bloqueado por A.
--    ROLLBACK;
-- A: ROLLBACK;
-- Reintentar B después del ROLLBACK A: UPDATE permitido y luego ROLLBACK.
-- Conservar snapshots completos anteriores/posteriores; sin otra alta/firma,
-- sin modificar ACL ni funciones para crear un doble permisivo.
