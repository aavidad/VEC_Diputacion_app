\set ON_ERROR_STOP on
-- Cronos, resolución de permisos (C7) y avisos de resolución archivables
-- (C9, parte de mensajes). Quien resuelve (jefatura o RRHH) consulta su
-- bandeja de solicitudes pendientes y concede o deniega con motivo, recibo e
-- historia de solo adición; la resolución final deja un aviso a la persona
-- empleada, que lo consulta y lo archiva.
--
-- Autorización: cada función consume en su misma transacción una decisión
-- V3 nueva de su audiencia (AD3-57). Para quien resuelve, el recurso lleva
-- los ámbitos {persona_ref, paso_resolucion}: RBAC decide qué paso puede
-- resolver un perfil. La relación «quién resuelve los permisos de quién» es
-- dato funcional del circuito (permiso_resolutor), publicado y versionado
-- por vigencia, que sólo RESTRINGE: sin asignación vigente no se ve ni se
-- resuelve nada. Nadie resuelve sus propias solicitudes y, en el circuito
-- J-A, RRHH no puede ser la misma persona que resolvió como responsable.
-- Requiere 000001–000008 y AD3-57. No crea cuentas LOGIN ni datos.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000009',0));
DO $pre$
DECLARE fachada text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_cronos_v1_auditor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR to_regprocedure('vec_cronos_v1.solicitar_permiso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_estado') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_resolutor') IS NOT NULL
    OR to_regprocedure('vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
   RAISE EXCEPTION 'Cronos 000009: preimagen incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH fachada IN ARRAY ARRAY['consumir_cronos_bandeja_permisos_v3_atestada','registrar_y_consumir_cronos_resolucion_permiso_v3_atestada',
   'consumir_cronos_avisos_propio_v3_atestada','registrar_y_consumir_cronos_archivo_aviso_v3_atestada'] LOOP
   IF to_regprocedure('vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
      OR NOT has_function_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
     RAISE EXCEPTION 'Cronos 000009: falta consumidor AD3-57 %',fachada USING ERRCODE='55000';
   END IF;
 END LOOP;
END $pre$;

-- Circuito publicado: quién resuelve, en cada paso, los permisos de cada
-- persona. La etiqueta es el nombre visible que verá quien resuelve; procede
-- de la fuente indicada y no sustituye a Personal. `sintetico` marca las
-- asignaciones que no proceden de RRHH. Solo adición: una asignación se
-- retira con una fila en permiso_resolutor_retirada.
CREATE TABLE vec_cronos_v1.permiso_resolutor (
  asignacion_ref text PRIMARY KEY CHECK (asignacion_ref ~ '^resolutor:cronos:[-A-Za-z0-9_.:]{1,128}$'),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  empleado_etiqueta text NOT NULL CHECK (length(empleado_etiqueta) BETWEEN 1 AND 120 AND empleado_etiqueta !~ '[[:cntrl:]]'),
  paso text NOT NULL CHECK (paso IN ('responsable','administracion')),
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  vigente_desde timestamptz(6) NOT NULL,
  vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
  sintetico boolean NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);
CREATE INDEX permiso_resolutor_resolutor_idx ON vec_cronos_v1.permiso_resolutor(resolutor_ref,paso,empleado_ref);
CREATE TABLE vec_cronos_v1.permiso_resolutor_retirada (
  asignacion_ref text PRIMARY KEY REFERENCES vec_cronos_v1.permiso_resolutor,
  retirada_en timestamptz(6) NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);

-- Resolución de un paso: hecho inmutable con motivo, versión previa y
-- resultante, asignación que acreditó la competencia y la decisión V3.
CREATE TABLE vec_cronos_v1.permiso_resolucion (
  resolucion_ref text PRIMARY KEY CHECK (resolucion_ref='permiso:cronos:resolucion:'||clave_operacion),
  clave_operacion text NOT NULL UNIQUE CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_solicitud,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  paso text NOT NULL CHECK (paso IN ('responsable','administracion')),
  decision text NOT NULL CHECK (decision IN ('aprobar','denegar')),
  motivo text CHECK (motivo IS NULL OR (char_length(motivo) BETWEEN 1 AND 500 AND motivo !~ '[[:cntrl:]]')),
  version_previa integer NOT NULL CHECK (version_previa BETWEEN 1 AND 999),
  version_resultante integer NOT NULL CHECK (version_resultante=version_previa+1),
  estado_resultante text NOT NULL CHECK (estado_resultante IN ('pendiente_administracion','concedido','denegado')),
  asignacion_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_resolutor,
  actor_ref text NOT NULL CHECK (actor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  perfil_ref text NOT NULL CHECK (perfil_ref ~ '^prf_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  registrada_en timestamptz(6) NOT NULL,
  UNIQUE (solicitud_ref,version_resultante),
  CHECK (decision='aprobar' OR motivo IS NOT NULL),
  CHECK ((decision='denegar')=(estado_resultante='denegado')),
  CHECK (paso='administracion' OR estado_resultante<>'concedido'),
  CHECK (paso='responsable' OR estado_resultante<>'pendiente_administracion')
);
CREATE INDEX permiso_resolucion_empleado_idx ON vec_cronos_v1.permiso_resolucion(empleado_ref);

-- Aviso a la persona de una resolución final y su archivo (una sola vez).
CREATE TABLE vec_cronos_v1.permiso_aviso (
  aviso_ref text PRIMARY KEY CHECK (aviso_ref ~ '^aviso:cronos:[0-9a-f-]{36}$'),
  resolucion_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.permiso_resolucion,
  solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_solicitud,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  creado_en timestamptz(6) NOT NULL
);
CREATE INDEX permiso_aviso_empleado_idx ON vec_cronos_v1.permiso_aviso(empleado_ref,creado_en);
CREATE TABLE vec_cronos_v1.permiso_aviso_archivo (
  archivo_ref text PRIMARY KEY CHECK (archivo_ref='aviso:cronos:archivo:'||clave_operacion),
  clave_operacion text NOT NULL UNIQUE CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  aviso_ref text NOT NULL UNIQUE REFERENCES vec_cronos_v1.permiso_aviso,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  actor_ref text NOT NULL,
  perfil_ref text NOT NULL,
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  archivado_en timestamptz(6) NOT NULL
);

-- Evidencia de cada acceso autorizado de quien resuelve y de cada replay.
CREATE TABLE vec_cronos_v1.bandeja_acceso (
  decision_ref text PRIMARY KEY,
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  paso text NOT NULL CHECK (paso IN ('responsable','administracion')),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.resolucion_replay (
  decision_ref text PRIMARY KEY,
  resolutor_ref text NOT NULL CHECK (resolutor_ref ~ '^per_[-A-Za-z0-9_]{22,128}$'),
  resolucion_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_resolucion,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.avisos_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);

-- Competencia vigente de quien resuelve (fijado sólo por las funciones de
-- esta migración tras consumir V3) sobre una persona en un paso o en
-- cualquiera. Base de las políticas RLS de quien resuelve.
CREATE FUNCTION vec_cronos_v1.resolutor_competente_v1(p_empleado text,p_paso text)
RETURNS boolean LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor r
   WHERE r.resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),'')
     AND r.empleado_ref=p_empleado AND (p_paso IS NULL OR r.paso=p_paso)
     AND r.vigente_desde<=clock_timestamp() AND (r.vigente_hasta IS NULL OR clock_timestamp()<r.vigente_hasta)
     AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=clock_timestamp()))
