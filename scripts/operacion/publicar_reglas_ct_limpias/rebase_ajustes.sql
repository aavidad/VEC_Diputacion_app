\set ON_ERROR_STOP on
-- Operación excepcional de DATOS inventados: conserva el conjunto CT148 y
-- registra su vínculo con la base CT v2. No invoca ni suplanta una decisión V3.
-- La admisión debe permanecer detenida durante ambas transacciones.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended('vec_contratacion_temporal:regla_base_activacion',0));
SELECT pg_catalog.pg_advisory_xact_lock(
 pg_catalog.hashtextextended('vec_contratacion_temporal:ajustes_reglas',0));
LOCK TABLE vec_contratacion_temporal.regla_ajuste_version_v1,
 vec_contratacion_temporal.regla_ajuste_outbox_v1 IN SHARE ROW EXCLUSIVE MODE;
DO $rebase$
DECLARE
 base record; previa record; anterior record; instante timestamptz(6); solicitud_h text;
 clave constant uuid := 'c418dad9-6857-4c7c-a5f9-39ab47cfba16';
 operacion constant text := 'operacion:datos-inventados:20261012';
 huella_base constant text := '6ecf24d17b889c8bbd4400580977ac8b1c2c863c824f09ac5498d70a3d4a7141';
 huella_anterior constant text := '5530995222b852169848d7729a024cdc30b631fbde48608b1329e517f1d50572';
BEGIN
 IF session_user=current_user OR current_user<>'vec_contratacion_temporal_propietario'
 THEN RAISE EXCEPTION 'CT: instalador técnico requerido' USING ERRCODE='42501'; END IF;
 SELECT a.*,b.fuente_sha256 INTO base
 FROM vec_contratacion_temporal.regla_base_activacion_v1 a
 JOIN vec_contratacion_temporal.regla_base_publicada_v1 b
   ON (b.catalogo_id,b.version,b.huella_sha256)=(a.catalogo_id,a.version,a.huella_sha256)
 ORDER BY a.secuencia DESC LIMIT 1;
 IF NOT FOUND OR NOT base.activa OR base.secuencia<>2 OR base.secuencia_esperada<>1
    OR base.catalogo_id<>'vec.contratacion_temporal.reglas'
    OR base.version<>2 OR base.huella_sha256<>huella_base
    OR base.fuente_sha256<>'a9192d1a68e79a21f17650be6d57632ad3520fa91cfeb820fee84e5d6b89768a'
    OR base.aprobacion_ref<>operacion
 THEN RAISE EXCEPTION 'CT: base v2 incompatible' USING ERRCODE='55000'; END IF;
 SELECT * INTO previa FROM vec_contratacion_temporal.regla_ajuste_version_v1
  WHERE catalogo_id='vec.contratacion_temporal.reglas.ajustes'
  ORDER BY version DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'CT: no hay ajustes que vincular' USING ERRCODE='55000'; END IF;
 IF previa.base_version=2 AND previa.base_huella_sha256=huella_base
    AND previa.clave_idempotencia=clave THEN
   SELECT * INTO anterior FROM vec_contratacion_temporal.regla_ajuste_version_v1
     WHERE catalogo_id=previa.catalogo_id AND version=previa.version-1;
   IF NOT FOUND OR anterior.base_version<>1 OR anterior.base_huella_sha256<>huella_anterior
      OR anterior.ajustes<>previa.ajustes OR anterior.ajustes_canonico<>previa.ajustes_canonico
      OR anterior.huella_sha256<>previa.huella_sha256
      OR previa.decision_ref<>operacion OR previa.referencia<>operacion
      OR previa.consumo_huella_sha256<>previa.solicitud_huella_sha256
      OR previa.auditoria_ref NOT LIKE 'provision:%'
      OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_cambio_v1 c
        WHERE c.catalogo_id=previa.catalogo_id AND c.version=previa.version)
      OR NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_outbox_v1 o
     WHERE o.catalogo_id=previa.catalogo_id AND o.version=previa.version
       AND o.huella_sha256=previa.huella_sha256
       AND o.tipo='contratacion_temporal.reglas.ajustadas' AND o.creada_en=previa.vigente_desde)
   THEN RAISE EXCEPTION 'CT: rebase incompleto' USING ERRCODE='55000'; END IF;
   RETURN;
 END IF;
 IF previa.base_version<>1 OR previa.base_huella_sha256<>huella_anterior
    OR previa.version NOT BETWEEN 1 AND 9999998
    OR (SELECT pg_catalog.count(*) FROM vec_contratacion_temporal.regla_ajuste_version_v1
        WHERE catalogo_id=previa.catalogo_id)<>previa.version
    OR previa.clave_idempotencia=clave
    OR previa.huella_sha256<>pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(previa.ajustes_canonico,'UTF8')),'hex')
    OR previa.ajustes<>previa.ajustes_canonico::jsonb
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_version_v1 WHERE clave_idempotencia=clave)
    OR NOT EXISTS (SELECT 1 FROM vec_contratacion_temporal.regla_ajuste_outbox_v1 o
        WHERE o.catalogo_id=previa.catalogo_id AND o.version=previa.version
          AND o.huella_sha256=previa.huella_sha256)
 THEN RAISE EXCEPTION 'CT: cabeza de ajustes incompatible' USING ERRCODE='55000'; END IF;
 instante:=pg_catalog.date_trunc('microseconds',pg_catalog.clock_timestamp());
 IF instante<=previa.vigente_desde
 THEN RAISE EXCEPTION 'CT: reloj incompatible' USING ERRCODE='40001'; END IF;
 solicitud_h:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   clave::text||'|'||previa.version::text||'|'||previa.huella_sha256||'|'||huella_base||'|'||operacion,
   'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.regla_ajuste_version_v1(
  catalogo_id,version,version_esperada,organizacion_ref,ajustes,ajustes_canonico,huella_sha256,
  base_version,base_huella_sha256,actor_ref,motivo_clave,referencia,vigente_desde,
  clave_idempotencia,solicitud_huella_sha256,recibo_ref,decision_ref,consumo_huella_sha256,auditoria_ref)
 VALUES(previa.catalogo_id,previa.version+1,previa.version,previa.organizacion_ref,
  previa.ajustes,previa.ajustes_canonico,previa.huella_sha256,
  2,huella_base,session_user::text,'rebase_operacion',operacion,instante,
  clave,solicitud_h,'recibo:'||pg_catalog.gen_random_uuid()::text,operacion,solicitud_h,
  'provision:'||pg_catalog.gen_random_uuid()::text);
 -- No se añade una fila de cambio: ninguna cifra, unidad ni cómputo ha variado.
 INSERT INTO vec_contratacion_temporal.regla_ajuste_outbox_v1(
  evento_ref,catalogo_id,version,huella_sha256,tipo,estado,creada_en)
 VALUES('evento:'||pg_catalog.gen_random_uuid()::text,previa.catalogo_id,previa.version+1,
  previa.huella_sha256,'contratacion_temporal.reglas.ajustadas','pendiente',instante);
END $rebase$;
COMMIT;
