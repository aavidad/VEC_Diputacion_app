\set ON_ERROR_STOP on
-- Vocabularios de Personal B2. Sin entradas precargadas: cada publicación o
-- retirada exige un acto nominal V3. Una revisión nunca sustituye otra.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000020:catalogos-b2',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.entrada_catalogo_registro_empleado_historia') IS NOT NULL
    OR to_regclass('vec_personal.registro_empleado_b2_recibo') IS NULL
    OR EXISTS (SELECT 1 FROM vec_personal.relacion_servicio_historia)
    OR EXISTS (SELECT 1 FROM vec_personal.ocupacion_empleado_historia)
    OR EXISTS (SELECT 1 FROM vec_personal.servicio_reconocido_historia)
    OR EXISTS (SELECT 1 FROM vec_personal.situacion_empleado_historia)
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 THEN RAISE EXCEPTION 'Personal 000020: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

ALTER TABLE vec_personal.relacion_servicio_historia ADD COLUMN catalogo_snapshot jsonb NOT NULL;
ALTER TABLE vec_personal.ocupacion_empleado_historia ADD COLUMN catalogo_snapshot jsonb NOT NULL;
ALTER TABLE vec_personal.servicio_reconocido_historia ADD COLUMN catalogo_snapshot jsonb NOT NULL;
ALTER TABLE vec_personal.situacion_empleado_historia ADD COLUMN catalogo_snapshot jsonb NOT NULL;

