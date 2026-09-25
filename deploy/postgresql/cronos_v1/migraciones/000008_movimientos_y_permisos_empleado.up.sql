\set ON_ERROR_STOP on
-- Cronos para la persona empleada, segundo corte: movimientos (calendario
-- anual, absentismos y solicitudes de corrección), solicitud de corrección de
-- un olvido de marcaje, permisos del año y solicitud de un permiso. Cada
-- función consume en su misma transacción una decisión V3 nueva de su
-- audiencia (AD3-70), revalida el contexto con un único empleado vigente,
-- fija la RLS de esa persona y sólo entonces lee o registra. La corrección
-- nunca toca el marcaje original: es un hecho nuevo pendiente de resolver.
-- La concesión de permisos y la resolución de correcciones son otro corte.
-- Requiere 000001–000007 y AD3-70. No crea cuentas LOGIN ni datos.
BEGIN;
SET LOCAL ROLE vec_cronos_v1_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:000008',0));
DO $pre$
DECLARE fachada text;
BEGIN
 IF to_regprocedure('vec_cronos_v1.consultar_saldo_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regclass('vec_cronos_v1.denegacion_frontera') IS NULL
    OR to_regclass('vec_cronos_v1.permiso_catalogo') IS NOT NULL
    OR to_regprocedure('vec_cronos_v1.solicitar_permiso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
   RAISE EXCEPTION 'Cronos 000008: preimagen incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH fachada IN ARRAY ARRAY['consumir_cronos_movimientos_propio_v3_atestada','registrar_y_consumir_cronos_correccion_v3_atestada',
   'consumir_cronos_permisos_propio_v3_atestada','registrar_y_consumir_cronos_permiso_v3_atestada'] LOOP
   IF to_regprocedure('vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
      OR NOT has_function_privilege('vec_cronos_v1_propietario','vec_autorizacion_atestada_v3.'||fachada||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') THEN
     RAISE EXCEPTION 'Cronos 000008: falta consumidor AD3-70 %',fachada USING ERRCODE='55000';
   END IF;
 END LOOP;
END $pre$;

-- Calendario laboral publicado: días no laborables de cada calendario y
-- calendario asignado a cada persona por año. Sin asignación, la pantalla
-- dice que falta el calendario; nunca se inventa.
CREATE TABLE vec_cronos_v1.calendario_dia (
  calendario_ref text NOT NULL CHECK (calendario_ref ~ '^calendario:cronos:[-A-Za-z0-9_.]{1,128}$'),
  fecha date NOT NULL,
  tipo text NOT NULL CHECK (tipo IN ('festivo','no_laborable')),
  nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 120),
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL,
  PRIMARY KEY (calendario_ref,fecha)
);
CREATE TABLE vec_cronos_v1.calendario_asignacion (
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  anio integer NOT NULL CHECK (anio BETWEEN 2000 AND 2100),
  calendario_ref text NOT NULL CHECK (calendario_ref ~ '^calendario:cronos:[-A-Za-z0-9_.]{1,128}$'),
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL,
  PRIMARY KEY (empleado_ref,anio)
);

-- Catálogo versionado de permisos. Cantidades de horas en minutos y de días
-- en días. `sintetico` marca valores que no proceden de RRHH.
CREATE TABLE vec_cronos_v1.permiso_catalogo (
  version_ref text PRIMARY KEY CHECK (version_ref ~ '^catalogo:cronos:[-A-Za-z0-9_.:]{1,128}$'),
  permiso_ref text NOT NULL CHECK (permiso_ref ~ '^permiso:cronos:[-A-Za-z0-9_.]{1,96}$'),
  nombre text NOT NULL CHECK (length(nombre) BETWEEN 1 AND 120),
  orden integer NOT NULL CHECK (orden BETWEEN 0 AND 100000),
  vigente_desde timestamptz(6) NOT NULL,
  vigente_hasta timestamptz(6) CHECK (vigente_hasta IS NULL OR vigente_hasta>vigente_desde),
  unidad text NOT NULL CHECK (unidad IN ('dia','hora')),
  computo text NOT NULL CHECK (computo IN ('laborables','naturales')),
  circuito text NOT NULL CHECK (circuito IN ('A','J-A')),
  minimo bigint NOT NULL CHECK (minimo>0),
  maximo_solicitud bigint CHECK (maximo_solicitud IS NULL OR maximo_solicitud>=minimo),
  maximo_mensual bigint CHECK (maximo_mensual IS NULL OR maximo_mensual>=minimo),
  maximo_anual bigint CHECK (maximo_anual IS NULL OR maximo_anual>=minimo),
  justificante_exigido boolean NOT NULL,
  solicitable boolean NOT NULL,
  sintetico boolean NOT NULL,
  fuente_ref text NOT NULL CHECK (fuente_ref ~ '^[-A-Za-z0-9_.:/#]{1,255}$'),
  publicada_en timestamptz(6) NOT NULL
);
CREATE INDEX permiso_catalogo_permiso_idx ON vec_cronos_v1.permiso_catalogo(permiso_ref,vigente_desde);
CREATE FUNCTION vec_cronos_v1.comprobar_catalogo_permiso_v1()
RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $f$
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_cronos_v1:catalogo:'||NEW.permiso_ref,0));
 IF EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_catalogo p WHERE p.permiso_ref=NEW.permiso_ref AND p.version_ref<>NEW.version_ref
     AND tstzrange(p.vigente_desde,p.vigente_hasta,'[)') && tstzrange(NEW.vigente_desde,NEW.vigente_hasta,'[)')) THEN
   RAISE EXCEPTION 'versiones de catálogo solapadas' USING ERRCODE='23P01';
 END IF;
 RETURN NEW;
END $f$;
CREATE TRIGGER catalogo_sin_solape BEFORE INSERT ON vec_cronos_v1.permiso_catalogo
 FOR EACH ROW EXECUTE FUNCTION vec_cronos_v1.comprobar_catalogo_permiso_v1();

-- Solicitudes de la persona: hecho inicial inmutable y estados de solo adición.
CREATE TABLE vec_cronos_v1.correccion_solicitud (
  solicitud_ref text PRIMARY KEY CHECK (solicitud_ref='correccion:cronos:'||clave_operacion),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  clave_operacion text NOT NULL UNIQUE CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  actor_ref text NOT NULL,
  perfil_ref text NOT NULL,
  marcaje_original_ref text REFERENCES vec_cronos_v1.marcaje_original,
  hueco_declarado boolean NOT NULL CHECK (hueco_declarado=(marcaje_original_ref IS NULL)),
  movimiento text NOT NULL CHECK (movimiento IN ('entrada','salida','inicio_pausa','fin_pausa')),
  fecha_civil date NOT NULL,
  hora_pretendida text NOT NULL CHECK (hora_pretendida ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
  motivo_codigo text NOT NULL CHECK (motivo_codigo='olvido_marcaje'),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  solicitada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.correccion_actuacion (
  actuacion_ref text PRIMARY KEY CHECK (actuacion_ref ~ '^correccion:actuacion:[0-9a-f-]{36}$'),
  solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.correccion_solicitud,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  version integer NOT NULL CHECK (version BETWEEN 1 AND 4),
  paso text NOT NULL CHECK (paso IN ('solicitud','decision_responsable','resolucion_rrhh','aplicacion')),
  estado text NOT NULL CHECK (estado IN ('pendiente_responsable','pendiente_rrhh','denegada_responsable','denegada_rrhh','pendiente_aplicacion','aplicada')),
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  registrada_en timestamptz(6) NOT NULL,
  UNIQUE (solicitud_ref,version),
  CHECK ((version=1)=(paso='solicitud') AND (version<>1 OR estado='pendiente_responsable'))
);
CREATE TABLE vec_cronos_v1.permiso_solicitud (
  solicitud_ref text PRIMARY KEY CHECK (solicitud_ref='permiso:cronos:solicitud:'||clave_operacion),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  clave_operacion text NOT NULL UNIQUE CHECK (clave_operacion ~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'),
  actor_ref text NOT NULL,
  perfil_ref text NOT NULL,
  catalogo_version_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_catalogo,
  permiso_ref text NOT NULL,
  desde date NOT NULL,
  hasta date NOT NULL,
  hora_inicio text CHECK (hora_inicio IS NULL OR hora_inicio ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
  hora_fin text CHECK (hora_fin IS NULL OR hora_fin ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
  cantidad bigint NOT NULL CHECK (cantidad>0),
  unidad text NOT NULL CHECK (unidad IN ('dia','hora')),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  decision_ref text NOT NULL UNIQUE,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  solicitada_en timestamptz(6) NOT NULL,
  CHECK (desde<=hasta AND extract(year FROM desde)=extract(year FROM hasta)),
  CHECK ((unidad='hora')=(hora_inicio IS NOT NULL) AND (hora_inicio IS NULL)=(hora_fin IS NULL))
);
CREATE INDEX permiso_solicitud_persona_idx ON vec_cronos_v1.permiso_solicitud(empleado_ref,desde);
CREATE TABLE vec_cronos_v1.permiso_estado (
  estado_ref text PRIMARY KEY CHECK (estado_ref ~ '^permiso:estado:[0-9a-f-]{36}$'),
  solicitud_ref text NOT NULL REFERENCES vec_cronos_v1.permiso_solicitud,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  version integer NOT NULL CHECK (version BETWEEN 1 AND 1000),
  estado text NOT NULL CHECK (estado IN ('solicitado','pendiente_administracion','concedido','denegado','cancelado')),
  pendiente_justificar boolean NOT NULL CHECK (NOT pendiente_justificar OR estado='concedido'),
  recibo_ref text NOT NULL UNIQUE CHECK (recibo_ref ~ '^recibo:cronos:[0-9a-f-]{36}$'),
  registrada_en timestamptz(6) NOT NULL,
  UNIQUE (solicitud_ref,version),
  CHECK (version<>1 OR (estado='solicitado' AND NOT pendiente_justificar))
);
CREATE TABLE vec_cronos_v1.solicitud_outbox (
  evento_ref text PRIMARY KEY CHECK (evento_ref ~ '^evento:cronos:[0-9a-f-]{36}$'),
  agregado_ref text NOT NULL,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  tipo text NOT NULL CHECK (tipo IN ('cronos.correccion.solicitada','cronos.permiso.solicitado')),
  carga_json jsonb NOT NULL,
  creada_en timestamptz(6) NOT NULL
);
-- Evidencia de cada acceso autorizado y de cada replay de una escritura.
CREATE TABLE vec_cronos_v1.movimientos_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  desde date NOT NULL,
  hasta date NOT NULL,
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL,
  CHECK (desde<=hasta)
);
CREATE TABLE vec_cronos_v1.permisos_acceso (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  anio integer NOT NULL CHECK (anio BETWEEN 2000 AND 2100),
  material_sha256 text NOT NULL CHECK (material_sha256 ~ '^[0-9a-f]{64}$'),
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  consultada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_cronos_v1.solicitud_replay (
  decision_ref text PRIMARY KEY,
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[-A-Za-z0-9_]{22,128}$'),
  agregado_ref text NOT NULL,
  auditoria_ref text NOT NULL UNIQUE,
  consumo_huella_sha256 text NOT NULL UNIQUE,
  registrada_en timestamptz(6) NOT NULL
);
DO $seguridad$
DECLARE tabla text;
BEGIN
 FOREACH tabla IN ARRAY ARRAY['calendario_asignacion','correccion_solicitud','correccion_actuacion','permiso_solicitud','permiso_estado',
   'solicitud_outbox','movimientos_acceso','permisos_acceso','solicitud_replay'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY lectura_propia ON vec_cronos_v1.%I FOR SELECT TO vec_cronos_v1_propietario USING (empleado_ref=nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
  EXECUTE format('CREATE POLICY adicion_propia ON vec_cronos_v1.%I FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (empleado_ref=nullif(current_setting(''vec.cronos.empleado_ref'',true),''''))',tabla);
 END LOOP;
 -- Catálogo y días de calendario no son datos personales: se leen enteros
 -- y sólo el propietario los publica (instalación gobernada, sin LOGIN).
 FOREACH tabla IN ARRAY ARRAY['calendario_dia','permiso_catalogo'] LOOP
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I ENABLE ROW LEVEL SECURITY',tabla);
  EXECUTE format('ALTER TABLE vec_cronos_v1.%I FORCE ROW LEVEL SECURITY',tabla);
  EXECUTE format('CREATE POLICY lectura_publicada ON vec_cronos_v1.%I FOR SELECT TO vec_cronos_v1_propietario USING (true)',tabla);
  EXECUTE format('CREATE POLICY publicacion ON vec_cronos_v1.%I FOR INSERT TO vec_cronos_v1_propietario WITH CHECK (true)',tabla);
 END LOOP;
 FOREACH tabla IN ARRAY ARRAY['calendario_dia','calendario_asignacion','permiso_catalogo','correccion_solicitud','correccion_actuacion',
   'permiso_solicitud','permiso_estado','solicitud_outbox','movimientos_acceso','permisos_acceso','solicitud_replay'] LOOP
  EXECUTE format('CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE OR TRUNCATE ON vec_cronos_v1.%I FOR EACH STATEMENT EXECUTE FUNCTION vec_cronos_v1.rechazar_mutacion_historia()',tabla);
  EXECUTE format('REVOKE ALL ON TABLE vec_cronos_v1.%I FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',tabla);
 END LOOP;
END $seguridad$;

-- La frontera audita también las cuatro rutas nuevas. Sólo amplía la lista.
ALTER TABLE vec_cronos_v1.denegacion_frontera DROP CONSTRAINT denegacion_frontera_ruta_check;
ALTER TABLE vec_cronos_v1.denegacion_frontera ADD CONSTRAINT denegacion_frontera_ruta_check CHECK (ruta IN (
  '/api/interna/cronos/saldos/propio','/api/interna/cronos/marcajes/remoto',
  '/api/interna/cronos/marcajes/remoto/disponibilidad','/api/interna/cronos/marcajes/remoto/recibo',
  '/api/interna/cronos/movimientos/propio','/api/interna/cronos/correcciones/propias',
  '/api/interna/cronos/permisos/propio','/api/interna/cronos/permisos/solicitudes','otra'));

-- Material común: objeto con exactamente las claves dadas, todas cadenas, y
-- actor, perfil y empleado canónicos. Sin EXECUTE para ningún LOGIN.
CREATE FUNCTION vec_cronos_v1.material_propio_v1(p_material text,p_claves text[])
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE m jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096 THEN
   RAISE EXCEPTION 'material Cronos inválido' USING ERRCODE='PC001';
 END IF;
 BEGIN
   m:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM jsonb_object_keys(m))<>cardinality(p_claves)
    OR m-p_claves<>'{}'::jsonb
    OR EXISTS (SELECT 1 FROM jsonb_each(m) e WHERE jsonb_typeof(e.value) IS DISTINCT FROM 'string')
    OR coalesce(m->>'actor_ref','') !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR coalesce(m->>'perfil_ref','') !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR coalesce(m->>'empleado_ref','') !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR coalesce(m->>'zona_horaria','') NOT IN ('Europe/Madrid','Atlantic/Canary') THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 RETURN m;
END $f$;

CREATE FUNCTION vec_cronos_v1.fecha_material_v1(p_valor text)
RETURNS date LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE f date;
BEGIN
 IF coalesce(p_valor,'') !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
   RAISE EXCEPTION 'fecha Cronos inválida' USING ERRCODE='PC001';
 END IF;
 f:=p_valor::date;
 IF to_char(f,'YYYY-MM-DD')<>p_valor OR extract(year FROM f) NOT BETWEEN 2000 AND 2100 THEN
   RAISE EXCEPTION 'fecha Cronos inválida' USING ERRCODE='PC001';
 END IF;
 RETURN f;
EXCEPTION WHEN data_exception THEN
 RAISE EXCEPTION 'fecha Cronos inválida' USING ERRCODE='PC001';
END $f$;

-- Consumo nominal común: acredita el empleado del contexto, comprueba la
-- decisión, espera los bloqueos indicados, consume la fachada AD3-70 de la
-- acción, revalida plazos y vínculo con reloj vivo y fija la RLS propia.
CREATE FUNCTION vec_cronos_v1.consumir_propio_v1(
    p_fachada text,p_accion text,p_tipo text,p_finalidad text,p_recurso text,p_bloqueos text[],
    m jsonb,p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea,
    OUT decision_ref text,OUT auditoria_ref text,OUT consumo_huella_sha256 text,OUT ahora timestamptz,OUT vence timestamptz,OUT material_sha256 text)
LANGUAGE plpgsql VOLATILE SET search_path=pg_catalog AS $f$
DECLARE c jsonb; d jsonb; x jsonb; vinculo jsonb; huella text; consumo record; b text;
BEGIN
 IF p_fachada NOT IN ('consumir_cronos_movimientos_propio_v3_atestada','registrar_y_consumir_cronos_correccion_v3_atestada',
     'consumir_cronos_permisos_propio_v3_atestada','registrar_y_consumir_cronos_permiso_v3_atestada') THEN
   RAISE EXCEPTION 'fachada Cronos desconocida' USING ERRCODE='PC003';
 END IF;
 BEGIN
   c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
   RAISE EXCEPTION 'autorización Cronos ilegible' USING ERRCODE='PC003';
 END;
 ahora:=date_trunc('microseconds',clock_timestamp());
 vinculo:=vec_cronos_v1.acreditar_empleado_contexto_v1(x,m->>'actor_ref',m->>'perfil_ref',m->>'empleado_ref',ahora);
 material_sha256:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 huella:=vec_cronos_v1.huella_contexto_empleado_v1(m->>'empleado_ref',material_sha256);
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
 IF ahora>=vence OR nullif(current_setting('vec.cronos.empleado_ref',true),'') IS NOT NULL THEN
   RAISE EXCEPTION 'vigencia o contexto Cronos no válidos' USING ERRCODE='PC003';
 END IF;
 PERFORM set_config('vec.cronos.empleado_ref',m->>'empleado_ref',true);
 decision_ref:=consumo.decision_ref; auditoria_ref:=consumo.auditoria_ref; consumo_huella_sha256:=consumo.consumo_huella_sha256;
END $f$;

-- Estado vigente de cada solicitud: la versión más alta.
CREATE FUNCTION vec_cronos_v1.estado_permiso_actual_v1(p_empleado text)
RETURNS TABLE(solicitud_ref text,version integer,estado text,pendiente_justificar boolean)
LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT DISTINCT ON (e.solicitud_ref) e.solicitud_ref,e.version,e.estado,e.pendiente_justificar
   FROM vec_cronos_v1.permiso_estado e WHERE e.empleado_ref=p_empleado
  ORDER BY e.solicitud_ref,e.version DESC
$f$;

-- Calendario del año asignado a la persona, o NULL si no se ha publicado.
CREATE FUNCTION vec_cronos_v1.calendario_de_v1(p_empleado text,p_anio integer)
RETURNS text LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT a.calendario_ref FROM vec_cronos_v1.calendario_asignacion a WHERE a.empleado_ref=p_empleado AND a.anio=p_anio
$f$;

CREATE FUNCTION vec_cronos_v1.consultar_movimientos_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
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
     SELECT (mo.instante_utc AT TIME ZONE zona)::date f,count(*) n FROM vec_cronos_v1.marcaje_original mo
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
END $f$;

CREATE FUNCTION vec_cronos_v1.solicitar_correccion_propia_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; ref text; fecha date; hoy date; previa vec_cronos_v1.correccion_solicitud%ROWTYPE;
 act vec_cronos_v1.correccion_actuacion%ROWTYPE; original text; actuacion text; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','marcaje_original_ref',
   'movimiento','fecha_civil','hora_pretendida','zona_horaria']);
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'movimiento' NOT IN ('entrada','salida','inicio_pausa','fin_pausa')
    OR m->>'hora_pretendida' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'
    OR (m->>'marcaje_original_ref'<>'' AND m->>'marcaje_original_ref' !~ '^marcaje:cronos:[A-Za-z0-9][A-Za-z0-9_-]{7,127}$') THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 fecha:=vec_cronos_v1.fecha_material_v1(m->>'fecha_civil');
 emp:=m->>'empleado_ref'; ref:='correccion:cronos:'||(m->>'clave_operacion');
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_propio_v1('registrar_y_consumir_cronos_correccion_v3_atestada','cronos.correccion.solicitar',
   'correccion_marcaje','solicitar_correccion_marcaje',ref,
   ARRAY['vec_cronos_v1:clave:'||(m->>'clave_operacion'),'vec_cronos_v1:empleado:'||emp],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.correccion_solicitud WHERE solicitud_ref=ref;
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   SELECT * INTO STRICT act FROM vec_cronos_v1.correccion_actuacion WHERE solicitud_ref=ref AND version=1;
   INSERT INTO vec_cronos_v1.solicitud_replay(decision_ref,empleado_ref,agregado_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,emp,ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('solicitud_ref',ref,'actuacion_ref',act.actuacion_ref,'recibo_ref',act.recibo_ref,'estado',act.estado,
     'version',act.version,'instante_utc',act.registrada_en,'replay',true);
 END IF;
 -- Un olvido se declara sobre un día ya vivido y reciente; nunca a futuro.
 hoy:=(k.ahora AT TIME ZONE (m->>'zona_horaria'))::date;
 IF fecha>hoy OR fecha<hoy-366 THEN
   RAISE EXCEPTION 'fecha de corrección fuera de plazo' USING ERRCODE='PC001';
 END IF;
 original:=nullif(m->>'marcaje_original_ref','');
 IF original IS NOT NULL AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.marcaje_original mo WHERE mo.marcaje_ref=original AND mo.empleado_ref=emp) THEN
   RAISE EXCEPTION 'marcaje original ajeno o inexistente' USING ERRCODE='PC001';
 END IF;
 actuacion:='correccion:actuacion:'||gen_random_uuid()::text; recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.correccion_solicitud(solicitud_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,marcaje_original_ref,hueco_declarado,
   movimiento,fecha_civil,hora_pretendida,motivo_codigo,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,solicitada_en)
 VALUES(ref,emp,m->>'clave_operacion',m->>'actor_ref',m->>'perfil_ref',original,original IS NULL,m->>'movimiento',fecha,m->>'hora_pretendida',
   'olvido_marcaje',k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 INSERT INTO vec_cronos_v1.correccion_actuacion(actuacion_ref,solicitud_ref,empleado_ref,version,paso,estado,recibo_ref,registrada_en)
 VALUES(actuacion,ref,emp,1,'solicitud','pendiente_responsable',recibo,k.ahora);
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,ref,emp,'cronos.correccion.solicitada',
   jsonb_build_object('solicitud_ref',ref,'recibo_ref',recibo,'version',1,'estado','pendiente_responsable'),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('solicitud_ref',ref,'actuacion_ref',actuacion,'recibo_ref',recibo,'estado','pendiente_responsable',
   'version',1,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

-- Versión del catálogo vigente en un instante.
CREATE FUNCTION vec_cronos_v1.catalogo_vigente_v1(p_instante timestamptz)
RETURNS SETOF vec_cronos_v1.permiso_catalogo LANGUAGE sql STABLE SET search_path=pg_catalog AS $f$
 SELECT * FROM vec_cronos_v1.permiso_catalogo p
  WHERE p.vigente_desde<=p_instante AND (p.vigente_hasta IS NULL OR p_instante<p.vigente_hasta)
$f$;

CREATE FUNCTION vec_cronos_v1.consultar_permisos_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; anio integer; referencia timestamptz; inicio timestamptz; fin timestamptz; resultado jsonb;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','anio','zona_horaria']);
 IF m->>'anio' !~ '^[0-9]{4}$' OR (m->>'anio')::integer NOT BETWEEN 2000 AND 2100 THEN
   RAISE EXCEPTION 'año Cronos inválido' USING ERRCODE='PC001';
 END IF;
 anio:=(m->>'anio')::integer; emp:=m->>'empleado_ref';
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_propio_v1('consumir_cronos_permisos_propio_v3_atestada','cronos.permisos.propio.consultar',
   'permisos_propio','consultar_permisos_propio','permisos:cronos:'||emp,NULL,m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 INSERT INTO vec_cronos_v1.permisos_acceso(decision_ref,empleado_ref,anio,material_sha256,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(k.decision_ref,emp,anio,k.material_sha256,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 -- El catálogo del año es el vigente hoy si el año está en curso; si no, el
 -- de su primer o último instante.
 inicio:=make_date(anio,1,1)::timestamp AT TIME ZONE (m->>'zona_horaria');
 fin:=make_date(anio+1,1,1)::timestamp AT TIME ZONE (m->>'zona_horaria');
 referencia:=CASE WHEN k.ahora<inicio THEN inicio WHEN k.ahora>=fin THEN fin-interval '1 microsecond' ELSE k.ahora END;
 resultado:=jsonb_build_object('empleado_ref',emp,'anio',anio,
  'catalogo',coalesce((SELECT jsonb_agg(jsonb_build_object('permiso_ref',c.permiso_ref,'version_ref',c.version_ref,'nombre',c.nombre,
      'vigente_desde',c.vigente_desde,'vigente_hasta',c.vigente_hasta,'unidad',c.unidad,'computo',c.computo,'circuito',c.circuito,
      'minimo',c.minimo,'maximo_solicitud',c.maximo_solicitud,'maximo_mensual',c.maximo_mensual,'maximo_anual',c.maximo_anual,
      'justificante_exigido',c.justificante_exigido,'solicitable',c.solicitable AND k.ahora>=c.vigente_desde AND (c.vigente_hasta IS NULL OR k.ahora<c.vigente_hasta),
      'sintetico',c.sintetico) ORDER BY c.orden,c.permiso_ref)
     FROM vec_cronos_v1.catalogo_vigente_v1(referencia) c),'[]'::jsonb),
  'solicitudes',coalesce((SELECT jsonb_agg(jsonb_build_object('solicitud_ref',s.solicitud_ref,'catalogo_version_ref',s.catalogo_version_ref,
      'permiso_ref',s.permiso_ref,'desde',to_char(s.desde,'YYYY-MM-DD'),'hasta',to_char(s.hasta,'YYYY-MM-DD'),
      'hora_inicio',s.hora_inicio,'hora_fin',s.hora_fin,'cantidad',s.cantidad,'unidad',s.unidad,'estado',e.estado,'version',e.version,
      'pendiente_justificar',e.pendiente_justificar,'solicitada_en',s.solicitada_en) ORDER BY s.desde,s.solicitud_ref)
     FROM vec_cronos_v1.permiso_solicitud s JOIN vec_cronos_v1.estado_permiso_actual_v1(emp) e ON e.solicitud_ref=s.solicitud_ref
    WHERE s.empleado_ref=emp AND extract(year FROM s.desde)=anio),'[]'::jsonb));
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN resultado;
END $f$;

CREATE FUNCTION vec_cronos_v1.solicitar_permiso_propio_v1(
    p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE m jsonb; k record; emp text; ref text; desde date; hasta date; hoy date; cat vec_cronos_v1.permiso_catalogo%ROWTYPE;
 previa vec_cronos_v1.permiso_solicitud%ROWTYPE; est vec_cronos_v1.permiso_estado%ROWTYPE; cantidad bigint; calendario text;
 hi text; hf text; usado bigint; recibo text;
BEGIN
 m:=vec_cronos_v1.material_propio_v1(p_material,ARRAY['actor_ref','perfil_ref','empleado_ref','clave_operacion','permiso_ref',
   'desde','hasta','hora_inicio','hora_fin','zona_horaria']);
 IF coalesce(m->>'clave_operacion','') !~ '^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$'
    OR m->>'permiso_ref' !~ '^permiso:cronos:[-A-Za-z0-9_.]{1,96}$'
    OR (m->>'hora_inicio'<>'' AND m->>'hora_inicio' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$')
    OR (m->>'hora_fin'<>'' AND m->>'hora_fin' !~ '^([01][0-9]|2[0-3]):[0-5][0-9]$')
    OR (m->>'hora_inicio'='')<>(m->>'hora_fin'='') THEN
   RAISE EXCEPTION 'estructura Cronos inválida' USING ERRCODE='PC001';
 END IF;
 desde:=vec_cronos_v1.fecha_material_v1(m->>'desde'); hasta:=vec_cronos_v1.fecha_material_v1(m->>'hasta');
 IF hasta<desde OR extract(year FROM desde)<>extract(year FROM hasta) THEN
   RAISE EXCEPTION 'periodo Cronos inválido' USING ERRCODE='PC001';
 END IF;
 emp:=m->>'empleado_ref'; ref:='permiso:cronos:solicitud:'||(m->>'clave_operacion');
 hi:=nullif(m->>'hora_inicio',''); hf:=nullif(m->>'hora_fin','');
 SELECT * INTO STRICT k FROM vec_cronos_v1.consumir_propio_v1('registrar_y_consumir_cronos_permiso_v3_atestada','cronos.permiso.solicitar',
   'solicitud_permiso','solicitar_permiso_propio',ref,
   ARRAY['vec_cronos_v1:clave:'||(m->>'clave_operacion'),'vec_cronos_v1:empleado:'||emp,'vec_cronos_v1:permisos:'||emp],m,p_material,
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 SELECT * INTO previa FROM vec_cronos_v1.permiso_solicitud WHERE solicitud_ref=ref;
 IF FOUND THEN
   IF previa.material_sha256 IS DISTINCT FROM k.material_sha256 THEN
     RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
   END IF;
   SELECT * INTO STRICT est FROM vec_cronos_v1.permiso_estado WHERE solicitud_ref=ref AND version=1;
   INSERT INTO vec_cronos_v1.solicitud_replay(decision_ref,empleado_ref,agregado_ref,auditoria_ref,consumo_huella_sha256,registrada_en)
   VALUES(k.decision_ref,emp,ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
   IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
   RETURN jsonb_build_object('solicitud_ref',ref,'recibo_ref',est.recibo_ref,'catalogo_version_ref',previa.catalogo_version_ref,
     'version',1,'estado','solicitado','cantidad',previa.cantidad,'unidad',previa.unidad,'instante_utc',est.registrada_en,'replay',true);
 END IF;
 SELECT * INTO cat FROM vec_cronos_v1.catalogo_vigente_v1(k.ahora) c WHERE c.permiso_ref=m->>'permiso_ref';
 IF NOT FOUND OR NOT cat.solicitable THEN
   RAISE EXCEPTION 'permiso no solicitable' USING ERRCODE='PC009';
 END IF;
 hoy:=(k.ahora AT TIME ZONE (m->>'zona_horaria'))::date;
 IF desde<hoy-366 OR hasta>hoy+366 OR (cat.maximo_mensual IS NOT NULL AND date_trunc('month',desde)<>date_trunc('month',hasta)) THEN
   RAISE EXCEPTION 'periodo de permiso inválido' USING ERRCODE='PC001';
 END IF;
 IF cat.unidad='hora' THEN
   IF hi IS NULL OR desde<>hasta OR hf<=hi THEN
     RAISE EXCEPTION 'tramo horario inválido' USING ERRCODE='PC001';
   END IF;
   cantidad:=(extract(epoch FROM (hf::time-hi::time))/60)::bigint;
 ELSE
   IF hi IS NOT NULL THEN
     RAISE EXCEPTION 'permiso en días sin tramo horario' USING ERRCODE='PC001';
   END IF;
   IF cat.computo='naturales' THEN
     cantidad:=hasta-desde+1;
   ELSE
     calendario:=vec_cronos_v1.calendario_de_v1(emp,extract(year FROM desde)::integer);
     IF calendario IS NULL THEN
       RAISE EXCEPTION 'calendario laboral no publicado' USING ERRCODE='PC008';
     END IF;
     SELECT count(*) INTO cantidad FROM generate_series(desde,hasta,interval '1 day') g
      WHERE extract(isodow FROM g)<6
        AND NOT EXISTS (SELECT 1 FROM vec_cronos_v1.calendario_dia cd WHERE cd.calendario_ref=calendario AND cd.fecha=g::date);
     IF cantidad=0 THEN
       RAISE EXCEPTION 'sin días laborables' USING ERRCODE='PC007';
     END IF;
   END IF;
   -- Dos permisos en días no pueden solaparse mientras sigan vivos.
   IF EXISTS (SELECT 1 FROM vec_cronos_v1.permiso_solicitud s JOIN vec_cronos_v1.estado_permiso_actual_v1(emp) e ON e.solicitud_ref=s.solicitud_ref
       WHERE s.empleado_ref=emp AND s.unidad='dia' AND e.estado IN ('solicitado','pendiente_administracion','concedido')
         AND s.desde<=hasta AND s.hasta>=desde) THEN
     RAISE EXCEPTION 'permiso solapado' USING ERRCODE='PC010';
   END IF;
 END IF;
 IF cantidad<cat.minimo OR (cat.maximo_solicitud IS NOT NULL AND cantidad>cat.maximo_solicitud) THEN
   RAISE EXCEPTION 'cantidad fuera de límites' USING ERRCODE='PC007';
 END IF;
 IF cat.maximo_anual IS NOT NULL THEN
   SELECT coalesce(sum(s.cantidad),0) INTO usado FROM vec_cronos_v1.permiso_solicitud s JOIN vec_cronos_v1.estado_permiso_actual_v1(emp) e ON e.solicitud_ref=s.solicitud_ref
    WHERE s.empleado_ref=emp AND s.permiso_ref=cat.permiso_ref AND extract(year FROM s.desde)=extract(year FROM desde)
      AND e.estado IN ('solicitado','pendiente_administracion','concedido');
   IF usado+cantidad>cat.maximo_anual THEN
     RAISE EXCEPTION 'cupo anual superado' USING ERRCODE='PC007';
   END IF;
 END IF;
 IF cat.maximo_mensual IS NOT NULL THEN
   SELECT coalesce(sum(s.cantidad),0) INTO usado FROM vec_cronos_v1.permiso_solicitud s JOIN vec_cronos_v1.estado_permiso_actual_v1(emp) e ON e.solicitud_ref=s.solicitud_ref
    WHERE s.empleado_ref=emp AND s.permiso_ref=cat.permiso_ref AND date_trunc('month',s.desde)=date_trunc('month',desde)
      AND e.estado IN ('solicitado','pendiente_administracion','concedido');
   IF usado+cantidad>cat.maximo_mensual THEN
     RAISE EXCEPTION 'cupo mensual superado' USING ERRCODE='PC007';
   END IF;
 END IF;
 recibo:='recibo:cronos:'||gen_random_uuid()::text;
 INSERT INTO vec_cronos_v1.permiso_solicitud(solicitud_ref,empleado_ref,clave_operacion,actor_ref,perfil_ref,catalogo_version_ref,permiso_ref,
   desde,hasta,hora_inicio,hora_fin,cantidad,unidad,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,solicitada_en)
 VALUES(ref,emp,m->>'clave_operacion',m->>'actor_ref',m->>'perfil_ref',cat.version_ref,cat.permiso_ref,desde,hasta,hi,hf,cantidad,cat.unidad,
   k.material_sha256,k.decision_ref,k.auditoria_ref,k.consumo_huella_sha256,k.ahora);
 INSERT INTO vec_cronos_v1.permiso_estado(estado_ref,solicitud_ref,empleado_ref,version,estado,pendiente_justificar,recibo_ref,registrada_en)
 VALUES('permiso:estado:'||gen_random_uuid()::text,ref,emp,1,'solicitado',false,recibo,k.ahora);
 INSERT INTO vec_cronos_v1.solicitud_outbox(evento_ref,agregado_ref,empleado_ref,tipo,carga_json,creada_en)
 VALUES('evento:cronos:'||gen_random_uuid()::text,ref,emp,'cronos.permiso.solicitado',
   jsonb_build_object('solicitud_ref',ref,'recibo_ref',recibo,'version',1,'circuito',cat.circuito,'catalogo_version_ref',cat.version_ref),k.ahora);
 IF clock_timestamp()>=k.vence THEN RAISE EXCEPTION 'vigencia Cronos agotada' USING ERRCODE='PC003'; END IF;
 RETURN jsonb_build_object('solicitud_ref',ref,'recibo_ref',recibo,'catalogo_version_ref',cat.version_ref,'version',1,'estado','solicitado',
   'cantidad',cantidad,'unidad',cat.unidad,'instante_utc',k.ahora,'replay',false);
EXCEPTION WHEN unique_violation THEN
 RAISE EXCEPTION 'clave Cronos en conflicto' USING ERRCODE='PC002';
END $f$;

DO $acl$
DECLARE f text;
BEGIN
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.comprobar_catalogo_permiso_v1()',
   'vec_cronos_v1.material_propio_v1(text,text[])',
   'vec_cronos_v1.fecha_material_v1(text)',
   'vec_cronos_v1.consumir_propio_v1(text,text,text,text,text,text[],jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.estado_permiso_actual_v1(text)',
   'vec_cronos_v1.calendario_de_v1(text,integer)',
   'vec_cronos_v1.catalogo_vigente_v1(timestamptz)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_ejecutor,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
 END LOOP;
 FOREACH f IN ARRAY ARRAY[
   'vec_cronos_v1.consultar_movimientos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.solicitar_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.consultar_permisos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
   'vec_cronos_v1.solicitar_permiso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'] LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,vec_cronos_v1_migrador,vec_cronos_v1_auditor',f);
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_cronos_v1_ejecutor',f);
 END LOOP;
END $acl$;
COMMIT;
