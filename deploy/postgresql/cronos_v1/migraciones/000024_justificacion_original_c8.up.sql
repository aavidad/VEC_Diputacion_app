\set ON_ERROR_STOP on
-- CRN24, C8 v2. Enlace y revisión de un original admitido por Documentos.
-- Sucesora prospectiva del borrador CRN14: no registra referencias externas
-- como originales ni modifica permisos, marcajes o sus recibos.
-- Requiere CRN22/23, dos consumidores nominales AD273 y fachada Documentos
-- para cotejar el original dentro de la transacción del efecto.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000024',0));
DO $pre$
DECLARE fachada text;
BEGIN
 IF current_user<>'vec_cronos_v1_propietario'
    OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
    OR to_regprocedure('vec_cronos_v1.validar_propietario_subida_documental_v1(text,bigint,text,text)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
      p.oid=to_regprocedure('vec_cronos_v1.validar_propietario_subida_documental_v1(text,bigint,text,text)')
      AND p.proowner='vec_cronos_v1_propietario'::regrole AND p.prosecdef AND p.provolatile='v'
      AND p.prorettype='boolean'::regtype
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
      AND encode(sha256(convert_to(p.prosrc,'UTF8')),'hex')='4c69bf8f49a3538bfe6affca0754a5d75dc7252f99fe5e9e45d7c321c4bbcd2a'
      AND p.proacl::text=
        '{vec_cronos_v1_propietario=X/vec_cronos_v1_propietario,vec_documentos_propietario=X/vec_cronos_v1_propietario}')
    OR to_regprocedure('vec_cronos_v1.resolver_acto_subida_documental_c8_v1(text,bigint,bytea,bytea,jsonb)') IS NOT NULL
    OR to_regclass('vec_cronos_v1.justificacion_operacion_c8') IS NOT NULL
    OR to_regclass('vec_cronos_v1.permiso_solicitud') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_estado') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_resolutor') IS NULL
    OR NOT has_schema_privilege('vec_cronos_v1_propietario','vec_documentos','USAGE')
    OR NOT has_schema_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3','USAGE')
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
      WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)')
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
        AND p.provolatile='v' AND p.prorettype='jsonb'::regtype
        AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
        AND has_function_privilege('vec_cronos_v1_propietario',p.oid,'EXECUTE')
        AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
          WHERE a.grantee=0 AND a.privilege_type='EXECUTE'))
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
      WHERE p.oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(bytea,bytea,jsonb)')
        AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
        AND p.provolatile='v' AND p.prorettype='jsonb'::regtype
        AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
        AND has_function_privilege('vec_cronos_v1_propietario',p.oid,'EXECUTE')
        AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
          WHERE a.grantee=0 AND a.privilege_type='EXECUTE'))
    OR to_regprocedure('vec_documentos.acreditar_original_cronos_v1(text,bigint,text,text,text)') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_proc p
      WHERE p.oid=to_regprocedure('vec_documentos.acreditar_original_cronos_v1(text,bigint,text,text,text)')
        AND p.proowner='vec_documentos_propietario'::regrole AND p.prosecdef
        AND p.provolatile='v' AND p.prorettype='jsonb'::regtype
        AND p.proconfig @> ARRAY['search_path=pg_catalog, pg_temp']::text[]
        AND has_function_privilege('vec_cronos_v1_propietario',p.oid,'EXECUTE')
        AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
          WHERE a.grantee=0 AND a.privilege_type='EXECUTE'))
 THEN RAISE EXCEPTION 'CRN24: preimagen o Documentos incompatible' USING ERRCODE='55000'; END IF;
 FOREACH fachada IN ARRAY ARRAY[
   'registrar_y_consumir_cronos_justificacion_anexo_v3_atestada',
   'registrar_y_consumir_cronos_justificacion_revision_v3_atestada'] LOOP
   IF NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname=fachada
         AND p.proargtypes='17 17 17 17 1700 1700 17 17 17 17'::oidvector
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
         AND has_function_privilege('vec_cronos_v1_propietario',p.oid,'EXECUTE')) THEN
     RAISE EXCEPTION 'CRN24: falta consumidor nominal %',fachada USING ERRCODE='55000';
   END IF;
 END LOOP;
END $pre$;

-- Motivos y conector esperados proceden de una publicación gobernada. Sin
-- fila vigente no se puede revisar ni enlazar un original.
CREATE TABLE vec_cronos_v1.justificacion_politica_c8 (
 politica_ref text NOT NULL CHECK (politica_ref ~ '^[-A-Za-z0-9_.:/#]{1,128}$'),
 version bigint NOT NULL CHECK (version>0),
 catalogo_version_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_catalogo,
 permiso_ref text NOT NULL,
 custodio_id text NOT NULL CHECK (custodio_id ~ '^[a-z][a-z0-9_.-]{1,127}$'),
 motivos_ref text[] NOT NULL CHECK (cardinality(motivos_ref) BETWEEN 1 AND 32
   AND array_position(motivos_ref,NULL) IS NULL),
 vigente_desde timestamptz(6) NOT NULL,
 vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
 fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
 sintetico boolean NOT NULL,
 publicada_en timestamptz(6) NOT NULL,
 PRIMARY KEY (politica_ref,version)
);
CREATE INDEX justificacion_politica_c8_permiso_idx ON vec_cronos_v1.justificacion_politica_c8(catalogo_version_ref,permiso_ref,vigente_desde);

