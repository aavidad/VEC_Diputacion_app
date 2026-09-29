\set ON_ERROR_STOP on
-- Documentos-9 (5.06): custodia del documento firmado de un expediente.
-- Un módulo productor (Contratación temporal, resolución firmada) entrega el
-- PDF ya firmado y verificado; Documentos lo custodia con custodia VEC y
-- consume en ESTA transacción la autorización V3 documentos.firmado.custodiar
-- (AD3-113), junto a la fila del documento, la de firmado, la auditoría y el
-- outbox. Los tipos reservados a esta ruta no pueden darse de alta por el alta
-- genérica: una comprobación diferida exige su fila de firmado.
-- La firma registrada es de prueba (GrxFirma) y no la del proveedor
-- corporativo: estado_firma sigue en pendiente_proveedor.
-- Requiere Documentos 000002 y AD3-113. Historia de solo adición.
BEGIN;
SET LOCAL ROLE vec_documentos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_documentos:migracion:000009',0));
DO $pre$
DECLARE f oid;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_documentos_propietario'
    OR to_regprocedure('vec_documentos.consumir_v3_v2(bytea,text,text,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_documentos.confirmar_alta_v2(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_documentos.custodiar_firmado_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regclass('vec_documentos.documento_firmado') IS NOT NULL
    OR to_regclass('vec_documentos.tipo_reservado_firmado') IS NOT NULL
    OR to_regclass('vec_documentos.referencia_externa') IS NULL
    OR f IS NULL OR strpos(pg_get_functiondef(f),'documentos.firmado.custodiar')=0
 THEN RAISE EXCEPTION 'Documentos-9: requiere Documentos 000002 y AD3-113, y no estar ya instalada' USING ERRCODE='55000'; END IF;
END $pre$;

-- Auditoría y outbox admiten la operación nueva. Se comprueba antes la
-- definición exacta vigente para no pisar otra ampliación.
DO $restricciones$
DECLARE a text; o text;
BEGIN
 SELECT pg_get_constraintdef(c.oid) INTO STRICT a FROM pg_constraint c
  WHERE c.conrelid='vec_documentos.auditoria_operacion'::regclass AND c.conname='auditoria_operacion_accion_check';
 SELECT pg_get_constraintdef(c.oid) INTO STRICT o FROM pg_constraint c
  WHERE c.conrelid='vec_documentos.outbox'::regclass AND c.conname='outbox_tipo_check';
 IF a IS DISTINCT FROM $x$CHECK ((accion = ANY (ARRAY['documentos.generado.alta'::text, 'documentos.expediente.listar'::text, 'documentos.original.descargar'::text, 'documentos.notificacion.preparar'::text, 'documentos.externo.registrar'::text])))$x$
    OR o IS DISTINCT FROM $x$CHECK ((tipo = ANY (ARRAY['documento_generado'::text, 'notificacion_preparada'::text, 'documento_externo_registrado'::text])))$x$
 THEN RAISE EXCEPTION 'Documentos-9: restricciones de auditoría u outbox distintas de las esperadas' USING ERRCODE='55000'; END IF;
END $restricciones$;
ALTER TABLE vec_documentos.auditoria_operacion DROP CONSTRAINT auditoria_operacion_accion_check;
ALTER TABLE vec_documentos.auditoria_operacion ADD CONSTRAINT auditoria_operacion_accion_check
 CHECK(accion IN ('documentos.generado.alta','documentos.expediente.listar','documentos.original.descargar','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar'));
ALTER TABLE vec_documentos.outbox DROP CONSTRAINT outbox_tipo_check;
ALTER TABLE vec_documentos.outbox ADD CONSTRAINT outbox_tipo_check
 CHECK(tipo IN ('documento_generado','notificacion_preparada','documento_externo_registrado','documento_firmado_custodiado'));

-- Tipos documentales reservados a la custodia de firmados. tipo_ref es la
-- referencia opaca del catálogo de conservación (ref:sha256 del tipo).
CREATE TABLE vec_documentos.tipo_reservado_firmado (
 tipo_ref text PRIMARY KEY CHECK(vec_documentos.referencia_opaca_v1(tipo_ref)),
 tipo text NOT NULL UNIQUE CHECK(tipo ~ '^[a-z][a-z0-9_.]{2,127}$'),
 modulo_id text NOT NULL CHECK(modulo_id ~ '^[a-z][a-z0-9_]{2,63}$')
);
INSERT INTO vec_documentos.tipo_reservado_firmado VALUES
 ('ref:f0074a505ef2a693dfd5bf2195228c47a7ce2a1435d4acf6a6b9f50e89e5663c',
  'contratacion_temporal.resolucion_firmada.v1','contratacion_temporal');

-- Datos de firma del documento custodiado, 1:1 con su documento.
CREATE TABLE vec_documentos.documento_firmado (
 documento_id text PRIMARY KEY REFERENCES vec_documentos.documento(id),
 expediente_ref text NOT NULL,
 modulo_id text NOT NULL,
 tipo_ref text NOT NULL REFERENCES vec_documentos.tipo_reservado_firmado(tipo_ref),
 firma_operacion_ref text NOT NULL CHECK(vec_documentos.referencia_opaca_v1(firma_operacion_ref)),
 huella_original_sha256 text NOT NULL CHECK(huella_original_sha256 ~ '^[0-9a-f]{64}$'),
 huella_firmado_sha256 text NOT NULL CHECK(huella_firmado_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ad3_ref text NOT NULL,
 custodiado_en timestamptz(6) NOT NULL,
 CHECK(huella_original_sha256<>huella_firmado_sha256),
 UNIQUE(modulo_id,firma_operacion_ref)
);
COMMENT ON TABLE vec_documentos.documento_firmado IS
 'Documento firmado custodiado por Documentos (5.06). Solo adición. La firma es de prueba (GrxFirma) y no del proveedor corporativo.';

DO $seguridad$
DECLARE t text; a record;
BEGIN
 FOREACH t IN ARRAY ARRAY['tipo_reservado_firmado','documento_firmado'] LOOP
  EXECUTE format('ALTER TABLE vec_documentos.%I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE vec_documentos.%I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('REVOKE ALL ON TABLE vec_documentos.%I FROM PUBLIC',t);
  FOR a IN SELECT DISTINCT x.grantee FROM pg_class c,
      LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) x
      WHERE c.oid=to_regclass('vec_documentos.'||t) AND x.grantee<>0 AND x.grantee<>c.relowner LOOP
   EXECUTE format('REVOKE ALL ON TABLE vec_documentos.%I FROM %I',t,pg_get_userbyid(a.grantee));
  END LOOP;
  EXECUTE format('REVOKE ALL ON TYPE vec_documentos.%I FROM PUBLIC',t);
  EXECUTE format('CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON vec_documentos.%I FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1()',t);
  EXECUTE format('CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_documentos.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_documentos.rechazar_mutacion_v1()',t);
 END LOOP;
END $seguridad$;
CREATE POLICY lectura ON vec_documentos.tipo_reservado_firmado FOR SELECT TO vec_documentos_propietario USING (true);
CREATE POLICY alcance_lectura ON vec_documentos.documento_firmado FOR SELECT TO vec_documentos_propietario
 USING (expediente_ref = nullif(current_setting('vec.documentos.expediente_ref',true),''));
CREATE POLICY alcance_alta ON vec_documentos.documento_firmado FOR INSERT TO vec_documentos_propietario
 WITH CHECK (expediente_ref = nullif(current_setting('vec.documentos.expediente_ref',true),''));

-- Un documento de tipo reservado sin su fila de firmado no llega a confirmarse:
-- ni el alta genérica ni un INSERT directo pueden custodiarlo como firmado.
CREATE FUNCTION vec_documentos.exigir_documento_firmado_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_documentos.tipo_reservado_firmado t WHERE t.tipo_ref=NEW.tipo_ref)
    AND NOT EXISTS (SELECT 1 FROM vec_documentos.documento_firmado f
                     WHERE f.documento_id=NEW.id AND f.tipo_ref=NEW.tipo_ref AND f.huella_firmado_sha256=NEW.huella_sha256
                       AND f.expediente_ref=NEW.expediente_ref AND f.modulo_id=NEW.modulo_id) THEN
  RAISE EXCEPTION 'documentos: tipo reservado a la custodia de documentos firmados' USING ERRCODE='42501';
 END IF;
 RETURN NULL;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.exigir_documento_firmado_v1() FROM PUBLIC;
CREATE CONSTRAINT TRIGGER exigir_documento_firmado AFTER INSERT ON vec_documentos.documento
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION vec_documentos.exigir_documento_firmado_v1();

-- Un tipo reservado tampoco puede anotarse como referencia de custodia externa.
CREATE FUNCTION vec_documentos.rechazar_externa_reservada_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_documentos.tipo_reservado_firmado t WHERE t.tipo_ref=NEW.tipo_ref) THEN
  RAISE EXCEPTION 'documentos: tipo reservado a la custodia de documentos firmados' USING ERRCODE='42501';
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.rechazar_externa_reservada_v1() FROM PUBLIC;
CREATE TRIGGER rechazar_tipo_reservado BEFORE INSERT ON vec_documentos.referencia_externa
 FOR EACH ROW EXECUTE FUNCTION vec_documentos.rechazar_externa_reservada_v1();

