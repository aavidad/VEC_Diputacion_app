\set ON_ERROR_STOP on
-- D18: una referencia externa de Méritos solo nace de la ingesta Mayan
-- confirmada por Documentos. Es una guardia prospectiva; no reescribe D3.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000018',0));
DO $pre$
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR to_regclass('vec_documentos.referencia_externa') IS NULL
    OR to_regclass('vec_documentos.plan_ingesta_mayan') IS NULL
    OR to_regclass('vec_documentos.carga_directa_operacion') IS NULL
    OR to_regprocedure('vec_documentos.registrar_referencia_externa_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_documentos.guardar_referencia_meritos_mayan_v1()') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_documentos.referencia_externa'::regclass
       AND tgname='inmutable' AND NOT tgisinternal AND tgenabled='O')
    OR NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgrelid='vec_documentos.referencia_externa'::regclass
       AND tgname='no_truncar' AND NOT tgisinternal AND tgenabled='O')
 THEN RAISE EXCEPTION 'D18: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- D17 la invoca solo después de consumir su decisión V3 de fase abrir y
-- fijar el ámbito en la misma transacción. Si D18 falta, D17 falla cerrado.
-- No se concede SELECT de plan_ingesta_mayan al ejecutor ni a D17.
CREATE FUNCTION vec_documentos.validar_plan_carga_para_lectura_v1(
 p_plan_ref text,p_carga_id text,p_persona_ref text,p_hecho_ref text,p_version_carga bigint)
RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE valido boolean;
BEGIN
 IF current_user<>'vec_documentos_propietario'
    OR p_hecho_ref IS DISTINCT FROM nullif(current_setting('vec.documentos.expediente_ref',true),'')
    OR p_version_carga IS DISTINCT FROM 5 THEN
  RAISE EXCEPTION 'D18: plan de lectura no autorizado' USING ERRCODE='42501';
 END IF;
 SELECT true INTO valido FROM vec_documentos.plan_ingesta_mayan p
  WHERE p.plan_ref=p_plan_ref AND p.carga_id=p_carga_id
    AND p.persona_ref=p_persona_ref AND p.principal_ref=p_persona_ref
    AND p.expediente_ref=p_hecho_ref AND p.carga_version=p_version_carga
    AND p.estado='preparado' AND p.version=1 FOR SHARE;
 IF valido IS DISTINCT FROM true THEN
  RAISE EXCEPTION 'D18: plan de lectura no autorizado' USING ERRCODE='42501';
 END IF;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.validar_plan_carga_para_lectura_v1(text,text,text,text,bigint)
 FROM PUBLIC,vec_documentos_ejecutor;

-- El login ejecutor no tiene DML sobre la tabla. Esta función solo se invoca
-- desde el INSERT D3, después del consumo V3 en su misma transacción.
CREATE FUNCTION vec_documentos.guardar_referencia_meritos_mayan_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
DECLARE p vec_documentos.plan_ingesta_mayan%ROWTYPE;
        custodia text; id_documento text; n_outbox bigint; n_auditoria bigint;
BEGIN
 IF TG_OP<>'INSERT' THEN
  RAISE EXCEPTION 'D18: operación no admitida' USING ERRCODE='42501';
 END IF;
 IF NEW.modulo_id<>'meritos' THEN RETURN NEW; END IF;
 -- El ámbito se fija por la fachada D3 tras consumir V3. Nunca se toma una
 -- referencia del navegador como autorización para leer otro plan.
 IF current_user<>'vec_documentos_propietario'
    OR NEW.expediente_ref IS DISTINCT FROM nullif(current_setting('vec.documentos.expediente_ref',true),'')
    OR NEW.custodio_id IS DISTINCT FROM 'mayan'
    OR NEW.version IS DISTINCT FROM 1
    OR NEW.mime IS NULL OR NEW.tamano IS NULL
 THEN RAISE EXCEPTION 'D18: custodia de mérito no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO p FROM vec_documentos.plan_ingesta_mayan
  WHERE plan_ref=NEW.clave_idempotencia AND expediente_ref=NEW.expediente_ref;
 IF NOT FOUND OR p.estado IS DISTINCT FROM 'confirmado'
    OR p.documento_id IS NULL OR p.archivo_id IS NULL OR p.version_id IS NULL
 THEN RAISE EXCEPTION 'D18: plan Mayan no confirmado' USING ERRCODE='42501'; END IF;
 custodia:='mayan:'||p.documento_id||':'||p.archivo_id||':'||p.version_id;
 id_documento:='ref:'||encode(sha256(convert_to(
  'vec.documentos.meritos.mayan.referencia.v1:'||p.plan_ref,'UTF8')),'hex');
 IF NEW.id IS DISTINCT FROM id_documento
    OR NEW.principal_ref IS DISTINCT FROM p.persona_ref
    OR p.principal_ref IS DISTINCT FROM p.persona_ref
    OR NEW.expediente_ref IS DISTINCT FROM p.expediente_ref
    OR NEW.tipo_ref IS DISTINCT FROM p.tipo_ref
    OR NEW.custodia_ref IS DISTINCT FROM custodia
    OR NEW.huella_sha256 IS DISTINCT FROM p.huella_sha256
    OR NEW.mime IS DISTINCT FROM p.mime OR NEW.tamano IS DISTINCT FROM p.tamano
    OR NEW.politica_ref IS DISTINCT FROM p.politica_ref
    OR NEW.version_politica IS DISTINCT FROM p.version_politica
    OR NEW.huella_politica_sha256 IS DISTINCT FROM p.huella_politica_sha256
    OR NEW.conservacion_hasta IS DISTINCT FROM p.conservacion_hasta
    OR NEW.estado_politica IS DISTINCT FROM p.estado_politica
    OR NEW.proteccion IS DISTINCT FROM p.proteccion
 THEN RAISE EXCEPTION 'D18: original, titular o política divergente' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO n_outbox FROM vec_documentos.outbox o
  WHERE o.tipo='ingesta_mayan_verificada' AND o.recurso_ref=p.plan_ref
    AND o.expediente_ref=p.expediente_ref AND o.huella_sha256=p.huella_sha256
    AND o.registrada_en=p.actualizada_en;
 SELECT count(*) INTO n_auditoria FROM vec_documentos.auditoria_operacion a
  WHERE a.accion IN ('documentos.ingesta_mayan.resolver','documentos.ingesta_mayan.conciliar')
    AND a.recurso_ref=p.plan_ref AND a.expediente_ref=p.expediente_ref
    AND a.principal_ref=p.persona_ref AND a.resultado='confirmado'
    AND a.registrada_en=p.actualizada_en;
 IF n_outbox<>1 OR n_auditoria<>1 THEN
  RAISE EXCEPTION 'D18: evidencia de confirmación incompleta' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.guardar_referencia_meritos_mayan_v1() FROM PUBLIC,vec_documentos_ejecutor;
CREATE TRIGGER guardia_meritos_mayan BEFORE INSERT ON vec_documentos.referencia_externa
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.guardar_referencia_meritos_mayan_v1();
COMMIT;