CREATE TABLE vec_cronos_v1.justificacion_operacion_c8 (
 operacion_ref text PRIMARY KEY CHECK (operacion_ref ~ '^justificacion:cronos:[0-9a-f-]{36}$'),
 clave_operacion text NOT NULL CHECK (clave_operacion ~ '^ref:[0-9a-f]{64}$'),
 solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_solicitud,
 empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
 persona_titular_ref text NOT NULL CHECK (persona_titular_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 accion text NOT NULL CHECK (accion IN ('anexar','revisar')),
 version_previa bigint NOT NULL CHECK (version_previa BETWEEN 0 AND 9223372036854775806),
 version bigint NOT NULL CHECK (version=version_previa+1),
 estado text NOT NULL CHECK (estado IN ('pendiente_revision','aceptada','rechazada')),
 motivo_ref text NOT NULL,
 documento_id text NOT NULL CHECK (documento_id ~ '^ref:[0-9a-f]{64}$'),
 documento_version bigint NOT NULL CHECK (documento_version>0),
 documento_sha256 text NOT NULL CHECK (documento_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_documento_ref text NOT NULL CHECK (recibo_documento_ref ~ '^[A-Za-z][A-Za-z0-9_.:-]{2,159}$'),
 custodio_id text NOT NULL CHECK (custodio_id ~ '^[a-z][a-z0-9_.-]{1,127}$'),
 politica_ref text NOT NULL,
 politica_version bigint NOT NULL,
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
 material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE (actor_ref,clave_operacion), UNIQUE (solicitud_ref,version),
 FOREIGN KEY (politica_ref,politica_version) REFERENCES vec_cronos_v1.justificacion_politica_c8,
 CHECK ((accion='anexar' AND estado='pendiente_revision' AND motivo_ref='' AND actor_ref=persona_titular_ref)
    OR (accion='revisar' AND estado IN ('aceptada','rechazada') AND motivo_ref<>'' AND actor_ref<>persona_titular_ref))
);
CREATE INDEX justificacion_operacion_c8_solicitud_idx ON vec_cronos_v1.justificacion_operacion_c8(solicitud_ref,version DESC);

CREATE TABLE vec_cronos_v1.justificacion_replay_c8 (
 decision_ref text PRIMARY KEY,
 operacion_ref text NOT NULL REFERENCES vec_cronos_v1.justificacion_operacion_c8,
 actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
 auditoria_ref text NOT NULL UNIQUE,
 consumo_huella_sha256 text NOT NULL UNIQUE,
 registrada_en timestamptz(6) NOT NULL
);

ALTER TABLE vec_cronos_v1.justificacion_politica_c8 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.justificacion_politica_c8 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.justificacion_politica_c8
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.justificacion_politica_c8 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_publicada ON vec_cronos_v1.justificacion_politica_c8 FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY publicacion ON vec_cronos_v1.justificacion_politica_c8 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL
  AND nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NULL);
ALTER TABLE vec_cronos_v1.justificacion_operacion_c8 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.justificacion_operacion_c8 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.justificacion_operacion_c8
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.justificacion_operacion_c8 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY lectura_nominal ON vec_cronos_v1.justificacion_operacion_c8 FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'')
  OR vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion'));
CREATE POLICY adicion_propia ON vec_cronos_v1.justificacion_operacion_c8 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (accion='anexar' AND empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_revisor ON vec_cronos_v1.justificacion_operacion_c8 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (accion='revisar' AND actor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),'')
   AND vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion'));
ALTER TABLE vec_cronos_v1.justificacion_replay_c8 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_cronos_v1.justificacion_replay_c8 FORCE ROW LEVEL SECURITY;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.justificacion_replay_c8
 FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia();
REVOKE ALL ON TABLE vec_cronos_v1.justificacion_replay_c8 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
CREATE POLICY replay_propio ON vec_cronos_v1.justificacion_replay_c8 FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (actor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),'') OR
   EXISTS (SELECT 1 FROM vec_cronos_v1.justificacion_operacion_c8 o
     WHERE o.operacion_ref=justificacion_replay_c8.operacion_ref
       AND o.empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'')));

CREATE POLICY outbox_justificacion_c8 ON vec_cronos_v1.solicitud_outbox FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (tipo IN ('cronos.justificacion.anexada','cronos.justificacion.revisada')
  AND (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),'')
    OR vec_cronos_v1.resolutor_competente_v1(empleado_ref,'administracion')));
DO $restricciones$
DECLARE actual text; esperado text;
 tipos text[]:=ARRAY['cronos.correccion.solicitada','cronos.correccion.actuada',
   'cronos.permiso.solicitado','cronos.permiso.resuelto','cronos.aviso.archivado',
   'cronos.notificacion.registrada','cronos.notificacion.atendida'];
 rutas text[]:=ARRAY[
  '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
  '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
  '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
  '/api/interna/cronos/correcciones/recibo','/api/interna/cronos/correcciones/bandeja',
  '/api/interna/cronos/correcciones/actuaciones','/api/interna/cronos/correcciones/historia',
  '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
  '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
  '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos',
  '/api/interna/cronos/notificaciones/propio','/api/interna/cronos/notificaciones/envios',
  '/api/interna/cronos/notificaciones/bandeja','/api/interna/cronos/notificaciones/atenciones','otra'];
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO actual FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.solicitud_outbox'::regclass
    AND c.conname='solicitud_outbox_tipo_check' AND c.contype='c' AND c.convalidated;
 SELECT 'CHECK (tipo = ANY (ARRAY['||string_agg(quote_literal(v)||'::text',', ' ORDER BY ord)||']))'
  INTO esperado FROM unnest(tipos) WITH ORDINALITY AS x(v,ord);
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'CRN24: tipos previos incompatibles' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO actual FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.denegacion_frontera'::regclass
    AND c.conname='denegacion_frontera_ruta_check' AND c.contype='c' AND c.convalidated;
 SELECT 'CHECK (ruta = ANY (ARRAY['||string_agg(quote_literal(v)||'::text',', ' ORDER BY ord)||']))'
  INTO esperado FROM unnest(rutas) WITH ORDINALITY AS x(v,ord);
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'CRN24: rutas previas incompatibles' USING ERRCODE='55000';
 END IF;
END $restricciones$;
ALTER TABLE vec_cronos_v1.solicitud_outbox DROP CONSTRAINT solicitud_outbox_tipo_check;
ALTER TABLE vec_cronos_v1.solicitud_outbox ADD CONSTRAINT solicitud_outbox_tipo_check CHECK (tipo IN (
 'cronos.correccion.solicitada','cronos.correccion.actuada','cronos.permiso.solicitado',
 'cronos.permiso.resuelto','cronos.aviso.archivado','cronos.notificacion.registrada',
 'cronos.notificacion.atendida','cronos.justificacion.anexada','cronos.justificacion.revisada'));
ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check CHECK (ruta IN (
 '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
 '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
 '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
 '/api/interna/cronos/correcciones/recibo','/api/interna/cronos/correcciones/bandeja',
 '/api/interna/cronos/correcciones/actuaciones','/api/interna/cronos/correcciones/historia',
 '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
 '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
 '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos',
 '/api/interna/cronos/notificaciones/propio','/api/interna/cronos/notificaciones/envios',
 '/api/interna/cronos/notificaciones/bandeja','/api/interna/cronos/notificaciones/atenciones',
 '/api/interna/cronos/permisos/justificantes','/api/interna/cronos/permisos/justificantes/revisiones',
 '/api/interna/cronos/permisos/justificantes/propio','/api/interna/cronos/permisos/justificantes/detalle',
 '/api/interna/cronos/permisos/justificantes/bandeja','otra'));