-- Preimagen canónica (claves exactas) que liga la autorización V3 al efecto.
CREATE FUNCTION vec_documentos.custodiar_firmado_v1(
 p_preimagen bytea,p_objeto jsonb,p_auth jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE m jsonb; v record; d vec_documentos.documento%ROWTYPE; fi vec_documentos.documento_firmado%ROWTYPE;
 ahora timestamptz(6):=date_trunc('microseconds',clock_timestamp()); h text; nuevo boolean:=false;
BEGIN
 IF current_user<>'vec_documentos_propietario' OR session_user=current_user
    OR p_auth IS NULL OR p_objeto IS NULL OR p_preimagen IS NULL
 THEN RAISE EXCEPTION 'documentos: custodia denegada' USING ERRCODE='42501'; END IF;
 BEGIN m:=convert_from(p_preimagen,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'documentos: preimagen inválida' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1)
    IS DISTINCT FROM ARRAY['accion','clave_idempotencia','conservacion_hasta','estado_politica','expediente_ref','firma_operacion_ref','huella_original_sha256','huella_politica_sha256','huella_sha256','id','mime','modulo_id','politica_ref','proteccion','tamano','tipo_ref','version','version_politica']
    OR m->>'accion' IS DISTINCT FROM 'documentos.firmado.custodiar'
    OR m->>'mime' IS DISTINCT FROM 'application/pdf'
    OR m->>'id' IS DISTINCT FROM p_auth->>'recurso_ref'
    OR m->>'expediente_ref' IS DISTINCT FROM p_auth->>'ambito_ref'
    OR p_auth->>'accion' IS DISTINCT FROM m->>'accion'
    OR p_auth->>'finalidad' IS DISTINCT FROM 'custodiar_documento_firmado'
    OR NOT EXISTS (SELECT 1 FROM vec_documentos.tipo_reservado_firmado t WHERE t.tipo_ref=m->>'tipo_ref' AND t.modulo_id=m->>'modulo_id')
    OR NOT vec_documentos.referencia_opaca_v1(m->>'firma_operacion_ref')
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR m->>'huella_original_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'huella_sha256' = m->>'huella_original_sha256'
    OR p_objeto->>'mime' IS DISTINCT FROM m->>'mime'
    OR p_objeto->>'tamano' IS DISTINCT FROM m->>'tamano'
    OR p_objeto->>'huella_sha256' IS DISTINCT FROM m->>'huella_sha256'
    OR p_objeto->>'objeto_ref' IS NULL OR p_objeto->>'objeto_version' IS NULL
    OR p_objeto->>'conector_ref' IS NULL OR p_objeto->>'recibo_objeto_ref' IS NULL
    OR p_objeto->>'recibo_objeto_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR jsonb_typeof(p_objeto->'inmovilizado') IS DISTINCT FROM 'boolean'
    OR m->>'estado_politica' NOT IN ('aprobada','provisional')
    OR (m->>'estado_politica'='aprobada' AND (jsonb_typeof(p_objeto->'retenido_hasta') IS DISTINCT FROM 'string'
        OR (p_objeto->>'retenido_hasta')::timestamptz < (m->>'conservacion_hasta')::timestamptz))
    OR (m->>'estado_politica'='provisional' AND (jsonb_typeof(p_objeto->'retenido_hasta') IS DISTINCT FROM 'null'
        OR p_objeto->>'inmovilizado' IS DISTINCT FROM 'false' OR m->>'proteccion' IS DISTINCT FROM 'conservacion'))
    OR (m->>'proteccion'='bloqueo' AND p_objeto->>'inmovilizado'<>'true')
    OR m->>'huella_politica_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'proteccion' NOT IN ('conservacion','bloqueo')
 THEN RAISE EXCEPTION 'documentos: custodia no ligada al objeto' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_documentos.consumir_v3_v2(
  p_preimagen,'documentos.firmado.custodiar',p_auth->>'recurso_ref',p_auth->>'principal_id',
  p_auth->>'perfil_activo_ref',p_auth->>'finalidad',p_auth->>'correlacion_ref',
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 PERFORM set_config('vec.documentos.expediente_ref',m->>'expediente_ref',true);
 h:=encode(sha256(p_preimagen),'hex');
 SELECT * INTO d FROM vec_documentos.documento
  WHERE principal_ref=p_auth->>'principal_id' AND clave_idempotencia=m->>'clave_idempotencia';
 IF v.consumo_nuevo IS FALSE AND NOT FOUND THEN
  RAISE EXCEPTION 'documentos: replay sin custodia confirmada' USING ERRCODE='42501';
 END IF;
 IF FOUND THEN
  SELECT * INTO fi FROM vec_documentos.documento_firmado WHERE documento_id=d.id;
  IF NOT FOUND
     OR (v.consumo_nuevo IS FALSE AND (d.decision_ref IS DISTINCT FROM v.decision_ref OR d.auditoria_ad3_ref IS DISTINCT FROM v.auditoria_ref))
     OR d.huella_preimagen_sha256 IS DISTINCT FROM h OR d.id IS DISTINCT FROM m->>'id'
     OR d.recibo_objeto_ref IS DISTINCT FROM p_objeto->>'recibo_objeto_ref'
     OR d.recibo_objeto_huella_sha256 IS DISTINCT FROM p_objeto->>'recibo_objeto_huella_sha256'
     OR d.conector_ref IS DISTINCT FROM p_objeto->>'conector_ref'
     OR d.objeto_ref IS DISTINCT FROM p_objeto->>'objeto_ref'
     OR d.objeto_version IS DISTINCT FROM p_objeto->>'objeto_version'
     OR d.objeto_retenido_hasta IS DISTINCT FROM (p_objeto->>'retenido_hasta')::timestamptz
     OR d.objeto_inmovilizado IS DISTINCT FROM (p_objeto->>'inmovilizado')::boolean
     OR fi.firma_operacion_ref IS DISTINCT FROM m->>'firma_operacion_ref'
  THEN RAISE EXCEPTION 'documentos: clave idempotente reutilizada' USING ERRCODE='23505'; END IF;
 ELSE
  INSERT INTO vec_documentos.documento(
    id,numero_vec,clave_idempotencia,principal_ref,modulo_id,expediente_ref,tipo_ref,version,mime,
    huella_sha256,tamano,objeto_ref,objeto_version,conector_ref,recibo_objeto_ref,recibo_objeto_huella_sha256,
    objeto_retenido_hasta,objeto_inmovilizado,
    politica_ref,version_politica,huella_politica_sha256,conservacion_hasta,proteccion,estado_politica,
    huella_preimagen_sha256,decision_ref,auditoria_ad3_ref,creada_en)
  VALUES(m->>'id','VEC-'||to_char(ahora,'YYYY')||'-'||nextval('vec_documentos.numero_interno_seq')::text,
    m->>'clave_idempotencia',p_auth->>'principal_id',m->>'modulo_id',m->>'expediente_ref',m->>'tipo_ref',(m->>'version')::bigint,m->>'mime',
    m->>'huella_sha256',(m->>'tamano')::bigint,p_objeto->>'objeto_ref',p_objeto->>'objeto_version',
    p_objeto->>'conector_ref',p_objeto->>'recibo_objeto_ref',p_objeto->>'recibo_objeto_huella_sha256',
    (p_objeto->>'retenido_hasta')::timestamptz,(p_objeto->>'inmovilizado')::boolean,
    m->>'politica_ref',(m->>'version_politica')::bigint,m->>'huella_politica_sha256',(m->>'conservacion_hasta')::timestamptz,
    m->>'proteccion',m->>'estado_politica',h,v.decision_ref,v.auditoria_ref,ahora) RETURNING * INTO d;
  INSERT INTO vec_documentos.documento_firmado VALUES(d.id,d.expediente_ref,d.modulo_id,d.tipo_ref,m->>'firma_operacion_ref',
    m->>'huella_original_sha256',d.huella_sha256,v.decision_ref,v.auditoria_ref,ahora);
  INSERT INTO vec_documentos.outbox(tipo,recurso_ref,expediente_ref,huella_sha256,registrada_en)
   VALUES('documento_firmado_custodiado',d.id,d.expediente_ref,d.huella_sha256,ahora);
  nuevo:=true;
 END IF;
 INSERT INTO vec_documentos.auditoria_operacion(accion,recurso_ref,expediente_ref,principal_ref,perfil_ref,finalidad,correlacion_ref,decision_ref,auditoria_ad3_ref,resultado,registrada_en)
 VALUES('documentos.firmado.custodiar',d.id,d.expediente_ref,p_auth->>'principal_id',p_auth->>'perfil_activo_ref',
  p_auth->>'finalidad',p_auth->>'correlacion_ref',v.decision_ref,v.auditoria_ref,
  CASE WHEN nuevo THEN 'creado' ELSE 'repetido' END,ahora);
 RETURN vec_documentos.proyectar_documento_v1(d) || jsonb_build_object(
  'firma_operacion_ref',m->>'firma_operacion_ref','huella_original_sha256',m->>'huella_original_sha256');
END $f$;
REVOKE ALL ON FUNCTION vec_documentos.custodiar_firmado_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_documentos.custodiar_firmado_v1(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_ejecutor;
COMMIT;
