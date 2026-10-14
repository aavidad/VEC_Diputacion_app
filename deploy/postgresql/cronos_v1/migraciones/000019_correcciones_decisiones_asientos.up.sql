\set ON_ERROR_STOP on
-- CRN19, C5b. Jefatura, RRHH y aplicación de correcciones por hechos
-- inmutables. Requiere CRN18 y las cuatro fachadas nominales AD273.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000019',0));
DO $pre$
DECLARE n text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_cronos_v1.correccion_solicitud') IS NULL
    OR to_regclass('vec_cronos_v1.correccion_actuacion') IS NULL
    OR to_regclass('vec_cronos_v1.correccion_resolutor') IS NOT NULL
    OR to_regprocedure('vec_cronos_v1.consultar_movimientos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)') IS NULL
 THEN RAISE EXCEPTION 'CRN19: preimagen incompatible' USING ERRCODE='55000'; END IF;
 IF (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex') FROM pg_constraint c
      WHERE c.conrelid='vec_cronos_v1.solicitud_outbox'::regclass AND c.conname='solicitud_outbox_tipo_check')
      <> '4bb48b1e56c6510a0b14e31cb0363959bfcb031e48655c9e1ad785a16bfda9a8'
    OR (SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex') FROM pg_constraint c
      WHERE c.conrelid='vec_cronos_v1.denegacion_frontera'::regclass AND c.conname='denegacion_frontera_ruta_check')
      <> 'c729a67a0717e6331395f7015aaf71da43757c142f5a90ac2a3e0224363cc859'
 THEN RAISE EXCEPTION 'CRN19: restricciones previas incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY[
   'consumir_cronos_bandeja_correcciones_v3_atestada',
   'registrar_y_consumir_cronos_decision_responsable_v3_atestada',
   'registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada',
   'registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada',
   'consumir_cronos_historia_correccion_propia_v3_atestada',
   'consumir_cronos_historia_correccion_equipo_v3_atestada'] LOOP
   IF to_regprocedure('vec_autorizacion_atestada_v3.'||n||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
      OR NOT has_function_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3.'||n||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
   THEN RAISE EXCEPTION 'CRN19: falta consumidor %',n USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- Fuente publicada por empleado: zona civil y origen que explican cada
-- asiento. No hay valor automático ni inferencia de la zona del resolutor.
CREATE TABLE vec_cronos_v1.correccion_fuente_version (
 version_ref text PRIMARY KEY CHECK (version_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
 zona_horaria text NOT NULL CHECK (zona_horaria IN ('Europe/Madrid','Atlantic/Canary')),
 canal_ref text NOT NULL CHECK (canal_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 origen_ref text NOT NULL CHECK (origen_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 calidad_ref text NOT NULL CHECK (calidad_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 fecha_desde date NOT NULL,
 fecha_hasta_exclusivo date NOT NULL CHECK (fecha_hasta_exclusivo>fecha_desde),
 publicada_en timestamptz(6) NOT NULL,
 UNIQUE (empleado_ref,version_ref)
);
CREATE INDEX correccion_fuente_empleado_idx ON vec_cronos_v1.correccion_fuente_version(empleado_ref,fecha_desde,fecha_hasta_exclusivo);
CREATE TABLE vec_cronos_v1.correccion_fuente_retirada (
 version_ref text PRIMARY KEY REFERENCES vec_cronos_v1.correccion_fuente_version,
 retirada_en timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
 publicada_en timestamptz(6) NOT NULL
);

-- Competencia específica de Correcciones, separada del circuito de permisos.
CREATE TABLE vec_cronos_v1.correccion_resolutor (
 asignacion_ref text PRIMARY KEY CHECK (asignacion_ref ~ '^resolutor:cronos:correccion:[-A-Za-z0-9_.:]{1,128}$'),
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 empleado_etiqueta text NOT NULL CHECK (char_length(empleado_etiqueta) BETWEEN 1 AND 120 AND empleado_etiqueta !~ '[[:cntrl:]]'),
 paso text NOT NULL CHECK (paso IN ('responsable','administracion')),
 resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
 sintetico boolean NOT NULL,
 fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
 publicada_en timestamptz(6) NOT NULL
);
CREATE INDEX correccion_resolutor_actor_idx ON vec_cronos_v1.correccion_resolutor(resolutor_ref,paso,empleado_ref);
CREATE TABLE vec_cronos_v1.correccion_resolutor_retirada (
 asignacion_ref text PRIMARY KEY REFERENCES vec_cronos_v1.correccion_resolutor,
 retirada_en timestamptz(6) NOT NULL,
 fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
 publicada_en timestamptz(6) NOT NULL
);

-- Cada actuación tiene versión, motivo, origen de competencia y consumo
-- V3 propios. La solicitud y el marcaje original siguen sin modificarse.
CREATE TABLE vec_cronos_v1.correccion_decision (
 decision_correccion_ref text PRIMARY KEY CHECK (decision_correccion_ref='correccion:cronos:actuacion:'||clave_operacion),
 clave_operacion text NOT NULL UNIQUE CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
 solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_solicitud,
 actuacion_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.correccion_actuacion,
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 paso text NOT NULL CHECK (paso IN ('decision_responsable','resolucion_rrhh','aplicacion')),
 resultado text CHECK (resultado IS NULL OR resultado IN ('favorable','desfavorable')),
 motivo text NOT NULL CHECK (char_length(motivo) BETWEEN 1 AND 500 AND motivo !~ '[[:cntrl:]]'),
 version_previa integer NOT NULL CHECK (version_previa BETWEEN 1 AND 3),
 version_resultante integer NOT NULL CHECK (version_resultante=version_previa+1),
 estado_resultante text NOT NULL CHECK (estado_resultante IN ('pendiente_rrhh','denegada_responsable','pendiente_aplicacion','denegada_rrhh','aplicada')),
 asignacion_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_resolutor,
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
 material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE (solicitud_ref,version_resultante),
 CHECK ((paso='decision_responsable' AND version_previa=1 AND estado_resultante IN ('pendiente_rrhh','denegada_responsable'))
    OR (paso='resolucion_rrhh' AND version_previa=2 AND estado_resultante IN ('pendiente_aplicacion','denegada_rrhh'))
    OR (paso='aplicacion' AND version_previa=3 AND estado_resultante='aplicada')),
 CHECK ((paso='aplicacion')=(resultado IS NULL)),
 CHECK (motivo=btrim(motivo))
);
CREATE INDEX correccion_decision_empleado_idx ON vec_cronos_v1.correccion_decision(empleado_ref,solicitud_ref);
CREATE TABLE vec_cronos_v1.correccion_asiento (
 asiento_ref text PRIMARY KEY CHECK (asiento_ref ~ '^correccion:asiento:[0-9a-f-]{36}$'),
 solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_solicitud,
 decision_correccion_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_decision,
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 tipo text NOT NULL CHECK (tipo IN ('anulacion','incorporacion')),
 marcaje_original_ref text REFERENCES vec_cronos_v1.marcaje_original,
 movimiento text CHECK (movimiento IS NULL OR movimiento IN ('entrada','salida','inicio_pausa','fin_pausa')),
 instante_utc timestamptz(6),
 fuente_version_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_fuente_version,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE (solicitud_ref,tipo),
 CHECK ((tipo='anulacion' AND marcaje_original_ref IS NOT NULL AND movimiento IS NULL AND instante_utc IS NULL)
    OR (tipo='incorporacion' AND marcaje_original_ref IS NULL AND movimiento IS NOT NULL AND instante_utc IS NOT NULL))
);
CREATE INDEX correccion_asiento_empleado_idx ON vec_cronos_v1.correccion_asiento(empleado_ref,instante_utc);
CREATE UNIQUE INDEX correccion_asiento_anulacion_original_idx ON vec_cronos_v1.correccion_asiento(marcaje_original_ref) WHERE tipo='anulacion';
CREATE TABLE vec_cronos_v1.correccion_bandeja_acceso (
 decision_ref text PRIMARY KEY,
 resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 paso text NOT NULL CHECK (paso IN ('responsable','administracion')),
 material_sha256 text NOT NULL,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.correccion_decision_replay (
 decision_ref text PRIMARY KEY,
 resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 decision_correccion_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_decision,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.correccion_historia_acceso (
 decision_ref text PRIMARY KEY,
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_solicitud,
 vista text NOT NULL CHECK (vista IN ('propia','equipo')),
 paso text CHECK (paso IS NULL OR paso IN ('responsable','administracion')),
 material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 consultada_en timestamptz(6) NOT NULL,
 CHECK ((vista='propia')=(paso IS NULL))
);

CREATE FUNCTION vec_cronos_v1.correccion_competente_v1(p_empleado text,p_paso text)
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_resolutor r
 WHERE r.resolutor_ref=nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'')
   AND r.empleado_ref=p_empleado AND (p_paso IS NULL OR r.paso=p_paso)
   AND r.vigente_desde<=clock_timestamp() AND (r.vigente_hasta IS NULL OR clock_timestamp()<r.vigente_hasta)
   AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=clock_timestamp()))
$f$;

ALTER TABLE vec_cronos_v1.correccion_fuente_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_fuente_version FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_fuente_version FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_fuente_version FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_fuente_retirada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_fuente_retirada FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_fuente_retirada FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_fuente_retirada FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_resolutor ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_resolutor FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_resolutor FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_resolutor FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_resolutor_retirada ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_resolutor_retirada FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_resolutor_retirada FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_resolutor_retirada FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_decision ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_decision FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_decision FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_decision FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_asiento ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_asiento FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_asiento FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_asiento FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_bandeja_acceso ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_bandeja_acceso FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_bandeja_acceso FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_bandeja_acceso FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_decision_replay ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_decision_replay FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_decision_replay FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_decision_replay FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;

ALTER TABLE vec_cronos_v1.correccion_historia_acceso ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.correccion_historia_acceso FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.correccion_historia_acceso FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.correccion_historia_acceso FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_publicada ON vec_cronos_v1.correccion_fuente_version FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY publicacion ON vec_cronos_v1.correccion_fuente_version FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
CREATE POLICY lectura_publicada ON vec_cronos_v1.correccion_fuente_retirada FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY publicacion ON vec_cronos_v1.correccion_fuente_retirada FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
CREATE POLICY lectura_asignada ON vec_cronos_v1.correccion_resolutor FOR SELECT TO vec_cronos_v1_propietario
 USING (resolutor_ref=nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),''));
CREATE POLICY publicacion ON vec_cronos_v1.correccion_resolutor FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
CREATE POLICY lectura_publicada ON vec_cronos_v1.correccion_resolutor_retirada FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY publicacion ON vec_cronos_v1.correccion_resolutor_retirada FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
CREATE POLICY lectura_resolutor ON vec_cronos_v1.correccion_decision FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.correccion_competente_v1(empleado_ref,NULL));
CREATE POLICY lectura_propia ON vec_cronos_v1.correccion_decision FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.correccion_decision FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (actor_ref=nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') AND vec_cronos_v1.correccion_competente_v1(empleado_ref,CASE paso WHEN 'decision_responsable' THEN 'responsable' ELSE 'administracion' END));
CREATE POLICY lectura_propia ON vec_cronos_v1.correccion_asiento FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY lectura_resolutor ON vec_cronos_v1.correccion_asiento FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.correccion_competente_v1(empleado_ref,NULL));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.correccion_asiento FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.correccion_competente_v1(empleado_ref,'administracion'));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.correccion_bandeja_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.correccion_decision_replay FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),''));
CREATE POLICY adicion_acceso ON vec_cronos_v1.correccion_historia_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK ((vista='propia' AND empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''))
   OR (vista='equipo' AND actor_ref=nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'')
       AND vec_cronos_v1.correccion_competente_v1(empleado_ref,paso)));