DO $post_restricciones$
DECLARE actual text; esperado text;
 tipos text[]:=ARRAY['cronos.correccion.solicitada','cronos.correccion.actuada',
   'cronos.permiso.solicitado','cronos.permiso.resuelto','cronos.aviso.archivado',
   'cronos.notificacion.registrada','cronos.notificacion.atendida',
   'cronos.justificacion.anexada','cronos.justificacion.revisada'];
 rutas text[]:=ARRAY[
  '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
  '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
  '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
  '/api/interna/cronos/correcciones/recibo','/api/interna/cronos/correcciones/bandeja',
  '/api/interna/cronos/correcciones/actuaciones','/api/interna/cronos/correcciones/historia',
  '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
  '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
  '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos',
  '/api/interna/cronos/notificaciones/propio','/api/interna/cronos/notificaciones/envios',
  '/api/interna/cronos/notificaciones/bandeja','/api/interna/cronos/notificaciones/atenciones',
  '/api/interna/cronos/permisos/justificantes','/api/interna/cronos/permisos/justificantes/revisiones',
  '/api/interna/cronos/permisos/justificantes/propio','/api/interna/cronos/permisos/justificantes/detalle',
  '/api/interna/cronos/permisos/justificantes/bandeja','otra'];
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO actual FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.solicitud_outbox'::regclass
    AND c.conname='solicitud_outbox_tipo_check' AND c.contype='c' AND c.convalidated;
 SELECT 'CHECK (tipo = ANY (ARRAY['||string_agg(quote_literal(v)||'::text',', ' ORDER BY ord)||']))'
  INTO esperado FROM unnest(tipos) WITH ORDINALITY AS x(v,ord);
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'CRN24: postimagen de tipos incompatible' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO actual FROM pg_constraint c
  WHERE c.conrelid='vec_cronos_v1.denegacion_frontera'::regclass
    AND c.conname='denegacion_frontera_ruta_check' AND c.contype='c' AND c.convalidated;
 SELECT 'CHECK (ruta = ANY (ARRAY['||string_agg(quote_literal(v)||'::text',', ' ORDER BY ord)||']))'
  INTO esperado FROM unnest(rutas) WITH ORDINALITY AS x(v,ord);
 IF actual IS DISTINCT FROM esperado THEN
  RAISE EXCEPTION 'CRN24: postimagen de rutas incompatible' USING ERRCODE='55000';
 END IF;
END $post_restricciones$;

CREATE FUNCTION vec_cronos_v1.huella_contexto_justificacion_c8_v1(
 p_empleado text,p_solicitud text,p_material_sha256 text)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{"empleado_ref":'||to_jsonb(p_empleado)::text||
   ',"solicitud_ref":'||to_jsonb(p_solicitud)::text||'},"atributos":{"material_sha256":"'||
   p_material_sha256||'"}}','UTF8')),'hex')
$f$;

