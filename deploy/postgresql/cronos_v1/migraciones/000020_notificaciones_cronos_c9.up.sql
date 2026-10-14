\set ON_ERROR_STOP on
-- CRN20: comunicaciones del empleado a RRHH y atención con recibos, historia
-- y decisiones nominales. Sucesora prospectiva del bloque C9 de CRN10; no
-- reinstala CRN9/10 ni cambia el circuito Jefatura → RRHH de CRN16.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000020',0));
DO $pre$
DECLARE n text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regclass('vec_cronos_v1.notificacion') IS NOT NULL
    OR to_regclass('vec_cronos_v1.permiso_resolutor') IS NULL
    OR to_regclass('vec_cronos_v1.correccion_resolutor') IS NULL
    OR to_regprocedure('vec_cronos_v1.resolutor_competente_v1(text,text)') IS NULL
 THEN RAISE EXCEPTION 'CRN20: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH n IN ARRAY ARRAY[
   'registrar_y_consumir_cronos_notificacion_v3_atestada',
   'consumir_cronos_notificaciones_propio_v3_atestada',
   'consumir_cronos_notificaciones_bandeja_v3_atestada',
   'registrar_y_consumir_cronos_atencion_notificacion_v3_atestada'] LOOP
   IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
       WHERE ns.nspname='vec_autorizacion_atestada_v3' AND p.proname=n
         AND p.proargtypes='17 17 17 17 1700 1700 17 17 17 17'::oidvector
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
         AND has_function_privilege('vec_cronos_v1_propietario',p.oid,'EXECUTE'))
   THEN RAISE EXCEPTION 'CRN20: falta consumidor nominal %',n USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;
CREATE TABLE vec_cronos_v1.notificacion_tipo (
  version_ref text PRIMARY KEY CHECK (version_ref ~ '^notificacion:cronos:tipo:[a-z0-9-]{1,64}:[A-Za-z0-9_.-]{1,64}$'),
  tipo_ref text NOT NULL CHECK (tipo_ref ~ '^notificacion:cronos:tipo:[a-z0-9-]{1,64}$' AND starts_with(version_ref,tipo_ref||':')),
  nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 120 AND nombre !~ '[[:cntrl:]]'),
  orden integer NOT NULL CHECK (orden BETWEEN 1 AND 10000),
  vigente_desde timestamptz(6) NOT NULL,
  vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
  sintetico boolean NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);