$f$;

DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['permiso_resolutor','permiso_resolutor_retirada','permiso_resolucion','permiso_aviso','permiso_aviso_archivo',
   'bandeja_acceso','resolucion_replay','avisos_acceso'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
END $seguridad$;
-- Circuito: lo publica el propietario (instalación gobernada, sin LOGIN) y
-- sólo lo lee quien resuelve, filtrado a sus propias asignaciones.
CREATE POLICY lectura_resolutor ON vec_cronos_v1.permiso_resolutor FOR SELECT TO vec_cronos_v1_propietario
 USING (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY publicacion ON vec_cronos_v1.permiso_resolutor FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
CREATE POLICY lectura_publicada ON vec_cronos_v1.permiso_resolutor_retirada FOR SELECT TO vec_cronos_v1_propietario USING (true);
CREATE POLICY publicacion ON vec_cronos_v1.permiso_resolutor_retirada FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (nullif(current_setting('vec.cronos.resolutor_ref',true),'') IS NULL AND nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NULL);
-- Resolución: la escribe quien resuelve en el paso exacto; la lee quien
-- resuelve a esa persona y la propia persona (motivo de su aviso).
CREATE POLICY lectura ON vec_cronos_v1.permiso_resolucion FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.resolutor_competente_v1(empleado_ref,NULL) OR empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.permiso_resolucion FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,paso) AND actor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY lectura_propia ON vec_cronos_v1.permiso_aviso FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.permiso_aviso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,NULL));
CREATE POLICY lectura_propia ON vec_cronos_v1.permiso_aviso_archivo FOR SELECT TO vec_cronos_v1_propietario
 USING (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.permiso_aviso_archivo FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.bandeja_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.resolucion_replay FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (resolutor_ref=nullif(current_setting('vec.cronos.resolutor_ref',true),''));
CREATE POLICY adicion_propia ON vec_cronos_v1.avisos_acceso FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (empleado_ref=nullif(current_setting('vec.cronos.empleado_ref',true),''));
-- Quien resuelve ve y amplía la historia sólo de las personas asignadas.
-- Las políticas propias de 000008 siguen intactas; éstas sólo se suman.
CREATE POLICY lectura_resolutor ON vec_cronos_v1.permiso_solicitud FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.resolutor_competente_v1(empleado_ref,NULL));
CREATE POLICY lectura_resolutor ON vec_cronos_v1.permiso_estado FOR SELECT TO vec_cronos_v1_propietario
 USING (vec_cronos_v1.resolutor_competente_v1(empleado_ref,NULL));