CREATE POLICY lectura_resolutor_correccion ON vec_cronos_v1.correccion_solicitud FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.correccion_competente_v1(empleado_ref,NULL));
CREATE POLICY lectura_resolutor_correccion ON vec_cronos_v1.correccion_actuacion FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.correccion_competente_v1(empleado_ref,NULL));
CREATE POLICY adicion_resolutor_correccion ON vec_cronos_v1.correccion_actuacion FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (version>1 AND vec_cronos_v1.correccion_competente_v1(empleado_ref,CASE paso WHEN 'decision_responsable' THEN 'responsable' ELSE 'administracion' END));
CREATE POLICY adicion_resolutor_correccion ON vec_cronos_v1.solicitud_outbox FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (tipo='cronos.correccion.actuada' AND vec_cronos_v1.correccion_competente_v1(empleado_ref,NULL));
ALTER TABLE vec_cronos_v1.solicitud_outbox DROP CONSTRAINT solicitud_outbox_tipo_check;
ALTER TABLE vec_cronos_v1.solicitud_outbox ADD CONSTRAINT solicitud_outbox_tipo_check CHECK (tipo IN
 ('cronos.correccion.solicitada','cronos.correccion.actuada','cronos.permiso.solicitado','cronos.permiso.resuelto','cronos.aviso.archivado'));
ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check CHECK (ruta IN (
 '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
 '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
 '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
 '/api/interna/cronos/correcciones/recibo','/api/interna/cronos/correcciones/bandeja',
 '/api/interna/cronos/correcciones/actuaciones','/api/interna/cronos/correcciones/historia',
 '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
 '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
 '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos','otra'));