-- Declaración de la persona. El adjunto es la referencia de custodia y la
-- huella del documento, nunca sus bytes.
CREATE TABLE vec_cronos_v1.notificacion (
  notificacion_ref text PRIMARY KEY CHECK (notificacion_ref ~ '^notificacion:cronos:[0-9a-f-]{36}$'),
  clave_operacion text NOT NULL CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
  tipo_version_ref text NOT NULL REFERENCES vec_cronos_v1.notificacion_tipo,
  fecha_referida date NOT NULL,
  texto text NOT NULL CHECK (char_length(texto) BETWEEN 1 AND 512 AND translate(texto,E'\n\t','') !~ '[[:cntrl:]]' AND btrim(texto,E' \n\t')<>''),
  adjunto_ref text CHECK (adjunto_ref IS NULL OR adjunto_ref ~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$'),
  adjunto_sha256 text CHECK (adjunto_sha256 IS NULL OR (adjunto_sha256 ~ '^[0-9a-f]{64}$' AND adjunto_sha256<>repeat('0',64))),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  registrada_en timestamptz(6) NOT NULL,
  UNIQUE (empleado_ref,clave_operacion),
  CHECK ((adjunto_ref IS NULL)=(adjunto_sha256 IS NULL))
);
CREATE INDEX notificacion_empleado_idx ON vec_cronos_v1.notificacion(empleado_ref,registrada_en);
-- Atención por RRHH: una sola por notificación, con recibo.
CREATE TABLE vec_cronos_v1.notificacion_atencion (
  atencion_ref text PRIMARY KEY CHECK (atencion_ref ~ '^notificacion:cronos:atencion:[0-9a-f-]{36}$'),
  clave_operacion text NOT NULL CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  notificacion_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.notificacion,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  asignacion_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_resolutor,
  actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  atendida_en timestamptz(6) NOT NULL,
  UNIQUE (actor_ref,clave_operacion)
);
CREATE INDEX notificacion_atencion_empleado_idx ON vec_cronos_v1.notificacion_atencion(empleado_ref);
-- Evidencia de cada acceso autorizado y de cada replay de RRHH.
CREATE TABLE vec_cronos_v1.notificaciones_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.notificaciones_bandeja_acceso (
  decision_ref text PRIMARY KEY,
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.atencion_replay (
  decision_ref text PRIMARY KEY,
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  atencion_ref text NOT NULL REFERENCES vec_cronos_v1.notificacion_atencion,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  registrada_en timestamptz(6) NOT NULL
);


DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['notificacion_tipo','notificacion','notificacion_atencion',
   'notificaciones_acceso','notificaciones_bandeja_acceso','atencion_replay'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
 CREATE POLICY lectura_publicada ON vec_cronos_v1.notificacion_tipo FOR SELECT TO vec_cronos_v1_propietario USING (true);
 CREATE POLICY publicacion ON vec_cronos_v1.notificacion_tipo FOR INSERT TO vec_cronos_v1_propietario
    WITH CHECK (nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NULL
      AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
END $seguridad$;
-- La persona escribe y lee lo suyo; RRHH competente lee y atiende.
CREATE POLICY lectura_propia ON vec_cronos_v1.notificacion FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'') OR vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion'));
CREATE POLICY adicion_propia ON vec_cronos_v1.notificacion FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY lectura ON vec_cronos_v1.notificacion_atencion FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'') OR vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion'));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.notificacion_atencion FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion') AND actor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.notificaciones_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.notificaciones_bandeja_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.atencion_replay FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_atencion ON vec_cronos_v1.solicitud_outbox FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion') AND tipo='cronos.notificacion.atendida');


-- CRN19 tiene la lista anterior. Esta migración la amplía sin perder tipos
-- de corrección, permiso ni aviso.
DO $restricciones$
DECLARE vieja text; esperada text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO vieja FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.solicitud_outbox'::regclass
    AND c.conname='solicitud_outbox_tipo_check' AND c.contype='c' AND c.convalidated;
 IF vieja IS NULL OR vieja NOT LIKE '%cronos.correccion.actuada%'
    OR vieja NOT LIKE '%cronos.permiso.resuelto%'
    OR vieja LIKE '%cronos.notificacion.registrada%' THEN
   RAISE EXCEPTION 'CRN20: tipos previos incompatibles' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO esperada FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.denegacion_frontera'::regclass
    AND c.conname='denegacion_frontera_ruta_check' AND c.contype='c' AND c.convalidated;
 IF esperada IS NULL OR esperada NOT LIKE '%/api/interna/cronos/correcciones/actuaciones%'
    OR esperada LIKE '%/api/interna/cronos/notificaciones/propio%' THEN
   RAISE EXCEPTION 'CRN20: rutas previas incompatibles' USING ERRCODE='55000';
 END IF;
END $restricciones$;
ALTER TABLE vec_cronos_v1.solicitud_outbox DROP CONSTRAINT solicitud_outbox_tipo_check;
ALTER TABLE vec_cronos_v1.solicitud_outbox ADD CONSTRAINT solicitud_outbox_tipo_check CHECK (tipo IN
 ('cronos.correccion.solicitada','cronos.correccion.actuada','cronos.permiso.solicitado',
  'cronos.permiso.resuelto','cronos.aviso.archivado','cronos.notificacion.registrada','cronos.notificacion.atendida'));
ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check CHECK (ruta IN (
 '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
 '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
 '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
 '/api/interna/cronos/correcciones/recibo','/api/interna/cronos/correcciones/bandeja',
 '/api/interna/cronos/correcciones/actuaciones',
 '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
 '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
 '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos',
 '/api/interna/cronos/notificaciones/propio','/api/interna/cronos/notificaciones/envios',
 '/api/interna/cronos/notificaciones/bandeja','/api/interna/cronos/notificaciones/atenciones','otra'));
CREATE FUNCTION vec_cronos_v1.consumir_notificacion_v1(
    p_fachada text,p_modo text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_bloqueos text[],
    m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,OUT ahora timestamptz,OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record; b text;
BEGIN
 IF NOT ((p_modo='propio' AND p_fachada IN ('registrar_y_consumir_cronos_notificacion_v3_atestada','consumir_cronos_notificaciones_propio_v3_atestada'))
      OR (p_modo='rrhh' AND p_fachada IN ('consumir_cronos_notificaciones_bandeja_v3_atestada','registrar_y_consumir_cronos_atencion_notificacion_v3_atestada'))) THEN
   RAISE EXCEPTION 'fachada Cronos desconocida' USING ERRCODE='PC003';
 END IF;
 IF nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL OR nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'contexto Cronos ya fijado' USING ERRCODE='PC003';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'autorización Cronos ilegible' USING ERRCODE='PC003';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 huella:=CASE p_modo WHEN 'propio' THEN vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',material_sha256)
   ELSE vec_cronos_v1.huella_contexto_resolutor_v1(m->>'actor_ref','administracion',material_sha256) END;
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,p_accion,p_tipo,p_finalidad,p_recurso,m->>'actor_ref',m->>'perfil_ref',huella);
 FOREACH b IN ARRAY coalesce(p_bloqueos,ARRAY[]::text[]) LOOP
   PERFORM pg_advisory_xact_lock(hashtextextended(b,0));
 END LOOP;
 EXECUTE format('SELECT * FROM vec_autorizacion_atestada_v3.%I($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)',p_fachada) INTO STRICT consumo
   USING p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_recurso OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence THEN
   RAISE EXCEPTION 'vigencia Cronos no válida' USING ERRCODE='PC003';
 END IF;
 IF p_modo='propio' THEN
   PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 ELSE
   PERFORM set_config('vec.cronos.resolutor_ref',m->>'actor_ref',true);
 END IF;
 decision_ref:=consumo.decision_ref; auditoria_ref:=consumo.auditoria_ref; consumo_huella_sha256:=consumo.consumo_huella_sha256;
END $f$;

-- Tipos vigentes en un instante, sin repetir tipo (la versión más reciente).
CREATE FUNCTION vec_cronos_v1.notificacion_tipos_vigentes_v1(p_instante timestamptz)
RETURNS SETOF vec_cronos_v1.notificacion_tipo LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT DISTINCT ON (t.tipo_ref) t.* FROM vec_cronos_v1.notificacion_tipo t
  WHERE t.vigente_desde<=p_instante AND (t.vigente_hasta IS NULL OR p_instante<t.vigente_hasta)
  ORDER BY t.tipo_ref,t.vigente_desde DESC,t.publicada_en DESC
$f$;

CREATE FUNCTION vec_cronos_v1.registrar_notificacion_propia_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; fecha date; texto text; adjunto text; huella text; tipo vec_cronos_v1.notificacion_tipo%ROWTYPE;
 previa vec_cronos_v1.notificacion%ROWTYPE; ref text; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','tipo_version_ref',
   'fecha_referida','texto','adjunto_ref','adjunto_sha256','zona_horaria']);
 texto:=m->>'texto'; adjunto:=nullif(m->>'adjunto_ref',''); huella:=nullif(m->>'adjunto_sha256','');
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'tipo_version_ref' !~ '^notificacion:cronos:tipo:[a-z0-9-]{1,64}:[A-Za-z0-9_.-]{1,64}$'
    OR char_length(texto) NOT BETWEEN 1 AND 512 OR translate(texto,E'\n\t','') ~ '[[:cntrl:]]' OR btrim(texto,E' \n\t')=''
    OR (adjunto IS NULL)<>(huella IS NULL)
    OR (adjunto IS NOT NULL AND (adjunto !~ '^[A-Za-z][A-Za-z0-9:_-]{2,127}$' OR huella !~ '^[0-9a-f]{64}$' OR huella=repeat('0',64))) THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 fecha:=vec_cronos_v1.fecha_material_v1(m->>'fecha_referida');
 emp:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('registrar_y_consumir_cronos_notificacion_v3_atestada','propio',
   'cronos.notificacion.propia.registrar','notificacion_propia','comunicar_incidencia_rrhh','notificacion:cronos:'||(m->>'clave_operacion'),
   ARRAY['vec_cronos_v1:notificacion:'||emp||':'||(m->>'clave_operacion')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 -- La clave es de la persona: la de otra persona no se ve ni se revela.
 SELECT * INTO previa FROM vec_cronos_v1.notificacion n WHERE n.empleado_ref=emp AND n.clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.solicitud_replay(decision_ref,empleado_ref,agregado_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,emp,previa.notificacion_ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('notificacion_ref',previa.notificacion_ref,'recibo_ref',previa.recibo_ref,'instante_utc',previa.registrada_en,'replay',true);
 END IF;
 SELECT * INTO tipo FROM vec_cronos_v1.notificacion_tipos_vigentes_v1(k.ahora) t WHERE t.version_ref=m->>'tipo_version_ref';
 IF NOT FOUND THEN
   RAISE EXCEPTION 'tipo de notificación Cronos no vigente' USING ERRCODE='PC011';
 END IF;
 ref:='notificacion:cronos:'||gen_random_uuid()::text; recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.notificacion(notificacion_ref,clave_operacion,empleado_ref,actor_ref,perfil_ref,tipo_version_ref,fecha_referida,texto,
   adjunto_ref,adjunto_sha256,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,registrada_en)
 VALUES(ref,m->>'clave_operacion',emp,m->>'actor_ref',m->>'perfil_ref',tipo.version_ref,fecha,texto,adjunto,huella,
   k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 -- El evento no lleva el texto ni el adjunto: sólo referencias.
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,ref,emp,'cronos.notificacion.registrada',
   jsonb_build_object('notificacion_ref',ref,'recibo_ref',recibo,'tipo_version_ref',tipo.version_ref),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('notificacion_ref',ref,'recibo_ref',recibo,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_notificaciones_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','zona_horaria']);
 emp:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('consumir_cronos_notificaciones_propio_v3_atestada','propio',
   'cronos.notificaciones.propio.consultar','notificaciones_propio','consultar_notificaciones_propio','notificaciones:cronos:'||emp,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.notificaciones_acceso(decision_ref,empleado_ref,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,emp,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('empleado_ref',emp,
   'tipos',coalesce((SELECT jsonb_agg(jsonb_build_object('tipo_version_ref',t.version_ref,'tipo_ref',t.tipo_ref,'nombre',t.nombre) ORDER BY t.orden,t.tipo_ref)
      FROM vec_cronos_v1.notificacion_tipos_vigentes_v1(k.ahora) t),'[]'::jsonb),
   'notificaciones',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'notificacion_ref',n.notificacion_ref,'tipo_ref',t.tipo_ref,'tipo_nombre',t.nombre,'fecha_referida',to_char(n.fecha_referida,'YYYY-MM-DD'),
      'texto',n.texto,'adjunto_ref',n.adjunto_ref,'adjunto_sha256',n.adjunto_sha256,'registrada_en',n.registrada_en,
      'estado',CASE WHEN a.atencion_ref IS NULL THEN 'registrada' ELSE 'atendida' END,'atendida_en',a.atendida_en)
      ORDER BY n.registrada_en DESC,n.notificacion_ref)
    FROM (SELECT * FROM vec_cronos_v1.notificacion n0 WHERE n0.empleado_ref=emp ORDER BY n0.registrada_en DESC,n0.notificacion_ref LIMIT 501) n
    JOIN vec_cronos_v1.notificacion_tipo t ON t.version_ref=n.tipo_version_ref
    LEFT JOIN vec_cronos_v1.notificacion_atencion a ON a.notificacion_ref=n.notificacion_ref),'[]'::jsonb));
 IF jsonb_array_length(resultado->'tipos')>100 THEN
   RAISE EXCEPTION 'catálogo Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF jsonb_array_length(resultado->'notificaciones')>500 THEN
   RAISE EXCEPTION 'historial Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_bandeja_notificaciones_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; propio text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','zona_horaria']);
 propio:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('consumir_cronos_notificaciones_bandeja_v3_atestada','rrhh',
   'cronos.notificaciones.bandeja.consultar','bandeja_notificaciones','consultar_bandeja_notificaciones','bandeja:cronos:notificaciones',NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.notificaciones_bandeja_acceso(decision_ref,resolutor_ref,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,m->>'actor_ref',k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 -- Primero lo pendiente de atender y, después, lo más reciente; nunca lo propio.
 resultado:=jsonb_build_object('notificaciones',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'notificacion_ref',n.notificacion_ref,'empleado_ref',n.empleado_ref,'empleado_etiqueta',
      (SELECT r.empleado_etiqueta FROM vec_cronos_v1.permiso_resolutor r WHERE r.empleado_ref=n.empleado_ref AND r.paso='administracion'
         AND r.resolutor_ref=m->>'actor_ref' ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1),
      'tipo_ref',t.tipo_ref,'tipo_nombre',t.nombre,'fecha_referida',to_char(n.fecha_referida,'YYYY-MM-DD'),'texto',n.texto,
      'adjunto_ref',n.adjunto_ref,'adjunto_sha256',n.adjunto_sha256,'registrada_en',n.registrada_en,
      'atendida',n.atendida_en IS NOT NULL,'atendida_en',n.atendida_en) ORDER BY n.atendida_en IS NOT NULL,n.registrada_en DESC,n.notificacion_ref)
    FROM (SELECT n0.*,a.atendida_en FROM vec_cronos_v1.notificacion n0
            LEFT JOIN vec_cronos_v1.notificacion_atencion a ON a.notificacion_ref=n0.notificacion_ref
           WHERE vec_cronos_v1.resolutor_competente_v1(n0.empleado_ref,'administracion') AND n0.empleado_ref<>propio
           ORDER BY a.atendida_en IS NOT NULL,n0.registrada_en DESC,n0.notificacion_ref LIMIT 501) n
    JOIN vec_cronos_v1.notificacion_tipo t ON t.version_ref=n.tipo_version_ref),'[]'::jsonb));
 -- Como la bandeja de permisos: más de 500 no se recorta en silencio.
 IF jsonb_array_length(resultado->'notificaciones')>500 THEN
   RAISE EXCEPTION 'bandeja Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.atender_notificacion_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; actor text; previa vec_cronos_v1.notificacion_atencion%ROWTYPE; n vec_cronos_v1.notificacion%ROWTYPE;
 asignacion text; ref text; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','notificacion_ref','zona_horaria']);
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$' OR m->>'notificacion_ref' !~ '^notificacion:cronos:[0-9a-f-]{36}$' THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 actor:=m->>'actor_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_notificacion_v1('registrar_y_consumir_cronos_atencion_notificacion_v3_atestada','rrhh',
   'cronos.notificacion.atender','atencion_notificacion','atender_notificacion','notificacion:cronos:atencion:'||(m->>'clave_operacion'),
   ARRAY['vec_cronos_v1:atencion:'||actor||':'||(m->>'clave_operacion'),'vec_cronos_v1:notificacion:'||(m->>'notificacion_ref')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.notificacion_atencion x WHERE x.actor_ref=actor AND x.clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.atencion_replay(decision_ref,resolutor_ref,atencion_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,actor,previa.atencion_ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('atencion_ref',previa.atencion_ref,'notificacion_ref',previa.notificacion_ref,'recibo_ref',previa.recibo_ref,
     'instante_utc',previa.atendida_en,'replay',true);
 END IF;
 -- Sin competencia la notificación no es visible: ajena e inexistente igual.
 SELECT * INTO n FROM vec_cronos_v1.notificacion x WHERE x.notificacion_ref=m->>'notificacion_ref'
   AND vec_cronos_v1.resolutor_competente_v1(x.empleado_ref,'administracion');
 IF NOT FOUND OR n.empleado_ref=m->>'empleado_ref' THEN
   RAISE EXCEPTION 'atención no competente' USING ERRCODE='PC012';
 END IF;
 SELECT r.asignacion_ref INTO asignacion FROM vec_cronos_v1.permiso_resolutor r
  WHERE r.empleado_ref=n.empleado_ref AND r.paso='administracion' AND r.resolutor_ref=actor
    AND r.vigente_desde<=k.ahora AND (r.vigente_hasta IS NULL OR k.ahora<r.vigente_hasta)
    AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=k.ahora)
  ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1;
 IF asignacion IS NULL THEN
   RAISE EXCEPTION 'atención no competente' USING ERRCODE='PC012';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_cronos_v1.notificacion_atencion x WHERE x.notificacion_ref=n.notificacion_ref) THEN
   RAISE EXCEPTION 'notificación ya atendida' USING ERRCODE='PC011';
 END IF;
 ref:='notificacion:cronos:atencion:'||gen_random_uuid()::text; recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.notificacion_atencion(atencion_ref,clave_operacion,notificacion_ref,empleado_ref,asignacion_ref,actor_ref,perfil_ref,
   material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,atendida_en)
 VALUES(ref,m->>'clave_operacion',n.notificacion_ref,n.empleado_ref,asignacion,actor,m->>'perfil_ref',
   k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,n.notificacion_ref,n.empleado_ref,'cronos.notificacion.atendida',
   jsonb_build_object('notificacion_ref',n.notificacion_ref,'atencion_ref',ref,'recibo_ref',recibo),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('atencion_ref',ref,'notificacion_ref',n.notificacion_ref,'recibo_ref',recibo,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;


DO $acl$
DECLARE f text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.consumir_notificacion_v1(text,text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.notificacion_tipos_vigentes_v1(timestamptz)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.registrar_notificacion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_notificaciones_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_bandeja_notificaciones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.atender_notificacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_ejecutor',f);
 END LOOP;
END $acl$;
COMMIT;