CREATE TABLE vec_personal.entrada_catalogo_registro_empleado_historia (
 organismo_ref text NOT NULL CHECK(organismo_ref ~ '^[a-z][a-z0-9_:-]{2,127}$'),
 tipo text NOT NULL CHECK(tipo IN ('regimen','modalidad','situacion','clase_servicio')),
 ref text NOT NULL CHECK(ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 version integer NOT NULL CHECK(version>0),
 revision integer NOT NULL CHECK(revision>0),
 estado text NOT NULL CHECK(estado IN ('publicada','retirada')),
 vigente_desde date NOT NULL CHECK(isfinite(vigente_desde)),
 vigente_hasta date CHECK(vigente_hasta IS NULL OR (isfinite(vigente_hasta) AND vigente_hasta>vigente_desde)),
 huella_sha256 text NOT NULL CHECK(huella_sha256 ~ '^[0-9a-f]{64}$'),
 denominacion text NOT NULL CHECK(octet_length(denominacion) BETWEEN 1 AND 256 AND denominacion=btrim(denominacion) AND denominacion !~ '[[:cntrl:]]'),
 acto_ref text NOT NULL CHECK(acto_ref ~ '^[a-z][a-z0-9_:-]{2,159}$'),
 actor_ref text NOT NULL,
 idempotencia_ref uuid NOT NULL UNIQUE,
 decision_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 auditoria_ref text NOT NULL,
 registrado_en timestamptz(6) NOT NULL CHECK(isfinite(registrado_en)),
 PRIMARY KEY(organismo_ref,tipo,ref,version,revision)
);
CREATE INDEX entrada_catalogo_registro_empleado_busqueda_idx
 ON vec_personal.entrada_catalogo_registro_empleado_historia(organismo_ref,tipo,ref,version,revision DESC);

-- Cabeza mutable de concurrencia. La historia conserva cada revisión; bloquear
-- esta fila fuerza 40001 al lector serializable ante una retirada simultánea.
CREATE TABLE vec_personal.entrada_catalogo_registro_empleado_actual (
 organismo_ref text NOT NULL,
 tipo text NOT NULL,
 ref text NOT NULL,
 version integer NOT NULL,
 revision integer NOT NULL,
 estado text NOT NULL CHECK(estado IN ('publicada','retirada')),
 PRIMARY KEY(organismo_ref,tipo,ref,version),
 FOREIGN KEY(organismo_ref,tipo,ref,version,revision)
  REFERENCES vec_personal.entrada_catalogo_registro_empleado_historia(organismo_ref,tipo,ref,version,revision)
);
ALTER TABLE vec_personal.entrada_catalogo_registro_empleado_actual ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.entrada_catalogo_registro_empleado_actual FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.entrada_catalogo_registro_empleado_actual
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.entrada_catalogo_registro_empleado_actual FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.validar_revision_catalogo_registro_empleado_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $f$
DECLARE anterior record; ultima_version integer;
BEGIN
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:'||NEW.organismo_ref||':'||NEW.tipo||':'||NEW.ref,0));
 SELECT max(version) INTO ultima_version FROM vec_personal.entrada_catalogo_registro_empleado_historia
  WHERE organismo_ref=NEW.organismo_ref AND tipo=NEW.tipo AND ref=NEW.ref;
 SELECT * INTO anterior FROM vec_personal.entrada_catalogo_registro_empleado_historia
  WHERE organismo_ref=NEW.organismo_ref AND tipo=NEW.tipo AND ref=NEW.ref AND version=NEW.version ORDER BY revision DESC LIMIT 1;
 IF NEW.estado='publicada' THEN
  IF FOUND OR NEW.revision<>1 OR NEW.version<>coalesce(ultima_version,0)+1 THEN
   RAISE EXCEPTION 'publicación de catálogo no consecutiva' USING ERRCODE='23514'; END IF;
  IF ultima_version IS NOT NULL AND EXISTS (
   SELECT 1 FROM vec_personal.entrada_catalogo_registro_empleado_historia e
   WHERE e.organismo_ref=NEW.organismo_ref AND e.tipo=NEW.tipo AND e.ref=NEW.ref AND e.version=ultima_version
     AND e.revision=(SELECT max(i.revision) FROM vec_personal.entrada_catalogo_registro_empleado_historia i
       WHERE i.organismo_ref=e.organismo_ref AND i.tipo=e.tipo AND i.ref=e.ref AND i.version=e.version)
     AND e.estado<>'retirada') THEN
   RAISE EXCEPTION 'versión previa aún publicada' USING ERRCODE='23514'; END IF;
 ELSE
  IF NOT FOUND OR anterior.estado<>'publicada' OR NEW.revision<>anterior.revision+1
     OR NEW.version<>ultima_version OR NEW.huella_sha256<>anterior.huella_sha256
     OR NEW.vigente_desde<>anterior.vigente_desde
     OR NEW.vigente_hasta IS DISTINCT FROM anterior.vigente_hasta
     OR NEW.denominacion<>anterior.denominacion THEN
   RAISE EXCEPTION 'retirada de catálogo inválida' USING ERRCODE='23514'; END IF;
 END IF;
 RETURN NEW;
END $f$;

CREATE FUNCTION vec_personal.actualizar_cabeza_catalogo_registro_empleado_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog AS $f$
BEGIN
 INSERT INTO vec_personal.entrada_catalogo_registro_empleado_actual(
  organismo_ref,tipo,ref,version,revision,estado)
 VALUES(NEW.organismo_ref,NEW.tipo,NEW.ref,NEW.version,NEW.revision,NEW.estado)
 ON CONFLICT (organismo_ref,tipo,ref,version) DO UPDATE
 SET revision=excluded.revision,estado=excluded.estado;
 RETURN NULL;
END $f$;

CREATE TRIGGER revision_continua BEFORE INSERT ON vec_personal.entrada_catalogo_registro_empleado_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.validar_revision_catalogo_registro_empleado_v1();
CREATE TRIGGER cabeza_actual AFTER INSERT ON vec_personal.entrada_catalogo_registro_empleado_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.actualizar_cabeza_catalogo_registro_empleado_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.entrada_catalogo_registro_empleado_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.entrada_catalogo_registro_empleado_historia
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.entrada_catalogo_registro_empleado_historia ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.entrada_catalogo_registro_empleado_historia FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.entrada_catalogo_registro_empleado_historia
 FOR ALL TO vec_personal_propietario USING (true) WITH CHECK (true);
REVOKE ALL ON TABLE vec_personal.entrada_catalogo_registro_empleado_historia FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.validar_entrada_registro_empleado_v1(
 p_organismo text,p_tipo text,p_ref text,p_version integer,p_fecha date) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog SET row_security=on AS $f$
DECLARE e vec_personal.entrada_catalogo_registro_empleado_historia%ROWTYPE; cabeza record;
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR p_organismo IS NULL OR p_organismo !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR p_tipo IS NULL OR p_tipo NOT IN ('regimen','modalidad','situacion','clase_servicio')
    OR p_ref IS NULL OR p_ref !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR p_version IS NULL OR p_version<1 OR p_fecha IS NULL OR NOT isfinite(p_fecha) THEN
  RAISE EXCEPTION 'entrada de catálogo inválida' USING ERRCODE='23514'; END IF;
 SELECT revision,estado INTO cabeza FROM vec_personal.entrada_catalogo_registro_empleado_actual
  WHERE organismo_ref=p_organismo AND tipo=p_tipo AND ref=p_ref AND version=p_version FOR SHARE;
 IF NOT FOUND OR cabeza.estado<>'publicada' THEN
  RAISE EXCEPTION 'entrada de catálogo ausente o retirada' USING ERRCODE='23514'; END IF;
 SELECT * INTO e FROM vec_personal.entrada_catalogo_registro_empleado_historia
 WHERE organismo_ref=p_organismo AND tipo=p_tipo AND ref=p_ref AND version=p_version AND revision=cabeza.revision;
 IF NOT FOUND OR e.estado<>'publicada' OR p_fecha<e.vigente_desde
    OR (e.vigente_hasta IS NOT NULL AND p_fecha>=e.vigente_hasta) THEN
  RAISE EXCEPTION 'entrada de catálogo ausente o retirada' USING ERRCODE='23514'; END IF;
 RETURN jsonb_build_object('organismo_ref',e.organismo_ref,'tipo',e.tipo,'ref',e.ref,'version',e.version,'revision',e.revision,
  'huella_sha256',e.huella_sha256,'vigente_desde',e.vigente_desde,
  'vigente_hasta',e.vigente_hasta,'estado',e.estado,'denominacion',e.denominacion);
END $f$;
REVOKE ALL ON FUNCTION vec_personal.validar_entrada_registro_empleado_v1(text,text,text,integer,date)
 FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
-- entrada/recibo son los campos de negocio V3; acceso_actual es el recibo
-- técnico del consumo del sobre en esta invocación, incluso en replay.
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; x jsonb; v record; previo record; anterior record;
 accion text; audiencia text; efecto text; recurso text; recurso_sha text; material_sha text;
 clave uuid; fecha_desde date; fecha_hasta date; ver integer; rev integer; estado text; huella_esperada text;
 cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' OR current_user<>'vec_personal_propietario'
    OR session_user=current_user OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 8192
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'acto de catálogo denegado' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
  clave:=(m->>'idempotencia_ref')::uuid;
  fecha_desde:=(m->>'vigente_desde')::date;
  fecha_hasta:=NULLIF(m->>'vigente_hasta','')::date;
  ver:=(m->>'version')::integer; rev:=(m->>'revision')::integer;
  cap_desde:=(c->>'emitida_en')::timestamptz;
  cap_hasta:=(c->>'expira_en')::timestamptz;
  dec_hasta:=(d->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de catálogo inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'acto_ref','actor_ref','denominacion','esquema','huella_sha256','idempotencia_ref','operacion','organismo_ref','ref','revision',
   'tipo','version','vigente_desde','vigente_hasta']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.catalogo-registro-empleado.v1'
    OR m->>'operacion' NOT IN ('publicar','retirar')
    OR m->>'tipo' NOT IN ('regimen','modalidad','situacion','clase_servicio')
    OR m->>'organismo_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR m->>'ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR m->>'huella_sha256' !~ '^[0-9a-f]{64}$'
    OR m->>'denominacion' IS NULL OR octet_length(m->>'denominacion') NOT BETWEEN 1 AND 256
    OR m->>'denominacion'<>btrim(m->>'denominacion') OR m->>'denominacion' ~ '[[:cntrl:]]'
    OR m->>'acto_ref' !~ '^[a-z][a-z0-9_:-]{2,159}$'
    OR ver<1 OR rev<1 OR fecha_desde IS NULL OR NOT isfinite(fecha_desde)
    OR (fecha_hasta IS NOT NULL AND (NOT isfinite(fecha_hasta) OR fecha_hasta<=fecha_desde))
    OR m->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR x->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR x->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
    OR cap_desde IS NULL OR cap_hasta IS NULL OR dec_hasta IS NULL
    OR NOT isfinite(cap_desde) OR NOT isfinite(cap_hasta) OR NOT isfinite(dec_hasta)
    OR cap_hasta<=cap_desde OR dec_hasta<=cap_desde
    OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta'
    OR clock_timestamp()<cap_desde OR clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta
    OR d->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'catálogo o identidad divergente' USING ERRCODE='42501'; END IF;
 estado:=CASE m->>'operacion' WHEN 'publicar' THEN 'publicada' ELSE 'retirada' END;
 accion:='personal.registro_empleado.catalogo.'||(m->>'operacion');
 audiencia:='vec_personal.registro_empleado.catalogo.'||(m->>'operacion')||'.v1';
 efecto:=(m->>'organismo_ref')||':'||(m->>'tipo')||':'||(m->>'ref')||':'||ver;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":'||to_jsonb(efecto)::text||',"organismo_ref":'||to_jsonb(m->>'organismo_ref')::text||'},"atributos":{"material_sha256":"'||material_sha||'","operacion":'||to_jsonb(m->>'operacion')::text||'}}';
 recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'efecto_ref' IS DISTINCT FROM efecto OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'accion' IS DISTINCT FROM accion OR d->>'recurso_ref' IS DISTINCT FROM efecto
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'tipo_recurso' IS DISTINCT FROM 'entrada_catalogo_empleado_rrhh'
    OR d->>'finalidad' IS DISTINCT FROM 'gobernar_catalogo_empleado'
    OR d->'campos_permitidos' IS DISTINCT FROM '["entrada","recibo"]'::jsonb THEN
  RAISE EXCEPTION 'acto de catálogo sin permiso nominal' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.efecto_ref IS DISTINCT FROM efecto
    OR v.huella_efecto_sha256 IS DISTINCT FROM recurso_sha THEN
  RAISE EXCEPTION 'consumo de catálogo divergente' USING ERRCODE='42501'; END IF;
 IF v.consumida_en<cap_desde OR v.consumida_en>=cap_hasta OR v.consumida_en>=dec_hasta THEN
  RAISE EXCEPTION 'consumo de catálogo fuera de vigencia' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:idempotencia:'||clave::text,0));
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de catálogo caducada' USING ERRCODE='42501'; END IF;
 SELECT * INTO previo FROM vec_personal.entrada_catalogo_registro_empleado_historia WHERE idempotencia_ref=clave;
 IF FOUND THEN
  IF previo.organismo_ref IS DISTINCT FROM m->>'organismo_ref' OR previo.tipo IS DISTINCT FROM m->>'tipo'
     OR previo.ref IS DISTINCT FROM m->>'ref' OR previo.version IS DISTINCT FROM ver
     OR previo.revision IS DISTINCT FROM rev OR previo.estado IS DISTINCT FROM estado
     OR previo.actor_ref IS DISTINCT FROM m->>'actor_ref' OR previo.acto_ref IS DISTINCT FROM m->>'acto_ref'
     OR previo.huella_sha256 IS DISTINCT FROM m->>'huella_sha256'
     OR previo.denominacion IS DISTINCT FROM m->>'denominacion'
     OR previo.vigente_desde IS DISTINCT FROM fecha_desde
     OR previo.vigente_hasta IS DISTINCT FROM fecha_hasta THEN
   RAISE EXCEPTION 'idempotencia de catálogo divergente' USING ERRCODE='23505'; END IF;
  RETURN jsonb_build_object('entrada',jsonb_build_object('organismo_ref',previo.organismo_ref,'tipo',previo.tipo,
   'ref',previo.ref,'version',previo.version,'revision',previo.revision,'estado',previo.estado,
   'denominacion',previo.denominacion,'huella_sha256',previo.huella_sha256,
   'vigente_desde',previo.vigente_desde,'vigente_hasta',previo.vigente_hasta),
   'recibo',jsonb_build_object('decision_ref',previo.decision_ref,'auditoria_ref',previo.auditoria_ref,
    'consumo_huella_sha256',previo.consumo_huella_sha256,'registrado_en',previo.registrado_en),
   'acceso_actual',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
    'consumo_huella_sha256',v.consumo_huella_sha256,'registrado_en',v.consumida_en,
   'estado_replay','replay'));
 END IF;
 -- El trigger de historia usa esta misma clave. Adquirirla antes de la última
 -- comprobación temporal impide que la espera del INSERT cruce la caducidad.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:catalogo-b2:'||
  (m->>'organismo_ref')||':'||(m->>'tipo')||':'||(m->>'ref'),0));
 IF estado='publicada' THEN
  huella_esperada:=encode(sha256(convert_to(concat_ws(E'\n',
   'vec.personal.catalogo-registro-empleado.entrada.v1',m->>'organismo_ref',m->>'tipo',m->>'ref',
   ver::text,rev::text,m->>'denominacion',fecha_desde::text,coalesce(fecha_hasta::text,''),m->>'acto_ref'),
   'UTF8')),'hex');
  IF m->>'huella_sha256' IS DISTINCT FROM huella_esperada THEN
   RAISE EXCEPTION 'huella de entrada divergente' USING ERRCODE='23514'; END IF;
 ELSE
  SELECT * INTO anterior FROM vec_personal.entrada_catalogo_registro_empleado_historia
   WHERE organismo_ref=m->>'organismo_ref' AND tipo=m->>'tipo' AND ref=m->>'ref'
     AND version=ver ORDER BY revision DESC LIMIT 1;
  IF NOT FOUND OR anterior.estado<>'publicada' OR rev<>anterior.revision+1
     OR m->>'huella_sha256' IS DISTINCT FROM anterior.huella_sha256
     OR m->>'denominacion' IS DISTINCT FROM anterior.denominacion
     OR fecha_desde IS DISTINCT FROM anterior.vigente_desde
     OR fecha_hasta IS DISTINCT FROM anterior.vigente_hasta THEN
   RAISE EXCEPTION 'retirada sin revisión vigente' USING ERRCODE='23514'; END IF;
 END IF;
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de catálogo caducada' USING ERRCODE='42501'; END IF;
 INSERT INTO vec_personal.entrada_catalogo_registro_empleado_historia(
  organismo_ref,tipo,ref,version,revision,estado,vigente_desde,vigente_hasta,huella_sha256,denominacion,
  acto_ref,actor_ref,idempotencia_ref,decision_ref,consumo_huella_sha256,auditoria_ref,registrado_en)
 VALUES(m->>'organismo_ref',m->>'tipo',m->>'ref',ver,rev,estado,fecha_desde,fecha_hasta,m->>'huella_sha256',m->>'denominacion',
  m->>'acto_ref',m->>'actor_ref',clave,v.decision_ref,v.consumo_huella_sha256,v.auditoria_ref,v.consumida_en);
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de catálogo caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('entrada',jsonb_build_object('organismo_ref',m->>'organismo_ref','tipo',m->>'tipo','ref',m->>'ref',
  'version',ver,'revision',rev,'estado',estado,'huella_sha256',m->>'huella_sha256',
  'denominacion',m->>'denominacion',
  'vigente_desde',fecha_desde,'vigente_hasta',fecha_hasta),
  'recibo',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'consumo_huella_sha256',v.consumo_huella_sha256,'registrado_en',v.consumida_en),
  'acceso_actual',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
  'consumo_huella_sha256',v.consumo_huella_sha256,'registrado_en',v.consumida_en,
  'estado_replay','registrado'));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.registrar_entrada_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_ejecutor;