-- La autoridad V3 se consume antes de leer filas personales. El contexto
-- propio sólo acredita a quien actúa; la asignación específica limita el
-- empleado afectado. Ninguna GUC viene del cliente ni del pool.
CREATE FUNCTION vec_cronos_v1.consumir_correccion_decision_v1(
 p_fachada text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_paso text,p_bloqueos text[],
 m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,OUT ahora timestamptz,
 OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record; b text;
BEGIN
 IF NOT ((p_paso='propia' AND p_fachada='consumir_cronos_historia_correccion_propia_v3_atestada')
   OR (p_paso='responsable' AND p_fachada IN ('consumir_cronos_bandeja_correcciones_v3_atestada',
      'consumir_cronos_historia_correccion_equipo_v3_atestada',
      'registrar_y_consumir_cronos_decision_responsable_v3_atestada'))
   OR (p_paso='administracion' AND p_fachada IN ('consumir_cronos_bandeja_correcciones_v3_atestada',
      'consumir_cronos_historia_correccion_equipo_v3_atestada',
      'registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada','registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada')))
   OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL
   OR nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL
   OR nullif(current_setting('vec.cronos.correccion_resolutor_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'frontera de corrección inválida' USING ERRCODE='PC003';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb;
   d:=convert_from(p_decision,'UTF8')::jsonb;
   x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'autorización Cronos ilegible' USING ERRCODE='PC003';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 huella:=CASE p_paso WHEN 'propia' THEN vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',material_sha256)
   ELSE vec_cronos_v1.huella_contexto_resolutor_v1(m->>'actor_ref',p_paso,material_sha256) END;
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,p_accion,p_tipo,p_finalidad,p_recurso,m->>'actor_ref',m->>'perfil_ref',huella);
 FOREACH b IN ARRAY coalesce(p_bloqueos,ARRAY[]::text[]) LOOP
   PERFORM pg_advisory_xact_lock(hashtextextended(b,0));
 END LOOP;
 EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',p_fachada) INTO STRICT consumo
   USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_recurso
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 IF p_paso='propia' THEN
   PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 ELSE
   PERFORM set_config('vec.cronos.correccion_resolutor_ref',m->>'actor_ref',true);
 END IF;
 decision_ref:=consumo.decision_ref;
 auditoria_ref:=consumo.auditoria_ref;
 consumo_huella_sha256:=consumo.consumo_huella_sha256;
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_bandeja_correcciones_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; paso text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','paso','zona_horaria']);
 paso:=m->>'paso';
 IF paso NOT IN ('responsable','administracion') THEN RAISE EXCEPTION 'paso inválido' USING ERRCODE='PC001'; END IF;
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_correccion_decision_v1(
   'consumir_cronos_bandeja_correcciones_v3_atestada','cronos.correccion.bandeja.consultar',
   'bandeja_correcciones','consultar_bandeja_correcciones','bandeja:cronos:correcciones:'||paso,paso,NULL,
   m,p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.correccion_bandeja_acceso(decision_ref,resolutor_ref,paso,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,m->>'actor_ref',paso,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('paso',paso,'pendientes',coalesce((SELECT jsonb_agg(jsonb_build_object(
   'solicitud_ref',s.solicitud_ref,'empleado_ref',s.empleado_ref,
   'empleado_etiqueta',(SELECT r.empleado_etiqueta FROM vec_cronos_v1.correccion_resolutor r
      WHERE r.empleado_ref=s.empleado_ref AND r.paso=paso AND r.resolutor_ref=m->>'actor_ref'
        AND r.vigente_desde<=k.ahora AND (r.vigente_hasta IS NULL OR k.ahora<r.vigente_hasta)
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_resolutor_retirada x
          WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=k.ahora)
      ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1),
   'marcaje_original_ref',coalesce(s.marcaje_original_ref,''),'hueco_declarado',s.hueco_declarado,
   'movimiento',s.movimiento,'fecha_civil',to_char(s.fecha_civil,'YYYY-MM-DD'),
   'hora_pretendida',s.hora_pretendida,'motivo_codigo',s.motivo_codigo,
   'estado',s.estado,'version',s.version,'solicitada_en',s.solicitada_en)
   ORDER BY s.solicitada_en,s.solicitud_ref)
   FROM (SELECT s0.*,a0.estado,a0.version FROM vec_cronos_v1.correccion_solicitud s0
     JOIN LATERAL (SELECT ca.estado,ca.version FROM vec_cronos_v1.correccion_actuacion ca
       WHERE ca.solicitud_ref=s0.solicitud_ref ORDER BY ca.version DESC LIMIT 1) a0 ON true
     WHERE s0.empleado_ref<>m->>'empleado_ref'
       AND vec_cronos_v1.correccion_competente_v1(s0.empleado_ref,paso)
       AND ((paso='responsable' AND a0.estado='pendiente_responsable')
          OR (paso='administracion' AND a0.estado IN ('pendiente_rrhh','pendiente_aplicacion')))
     ORDER BY s0.solicitada_en,s0.solicitud_ref LIMIT 501) s),'[]'::jsonb));
 IF jsonb_array_length(resultado->'pendientes')>500 THEN
   RAISE EXCEPTION 'bandeja Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.registrar_actuacion_correccion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; paso text; ambito text; fachada text; accion text; finalidad text;
 ref text; previa vec_cronos_v1.correccion_decision%ROWTYPE; sol vec_cronos_v1.correccion_solicitud%ROWTYPE;
 actual vec_cronos_v1.correccion_actuacion%ROWTYPE; fuente vec_cronos_v1.correccion_fuente_version%ROWTYPE;
 asignacion text; estado text; version_esperada integer; actuacion text; recibo text; instante timestamptz; fuentes integer;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','solicitud_ref',
  'clave_operacion','paso','resultado','motivo','version_esperada','zona_horaria']);
 paso:=m->>'paso';
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
   OR m->>'solicitud_ref' !~ '^correccion:cronos:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
   OR paso NOT IN ('decision_responsable','resolucion_rrhh','aplicacion')
   OR m->>'version_esperada' NOT IN ('1','2','3')
   OR (m->>'motivo') IS DISTINCT FROM btrim(m->>'motivo')
   OR char_length(m->>'motivo')>500 OR m->>'motivo' ~ '[[:cntrl:]]'
   OR m->>'motivo'=''
   OR (paso='aplicacion' AND (m->>'version_esperada'<>'3' OR m->>'resultado'<>''))
   OR (paso='decision_responsable' AND (m->>'version_esperada'<>'1' OR m->>'resultado' NOT IN ('favorable','desfavorable')))
   OR (paso='resolucion_rrhh' AND (m->>'version_esperada'<>'2' OR m->>'resultado' NOT IN ('favorable','desfavorable')))
   THEN
   RAISE EXCEPTION 'actuación Cronos inválida' USING ERRCODE='PC001';
 END IF;
 version_esperada:=(m->>'version_esperada')::integer;
 ambito:=CASE paso WHEN 'decision_responsable' THEN 'responsable' ELSE 'administracion' END;
 fachada:=CASE paso WHEN 'decision_responsable' THEN 'registrar_y_consumir_cronos_decision_responsable_v3_atestada'
   WHEN 'resolucion_rrhh' THEN 'registrar_y_consumir_cronos_resolucion_rrhh_v3_atestada'
   ELSE 'registrar_y_consumir_cronos_aplicacion_correccion_v3_atestada' END;
 accion:=CASE paso WHEN 'decision_responsable' THEN 'cronos.correccion.responsable.decidir'
   WHEN 'resolucion_rrhh' THEN 'cronos.correccion.rrhh.resolver' ELSE 'cronos.correccion.aplicar' END;
 finalidad:=CASE paso WHEN 'decision_responsable' THEN 'decidir_correccion_responsable'
   WHEN 'resolucion_rrhh' THEN 'resolver_correccion_rrhh' ELSE 'aplicar_correccion' END;
 ref:='correccion:cronos:actuacion:'||(m->>'clave_operacion');
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_correccion_decision_v1(
   fachada,accion,'actuacion_correccion',finalidad,ref,ambito,
   ARRAY['vec_cronos_v1:clave:'||(m->>'clave_operacion'),'vec_cronos_v1:solicitud:'||(m->>'solicitud_ref')],
   m,p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.correccion_decision WHERE decision_correccion_ref=ref;
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 OR
      NOT vec_cronos_v1.correccion_competente_v1(previa.empleado_ref,ambito) THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.correccion_decision_replay(decision_ref,resolutor_ref,decision_correccion_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,m->>'actor_ref',ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   SELECT * INTO STRICT actual FROM vec_cronos_v1.correccion_actuacion WHERE actuacion_ref=previa.actuacion_ref;
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('solicitud_ref',previa.solicitud_ref,'actuacion_ref',previa.actuacion_ref,
      'recibo_ref',previa.recibo_ref,'estado',previa.estado_resultante,'version',previa.version_resultante,
      'instante_utc',previa.registrada_en,'replay',true);
 END IF;
 SELECT * INTO sol FROM vec_cronos_v1.correccion_solicitud s
   WHERE s.solicitud_ref=m->>'solicitud_ref' AND vec_cronos_v1.correccion_competente_v1(s.empleado_ref,ambito);
 IF NOT FOUND OR sol.empleado_ref=m->>'empleado_ref' OR sol.actor_ref=m->>'actor_ref' THEN
   RAISE EXCEPTION 'corrección no competente' USING ERRCODE='PC012';
 END IF;
 SELECT r.asignacion_ref INTO asignacion FROM vec_cronos_v1.correccion_resolutor r
   WHERE r.empleado_ref=sol.empleado_ref AND r.paso=ambito AND r.resolutor_ref=m->>'actor_ref'
     AND r.vigente_desde<=k.ahora AND (r.vigente_hasta IS NULL OR k.ahora<r.vigente_hasta)
     AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=k.ahora)
   ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1;
 IF asignacion IS NULL THEN RAISE EXCEPTION 'corrección no competente' USING ERRCODE='PC012'; END IF;
 SELECT * INTO STRICT actual FROM vec_cronos_v1.correccion_actuacion a
   WHERE a.solicitud_ref=sol.solicitud_ref ORDER BY a.version DESC LIMIT 1;
 IF actual.version<>version_esperada THEN RAISE EXCEPTION 'versión Cronos en conflicto' USING ERRCODE='PC011'; END IF;
 IF (paso='decision_responsable' AND actual.estado='pendiente_responsable') THEN
   estado:=CASE m->>'resultado' WHEN 'favorable' THEN 'pendiente_rrhh' ELSE 'denegada_responsable' END;
 ELSIF (paso='resolucion_rrhh' AND actual.estado='pendiente_rrhh') THEN
   IF EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_decision d
     WHERE d.solicitud_ref=sol.solicitud_ref AND d.actor_ref=m->>'actor_ref') THEN
      RAISE EXCEPTION 'corrección no competente' USING ERRCODE='PC012';
   END IF;
   estado:=CASE m->>'resultado' WHEN 'favorable' THEN 'pendiente_aplicacion' ELSE 'denegada_rrhh' END;
 ELSIF (paso='aplicacion' AND actual.estado='pendiente_aplicacion') THEN
   IF EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_decision d
     WHERE d.solicitud_ref=sol.solicitud_ref AND d.actor_ref=m->>'actor_ref' AND d.paso='decision_responsable') THEN
      RAISE EXCEPTION 'corrección no competente' USING ERRCODE='PC012';
   END IF;
   estado:='aplicada';
 ELSE RAISE EXCEPTION 'estado Cronos en conflicto' USING ERRCODE='PC011'; END IF;
 IF paso='aplicacion' THEN
   SELECT count(*) INTO fuentes FROM vec_cronos_v1.correccion_fuente_version f
     WHERE f.empleado_ref=sol.empleado_ref AND sol.fecha_civil>=f.fecha_desde
       AND sol.fecha_civil<f.fecha_hasta_exclusivo AND f.publicada_en<=k.ahora
       AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_fuente_retirada x
         WHERE x.version_ref=f.version_ref AND x.retirada_en<=k.ahora);
   IF fuentes<>1 THEN RAISE EXCEPTION 'fuente de corrección ausente o ambigua' USING ERRCODE='PC008'; END IF;
   SELECT * INTO STRICT fuente FROM vec_cronos_v1.correccion_fuente_version f
     WHERE f.empleado_ref=sol.empleado_ref AND sol.fecha_civil>=f.fecha_desde
       AND sol.fecha_civil<f.fecha_hasta_exclusivo AND f.publicada_en<=k.ahora
       AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_fuente_retirada x
         WHERE x.version_ref=f.version_ref AND x.retirada_en<=k.ahora);
   instante:=(sol.fecha_civil::text||' '||sol.hora_pretendida)::timestamp AT TIME ZONE fuente.zona_horaria;
   IF to_char(instante AT TIME ZONE fuente.zona_horaria,'YYYY-MM-DD HH24:MI') IS DISTINCT FROM
      to_char(sol.fecha_civil,'YYYY-MM-DD')||' '||sol.hora_pretendida
      OR to_char((instante+interval '1 hour') AT TIME ZONE fuente.zona_horaria,'YYYY-MM-DD HH24:MI') IS NOT DISTINCT FROM
         to_char(sol.fecha_civil,'YYYY-MM-DD')||' '||sol.hora_pretendida
      OR to_char((instante-interval '1 hour') AT TIME ZONE fuente.zona_horaria,'YYYY-MM-DD HH24:MI') IS NOT DISTINCT FROM
         to_char(sol.fecha_civil,'YYYY-MM-DD')||' '||sol.hora_pretendida THEN
      RAISE EXCEPTION 'hora civil ambigua o inexistente' USING ERRCODE='PC001';
   END IF;
 END IF;
 actuacion:='correccion:actuacion:'||gen_random_uuid()::text;
 recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.correccion_actuacion(actuacion_ref,solicitud_ref,empleado_ref,version,paso,estado,recibo_ref,registrada_en)
 VALUES(actuacion,sol.solicitud_ref,sol.empleado_ref,version_esperada+1,paso,estado,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.correccion_decision(decision_correccion_ref,clave_operacion,solicitud_ref,actuacion_ref,empleado_ref,paso,resultado,motivo,
   version_previa,version_resultante,estado_resultante,asignacion_ref,actor_ref,perfil_ref,material_sha256,decision_ref,auditoria_ref,
   consumo_huella_sha256,recibo_ref,registrada_en)
 VALUES(ref,m->>'clave_operacion',sol.solicitud_ref,actuacion,sol.empleado_ref,paso,nullif(m->>'resultado',''),nullif(m->>'motivo',''),
   version_esperada,version_esperada+1,estado,asignacion,m->>'actor_ref',m->>'perfil_ref',k.material_sha256,k.decision_ref,k.auditoria_ref,
   k.consumo_huella_sha256,recibo,k.ahora);
 IF paso='aplicacion' THEN
   IF sol.marcaje_original_ref IS NOT NULL THEN
     INSERT INTO vec_cronos_v1.correccion_asiento(asiento_ref,solicitud_ref,decision_correccion_ref,empleado_ref,tipo,marcaje_original_ref,
       fuente_version_ref,registrada_en)
     VALUES('correccion:asiento:'||gen_random_uuid()::text,sol.solicitud_ref,ref,sol.empleado_ref,'anulacion',sol.marcaje_original_ref,
       fuente.version_ref,k.ahora);
   END IF;
   INSERT INTO vec_cronos_v1.correccion_asiento(asiento_ref,solicitud_ref,decision_correccion_ref,empleado_ref,tipo,movimiento,instante_utc,
       fuente_version_ref,registrada_en)
   VALUES('correccion:asiento:'||gen_random_uuid()::text,sol.solicitud_ref,ref,sol.empleado_ref,'incorporacion',sol.movimiento,instante,
       fuente.version_ref,k.ahora);
 END IF;
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,sol.solicitud_ref,sol.empleado_ref,'cronos.correccion.actuada',
   jsonb_build_object('solicitud_ref',sol.solicitud_ref,'actuacion_ref',actuacion,'recibo_ref',recibo,
      'paso',paso,'estado',estado,'version',version_esperada+1),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('solicitud_ref',sol.solicitud_ref,'actuacion_ref',actuacion,'recibo_ref',recibo,
   'estado',estado,'version',version_esperada+1,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_historia_correccion_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; s vec_cronos_v1.correccion_solicitud%ROWTYPE;
 a vec_cronos_v1.correccion_actuacion%ROWTYPE; vista text; paso text; ambito text;
 fachada text; accion text; tipo text; finalidad text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref',
   'solicitud_ref','vista','paso','zona_horaria']);
 vista:=m->>'vista'; paso:=m->>'paso';
 IF m->>'solicitud_ref' !~ '^correccion:cronos:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR NOT ((vista='propia' AND paso='') OR (vista='equipo' AND paso IN ('responsable','administracion'))) THEN
   RAISE EXCEPTION 'historia Cronos inválida' USING ERRCODE='PC001';
 END IF;
 ambito:=CASE vista WHEN 'propia' THEN 'propia' ELSE paso END;
 fachada:=CASE vista WHEN 'propia' THEN 'consumir_cronos_historia_correccion_propia_v3_atestada'
   ELSE 'consumir_cronos_historia_correccion_equipo_v3_atestada' END;
 accion:=CASE vista WHEN 'propia' THEN 'cronos.correccion.historia.propia.consultar'
   ELSE 'cronos.correccion.historia.equipo.consultar' END;
 tipo:=CASE vista WHEN 'propia' THEN 'historia_correccion_propia' ELSE 'historia_correccion_equipo' END;
 finalidad:=CASE vista WHEN 'propia' THEN 'consultar_historia_correccion_propia'
   ELSE 'consultar_historia_correccion_equipo' END;
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_correccion_decision_v1(
   fachada,accion,tipo,finalidad,'historia:cronos:'||(m->>'solicitud_ref'),ambito,NULL,
   m,p_material,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO s FROM vec_cronos_v1.correccion_solicitud cs WHERE cs.solicitud_ref=m->>'solicitud_ref'
   AND ((vista='propia' AND cs.empleado_ref=m->>'empleado_ref')
     OR (vista='equipo' AND cs.empleado_ref<>m->>'empleado_ref' AND vec_cronos_v1.correccion_competente_v1(cs.empleado_ref,paso)));
 IF NOT FOUND THEN
   IF vista='propia' THEN RAISE EXCEPTION 'historia propia no autorizada' USING ERRCODE='PC004'; END IF;
   RAISE EXCEPTION 'historia de equipo no competente' USING ERRCODE='PC012';
 END IF;
 SELECT * INTO STRICT a FROM vec_cronos_v1.correccion_actuacion ca
   WHERE ca.solicitud_ref=s.solicitud_ref ORDER BY ca.version DESC LIMIT 1;
 INSERT INTO vec_cronos_v1.correccion_historia_acceso(decision_ref,actor_ref,empleado_ref,solicitud_ref,vista,paso,
   material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,m->>'actor_ref',s.empleado_ref,s.solicitud_ref,vista,nullif(paso,''),
   k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('solicitud_ref',s.solicitud_ref,'empleado_ref',s.empleado_ref,
   'estado',a.estado,'version',a.version,'fecha_civil',to_char(s.fecha_civil,'YYYY-MM-DD'),
   'hora_pretendida',s.hora_pretendida,'movimiento',s.movimiento,'motivo_codigo',s.motivo_codigo,
   'actuaciones',coalesce((SELECT jsonb_agg(
     jsonb_build_object('paso',h.paso,'resultado',coalesce(d.resultado,''),'motivo',coalesce(d.motivo,''),
       'estado',h.estado,'version',h.version,'recibo_ref',h.recibo_ref,'instante_utc',h.registrada_en)
     || CASE WHEN vista='equipo' THEN jsonb_build_object('actor_ref',coalesce(d.actor_ref,s.actor_ref),
          'perfil_ref',coalesce(d.perfil_ref,s.perfil_ref)) ELSE '{}'::jsonb END
     ORDER BY h.version)
     FROM vec_cronos_v1.correccion_actuacion h
     LEFT JOIN vec_cronos_v1.correccion_decision d ON d.actuacion_ref=h.actuacion_ref
     WHERE h.solicitud_ref=s.solicitud_ref),'[]'::jsonb),
   'asientos',coalesce((SELECT jsonb_agg(jsonb_build_object('tipo',c.tipo,
     'marcaje_original_ref',coalesce(c.marcaje_original_ref,''),'movimiento',coalesce(c.movimiento,''),
     'instante_utc',c.instante_utc,'fuente_version_ref',c.fuente_version_ref)
     ORDER BY c.registrada_en,c.tipo) FROM vec_cronos_v1.correccion_asiento c
     WHERE c.solicitud_ref=s.solicitud_ref),'[]'::jsonb));
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

-- Proyección efectiva explicable: los dos asientos de una edición anulan
-- visualmente el hecho original y aportan el nuevo. Nada reescribe historia.
CREATE FUNCTION vec_cronos_v1.marcajes_efectivos_v1(p_empleado text)
RETURNS TABLE(marcaje_ref text,empleado_ref text,movimiento text,instante_utc timestamptz,material text,tipo_origen text)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT o.marcaje_ref,o.empleado_ref,o.movimiento,o.instante_utc,o.material,o.tipo_origen
 FROM vec_cronos_v1.marcaje_original o
 WHERE o.empleado_ref=p_empleado AND p_empleado=nullif(current_setting('vec.cronos.empleado_ref',true),'')
   AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.correccion_asiento a
      WHERE a.empleado_ref=o.empleado_ref AND a.tipo='anulacion' AND a.marcaje_original_ref=o.marcaje_ref)
 UNION ALL
 SELECT a.asiento_ref,a.empleado_ref,a.movimiento,a.instante_utc,
   jsonb_build_object('canal',jsonb_build_object('politica_version_ref',f.version_ref,
      'canal_ref',f.canal_ref,'origen_ref',f.origen_ref,'calidad_ref',f.calidad_ref))::text,NULL::text
 FROM vec_cronos_v1.correccion_asiento a
 JOIN vec_cronos_v1.correccion_fuente_version f ON f.version_ref=a.fuente_version_ref
 WHERE a.empleado_ref=p_empleado AND p_empleado=nullif(current_setting('vec.cronos.empleado_ref',true),'')
   AND a.tipo='incorporacion'
$f$;

-- Reanclaje exacto de dos lectores instalados. Conserva OID, ACL, firma y
-- cuerpo restante; sólo cambia la fuente de marcajes por la proyección.
DO $proyeccion_pre$
DECLARE f oid; original text; metadatos jsonb; dependencias jsonb;
BEGIN
 f:='vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)'::regprocedure;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,metadatos FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'0704a06aa128516aac326ca09da184250720e232e06e05f903f4085241c2e56a' OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>current_user::regrole
 THEN RAISE EXCEPTION 'CRN19: preimagen de proyección 0 incompatible' USING ERRCODE='55000'; END IF;
 dependencias:=(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f);
 PERFORM set_config('vec_crn19.preimagen_0',jsonb_build_object('metadatos',metadatos,'dependencias',dependencias)::text,true);
 f:='vec_cronos_v1.consultar_movimientos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,metadatos FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'e99811f1b92ca8ae8749fd90c37094db3ca2e8350fbb7014fb70ef02a469b13a' OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>current_user::regrole
 THEN RAISE EXCEPTION 'CRN19: preimagen de proyección 1 incompatible' USING ERRCODE='55000'; END IF;
 dependencias:=(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f);
 PERFORM set_config('vec_crn19.preimagen_1',jsonb_build_object('metadatos',metadatos,'dependencias',dependencias)::text,true);
 f:='vec_cronos_v1.secuencia_remota_v1(text,timestamp with time zone)'::regprocedure;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc' INTO STRICT original,metadatos FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'be609c61bf643cafd3007d4a88d732be4e417f4da88dddd99857879564fc1e3e' OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f)<>current_user::regrole
 THEN RAISE EXCEPTION 'CRN19: preimagen de proyección 2 incompatible' USING ERRCODE='55000'; END IF;
 dependencias:=(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f);
 PERFORM set_config('vec_crn19.preimagen_2',jsonb_build_object('metadatos',metadatos,'dependencias',dependencias)::text,true);
END $proyeccion_pre$;

CREATE OR REPLACE FUNCTION vec_cronos_v1.consultar_libro_saldo_interno_v1(p_empleado_ref text, p_desde date, p_hasta date, p_zona_horaria text)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE jornadas jsonb; marcajes jsonb; movimientos jsonb; pendientes integer;
BEGIN
    IF p_empleado_ref IS NULL OR p_empleado_ref !~ '^emp_[-A-Za-z0-9_]{22,128}$'
       OR p_desde IS NULL OR p_hasta IS NULL OR p_hasta<p_desde OR p_hasta-p_desde>366
       OR p_zona_horaria IS NULL OR p_zona_horaria NOT IN ('Europe/Madrid','Atlantic/Canary')
       OR p_empleado_ref IS DISTINCT FROM nullif(current_setting('vec.cronos.empleado_ref',true),'') THEN
        RAISE EXCEPTION 'consulta Cronos inválida' USING ERRCODE='PC003';
    END IF;
    -- Una segunda versión no sustituye silenciosamente el asiento histórico:
    -- esa fecha queda no disponible hasta un recálculo gobernado y versionado.
    WITH vigente AS (
      SELECT p.programacion_ref,p.fecha,p.turno_ref,p.politica_version_ref,
             p.fuente_ref,p.zona_horaria,p.minutos_previstos,p.version
      FROM vec_cronos_v1.programacion_jornada p
      WHERE p.empleado_ref=p_empleado_ref AND p.fecha BETWEEN p_desde AND p_hasta
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada p2
          WHERE p2.empleado_ref=p.empleado_ref AND p2.fecha=p.fecha
            AND p2.programacion_ref<>p.programacion_ref)
    )
    SELECT coalesce(jsonb_agg(jsonb_build_object('programacion_ref',programacion_ref,'fecha',fecha,
      'turno_ref',turno_ref,'politica_version_ref',politica_version_ref,'fuente_ref',fuente_ref,
      'zona_horaria',zona_horaria,'minutos_previstos',minutos_previstos,'version',version)
      ORDER BY fecha),'[]'::jsonb) INTO jornadas FROM vigente;
    -- Los hechos limítrofes permiten parejas nocturnas. También se incluye
    -- el último hecho anterior si deja una secuencia abierta, aunque sea
    -- más antiguo que el margen. La zona local nunca depende del pool.
    SELECT coalesce(jsonb_agg(jsonb_build_object('marcaje_ref',m.marcaje_ref,'movimiento',m.movimiento,
      'instante_utc',m.instante_utc,'canal',m.material::jsonb->'canal',
      'tipo_origen',m.tipo_origen) ORDER BY m.instante_utc,m.marcaje_ref),'[]'::jsonb)
      INTO marcajes FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref) m
      WHERE m.empleado_ref=p_empleado_ref
        AND ((m.instante_utc>=((p_desde-2)::timestamp AT TIME ZONE p_zona_horaria)
          AND m.instante_utc<((p_hasta+3)::timestamp AT TIME ZONE p_zona_horaria))
          OR (m.movimiento IN ('entrada','inicio_pausa','fin_pausa')
            AND m.marcaje_ref=(SELECT anterior.marcaje_ref
              FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref) anterior
              WHERE anterior.empleado_ref=p_empleado_ref
                AND anterior.instante_utc<(p_desde::timestamp AT TIME ZONE p_zona_horaria)
              ORDER BY anterior.instante_utc DESC,anterior.marcaje_ref DESC LIMIT 1)));
    WITH ordenados AS (
      SELECT marcaje_ref,movimiento,instante_utc,
        lead(marcaje_ref) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_ref,
        lead(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_movimiento,
        lead(instante_utc) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_utc
      FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref)
      WHERE empleado_ref=p_empleado_ref
        -- Una pareja que aporte tiempo al intervalo dura como máximo 24 h.
        -- El sucesor puede estar fuera del intervalo local; incluir 24 h UTC
        -- a ambos lados conserva su LEAD y los empates por referencia.
        AND instante_utc >= (p_desde::timestamp AT TIME ZONE p_zona_horaria) - interval '24 hours'
        AND instante_utc < ((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria) + interval '24 hours'
    ), tramos AS (
      SELECT marcaje_ref,siguiente_ref,instante_utc,siguiente_utc FROM ordenados
      WHERE (movimiento='entrada' AND siguiente_movimiento IN ('salida','inicio_pausa'))
         OR (movimiento='fin_pausa' AND siguiente_movimiento IN ('salida','inicio_pausa'))
    ), partidos AS (
      SELECT t.marcaje_ref,t.siguiente_ref,g.fecha::date AS fecha,
        (extract(epoch FROM (least(t.siguiente_utc,(((g.fecha::date+1)::timestamp) AT TIME ZONE p_zona_horaria))
                  - greatest(t.instante_utc,((g.fecha::date)::timestamp AT TIME ZONE p_zona_horaria))))*1000000)::bigint AS microsegundos
      FROM tramos t CROSS JOIN LATERAL generate_series(
        greatest(p_desde,(t.instante_utc AT TIME ZONE p_zona_horaria)::date),
        least(p_hasta,((t.siguiente_utc - interval '1 microsecond') AT TIME ZONE p_zona_horaria)::date),
        interval '1 day') g(fecha)
      WHERE t.siguiente_utc>t.instante_utc AND t.siguiente_utc-t.instante_utc<=interval '24 hours'
    ), trabajo AS (
      SELECT fecha,sum(microsegundos)::bigint AS microsegundos,
        array_agg(marcaje_ref||'/'||siguiente_ref ORDER BY marcaje_ref,siguiente_ref) AS fuentes
      FROM partidos WHERE microsegundos>0 GROUP BY fecha
    ), previsto AS (
      SELECT p.fecha,p.programacion_ref,p.version,p.minutos_previstos
      FROM vec_cronos_v1.programacion_jornada p
      WHERE p.empleado_ref=p_empleado_ref AND p.fecha BETWEEN p_desde AND p_hasta
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.programacion_jornada p2
          WHERE p2.empleado_ref=p.empleado_ref AND p2.fecha=p.fecha
            AND p2.programacion_ref<>p.programacion_ref)
    ), libro AS (
      SELECT fecha,'trabajado'::text AS tipo,microsegundos AS delta_microsegundos,
        to_jsonb(fuentes) AS fuentes FROM trabajo
      UNION ALL
      SELECT fecha,'previsto',-(minutos_previstos::bigint*60000000),jsonb_build_array(programacion_ref||'/v'||version)
      FROM previsto
    )
    SELECT coalesce(jsonb_agg(jsonb_build_object('fecha',fecha,'tipo',tipo,
       'delta_microsegundos',delta_microsegundos,'fuentes',fuentes) ORDER BY fecha,tipo),'[]'::jsonb)
      INTO movimientos FROM libro;
    -- No dar saldo definitivo cuando hay una secuencia inválida o falta
    -- programación. El lector verá hechos y un estado explícito.
    WITH seleccion AS (
      -- Los dos últimos anteriores aportan el hecho que aún solapa B y su
      -- LAG real, incluso si la apertura ocurrió años antes.
      SELECT marcaje_ref,movimiento,instante_utc FROM (
        SELECT marcaje_ref,movimiento,instante_utc
        FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref)
        WHERE empleado_ref=p_empleado_ref
          AND instante_utc<(p_desde::timestamp AT TIME ZONE p_zona_horaria)
        ORDER BY instante_utc DESC,marcaje_ref DESC LIMIT 2
      ) anteriores
      UNION ALL
      SELECT marcaje_ref,movimiento,instante_utc
      FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref)
      WHERE empleado_ref=p_empleado_ref
        AND instante_utc>=(p_desde::timestamp AT TIME ZONE p_zona_horaria)
        AND instante_utc<((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
      UNION ALL
      -- El primer sucesor exterior es el LEAD real del último hecho de B.
      SELECT marcaje_ref,movimiento,instante_utc FROM (
        SELECT marcaje_ref,movimiento,instante_utc
        FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref)
        WHERE empleado_ref=p_empleado_ref
          AND instante_utc>=((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
        ORDER BY instante_utc,marcaje_ref LIMIT 1
      ) sucesor
    ), o AS (
      SELECT movimiento,lead(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente,
       lag(movimiento) OVER (ORDER BY instante_utc,marcaje_ref) AS anterior,
       lead(instante_utc) OVER (ORDER BY instante_utc,marcaje_ref) AS siguiente_utc,
       instante_utc FROM seleccion
    )
    -- Conserva la condición de solape original: una entrada antigua abierta
    -- sigue afectando al día pedido aunque no haya hechos recientes.
    SELECT count(*) INTO pendientes FROM o
    WHERE instante_utc<((p_hasta+1)::timestamp AT TIME ZONE p_zona_horaria)
      AND coalesce(siguiente_utc,'infinity'::timestamptz)>(p_desde::timestamp AT TIME ZONE p_zona_horaria)
      AND ((movimiento IN ('entrada','fin_pausa') AND
          (siguiente IS DISTINCT FROM 'salida' AND siguiente IS DISTINCT FROM 'inicio_pausa'
           OR siguiente_utc<=instante_utc OR siguiente_utc-instante_utc>interval '24 hours'))
        OR (movimiento='inicio_pausa' AND siguiente IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='salida' AND anterior IS DISTINCT FROM 'entrada' AND anterior IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='inicio_pausa' AND anterior IS DISTINCT FROM 'entrada' AND anterior IS DISTINCT FROM 'fin_pausa')
        OR (movimiento='fin_pausa' AND anterior IS DISTINCT FROM 'inicio_pausa'));
    RETURN jsonb_build_object('empleado_ref',p_empleado_ref,'desde',p_desde,'hasta',p_hasta,
      'zona_horaria',p_zona_horaria,'jornadas',jornadas,'marcajes',marcajes,
      'movimientos_saldo',movimientos,'completo',
      pendientes=0 AND jsonb_array_length(jornadas)=p_hasta-p_desde+1
      AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(jornadas) j
        WHERE j->>'zona_horaria' IS DISTINCT FROM p_zona_horaria));
END $function$;

CREATE OR REPLACE FUNCTION vec_cronos_v1.consultar_movimientos_propio_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
#variable_conflict use_variable
DECLARE m jsonb; k record; desde date; hasta date; emp text; zona text; anios integer[]; sin_calendario boolean; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','desde','hasta','zona_horaria']);
 desde:=vec_cronos_v1.fecha_material_v1(m->>'desde'); hasta:=vec_cronos_v1.fecha_material_v1(m->>'hasta');
 IF hasta<desde OR hasta-desde>366 THEN
   RAISE EXCEPTION 'periodo Cronos inválido' USING ERRCODE='PC001';
 END IF;
 emp:=m->>'empleado_ref'; zona:=m->>'zona_horaria';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_propio_v1('consumir_cronos_movimientos_propio_v3_atestada','cronos.movimientos.propio.consultar',
   'movimientos_propio','consultar_movimientos_propio','movimientos:cronos:'||emp,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.movimientos_acceso(decision_ref,empleado_ref,desde,hasta,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,emp,desde,hasta,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 anios:=ARRAY(SELECT generate_series(extract(year FROM desde)::integer,extract(year FROM hasta)::integer));
 sin_calendario:=EXISTS (SELECT 1 FROM unnest(anios) a WHERE vec_cronos_v1.calendario_de_v1(emp,a) IS NULL);
 resultado:=jsonb_build_object(
  'empleado_ref',emp,'desde',to_char(desde,'YYYY-MM-DD'),'hasta',to_char(hasta,'YYYY-MM-DD'),'zona_horaria',zona,
  'calendario',jsonb_build_object('disponible',NOT sin_calendario,'dias',CASE WHEN sin_calendario THEN '[]'::jsonb ELSE coalesce((
     SELECT jsonb_agg(jsonb_build_object('fecha',to_char(cd.fecha,'YYYY-MM-DD'),'tipo',cd.tipo,'nombre',cd.nombre) ORDER BY cd.fecha)
       FROM vec_cronos_v1.calendario_dia cd
      WHERE cd.fecha BETWEEN desde AND hasta AND cd.calendario_ref=vec_cronos_v1.calendario_de_v1(emp,extract(year FROM cd.fecha)::integer)),'[]'::jsonb) END),
  'marcajes_por_dia',coalesce((SELECT jsonb_agg(jsonb_build_object('fecha',to_char(f,'YYYY-MM-DD'),'marcajes',n) ORDER BY f) FROM (
     SELECT (mo.instante_utc AT TIME ZONE zona)::date f,count(*) n FROM vec_cronos_v1.marcajes_efectivos_v1(emp) mo
      WHERE mo.empleado_ref=emp AND (mo.instante_utc AT TIME ZONE zona)::date BETWEEN desde AND hasta
        AND mo.instante_utc>=(desde-1)::timestamp AT TIME ZONE 'UTC' AND mo.instante_utc<(hasta+2)::timestamp AT TIME ZONE 'UTC'
      GROUP BY 1) q),'[]'::jsonb),
  'absentismos',coalesce((SELECT jsonb_agg(jsonb_build_object('solicitud_ref',s.solicitud_ref,'permiso_ref',s.permiso_ref,'nombre',pc.nombre,
       'desde',to_char(s.desde,'YYYY-MM-DD'),'hasta',to_char(s.hasta,'YYYY-MM-DD'),'cantidad',s.cantidad,'unidad',s.unidad,
       'pendiente_justificar',e.pendiente_justificar) ORDER BY s.desde,s.solicitud_ref)
     FROM vec_cronos_v1.permiso_solicitud s JOIN vec_cronos_v1.estado_permiso_actual_v1(emp) e ON e.solicitud_ref=s.solicitud_ref
     JOIN vec_cronos_v1.permiso_catalogo pc ON pc.version_ref=s.catalogo_version_ref
    WHERE s.empleado_ref=emp AND e.estado='concedido' AND s.desde<=hasta AND s.hasta>=desde),'[]'::jsonb),
  'correcciones',coalesce((SELECT jsonb_agg(jsonb_build_object('solicitud_ref',cs.solicitud_ref,'fecha_civil',to_char(cs.fecha_civil,'YYYY-MM-DD'),
       'hora_pretendida',cs.hora_pretendida,'movimiento',cs.movimiento,'estado',a.estado,'version',a.version,'solicitada_en',cs.solicitada_en)
       ORDER BY cs.fecha_civil,cs.hora_pretendida,cs.solicitud_ref)
     FROM vec_cronos_v1.correccion_solicitud cs
     JOIN LATERAL (SELECT ca.estado,ca.version FROM vec_cronos_v1.correccion_actuacion ca WHERE ca.solicitud_ref=cs.solicitud_ref ORDER BY ca.version DESC LIMIT 1) a ON true
    WHERE cs.empleado_ref=emp AND cs.fecha_civil BETWEEN desde AND hasta),'[]'::jsonb));
 IF clock_timestamp()>=k.vence THEN
   RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003';
 END IF;
 RETURN resultado;
END $function$;

CREATE OR REPLACE FUNCTION vec_cronos_v1.secuencia_remota_v1(p_empleado text, p_desde timestamp with time zone, OUT continuidad boolean, OUT permitidos text[], OUT ultimo_instante timestamp with time zone)
 RETURNS record
 LANGUAGE plpgsql
 STABLE
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE ultimo record; hay boolean;
BEGIN
 SELECT m.movimiento,m.instante_utc INTO ultimo FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado) m
  WHERE m.empleado_ref=p_empleado ORDER BY m.instante_utc DESC,m.marcaje_ref DESC LIMIT 1;
 hay:=FOUND;
 ultimo_instante:=CASE WHEN hay THEN ultimo.instante_utc END;
 IF NOT hay OR ultimo.movimiento='salida' THEN
   continuidad:=true; permitidos:=ARRAY['entrada'];
 ELSIF ultimo.instante_utc<p_desde THEN
   continuidad:=false; permitidos:=ARRAY[]::text[];
 ELSIF ultimo.movimiento IN ('entrada','fin_pausa') THEN
   continuidad:=true; permitidos:=ARRAY['salida','inicio_pausa'];
 ELSE
   continuidad:=true; permitidos:=ARRAY['fin_pausa'];
 END IF;
END $function$;

DO $proyeccion_post$
DECLARE f oid; metadatos jsonb; dependencias jsonb; esperado jsonb;
BEGIN
 f:='vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)'::regprocedure;
 esperado:=current_setting('vec_crn19.preimagen_0')::jsonb;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT metadatos FROM pg_catalog.pg_proc p WHERE p.oid=f;
 dependencias:=(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f);
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'8f174843e8931a3ed0fdadecbf2216f555ca0304bc6fad5db070bb3a11580909'
 OR metadatos IS DISTINCT FROM jsonb_set(esperado->'metadatos','{proconfig}','["search_path=pg_catalog, pg_temp", "row_security=on", "TimeZone=UTC", "lock_timeout=2s"]'::jsonb)
 OR dependencias IS DISTINCT FROM esperado->'dependencias'
 THEN RAISE EXCEPTION 'CRN19: postimagen de proyección 0 divergente' USING ERRCODE='55000'; END IF;
 f:='vec_cronos_v1.consultar_movimientos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 esperado:=current_setting('vec_crn19.preimagen_1')::jsonb;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT metadatos FROM pg_catalog.pg_proc p WHERE p.oid=f;
 dependencias:=(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f);
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'878a8253d6cb0025966c2f92632e4073b202243de8e91e9d8c374d3dbfb4e108'
 OR metadatos IS DISTINCT FROM jsonb_set(esperado->'metadatos','{proconfig}','["search_path=pg_catalog, pg_temp", "row_security=on", "TimeZone=UTC", "lock_timeout=2s"]'::jsonb)
 OR dependencias IS DISTINCT FROM esperado->'dependencias'
 THEN RAISE EXCEPTION 'CRN19: postimagen de proyección 1 divergente' USING ERRCODE='55000'; END IF;
 f:='vec_cronos_v1.secuencia_remota_v1(text,timestamp with time zone)'::regprocedure;
 esperado:=current_setting('vec_crn19.preimagen_2')::jsonb;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT metadatos FROM pg_catalog.pg_proc p WHERE p.oid=f;
 dependencias:=(SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_catalog.pg_depend d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f);
 IF encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'a87f2790c92c24f57a04eea2adae0c3aef6701baaa0905e5c0aae6dc38609350'
 OR metadatos IS DISTINCT FROM jsonb_set(esperado->'metadatos','{proconfig}','["search_path=pg_catalog, pg_temp"]'::jsonb)
 OR dependencias IS DISTINCT FROM esperado->'dependencias'
 THEN RAISE EXCEPTION 'CRN19: postimagen de proyección 2 divergente' USING ERRCODE='55000'; END IF;
END $proyeccion_post$;

DO $acl$
DECLARE f regprocedure; tabla text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_cronos_v1.correccion_competente_v1(text,text)'::regprocedure,
  'vec_cronos_v1.consumir_correccion_decision_v1(text,text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_cronos_v1.marcajes_efectivos_v1(text)'::regprocedure] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f::text);
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
  'vec_cronos_v1.consultar_bandeja_correcciones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_cronos_v1.registrar_actuacion_correccion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_cronos_v1.consultar_historia_correccion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f::text);
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_ejecutor',f::text);
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['correccion_fuente_version','correccion_fuente_retirada','correccion_resolutor','correccion_resolutor_retirada',
  'correccion_decision','correccion_asiento','correccion_bandeja_acceso','correccion_decision_replay','correccion_historia_acceso'] LOOP
   EXECUTE format('REVOKE ALL ON TYPE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
END $acl$;
COMMIT;