CREATE POLICY adicion_resolutor ON vec_cronos_v1.permiso_estado FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,NULL) AND version>1);
CREATE POLICY adicion_resolutor ON vec_cronos_v1.solicitud_outbox FOR INSERT TO vec_cronos_v1_propietario
 WITH CHECK (vec_cronos_v1.resolutor_competente_v1(empleado_ref,NULL) AND tipo='cronos.permiso.resuelto');

ALTER TABLE vec_cronos_v1.solicitud_outbox DROP CONSTRAINT solicitud_outbox_tipo_check;
ALTER TABLE vec_cronos_v1.solicitud_outbox ADD CONSTRAINT solicitud_outbox_tipo_check
 CHECK (tipo IN ('cronos.correccion.solicitada','cronos.permiso.solicitado','cronos.permiso.resuelto','cronos.aviso.archivado'));
-- La frontera audita también las cuatro rutas nuevas. Sólo amplía la lista.
ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check CHECK (ruta IN (
  '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
  '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
  '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
  '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes',
  '/api/interna/cronos/permisos/bandeja','/api/interna/cronos/permisos/resoluciones',
  '/api/interna/cronos/avisos/propio','/api/interna/cronos/avisos/archivos','otra'));

-- Reproduce RecursoAutorizable.HuellaContextoAutorizacionSHA256 de Go para
-- ámbitos {paso_resolucion, persona_ref} (claves en orden) y material_sha256.
CREATE FUNCTION vec_cronos_v1.huella_contexto_resolutor_v1(p_persona text,p_paso text,p_material_sha256 text)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT encode(sha256(convert_to('{"ambitos":{"paso_resolucion":'||to_jsonb(p_paso)::text||',"persona_ref":'||to_jsonb(p_persona)::text||
   '},"atributos":{"material_sha256":"'||p_material_sha256||'"}}','UTF8')),'hex')