CREATE FUNCTION vec_cronos_v1.consumir_justificacion_original_c8_v1(
 p_modo text,m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
 OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,
 OUT ahora timestamptz,OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record;
 accion text; finalidad text;
BEGIN
 IF p_modo NOT IN ('anexar','revisar') OR
    nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL OR
    nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'contexto justificación Cronos incompatible' USING ERRCODE='PC003';
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
 IF (p_modo='anexar' AND (m->>'actor_ref' IS DISTINCT FROM m->>'persona_titular_ref' OR
       m->>'empleado_ref' IS DISTINCT FROM m->>'empleado_objetivo_ref')) OR
    (p_modo='revisar' AND (m->>'actor_ref' IS NOT DISTINCT FROM m->>'persona_titular_ref' OR
       m->>'empleado_ref' IS NOT DISTINCT FROM m->>'empleado_objetivo_ref')) THEN
   RAISE EXCEPTION 'justificación Cronos no competente' USING ERRCODE='PC012';
 END IF;
 material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 huella:=vec_cronos_v1.huella_contexto_justificacion_c8_v1(m->>'empleado_objetivo_ref',m->>'solicitud_ref',material_sha256);
 accion:='cronos.justificacion.'||p_modo;
 finalidad:=CASE p_modo WHEN 'anexar' THEN 'anexar_justificacion' ELSE 'revisar_justificacion' END;
 PERFORM vec_cronos_v1.comprobar_decision_cronos_v1(d,accion,'justificacion',finalidad,m->>'solicitud_ref',
   m->>'actor_ref',m->>'perfil_ref',huella);
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:justificacion:clave:'||m->>'actor_ref'||':'||m->>'clave_operacion',0));
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:solicitud:'||m->>'solicitud_ref',0));
 IF p_modo='anexar' THEN
   SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_anexo_v3_atestada(
     p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
   SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_justificacion_revision_v3_atestada(
     p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM m->>'solicitud_ref' OR
    consumo.huella_efecto_sha256 IS DISTINCT FROM huella THEN
   RAISE EXCEPTION 'consumo Cronos divergente' USING ERRCODE='PC003';
 END IF;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 vence:=vec_cronos_v1.vence_autorizacion_v1(c,d,x,vinculo);
 IF ahora>=vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 IF p_modo='anexar' THEN
   PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_objetivo_ref',true);
 ELSE
   PERFORM set_config('vec.cronos.resolutor_ref',m->>'actor_ref',true);
   IF NOT vec_cronos_v1.resolutor_competente_v1(m->>'empleado_objetivo_ref','administracion') THEN
     RAISE EXCEPTION 'justificación Cronos no competente' USING ERRCODE='PC012';
   END IF;
 END IF;
 decision_ref:=consumo.decision_ref;
 auditoria_ref:=consumo.auditoria_ref;
 consumo_huella_sha256:=consumo.consumo_huella_sha256;
END $f$;

CREATE FUNCTION vec_cronos_v1.material_justificacion_original_c8_v1(p_material text,p_modo text)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE m jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY[
   'actor_ref','perfil_ref','empleado_ref','empleado_objetivo_ref','solicitud_ref',
   'clave_operacion','version_esperada','documento_id','documento_version',
   'documento_sha256','recibo_documento_ref','custodio_id','persona_titular_ref',
   'persona_aportante_ref','contexto_ref','accion','decision','motivo_ref','zona_horaria']);
 IF p_modo NOT IN ('anexar','revisar') OR m->>'accion' IS DISTINCT FROM 'cronos.justificacion.'||p_modo
    OR m->>'solicitud_ref' !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'clave_operacion' !~ '^ref:[0-9a-f]{64}$'
    OR m->>'clave_operacion'='ref:'||repeat('0',64)
    OR m->>'empleado_objetivo_ref' !~ '^emp_[-A-Za-z0-9_]{22,128}$'
    OR m->>'persona_titular_ref' !~ '^per_[-A-Za-z0-9_]{22,128}$'
    OR m->>'persona_aportante_ref' IS DISTINCT FROM m->>'persona_titular_ref'
    OR m->>'contexto_ref' !~ '^rca_[-A-Za-z0-9_]{22,128}$'
    OR m->>'documento_id' !~ '^ref:[0-9a-f]{64}$'
    OR m->>'documento_id'='ref:'||repeat('0',64)
    OR m->>'documento_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'documento_sha256'=repeat('0',64)
    OR m->>'recibo_documento_ref' !~ '^[A-Za-z][A-Za-z0-9_.:-]{2,159}$'
    OR m->>'custodio_id' !~ '^[a-z][a-z0-9_.-]{1,127}$'
    OR m->>'version_esperada' !~ '^(0|[1-9][0-9]{0,17})$'
    OR m->>'documento_version' !~ '^[1-9][0-9]{0,17}$'
    OR (p_modo='anexar' AND (m->>'decision'<>'' OR m->>'motivo_ref'<>''))
    OR (p_modo='revisar' AND (m->>'decision' NOT IN ('aceptada','rechazada')
      OR m->>'motivo_ref' !~ '^[-A-Za-z0-9_.:/#]{1,160}$')) THEN
   RAISE EXCEPTION 'material justificación Cronos inválido' USING ERRCODE='PC001';
 END IF;
 RETURN m;
END $f$;

CREATE FUNCTION vec_cronos_v1.confirmar_justificacion_original_c8_v1(
 p_modo text,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; solicitud vec_cronos_v1.permiso_solicitud%ROWTYPE;
 permiso vec_cronos_v1.permiso_estado%ROWTYPE; catalogo vec_cronos_v1.permiso_catalogo%ROWTYPE;
 politica vec_cronos_v1.justificacion_politica_c8%ROWTYPE;
 anterior vec_cronos_v1.justificacion_operacion_c8%ROWTYPE;
 replay vec_cronos_v1.justificacion_operacion_c8%ROWTYPE;
 original jsonb; cantidad integer; version_previa bigint; doc_version bigint;
 ref text; recibo text; nuevo_estado text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR
    current_setting('transaction_read_only')<>'off' THEN
   RAISE EXCEPTION 'transacción de justificación incompatible' USING ERRCODE='PC003';
 END IF;
 m:=vec_cronos_v1.material_justificacion_original_c8_v1(p_material,p_modo);
 doc_version:=(m->>'documento_version')::bigint;
 version_previa:=(m->>'version_esperada')::bigint;
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_justificacion_original_c8_v1(p_modo,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO solicitud FROM vec_cronos_v1.permiso_solicitud s
  WHERE s.solicitud_ref=m->>'solicitud_ref' AND s.empleado_ref=m->>'empleado_objetivo_ref'
    AND s.actor_ref=m->>'persona_titular_ref';
 IF NOT FOUND THEN RAISE EXCEPTION 'solicitud no competente' USING ERRCODE='PC012'; END IF;
 SELECT * INTO permiso FROM vec_cronos_v1.permiso_estado e
  WHERE e.solicitud_ref=solicitud.solicitud_ref AND e.empleado_ref=solicitud.empleado_ref
  ORDER BY e.version DESC LIMIT 1;
 SELECT * INTO catalogo FROM vec_cronos_v1.permiso_catalogo c
  WHERE c.version_ref=solicitud.catalogo_version_ref;
 IF permiso.estado IS DISTINCT FROM 'concedido' OR permiso.pendiente_justificar IS NOT TRUE
    OR catalogo.justificante_exigido IS NOT TRUE THEN
   RAISE EXCEPTION 'solicitud no justificable' USING ERRCODE='PC011';
 END IF;
 -- La lectura documental vuelve a autorizar el original y sus bytes en la
 -- misma transacción del asiento Cronos. Un dato declarado no la sustituye.
 SELECT vec_documentos.acreditar_original_cronos_v1(m->>'documento_id',doc_version,
   solicitud.solicitud_ref,m->>'actor_ref',m->>'contexto_ref') INTO STRICT original;
 IF jsonb_typeof(original) IS DISTINCT FROM 'object' OR
    (SELECT count(*) FROM jsonb_object_keys(original))<>12 OR
    (original ?& ARRAY['documento_id','documento_version','huella_sha256_bytes',
     'recibo_ref','recurso_ref','persona_titular_ref','persona_aportante_ref',
     'custodio_id','modulo_id','tipo_recurso','finalidad','estado_admision']) IS NOT TRUE OR
    original->>'documento_id' IS DISTINCT FROM m->>'documento_id' OR
    original->>'documento_version' IS DISTINCT FROM m->>'documento_version' OR
    original->>'huella_sha256_bytes' IS DISTINCT FROM m->>'documento_sha256' OR
    original->>'recibo_ref' IS DISTINCT FROM m->>'recibo_documento_ref' OR
    original->>'recurso_ref' IS DISTINCT FROM solicitud.solicitud_ref OR
    original->>'persona_titular_ref' IS DISTINCT FROM solicitud.actor_ref OR
    original->>'persona_aportante_ref' IS DISTINCT FROM solicitud.actor_ref OR
    original->>'custodio_id' IS DISTINCT FROM m->>'custodio_id' OR
    original->>'modulo_id' IS DISTINCT FROM 'cronos' OR
    original->>'tipo_recurso' IS DISTINCT FROM 'justificacion' OR
    original->>'finalidad' IS DISTINCT FROM 'custodiar_justificacion_cronos' OR
    original->>'estado_admision' IS DISTINCT FROM 'admitido' THEN
   RAISE EXCEPTION 'original documental incompatible' USING ERRCODE='PC003';
 END IF;
 SELECT * INTO replay FROM vec_cronos_v1.justificacion_operacion_c8 o
  WHERE o.actor_ref=m->>'actor_ref' AND o.clave_operacion=m->>'clave_operacion';
 IF FOUND THEN
   IF replay.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave de justificación en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.justificacion_replay_c8(decision_ref,operacion_ref,actor_ref,auditoria_ref,
     consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,replay.operacion_ref,m->>'actor_ref',k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('solicitud_ref',replay.solicitud_ref,'documento_id',replay.documento_id,
     'documento_version',replay.documento_version,'sha256',replay.documento_sha256,
     'estado',replay.estado,'version',replay.version,'recibo_ref',replay.recibo_ref,
     'instante_utc',replay.registrada_en,'replay',true);
 END IF;
 SELECT * INTO anterior FROM vec_cronos_v1.justificacion_operacion_c8 o
  WHERE o.solicitud_ref=solicitud.solicitud_ref ORDER BY o.version DESC LIMIT 1;
 IF (NOT FOUND AND version_previa<>0) OR
    (FOUND AND anterior.version IS DISTINCT FROM version_previa) THEN
   RAISE EXCEPTION 'versión de justificación en conflicto' USING ERRCODE='PC011';
 END IF;
 IF p_modo='anexar' THEN
   IF anterior.operacion_ref IS NOT NULL AND anterior.documento_id=m->>'documento_id'
      AND anterior.documento_version=doc_version THEN
     RAISE EXCEPTION 'original ya enlazado' USING ERRCODE='PC011';
   END IF;
   SELECT count(*) INTO cantidad FROM vec_cronos_v1.justificacion_politica_c8 p
    WHERE p.catalogo_version_ref=solicitud.catalogo_version_ref AND p.permiso_ref=solicitud.permiso_ref
      AND p.vigente_desde<=k.ahora AND (p.vigente_hasta IS NULL OR k.ahora<p.vigente_hasta);
   IF cantidad<>1 THEN RAISE EXCEPTION 'política de justificación no determinada' USING ERRCODE='PC011'; END IF;
   SELECT * INTO STRICT politica FROM vec_cronos_v1.justificacion_politica_c8 p
    WHERE p.catalogo_version_ref=solicitud.catalogo_version_ref AND p.permiso_ref=solicitud.permiso_ref
      AND p.vigente_desde<=k.ahora AND (p.vigente_hasta IS NULL OR k.ahora<p.vigente_hasta);
   nuevo_estado:='pendiente_revision';
 ELSE
   IF anterior.operacion_ref IS NULL OR anterior.estado<>'pendiente_revision' OR
      anterior.documento_id IS DISTINCT FROM m->>'documento_id' OR
      anterior.documento_version IS DISTINCT FROM doc_version OR
      anterior.documento_sha256 IS DISTINCT FROM m->>'documento_sha256' OR
      anterior.recibo_documento_ref IS DISTINCT FROM m->>'recibo_documento_ref' OR
      anterior.custodio_id IS DISTINCT FROM m->>'custodio_id' THEN
     RAISE EXCEPTION 'revisión sin original pendiente' USING ERRCODE='PC011';
   END IF;
   SELECT * INTO STRICT politica FROM vec_cronos_v1.justificacion_politica_c8 p
    WHERE p.politica_ref=anterior.politica_ref AND p.version=anterior.politica_version;
   IF (m->>'motivo_ref'=ANY(politica.motivos_ref)) IS NOT TRUE THEN
     RAISE EXCEPTION 'motivo de revisión no publicado' USING ERRCODE='PC011';
   END IF;
   nuevo_estado:=m->>'decision';
 END IF;
 IF politica.custodio_id IS DISTINCT FROM m->>'custodio_id' THEN
   RAISE EXCEPTION 'custodio documental incompatible' USING ERRCODE='PC011';
 END IF;
 ref:='justificacion:cronos:'||gen_random_uuid()::text;
 recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.justificacion_operacion_c8(operacion_ref,clave_operacion,solicitud_ref,
   empleado_ref,persona_titular_ref,accion,version_previa,version,estado,motivo_ref,documento_id,
   documento_version,documento_sha256,recibo_documento_ref,custodio_id,politica_ref,politica_version,
   actor_ref,perfil_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,registrada_en)
 VALUES(ref,m->>'clave_operacion',solicitud.solicitud_ref,solicitud.empleado_ref,solicitud.actor_ref,
   p_modo,version_previa,version_previa+1,nuevo_estado,m->>'motivo_ref',m->>'documento_id',
   doc_version,m->>'documento_sha256',m->>'recibo_documento_ref',m->>'custodio_id',
   politica.politica_ref,politica.version,m->>'actor_ref',m->>'perfil_ref',k.material_sha256,
   k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,solicitud.solicitud_ref,solicitud.empleado_ref,
   CASE p_modo WHEN 'anexar' THEN 'cronos.justificacion.anexada' ELSE 'cronos.justificacion.revisada' END,
   jsonb_build_object('solicitud_ref',solicitud.solicitud_ref,'operacion_ref',ref,
     'documento_id',m->>'documento_id','documento_version',doc_version,'recibo_ref',recibo),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('solicitud_ref',solicitud.solicitud_ref,'documento_id',m->>'documento_id',
   'documento_version',doc_version,'sha256',m->>'documento_sha256','estado',nuevo_estado,
   'version',version_previa+1,'recibo_ref',recibo,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave de justificación en conflicto' USING ERRCODE='PC002';
END $f$;

CREATE FUNCTION vec_cronos_v1.anexar_justificacion_original_c8_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
 SELECT vec_cronos_v1.confirmar_justificacion_original_c8_v1('anexar',p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

CREATE FUNCTION vec_cronos_v1.revisar_justificacion_original_c8_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
 SELECT vec_cronos_v1.confirmar_justificacion_original_c8_v1('revisar',p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz)
$f$;

-- Lectura nominal de la historia: el servidor aporta contexto/vínculo de
-- sesión y evaluación actual. El núcleo asienta la auditoría antes de que
-- salga cualquier fila. Una decisión de escritura no sirve para consultar.
CREATE FUNCTION vec_cronos_v1.actor_lectura_justificacion_c8_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb,
 OUT persona_ref text,OUT empleado_ref text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog,pg_temp AS $f$
DECLARE x jsonb; pre jsonb; previa jsonb; cantidad integer; ahora timestamptz;
BEGIN
 previa:=CASE WHEN p_orden->>'accion'='cronos.justificacion.subida.preparar'
   THEN p_orden-'version_esperada' ELSE p_orden END;
 SELECT vec_autorizacion_atestada_v3.acreditar_sesion_lectura_justificacion_cronos_v1(
   p_contexto,p_vinculo,previa) INTO STRICT pre;
 IF jsonb_typeof(pre) IS DISTINCT FROM 'object' OR
    (SELECT count(*) FROM jsonb_object_keys(pre))<>7 OR
    (pre ?& ARRAY['persona_ref','empleado_ref','perfil_ref','registro_contexto_ref',
      'contexto_sha256','autenticacion_ref','sesion_ref']) IS NOT TRUE OR
    pre->>'registro_contexto_ref' IS DISTINCT FROM p_orden->>'registro_contexto_ref' OR
    pre->>'contexto_sha256' IS DISTINCT FROM encode(sha256(p_contexto),'hex') OR
    pre->>'autenticacion_ref' IS DISTINCT FROM p_orden->>'autenticacion_ref' OR
    pre->>'sesion_ref' IS DISTINCT FROM p_orden->>'sesion_ref' THEN
   RAISE EXCEPTION 'sesión de lectura Cronos no acreditada' USING ERRCODE='PC003';
 END IF;
 BEGIN
   x:=convert_from(p_contexto,'UTF8')::jsonb;
   ahora:=date_trunc('microseconds',clock_timestamp());
   SELECT count(*),min(v.item->>'referencia') INTO cantidad,empleado_ref
    FROM jsonb_array_elements(x->'vinculos') AS v(item)
    WHERE v.item->>'tipo'='empleado' AND v.item->>'estado'='activo'
      AND (v.item->>'vigente_desde')::timestamptz<=ahora
      AND ahora<(v.item->>'vigente_hasta')::timestamptz;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'contexto de lectura Cronos ilegible' USING ERRCODE='PC003';
 END;
 persona_ref:=x->>'persona_ref';
 IF cantidad<>1 OR persona_ref IS NULL OR empleado_ref IS NULL OR
    pre->>'persona_ref' IS DISTINCT FROM persona_ref OR
    pre->>'empleado_ref' IS DISTINCT FROM empleado_ref OR
    pre->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref' THEN
   RAISE EXCEPTION 'empleado de lectura no acreditado' USING ERRCODE='PC003';
 END IF;
 PERFORM vec_cronos_v1.acreditar_empleado_contexto_v1(x,persona_ref,x->>'perfil_activo_ref',empleado_ref,ahora);
END $f$;

-- Preparación documental: la selección viene de la ruta, pero solo la
-- sesión certificada y una lectura nominal positiva pueden revelar el acto.
-- La llamada de CRN22 y el apunte común forman parte de esta transacción.
CREATE FUNCTION vec_cronos_v1.resolver_acto_subida_documental_c8_v1(
 p_solicitud_ref text,p_version bigint,p_contexto bytea,p_vinculo bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE actor record; solicitud vec_cronos_v1.permiso_solicitud%ROWTYPE;
 permiso vec_cronos_v1.permiso_estado%ROWTYPE; exigido boolean; audit jsonb;
 orden jsonb; anterior text;
BEGIN
 IF session_user<>'vec_interno_documentos_cronos_datos'
    OR NOT EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
      AND r.rolinherit AND NOT r.rolsuper AND NOT r.rolbypassrls AND NOT r.rolcreaterole
      AND NOT r.rolcreatedb AND NOT r.rolreplication AND r.rolconfig IS NULL)
    OR NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
      AND m.roleid='vec_documentos_ejecutor'::regrole AND m.inherit_option
      AND NOT m.set_option AND NOT m.admin_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR p_solicitud_ref !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR p_version NOT BETWEEN 1 AND 1000
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 65536
    OR p_vinculo IS NULL OR octet_length(p_vinculo) NOT BETWEEN 1 AND 16384
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'accion' IS DISTINCT FROM 'cronos.justificacion.subida.preparar'
    OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'preparar_subida_justificacion'
    OR p_orden->>'modulo_id' IS DISTINCT FROM 'cronos'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM p_solicitud_ref
    OR jsonb_typeof(p_orden->'version_esperada') IS DISTINCT FROM 'number'
    OR p_orden->>'version_esperada' IS DISTINCT FROM p_version::text
    OR p_orden ? 'titular_persona_ref' OR p_orden ? 'empleado_objetivo_ref'
 THEN RAISE EXCEPTION 'preparación documental Cronos inválida' USING ERRCODE='PC001'; END IF;
 SELECT * INTO STRICT actor FROM vec_cronos_v1.actor_lectura_justificacion_c8_v1(p_contexto,p_vinculo,p_orden);
 IF p_orden->>'registro_contexto_ref' !~ '^rca_[-A-Za-z0-9_]{22,128}$' OR
    vec_cronos_v1.validar_propietario_subida_documental_v1(p_solicitud_ref,p_version,
      actor.persona_ref,p_orden->>'registro_contexto_ref') IS NOT TRUE THEN
   RAISE EXCEPTION 'preparación documental Cronos no competente' USING ERRCODE='PC012';
 END IF;
 anterior:=current_setting('vec.cronos.documental_persona_ref',true);
 PERFORM set_config('vec.cronos.documental_persona_ref',actor.persona_ref,true);
 SELECT * INTO STRICT solicitud FROM vec_cronos_v1.permiso_solicitud s
  WHERE s.solicitud_ref=p_solicitud_ref AND s.actor_ref=actor.persona_ref
    AND s.empleado_ref=actor.empleado_ref FOR SHARE;
 SELECT * INTO STRICT permiso FROM vec_cronos_v1.permiso_estado e
  WHERE e.solicitud_ref=solicitud.solicitud_ref AND e.empleado_ref=solicitud.empleado_ref
  ORDER BY e.version DESC LIMIT 1 FOR SHARE;
 SELECT c.justificante_exigido INTO STRICT exigido FROM vec_cronos_v1.permiso_catalogo c
  WHERE c.version_ref=solicitud.catalogo_version_ref;
 PERFORM set_config('vec.cronos.documental_persona_ref',coalesce(anterior,''),true);
 IF permiso.version IS DISTINCT FROM p_version OR permiso.estado IS DISTINCT FROM 'concedido'
    OR permiso.pendiente_justificar IS NOT TRUE OR exigido IS NOT TRUE THEN
   RAISE EXCEPTION 'preparación documental Cronos cambiada' USING ERRCODE='PC011';
 END IF;
 orden:=p_orden||jsonb_build_object('titular_persona_ref',solicitud.actor_ref,
   'empleado_objetivo_ref',solicitud.empleado_ref,'version_permiso',permiso.version);
 SELECT vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(
   p_contexto,p_vinculo,orden) INTO STRICT audit;
 IF jsonb_typeof(audit) IS DISTINCT FROM 'object' OR
    audit->>'auditoria_ref' IS NULL OR audit->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR
    audit->>'correlacion_ref' IS DISTINCT FROM p_orden->>'correlacion_ref' OR
    audit->>'registrada_en' IS NULL THEN
   RAISE EXCEPTION 'auditoría de preparación incompatible' USING ERRCODE='PC003';
 END IF;
 RETURN jsonb_build_object('solicitud_ref',solicitud.solicitud_ref,'version',permiso.version,
   'persona_ref',solicitud.actor_ref,'empleado_ref',solicitud.empleado_ref,'auditoria',audit);
EXCEPTION WHEN OTHERS THEN
 PERFORM set_config('vec.cronos.documental_persona_ref',coalesce(anterior,''),true);
 RAISE;
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_justificacion_original_c8_v1(
 p_solicitud_ref text,p_contexto bytea,p_vinculo bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE propia boolean; actor record; solicitud vec_cronos_v1.permiso_solicitud%ROWTYPE;
 permiso vec_cronos_v1.permiso_estado%ROWTYPE; historia jsonb; audit jsonb; orden jsonb;
 justificante_exigido boolean;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR p_solicitud_ref !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 65536
    OR p_vinculo IS NULL OR octet_length(p_vinculo) NOT BETWEEN 1 AND 16384
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM p_solicitud_ref
    OR p_orden->>'modulo_id' IS DISTINCT FROM 'cronos'
    OR p_orden->>'accion' NOT IN ('cronos.justificacion.propia.consultar','cronos.justificacion.rrhh.consultar')
    OR p_orden ? 'titular_persona_ref' OR p_orden ? 'empleado_objetivo_ref' THEN
   RAISE EXCEPTION 'lectura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 SELECT * INTO STRICT actor FROM vec_cronos_v1.actor_lectura_justificacion_c8_v1(p_contexto,p_vinculo,p_orden);
 propia:=p_orden->>'accion'='cronos.justificacion.propia.consultar';
 IF propia THEN
   PERFORM set_config('vec.cronos.empleado_ref',actor.empleado_ref,true);
 ELSE
   PERFORM set_config('vec.cronos.resolutor_ref',actor.persona_ref,true);
 END IF;
 SELECT * INTO solicitud FROM vec_cronos_v1.permiso_solicitud s
  WHERE s.solicitud_ref=p_solicitud_ref;
 IF NOT FOUND OR (propia AND (solicitud.actor_ref IS DISTINCT FROM actor.persona_ref
    OR solicitud.empleado_ref IS DISTINCT FROM actor.empleado_ref)) OR
    (NOT propia AND (solicitud.actor_ref IS NOT DISTINCT FROM actor.persona_ref
      OR vec_cronos_v1.resolutor_competente_v1(solicitud.empleado_ref,'administracion') IS NOT TRUE)) THEN
   RAISE EXCEPTION 'lectura Cronos no competente' USING ERRCODE='PC012';
 END IF;
 SELECT * INTO permiso FROM vec_cronos_v1.permiso_estado e
  WHERE e.solicitud_ref=solicitud.solicitud_ref ORDER BY e.version DESC LIMIT 1;
 SELECT c.justificante_exigido INTO justificante_exigido FROM vec_cronos_v1.permiso_catalogo c
  WHERE c.version_ref=solicitud.catalogo_version_ref;
 IF permiso.estado IS DISTINCT FROM 'concedido' OR permiso.pendiente_justificar IS NOT TRUE
    OR justificante_exigido IS NOT TRUE THEN
   RAISE EXCEPTION 'justificación no disponible' USING ERRCODE='PC011';
 END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'version',h.version,'estado',h.estado,'motivo_ref',h.motivo_ref,
   'documento_id',h.documento_id,'documento_version',h.documento_version,
   'sha256',h.documento_sha256,'recibo_documento_ref',h.recibo_documento_ref,
   'custodio_id',h.custodio_id,'recibo_ref',h.recibo_ref,'instante_utc',h.registrada_en)
   ORDER BY h.version),'[]'::jsonb) INTO historia
 FROM (SELECT * FROM vec_cronos_v1.justificacion_operacion_c8 x
       WHERE x.solicitud_ref=solicitud.solicitud_ref ORDER BY x.version LIMIT 501) h;
 IF jsonb_array_length(historia)>500 THEN
   RAISE EXCEPTION 'historia de justificación demasiado grande' USING ERRCODE='PC013';
 END IF;
 orden:=p_orden||jsonb_build_object('titular_persona_ref',solicitud.actor_ref,
   'empleado_objetivo_ref',solicitud.empleado_ref);
 SELECT vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(
   p_contexto,p_vinculo,orden) INTO STRICT audit;
 IF jsonb_typeof(audit) IS DISTINCT FROM 'object' OR
    audit->>'auditoria_ref' IS NULL OR audit->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR
    audit->>'correlacion_ref' IS DISTINCT FROM p_orden->>'correlacion_ref' OR
    audit->>'registrada_en' IS NULL THEN
   RAISE EXCEPTION 'auditoría de lectura incompatible' USING ERRCODE='PC003';
 END IF;
 RETURN jsonb_build_object('solicitud_ref',solicitud.solicitud_ref,'empleado_ref',solicitud.empleado_ref,
   'persona_titular_ref',solicitud.actor_ref,'estado_permiso',permiso.estado,
   'version_permiso',permiso.version,'pendiente_justificar',permiso.pendiente_justificar,
   'justificante_exigido',justificante_exigido,
   'historia',historia,'auditoria',audit);
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_bandeja_justificaciones_c8_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE pendientes jsonb; audit jsonb; actor record; ahora timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 1 AND 65536
    OR p_vinculo IS NULL OR octet_length(p_vinculo) NOT BETWEEN 1 AND 16384
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'accion' IS DISTINCT FROM 'cronos.justificacion.bandeja.consultar'
    OR p_orden->>'modulo_id' IS DISTINCT FROM 'cronos'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM 'bandeja:cronos:justificaciones'
    OR p_orden ? 'titular_persona_ref' OR p_orden ? 'empleado_objetivo_ref' THEN
   RAISE EXCEPTION 'bandeja Cronos inválida' USING ERRCODE='PC001';
 END IF;
 SELECT * INTO STRICT actor FROM vec_cronos_v1.actor_lectura_justificacion_c8_v1(p_contexto,p_vinculo,p_orden);
 PERFORM set_config('vec.cronos.resolutor_ref',actor.persona_ref,true);
 ahora:=date_trunc('microseconds',clock_timestamp());
 WITH asignados AS MATERIALIZED (
   SELECT DISTINCT r.empleado_ref FROM vec_cronos_v1.permiso_resolutor r
    WHERE r.resolutor_ref=actor.persona_ref AND r.paso='administracion'
      AND r.vigente_desde<=ahora AND (r.vigente_hasta IS NULL OR ahora<r.vigente_hasta)
      AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x
        WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=ahora)
 )
 SELECT coalesce(jsonb_agg(jsonb_build_object(
   'solicitud_ref',h.solicitud_ref,'empleado_etiqueta',et.empleado_etiqueta,
   'documento_id',h.documento_id,'documento_version',h.documento_version,
   'sha256',h.documento_sha256,'estado',h.estado,'version',h.version,
   'registrada_en',h.registrada_en) ORDER BY h.registrada_en DESC,h.solicitud_ref),'[]'::jsonb)
 INTO pendientes
 FROM (SELECT * FROM
   (SELECT DISTINCT ON (o.solicitud_ref) o.* FROM vec_cronos_v1.justificacion_operacion_c8 o
      JOIN asignados a ON a.empleado_ref=o.empleado_ref
      WHERE o.persona_titular_ref<>actor.persona_ref
        AND vec_cronos_v1.resolutor_competente_v1(o.empleado_ref,'administracion')
      ORDER BY o.solicitud_ref,o.version DESC) ultimos
   WHERE ultimos.estado='pendiente_revision'
   ORDER BY ultimos.registrada_en DESC,ultimos.solicitud_ref LIMIT 501) h
 LEFT JOIN LATERAL (SELECT r.empleado_etiqueta FROM vec_cronos_v1.permiso_resolutor r
   WHERE r.empleado_ref=h.empleado_ref AND r.paso='administracion'
     AND r.resolutor_ref=actor.persona_ref
   ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1) et ON true;
 IF jsonb_array_length(pendientes)>500 THEN
   RAISE EXCEPTION 'bandeja de justificación demasiado grande' USING ERRCODE='PC013';
 END IF;
 SELECT vec_autorizacion_atestada_v3.registrar_lectura_justificacion_cronos_v1(
   p_contexto,p_vinculo,p_orden) INTO STRICT audit;
 IF jsonb_typeof(audit) IS DISTINCT FROM 'object' OR
    audit->>'auditoria_ref' IS NULL OR audit->>'huella_sha256' !~ '^[0-9a-f]{64}$' OR
    audit->>'correlacion_ref' IS DISTINCT FROM p_orden->>'correlacion_ref' OR
    audit->>'registrada_en' IS NULL THEN
   RAISE EXCEPTION 'auditoría de bandeja incompatible' USING ERRCODE='PC003';
 END IF;
 RETURN jsonb_build_object('pendientes',pendientes,'auditoria',audit);
END $f$;

REVOKE ALL ON FUNCTION vec_cronos_v1.huella_contexto_justificacion_c8_v1(text,text,text)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.material_justificacion_original_c8_v1(text,text)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.consumir_justificacion_original_c8_v1(text,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.confirmar_justificacion_original_c8_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.actor_lectura_justificacion_c8_v1(bytea,bytea,jsonb)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.resolver_acto_subida_documental_c8_v1(text,bigint,bytea,bytea,jsonb)
 FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.anexar_justificacion_original_c8_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.revisar_justificacion_original_c8_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_justificacion_original_c8_v1(text,bytea,bytea,jsonb)
 FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
REVOKE ALL ON FUNCTION vec_cronos_v1.consultar_bandeja_justificaciones_c8_v1(bytea,bytea,jsonb)
 FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.anexar_justificacion_original_c8_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.revisar_justificacion_original_c8_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.consultar_justificacion_original_c8_v1(text,bytea,bytea,jsonb)
 TO vec_cronos_v1_ejecutor;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.consultar_bandeja_justificaciones_c8_v1(bytea,bytea,jsonb)
 TO vec_cronos_v1_ejecutor;
GRANT USAGE ON SCHEMA vec_cronos_v1 TO vec_interno_documentos_cronos_datos;
GRANT EXECUTE ON FUNCTION vec_cronos_v1.resolver_acto_subida_documental_c8_v1(text,bigint,bytea,bytea,jsonb)
 TO vec_interno_documentos_cronos_datos;
DO $post$
BEGIN
 IF (SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
    WHERE n.nspname='vec_cronos_v1' AND p.proname IN
      ('anexar_justificacion_original_c8_v1','revisar_justificacion_original_c8_v1')
      AND p.proargtypes='25 17 17 17 17 1700 1700 17 17 17 17'::oidvector
      AND p.proowner='vec_cronos_v1_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
      AND p.proacl::text=
        '{vec_cronos_v1_propietario=X/vec_cronos_v1_propietario,vec_cronos_v1_ejecutor=X/vec_cronos_v1_propietario}')<>2 THEN
    RAISE EXCEPTION 'CRN24: postimagen de funciones incompatible' USING ERRCODE='55000';
 END IF;
 IF (SELECT count(*) FROM pg_proc p WHERE p.oid IN (
     'vec_cronos_v1.consultar_justificacion_original_c8_v1(text,bytea,bytea,jsonb)'::regprocedure,
     'vec_cronos_v1.consultar_bandeja_justificaciones_c8_v1(bytea,bytea,jsonb)'::regprocedure)
      AND p.proowner='vec_cronos_v1_propietario'::regrole AND p.prosecdef
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
      AND p.proacl::text=
        '{vec_cronos_v1_propietario=X/vec_cronos_v1_propietario,vec_cronos_v1_ejecutor=X/vec_cronos_v1_propietario}')<>2 THEN
    RAISE EXCEPTION 'CRN24: postimagen de lecturas incompatible' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE
      p.oid='vec_cronos_v1.resolver_acto_subida_documental_c8_v1(text,bigint,bytea,bytea,jsonb)'::regprocedure
      AND p.proowner='vec_cronos_v1_propietario'::regrole AND p.prosecdef AND p.provolatile='v'
      AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
      AND p.proacl::text=
        '{vec_cronos_v1_propietario=X/vec_cronos_v1_propietario,vec_interno_documentos_cronos_datos=X/vec_cronos_v1_propietario}'
      AND has_schema_privilege('vec_interno_documentos_cronos_datos','vec_cronos_v1','USAGE')) THEN
   RAISE EXCEPTION 'CRN24: postimagen de preparación documental incompatible' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
