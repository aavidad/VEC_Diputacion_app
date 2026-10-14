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

DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['correccion_fuente_version','correccion_fuente_retirada','correccion_resolutor','correccion_resolutor_retirada',
   'correccion_decision','correccion_asiento','correccion_bandeja_acceso','correccion_decision_replay','correccion_historia_acceso'] LOOP
   EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
   EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
   EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
   EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
END $seguridad$;
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
DO $proyeccion$
DECLARE f oid; original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid;
BEGIN
 f:='vec_cronos_v1.consultar_libro_saldo_interno_v1(text,date,date,text)'::regprocedure;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner INTO STRICT original,meta,acl,propietario FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
      <>'37d8975b2beb7676f27442dcacd6440442aa8b4c40494c5fd12d066295407375'
    OR propietario<>current_user::regrole
    OR (length(original)-length(replace(original,'FROM vec_cronos_v1.marcaje_original','')))<>6*length('FROM vec_cronos_v1.marcaje_original')
 THEN RAISE EXCEPTION 'CRN19: libro de saldo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,'FROM vec_cronos_v1.marcaje_original','FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref)');
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,'FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado_ref)','FROM vec_cronos_v1.marcaje_original') IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'CRN19: libro de saldo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
 f:='vec_cronos_v1.consultar_movimientos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner INTO STRICT original,meta,acl,propietario FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex')<>'e99811f1b92ca8ae8749fd90c37094db3ca2e8350fbb7014fb70ef02a469b13a'
    OR propietario<>current_user::regrole
    OR (length(original)-length(replace(original,'FROM vec_cronos_v1.marcaje_original mo','')))<>length('FROM vec_cronos_v1.marcaje_original mo')
 THEN RAISE EXCEPTION 'CRN19: movimientos incompatibles' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,'FROM vec_cronos_v1.marcaje_original mo','FROM vec_cronos_v1.marcajes_efectivos_v1(emp) mo');
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,'FROM vec_cronos_v1.marcajes_efectivos_v1(emp) mo','FROM vec_cronos_v1.marcaje_original mo') IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'CRN19: movimientos alterados fuera del contrato' USING ERRCODE='55000'; END IF;
 f:='vec_cronos_v1.secuencia_remota_v1(text,timestamptz)'::regprocedure;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner INTO STRICT original,meta,acl,propietario FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc WHERE oid=f)
      <>'bf3ecb698cc7d627a9e44642156832f12ac67597426045f8ecd4b016386d8feb'
    OR propietario<>current_user::regrole
    OR (length(original)-length(replace(original,'FROM vec_cronos_v1.marcaje_original m','')))<>length('FROM vec_cronos_v1.marcaje_original m')
 THEN RAISE EXCEPTION 'CRN19: secuencia remota incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,'FROM vec_cronos_v1.marcaje_original m','FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado) m');
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,'FROM vec_cronos_v1.marcajes_efectivos_v1(p_empleado) m','FROM vec_cronos_v1.marcaje_original m') IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
       FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'CRN19: secuencia remota alterada fuera del contrato' USING ERRCODE='55000'; END IF;
END $proyeccion$;

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