$f$;

-- Consumo nominal de este corte. Modo 'propio': la persona sobre sí misma
-- (fija la RLS del empleado). Modo 'resolutor': quien resuelve en un paso
-- (fija la RLS de sus asignaciones). En ambos, el contexto debe traer un
-- único vínculo empleado vigente: el de quien actúa.
CREATE FUNCTION vec_cronos_v1.consumir_resolucion_v1(
    p_fachada text,p_modo text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_bloqueos text[],
    m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,OUT ahora timestamptz,OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record; b text;
BEGIN
 IF NOT ((p_modo='propio' AND p_fachada IN ('consumir_cronos_avisos_propio_v3_atestada','registrar_y_consumir_cronos_archivo_aviso_v3_atestada'))
      OR (p_modo='resolutor' AND p_fachada IN ('consumir_cronos_bandeja_permisos_v3_atestada','registrar_y_consumir_cronos_resolucion_permiso_v3_atestada'))) THEN
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
   ELSE vec_cronos_v1.huella_contexto_resolutor_v1(m->>'actor_ref',m->>'paso',material_sha256) END;
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

-- Estado vigente de las solicitudes visibles (la versión más alta).
CREATE FUNCTION vec_cronos_v1.estado_permiso_visible_v1()
RETURNS TABLE(solicitud_ref text,version integer,estado text,pendiente_justificar boolean)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT DISTINCT ON (e.solicitud_ref) e.solicitud_ref,e.version,e.estado,e.pendiente_justificar
   FROM vec_cronos_v1.permiso_estado e ORDER BY e.solicitud_ref,e.version DESC
$f$;

CREATE FUNCTION vec_cronos_v1.consultar_bandeja_permisos_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; paso text; propio text; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','paso','zona_horaria']);
 paso:=m->>'paso'; propio:=m->>'empleado_ref';
 IF paso NOT IN ('responsable','administracion') THEN
   RAISE EXCEPTION 'paso Cronos inválido' USING ERRCODE='PC001';
 END IF;
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_resolucion_v1('consumir_cronos_bandeja_permisos_v3_atestada','resolutor',
   'cronos.permisos.bandeja.consultar','bandeja_permisos','consultar_bandeja_permisos','bandeja:cronos:permisos:'||paso,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.bandeja_acceso(decision_ref,resolutor_ref,paso,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,m->>'actor_ref',paso,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('paso',paso,'pendientes',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'solicitud_ref',s.solicitud_ref,'empleado_ref',s.empleado_ref,'empleado_etiqueta',
      (SELECT r.empleado_etiqueta FROM vec_cronos_v1.permiso_resolutor r WHERE r.empleado_ref=s.empleado_ref AND r.paso=paso
         AND vec_cronos_v1.resolutor_competente_v1(s.empleado_ref,paso) ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1),
      'permiso_ref',s.permiso_ref,'nombre',pc.nombre,'circuito',pc.circuito,'justificante_exigido',pc.justificante_exigido,
      'desde',to_char(s.desde,'YYYY-MM-DD'),'hasta',to_char(s.hasta,'YYYY-MM-DD'),'hora_inicio',s.hora_inicio,'hora_fin',s.hora_fin,
      'cantidad',s.cantidad,'unidad',s.unidad,'estado',e.estado,'version',e.version,'solicitada_en',s.solicitada_en)
      ORDER BY s.solicitada_en,s.solicitud_ref)
    FROM (SELECT * FROM vec_cronos_v1.permiso_solicitud s0
           WHERE vec_cronos_v1.resolutor_competente_v1(s0.empleado_ref,paso) AND s0.empleado_ref<>propio
           ORDER BY s0.solicitada_en,s0.solicitud_ref) s
    JOIN vec_cronos_v1.estado_permiso_visible_v1() e ON e.solicitud_ref=s.solicitud_ref
    JOIN vec_cronos_v1.permiso_catalogo pc ON pc.version_ref=s.catalogo_version_ref
   WHERE (paso='responsable' AND pc.circuito='J-A' AND e.estado='solicitado')
      OR (paso='administracion' AND ((pc.circuito='A' AND e.estado='solicitado') OR (pc.circuito='J-A' AND e.estado='pendiente_administracion')))),'[]'::jsonb));
 IF jsonb_array_length(resultado->'pendientes')>500 THEN
   RAISE EXCEPTION 'bandeja Cronos demasiado grande' USING ERRCODE='PC013';
 END IF;
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.resolver_permiso_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; ref text; paso text; decision text; motivo text; esperada integer;
 previa vec_cronos_v1.permiso_resolucion%ROWTYPE; sol vec_cronos_v1.permiso_solicitud%ROWTYPE; cat vec_cronos_v1.permiso_catalogo%ROWTYPE;
 asignacion text; actual record; nuevo text; recibo text; aviso text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','solicitud_ref',
   'paso','decision','motivo','version_esperada','zona_horaria']);
 paso:=m->>'paso'; decision:=m->>'decision'; motivo:=nullif(m->>'motivo','');
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'solicitud_ref' !~ '^permiso:cronos:solicitud:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR paso NOT IN ('responsable','administracion') OR decision NOT IN ('aprobar','denegar')
    OR m->>'version_esperada' !~ '^[1-9][0-9]{0,2}$'
    OR (motivo IS NOT NULL AND (char_length(motivo)>500 OR motivo ~ '[[:cntrl:]]' OR motivo<>btrim(motivo)))
    OR (decision='denegar' AND motivo IS NULL) THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 esperada:=(m->>'version_esperada')::integer; ref:='permiso:cronos:resolucion:'||(m->>'clave_operacion');
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_resolucion_v1('registrar_y_consumir_cronos_resolucion_permiso_v3_atestada','resolutor',
   'cronos.permiso.resolver','resolucion_permiso','resolver_permiso',ref,
   ARRAY['vec_cronos_v1:clave:'||(m->>'clave_operacion'),'vec_cronos_v1:solicitud:'||(m->>'solicitud_ref')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.permiso_resolucion WHERE resolucion_ref=ref;
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.resolucion_replay(decision_ref,resolutor_ref,resolucion_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,m->>'actor_ref',ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('resolucion_ref',ref,'solicitud_ref',previa.solicitud_ref,'recibo_ref',previa.recibo_ref,
     'estado',previa.estado_resultante,'version',previa.version_resultante,'instante_utc',previa.registrada_en,'replay',true);
 END IF;
 -- Sin competencia vigente la solicitud no es visible: inexistente y ajena
 -- responden igual, sin revelar si existe.
 SELECT * INTO sol FROM vec_cronos_v1.permiso_solicitud s WHERE s.solicitud_ref=m->>'solicitud_ref' AND vec_cronos_v1.resolutor_competente_v1(s.empleado_ref,paso);
 IF NOT FOUND OR sol.empleado_ref=m->>'empleado_ref' THEN
   RAISE EXCEPTION 'resolución no competente' USING ERRCODE='PC012';
 END IF;
 SELECT r.asignacion_ref INTO STRICT asignacion FROM vec_cronos_v1.permiso_resolutor r
  WHERE r.empleado_ref=sol.empleado_ref AND r.paso=paso AND r.resolutor_ref=m->>'actor_ref'
    AND r.vigente_desde<=k.ahora AND (r.vigente_hasta IS NULL OR k.ahora<r.vigente_hasta)
    AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolutor_retirada x WHERE x.asignacion_ref=r.asignacion_ref AND x.retirada_en<=k.ahora)
  ORDER BY r.publicada_en DESC,r.asignacion_ref LIMIT 1;
 SELECT * INTO STRICT cat FROM vec_cronos_v1.permiso_catalogo WHERE version_ref=sol.catalogo_version_ref;
 SELECT e.version,e.estado INTO STRICT actual FROM vec_cronos_v1.permiso_estado e WHERE e.solicitud_ref=sol.solicitud_ref ORDER BY e.version DESC LIMIT 1;
 IF actual.version<>esperada THEN
   RAISE EXCEPTION 'versión Cronos en conflicto' USING ERRCODE='PC011';
 END IF;
 IF cat.circuito='J-A' AND actual.estado='solicitado' AND paso='responsable' THEN
   nuevo:=CASE decision WHEN 'aprobar' THEN 'pendiente_administracion' ELSE 'denegado' END;
 ELSIF ((cat.circuito='J-A' AND actual.estado='pendiente_administracion') OR (cat.circuito='A' AND actual.estado='solicitado')) AND paso='administracion' THEN
   nuevo:=CASE decision WHEN 'aprobar' THEN 'concedido' ELSE 'denegado' END;
   -- Separación de funciones: quien resolvió como responsable no concede
   -- ni deniega después como RRHH la misma solicitud.
   IF EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_resolucion r WHERE r.solicitud_ref=sol.solicitud_ref AND r.actor_ref=m->>'actor_ref') THEN
     RAISE EXCEPTION 'resolución no competente' USING ERRCODE='PC012';
   END IF;
 ELSE
   RAISE EXCEPTION 'estado Cronos en conflicto' USING ERRCODE='PC011';
 END IF;
 recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.permiso_resolucion(resolucion_ref,clave_operacion,solicitud_ref,empleado_ref,paso,decision,motivo,version_previa,
   version_resultante,estado_resultante,asignacion_ref,actor_ref,perfil_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,registrada_en)
 VALUES(ref,m->>'clave_operacion',sol.solicitud_ref,sol.empleado_ref,paso,decision,motivo,actual.version,actual.version+1,nuevo,asignacion,
   m->>'actor_ref',m->>'perfil_ref',k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.permiso_estado(estado_ref,solicitud_ref,empleado_ref,version,estado,pendiente_justificar,recibo_ref,registrada_en)
 VALUES('permiso:estado:'||gen_random_uuid()::text,sol.solicitud_ref,sol.empleado_ref,actual.version+1,nuevo,
   nuevo='concedido' AND cat.justificante_exigido,'recibo:cronos:'||gen_random_uuid()::text,k.ahora);
 IF nuevo IN ('concedido','denegado') THEN
   aviso:='aviso:cronos:'||gen_random_uuid()::text;
   INSERT INTO vec_cronos_v1.permiso_aviso(aviso_ref,resolucion_ref,solicitud_ref,empleado_ref,creado_en)
   VALUES(aviso,ref,sol.solicitud_ref,sol.empleado_ref,k.ahora);
 END IF;
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,sol.solicitud_ref,sol.empleado_ref,'cronos.permiso.resuelto',
   jsonb_build_object('solicitud_ref',sol.solicitud_ref,'resolucion_ref',ref,'recibo_ref',recibo,'paso',paso,'estado',nuevo,
     'version',actual.version+1,'aviso_ref',aviso),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('resolucion_ref',ref,'solicitud_ref',sol.solicitud_ref,'recibo_ref',recibo,'estado',nuevo,
   'version',actual.version+1,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

CREATE FUNCTION vec_cronos_v1.consultar_avisos_propio_v1(
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
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_resolucion_v1('consumir_cronos_avisos_propio_v3_atestada','propio',
   'cronos.avisos.propio.consultar','avisos_propio','consultar_avisos_propio','avisos:cronos:'||emp,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.avisos_acceso(decision_ref,empleado_ref,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,emp,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 resultado:=jsonb_build_object('empleado_ref',emp,'avisos',coalesce((SELECT jsonb_agg(jsonb_build_object(
      'aviso_ref',a.aviso_ref,'solicitud_ref',a.solicitud_ref,'estado',r.estado_resultante,'motivo',r.motivo,'resuelto_en',r.registrada_en,
      'permiso_ref',s.permiso_ref,'nombre',pc.nombre,'desde',to_char(s.desde,'YYYY-MM-DD'),'hasta',to_char(s.hasta,'YYYY-MM-DD'),
      'hora_inicio',s.hora_inicio,'hora_fin',s.hora_fin,'cantidad',s.cantidad,'unidad',s.unidad,
      'archivado',x.aviso_ref IS NOT NULL,'archivado_en',x.archivado_en) ORDER BY a.creado_en DESC,a.aviso_ref)
    FROM (SELECT * FROM vec_cronos_v1.permiso_aviso a0 WHERE a0.empleado_ref=emp ORDER BY a0.creado_en DESC,a0.aviso_ref LIMIT 500) a
    JOIN vec_cronos_v1.permiso_resolucion r ON r.resolucion_ref=a.resolucion_ref
    JOIN vec_cronos_v1.permiso_solicitud s ON s.solicitud_ref=a.solicitud_ref
    JOIN vec_cronos_v1.permiso_catalogo pc ON pc.version_ref=s.catalogo_version_ref
    LEFT JOIN vec_cronos_v1.permiso_aviso_archivo x ON x.aviso_ref=a.aviso_ref),'[]'::jsonb));
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.archivar_aviso_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; ref text; previa vec_cronos_v1.permiso_aviso_archivo%ROWTYPE; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','aviso_ref','zona_horaria']);
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$' OR m->>'aviso_ref' !~ '^aviso:cronos:[0-9a-f-]{36}$' THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 emp:=m->>'empleado_ref'; ref:='aviso:cronos:archivo:'||(m->>'clave_operacion');
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_resolucion_v1('registrar_y_consumir_cronos_archivo_aviso_v3_atestada','propio',
   'cronos.aviso.propio.archivar','archivo_aviso','archivar_aviso_propio',ref,
   ARRAY['vec_cronos_v1:clave:'||(m->>'clave_operacion'),'vec_cronos_v1:aviso:'||(m->>'aviso_ref')],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.permiso_aviso_archivo WHERE archivo_ref=ref;
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   INSERT INTO vec_cronos_v1.solicitud_replay(decision_ref,empleado_ref,agregado_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,emp,ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('archivo_ref',ref,'aviso_ref',previa.aviso_ref,'recibo_ref',previa.recibo_ref,
     'instante_utc',previa.archivado_en,'replay',true);
 END IF;
 IF NOT EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_aviso a WHERE a.aviso_ref=m->>'aviso_ref' AND a.empleado_ref=emp) THEN
   RAISE EXCEPTION 'aviso ajeno o inexistente' USING ERRCODE='PC001';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_aviso_archivo x WHERE x.aviso_ref=m->>'aviso_ref') THEN
   RAISE EXCEPTION 'aviso ya archivado' USING ERRCODE='PC011';
 END IF;
 recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.permiso_aviso_archivo(archivo_ref,clave_operacion,aviso_ref,empleado_ref,actor_ref,perfil_ref,material_sha256,
   decision_ref,auditoria_ref,consumo_huella_sha256,recibo_ref,archivado_en)
 VALUES(ref,m->>'clave_operacion',m->>'aviso_ref',emp,m->>'actor_ref',m->>'perfil_ref',k.material_sha256,
   k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,m->>'aviso_ref',emp,'cronos.aviso.archivado',
   jsonb_build_object('aviso_ref',m->>'aviso_ref','archivo_ref',ref,'recibo_ref',recibo),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('archivo_ref',ref,'aviso_ref',m->>'aviso_ref','recibo_ref',recibo,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

DO $acl$
DECLARE f text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.resolutor_competente_v1(text,text)',
   'vec_cronos_v1.huella_contexto_resolutor_v1(text,text,text)',
   'vec_cronos_v1.consumir_resolucion_v1(text,text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.estado_permiso_visible_v1()'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.consultar_bandeja_permisos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_avisos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.archivar_aviso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_ejecutor',f);
 END LOOP;
END $acl$;
COMMIT;