CREATE FUNCTION vec_personal.consultar_catalogo_empleado_rrhh_v1(
 p_filtro text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE m jsonb; c jsonb; d jsonb; v record; e record;
 v_org text; v_tipo text; v_estado text; v_cursor_ref text; v_cursor_version integer; v_limite integer;
 efecto text; recurso text; recurso_sha text; material_sha text; entradas jsonb:='[]'::jsonb;
 cursor_siguiente jsonb:=NULL; contador integer:=0;
 cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC' OR current_user<>'vec_personal_propietario'
    OR session_user=current_user OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_filtro IS NULL OR octet_length(p_filtro) NOT BETWEEN 1 AND 4096
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'consulta de catálogo denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_filtro::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  v_org:=m->>'organismo_ref'; v_tipo:=m->>'tipo'; v_estado:=m->>'estado';
  v_cursor_ref:=m->>'cursor_ref'; v_cursor_version:=(m->>'cursor_version')::integer;
  v_limite:=(m->>'limite')::integer;
  cap_desde:=(c->>'emitida_en')::timestamptz;
  cap_hasta:=(c->>'expira_en')::timestamptz;
  dec_hasta:=(d->>'valida_hasta')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'filtro de catálogo inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
    'cursor_ref','cursor_version','esquema','estado','limite','organismo_ref','tipo']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.catalogo-registro-empleado.consulta.v1'
    OR v_org IS NULL OR v_org !~ '^[a-z][a-z0-9_:-]{2,127}$'
    OR v_tipo IS NULL OR v_tipo NOT IN ('regimen','modalidad','situacion','clase_servicio')
    OR (v_estado IS NOT NULL AND v_estado NOT IN ('publicada','retirada'))
    OR (v_cursor_ref IS NULL)<>(v_cursor_version IS NULL)
    OR (v_cursor_ref IS NOT NULL AND v_cursor_ref !~ '^[a-z][a-z0-9_:-]{2,159}$')
    OR (v_cursor_version IS NOT NULL AND v_cursor_version<1)
    OR v_limite IS NULL OR v_limite NOT BETWEEN 1 AND 100
    OR cap_desde IS NULL OR cap_hasta IS NULL OR dec_hasta IS NULL
    OR NOT isfinite(cap_desde) OR NOT isfinite(cap_hasta) OR NOT isfinite(dec_hasta)
    OR cap_hasta<=cap_desde OR dec_hasta<=cap_desde
    OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta' THEN
  RAISE EXCEPTION 'filtro de catálogo inválido' USING ERRCODE='22023'; END IF;
 IF clock_timestamp()<cap_desde OR clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de consulta caducada' USING ERRCODE='42501'; END IF;
 efecto:=v_org||':'||v_tipo;
 material_sha:=encode(sha256(convert_to(p_filtro,'UTF8')),'hex');
 recurso:='{"ambitos":{"objetivo_ref":'||to_jsonb(efecto)::text||',"organismo_ref":'||to_jsonb(v_org)::text||'},"atributos":{"material_sha256":"'||material_sha||'","operacion":"consultar"}}';
 recurso_sha:=encode(sha256(convert_to(recurso,'UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM 'personal.registro_empleado.catalogo.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.registro_empleado.catalogo.consultar.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM efecto OR c->>'huella_efecto_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'accion' IS DISTINCT FROM 'personal.registro_empleado.catalogo.consultar'
    OR d->>'recurso_ref' IS DISTINCT FROM efecto
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
    OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_empleado_rrhh'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_catalogo_empleado'
    OR d->'campos_permitidos' IS DISTINCT FROM '["cursor_siguiente","entradas","evidencia","organismo_ref"]'::jsonb
    OR d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'consulta de catálogo sin permiso nominal' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v FROM vec_autorizacion_atestada_v3.consumir_catalogo_registro_empleado_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v.consumo_nuevo IS NOT TRUE OR v.efecto_ref IS DISTINCT FROM efecto
    OR v.huella_efecto_sha256 IS DISTINCT FROM recurso_sha THEN
  RAISE EXCEPTION 'consumo de consulta divergente' USING ERRCODE='42501'; END IF;
 IF v.consumida_en<cap_desde OR v.consumida_en>=cap_hasta OR v.consumida_en>=dec_hasta THEN
  RAISE EXCEPTION 'consumo de consulta fuera de vigencia' USING ERRCODE='42501'; END IF;
 FOR e IN
  SELECT h.organismo_ref,h.tipo,h.ref,h.version,h.revision,h.estado,h.denominacion,
         h.huella_sha256,h.vigente_desde,h.vigente_hasta
   FROM vec_personal.entrada_catalogo_registro_empleado_actual a
   JOIN vec_personal.entrada_catalogo_registro_empleado_historia h
    ON h.organismo_ref=a.organismo_ref AND h.tipo=a.tipo AND h.ref=a.ref
       AND h.version=a.version AND h.revision=a.revision
   WHERE a.organismo_ref=v_org AND a.tipo=v_tipo AND (v_estado IS NULL OR a.estado=v_estado)
     AND (v_cursor_ref IS NULL OR (a.ref,a.version)>(v_cursor_ref,v_cursor_version))
   ORDER BY a.ref,a.version LIMIT v_limite+1
 LOOP
  contador:=contador+1;
  IF contador>v_limite THEN EXIT; END IF;
  entradas:=entradas||jsonb_build_array(jsonb_build_object(
   'organismo_ref',e.organismo_ref,'tipo',e.tipo,'ref',e.ref,'version',e.version,
   'revision',e.revision,'estado',e.estado,'denominacion',e.denominacion,
   'huella_sha256',e.huella_sha256,'vigente_desde',e.vigente_desde,'vigente_hasta',e.vigente_hasta));
  cursor_siguiente:=jsonb_build_object('ref',e.ref,'version',e.version);
 END LOOP;
 IF contador<=v_limite THEN cursor_siguiente:=NULL; END IF;
 IF clock_timestamp()>=cap_hasta OR clock_timestamp()>=dec_hasta THEN
  RAISE EXCEPTION 'capacidad de consulta caducada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('organismo_ref',v_org,'entradas',entradas,'cursor_siguiente',cursor_siguiente,
  'evidencia',jsonb_build_object('decision_ref',v.decision_ref,'auditoria_ref',v.auditoria_ref,
   'consumo_huella_sha256',v.consumo_huella_sha256,'efecto_ref',v.efecto_ref,
   'consultada_en',v.consumida_en));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_catalogo_empleado_rrhh_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_ejecutor;
COMMIT;
