\set ON_ERROR_STOP on
-- Personal 000027: lectura nominal de relación laboral para RPT.
-- Fuente Go del lector: 1871cab4e14eabe5394bba072b7a714dd9f25c86.
-- Contrato de intentos y vigencia actual: 9f0ae4d688a830e9e1be36291957c45205e112fb.
-- Orden causal SQL: POST149 -> AD154 -> Personal27. El runtime de intentos
-- requiere AD169/CA26/IS13 y su LOGIN/pool común, provisionado fuera de Git.
-- La reserva 27 comprende el control de generaciones y los recibos.
-- No crea destino ni rol de auditoría. Conserva Personal13/17/19/22 y B2.
-- AD154 parte de la preimagen causal POST149 del clon principal PG18.4.
-- Integrar e instalar sólo tras revisión y ensayo del hash final.
-- No reaplicar sobre una instalación existente ni retirar historia conservada.

BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000027:relacion-rpt',0));

SET LOCAL ROLE vec_personal_propietario;
DO $pre$
DECLARE consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_personal_propietario'
 OR consumidor IS NULL
 OR to_regclass('vec_personal.relacion_servicio_historia') IS NULL
 OR to_regprocedure('vec_personal.rechazar_mutacion_registro_empleado_v1()') IS NULL
 OR to_regprocedure('vec_personal.validar_revision_registro_empleado_v1()') IS NULL
 OR to_regprocedure('vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regclass('vec_personal.control_generacion_relacion_rpt') IS NOT NULL
 OR to_regclass('vec_personal.recibo_relacion_para_rpt') IS NOT NULL
 OR to_regprocedure('vec_personal.avanzar_generacion_relacion_rpt_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'PARO clave=Personal27.preimagen, actual=%/%/%, esperado=true/true/true',
  current_user='vec_personal_propietario',consumidor IS NOT NULL,
  to_regclass('vec_personal.relacion_servicio_historia') IS NOT NULL
  USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario'
   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
   AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_ejecutor'
   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
   AND NOT rolreplication AND NOT rolbypassrls)
 OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_personal_ejecutor'::regrole)
 OR NOT has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE')
 OR NOT has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE')
 OR (SELECT proowner FROM pg_proc WHERE oid=consumidor) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (SELECT prosecdef FROM pg_proc WHERE oid=consumidor) IS NOT TRUE
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=consumidor)<>2
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=consumidor AND (a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner
     OR a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole)))
 OR (SELECT relowner FROM pg_class WHERE oid='vec_personal.relacion_servicio_historia'::regclass) IS DISTINCT FROM 'vec_personal_propietario'::regrole
 OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid='vec_personal.relacion_servicio_historia'::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity)
 OR NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid='vec_personal.relacion_servicio_historia'::regclass
   AND p.polname='propietario_interno' AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid]
   AND p.polcmd='*' AND p.polpermissive)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.relacion_servicio_historia'::regclass
   AND t.tgname='historia_inmutable' AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=27
   AND t.tgfoid='vec_personal.rechazar_mutacion_registro_empleado_v1()'::regprocedure)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.relacion_servicio_historia'::regclass
   AND t.tgname='no_truncar' AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=34
   AND t.tgfoid='vec_personal.rechazar_mutacion_registro_empleado_v1()'::regprocedure)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.relacion_servicio_historia'::regclass
   AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=7
   AND t.tgfoid='vec_personal.validar_revision_registro_empleado_v1()'::regprocedure)
 OR (SELECT count(*) FROM pg_attribute a WHERE a.attrelid='vec_personal.relacion_servicio_historia'::regclass
   AND NOT a.attisdropped AND a.attnum>0 AND a.attname IN ('relacion_ref','revision','empleado_ref','organismo_ref',
     'estado','vigente_desde','vigente_hasta','conocido_desde','acto_ref','fuente_ref','fuente_version',
     'firma_oficial','eficacia_administrativa'))<>13
 OR NOT EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid='vec_personal.relacion_servicio_historia'::regclass
   AND a.attname='revision' AND NOT a.attisdropped AND a.attnotnull AND a.atttypid='integer'::regtype)
 THEN RAISE EXCEPTION 'PARO clave=Personal27.fuente_roles, actual=%/%/%, esperado=true/13/true',
  has_function_privilege('vec_personal_propietario',consumidor,'EXECUTE'),
  (SELECT count(*) FROM pg_attribute a WHERE a.attrelid='vec_personal.relacion_servicio_historia'::regclass
    AND NOT a.attisdropped AND a.attnum>0 AND a.attname IN ('relacion_ref','revision','empleado_ref','organismo_ref',
      'estado','vigente_desde','vigente_hasta','conocido_desde','acto_ref','fuente_ref','fuente_version',
      'firma_oficial','eficacia_administrativa')),
  NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_personal_ejecutor'::regrole)
  USING ERRCODE='55000'; END IF;
END $pre$;

-- El control contiene sólo referencia y generación, sin datos laborales.
-- FOR SHARE sobre una generación actualizada desde la instantánea SERIALIZABLE
-- produce 40001. El lector nunca crea ni avanza este control.
CREATE TABLE vec_personal.control_generacion_relacion_rpt (
 relacion_ref text PRIMARY KEY CHECK(relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
 generacion bigint NOT NULL CHECK(generacion BETWEEN 1 AND 2147483647)
);
ALTER TABLE vec_personal.control_generacion_relacion_rpt ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.control_generacion_relacion_rpt FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.control_generacion_relacion_rpt
 FOR ALL TO vec_personal_propietario USING(true) WITH CHECK(true);
-- La generación puede avanzar por INSERT de historia, pero no desaparecer.
CREATE TRIGGER no_borrar BEFORE DELETE ON vec_personal.control_generacion_relacion_rpt
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.control_generacion_relacion_rpt
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
REVOKE ALL ON TABLE vec_personal.control_generacion_relacion_rpt FROM PUBLIC,vec_personal_ejecutor;
REVOKE ALL ON TYPE vec_personal.control_generacion_relacion_rpt FROM PUBLIC,vec_personal_ejecutor;

-- Excluye INSERT antes del backfill y hasta publicar el AFTER INSERT. Se
-- conservan todas las filas y los triggers originales de Personal17.
LOCK TABLE vec_personal.relacion_servicio_historia IN SHARE ROW EXCLUSIVE MODE;
INSERT INTO vec_personal.control_generacion_relacion_rpt(relacion_ref,generacion)
 SELECT relacion_ref,max(revision)::bigint FROM vec_personal.relacion_servicio_historia GROUP BY relacion_ref;

CREATE FUNCTION vec_personal.avanzar_generacion_relacion_rpt_v1() RETURNS trigger
LANGUAGE plpgsql VOLATILE SECURITY INVOKER SET search_path=pg_catalog, pg_temp SET row_security=on AS $f$
DECLARE anterior bigint;
BEGIN
 IF current_user<>'vec_personal_propietario' OR TG_TABLE_SCHEMA<>'vec_personal'
 OR TG_TABLE_NAME<>'relacion_servicio_historia' OR TG_OP<>'INSERT' OR TG_WHEN<>'AFTER' THEN
  RAISE EXCEPTION 'Personal27: avance de generación inválido' USING ERRCODE='42501'; END IF;
 -- Personal17 ya tomó el mismo lock exclusivo antes de validar la revisión.
 -- Repetirlo conserva el contrato si cambia el orden de los triggers.
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:registro-b2:relacion_servicio_historia:'||NEW.relacion_ref,0));
 SELECT generacion INTO anterior FROM vec_personal.control_generacion_relacion_rpt
  WHERE relacion_ref=NEW.relacion_ref FOR UPDATE;
 IF NOT FOUND THEN
  IF NEW.revision<>1 THEN RAISE EXCEPTION 'Personal27: generación ausente' USING ERRCODE='55000'; END IF;
  INSERT INTO vec_personal.control_generacion_relacion_rpt(relacion_ref,generacion) VALUES(NEW.relacion_ref,1);
 ELSE
  IF NEW.revision::bigint<>anterior+1 THEN
   RAISE EXCEPTION 'Personal27: generación divergente' USING ERRCODE='55000'; END IF;
  UPDATE vec_personal.control_generacion_relacion_rpt SET generacion=NEW.revision WHERE relacion_ref=NEW.relacion_ref;
 END IF;
 RETURN NEW;
END $f$;
REVOKE ALL ON FUNCTION vec_personal.avanzar_generacion_relacion_rpt_v1() FROM PUBLIC,vec_personal_ejecutor;
CREATE TRIGGER relacion_rpt_generacion_insertada AFTER INSERT ON vec_personal.relacion_servicio_historia
 FOR EACH ROW EXECUTE FUNCTION vec_personal.avanzar_generacion_relacion_rpt_v1();

CREATE TABLE vec_personal.recibo_relacion_para_rpt (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^relacionrpt:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'),
 relacion_ref text NOT NULL CHECK(relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL CHECK(length(decision_ref) BETWEEN 1 AND 512),
 auditoria_ref text NOT NULL CHECK(length(auditoria_ref) BETWEEN 1 AND 512),
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consultada_en timestamptz(6) NOT NULL CHECK(isfinite(consultada_en))
);
-- El recibo es de solo adición y sólo el propietario accede a su tabla.
-- Los intentos tras rollback pertenecen a la auditoría común AD169.
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_relacion_para_rpt
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_relacion_para_rpt
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_registro_empleado_v1();
ALTER TABLE vec_personal.recibo_relacion_para_rpt ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_relacion_para_rpt FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.recibo_relacion_para_rpt
 FOR ALL TO vec_personal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_personal.recibo_relacion_para_rpt FROM PUBLIC,vec_personal_ejecutor;
REVOKE ALL ON TYPE vec_personal.recibo_relacion_para_rpt FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.consultar_relacion_para_rpt_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog, pg_temp SET row_security=on SET timezone='UTC'
 SET lock_timeout='2s' SET statement_timeout='5s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; x jsonb; consumo record; hecho vec_personal.relacion_servicio_historia%ROWTYPE;
 fecha date; conocido timestamptz(6); esperada bigint; generacion bigint;
 empleado text; relacion text; organismo text; persona text;
 material_canon text; material_sha text; contexto_canon text; contexto_sha text;
 ahora timestamptz(6); cap_desde timestamptz; cap_hasta timestamptz; dec_hasta timestamptz;
 actor_desde timestamptz; actor_hasta timestamptz; actor_resuelto timestamptz;
 vinculo jsonb; recibo text;
 campos constant jsonb:='["cobertura","corte","estado","periodo","procedencia","version"]';
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolcanlogin
   AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_auth_members WHERE member=session_user::regrole
   AND roleid='vec_personal_ejecutor'::regrole AND inherit_option AND NOT set_option AND NOT admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member='vec_personal_ejecutor'::regrole)
 OR has_schema_privilege(session_user,'vec_personal','CREATE')
 OR has_database_privilege(session_user,current_database(),'TEMPORARY')
 OR p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 4096
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 OR p_contexto IS NULL OR octet_length(p_contexto) NOT BETWEEN 2 AND 32768
 OR p_motivo IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL
 OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'Personal27: lectura nominal denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal27: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
 OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(x) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'Personal27: material no es objeto' USING ERRCODE='22023'; END IF;
 IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','conocido_en','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
   'empleado_ref','esquema','operacion','organismo_ref','perfil_ref','perfil_version',
   'persona_ref','persona_version','relacion_ref','version_esperada','vigente_en']
 OR EXISTS (SELECT 1 FROM jsonb_each(m) e WHERE
   (e.key IN ('version_esperada','contexto_version','cuenta_version','perfil_version','persona_version') AND jsonb_typeof(e.value)<>'number')
   OR (e.key NOT IN ('version_esperada','contexto_version','cuenta_version','perfil_version','persona_version') AND jsonb_typeof(e.value)<>'string'))
 OR m->>'esquema' IS DISTINCT FROM 'vec.personal.relacion-rpt.consulta.v1'
 OR m->>'operacion' IS DISTINCT FROM 'relacion_para_rpt'
 OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
 OR m->>'relacion_ref' !~ '^rel_[A-Za-z0-9_-]{22,128}$'
 OR m->>'organismo_ref' !~ '^[a-z][a-z0-9_:-]{2,127}$'
 OR m->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR m->>'contexto_actor_ref' !~ '^vca_[A-Za-z0-9_-]{22,128}$'
 OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
 OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
 OR m->>'vigente_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
 OR m->>'conocido_en' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z$'
 OR m->>'version_esperada' !~ '^[1-9][0-9]{0,18}$'
 OR EXISTS (SELECT 1 FROM jsonb_each_text(m) e WHERE e.key IN ('contexto_version','cuenta_version','perfil_version','persona_version')
   AND e.value !~ '^[1-9][0-9]{0,19}$') THEN
  RAISE EXCEPTION 'Personal27: material incompatible' USING ERRCODE='22023'; END IF;
 BEGIN
  fecha:=(m->>'vigente_en')::date; conocido:=(m->>'conocido_en')::timestamptz;
  esperada:=(m->>'version_esperada')::bigint;
  cap_desde:=(c->>'emitida_en')::timestamptz; cap_hasta:=(c->>'expira_en')::timestamptz;
  dec_hasta:=(d->>'valida_hasta')::timestamptz;
  actor_desde:=(x->>'vigente_desde')::timestamptz; actor_hasta:=(x->>'vigente_hasta')::timestamptz;
  actor_resuelto:=(x->>'resuelto_en')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'Personal27: fechas o versiones inválidas' USING ERRCODE='22023'; END;
 ahora:=clock_timestamp();
 empleado:=m->>'empleado_ref'; relacion:=m->>'relacion_ref'; organismo:=m->>'organismo_ref'; persona:=m->>'persona_ref';
 IF fecha IS NULL OR NOT isfinite(fecha) OR conocido IS NULL OR NOT isfinite(conocido) OR conocido>ahora
 OR fecha::text IS DISTINCT FROM m->>'vigente_en'
 OR to_char(conocido AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM m->>'conocido_en'
 OR esperada IS NULL OR esperada<1
 OR EXISTS (SELECT 1 FROM jsonb_each_text(m) e WHERE CASE
   WHEN e.key IN ('contexto_version','cuenta_version','perfil_version','persona_version')
   THEN e.value::numeric>18446744073709551615 ELSE false END)
 OR m->>'persona_version' IS DISTINCT FROM p_persona_version::text
 OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
 OR m->>'actor_ref' IS DISTINCT FROM persona
 OR m->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
 OR m->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
 OR m->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
 OR m->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
 OR m->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
 OR m->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
 OR m->>'perfil_version' IS DISTINCT FROM x->>'perfil_version'
 OR persona IS DISTINCT FROM x->>'persona_ref'
 OR m->>'persona_version' IS DISTINCT FROM x->>'persona_version'
 OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
 OR x->>'estado' IS DISTINCT FROM 'activo'
 OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 OR cap_desde IS NULL OR NOT isfinite(cap_desde) OR cap_hasta IS NULL OR NOT isfinite(cap_hasta)
 OR dec_hasta IS NULL OR NOT isfinite(dec_hasta)
 OR actor_desde IS NULL OR NOT isfinite(actor_desde) OR actor_hasta IS NULL OR NOT isfinite(actor_hasta)
 OR actor_resuelto IS NULL OR NOT isfinite(actor_resuelto)
 OR ahora<cap_desde OR ahora>=cap_hasta OR ahora>=dec_hasta
 OR ahora<actor_desde OR ahora>=actor_hasta OR ahora<actor_resuelto
 OR c->>'decision_valida_hasta' IS DISTINCT FROM d->>'valida_hasta'
 OR d->>'principal_id' IS DISTINCT FROM m->>'actor_ref'
 OR d->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
 OR d->'concedida' IS DISTINCT FROM 'true'::jsonb
 OR d->>'modulo_id' IS DISTINCT FROM 'personal'
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'Personal27: actor o vigencia divergente' USING ERRCODE='42501'; END IF;
 -- Canon exacto json.Marshal de los 17 campos; rechaza extras, duplicados,
 -- espacios, números alternativos, NULL y orden distinto del contrato Go.
 material_canon:='{"esquema":"vec.personal.relacion-rpt.consulta.v1","operacion":"relacion_para_rpt","empleado_ref":'||to_jsonb(empleado)::text||
  ',"relacion_ref":'||to_jsonb(relacion)::text||',"organismo_ref":'||to_jsonb(organismo)::text||
  ',"version_esperada":'||esperada::text||',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||
  ',"conocido_en":'||to_jsonb(m->>'conocido_en')::text||',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||
  ',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||',"contexto_version":'||(m->>'contexto_version')||
  ',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||',"cuenta_version":'||(m->>'cuenta_version')||
  ',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||',"perfil_version":'||(m->>'perfil_version')||
  ',"persona_ref":'||to_jsonb(persona)::text||',"persona_version":'||(m->>'persona_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'Personal27: material no canónico' USING ERRCODE='22023'; END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||',"organismo_ref":'||to_jsonb(organismo)::text||
  ',"relacion_ref":'||to_jsonb(relacion)::text||'},"atributos":{"conocido_en":'||to_jsonb(m->>'conocido_en')::text||
  ',"material_sha256":"'||material_sha||'","operacion":"relacion_para_rpt","version_esperada":'||to_jsonb(esperada::text)::text||
  ',"vigente_en":'||to_jsonb(m->>'vigente_en')::text||'}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM 'personal.relacion_rpt.consultar'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'relacion_para_rpt'
 OR d->>'finalidad' IS DISTINCT FROM 'conciliar_relacion_laboral_para_rpt'
 OR d->'campos_permitidos' IS DISTINCT FROM campos
 OR d->>'recurso_ref' IS DISTINCT FROM relacion
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
 OR c->>'operacion' IS DISTINCT FROM 'personal.relacion_rpt.consultar'
 OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.relacion_rpt.v1'
 OR c->>'efecto_ref' IS DISTINCT FROM relacion
 OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'Personal27: concesión nominal divergente' USING ERRCODE='42501'; END IF;

 -- Orden de locks compatible con Personal19: consumir primero V3 y después
 -- bloquear relación/control. AD154 revalida identidad y políticas registradas
 -- AHORA; ConocidoEn sólo determina el hecho histórico solicitado.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_relacion_para_rpt_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR consumo.efecto_ref IS DISTINCT FROM relacion OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 OR consumo.auditoria_ref IS NULL OR consumo.auditoria_ref=''
 OR consumo.consumo_huella_sha256 IS NULL OR consumo.consumo_huella_sha256 !~ '^[0-9a-f]{64}$' THEN
  RAISE EXCEPTION 'Personal27: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_personal:registro-b2:relacion_servicio_historia:'||relacion,0));
 SELECT g.generacion INTO generacion FROM vec_personal.control_generacion_relacion_rpt g
  WHERE g.relacion_ref=relacion FOR SHARE;
 IF NOT FOUND THEN
  IF EXISTS (SELECT 1 FROM vec_personal.relacion_servicio_historia h WHERE h.relacion_ref=relacion) THEN
   RAISE EXCEPTION 'Personal27: control de relación ausente' USING ERRCODE='55000'; END IF;
  RAISE EXCEPTION 'Personal27: relación no disponible' USING ERRCODE='42501';
 END IF;
 -- Seleccionar por relación y conocimiento, sin filtrar primero por empleado,
 -- organismo, versión, estado ni vigencia. Nunca rescatar una revisión anterior.
 SELECT h.* INTO hecho FROM vec_personal.relacion_servicio_historia h
  WHERE h.relacion_ref=relacion AND h.conocido_desde<=conocido
  ORDER BY h.conocido_desde DESC,h.revision DESC LIMIT 1;
 IF NOT FOUND THEN RAISE EXCEPTION 'Personal27: relación no disponible' USING ERRCODE='42501'; END IF;
 IF hecho.empleado_ref IS DISTINCT FROM empleado OR hecho.relacion_ref IS DISTINCT FROM relacion
 OR hecho.organismo_ref IS DISTINCT FROM organismo OR hecho.revision::bigint IS DISTINCT FROM esperada THEN
  RAISE EXCEPTION 'Personal27: objetivo o versión divergente' USING ERRCODE='42501'; END IF;
 IF generacion<hecho.revision OR hecho.firma_oficial IS DISTINCT FROM false
 OR hecho.eficacia_administrativa IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'Personal27: fuente incompatible' USING ERRCODE='55000'; END IF;
 -- La fecha civil del corte no altera el estado almacenado ni su periodo
 -- semiabierto. NULL en origen se traduce únicamente a hasta="".
 ahora:=date_trunc('microseconds',clock_timestamp());
 IF ahora<cap_desde OR ahora>=cap_hasta OR ahora>=dec_hasta OR ahora<actor_desde OR ahora>=actor_hasta
 OR ahora<actor_resuelto OR ahora<conocido THEN
  RAISE EXCEPTION 'Personal27: autorización caducada' USING ERRCODE='42501'; END IF;
 FOR vinculo IN SELECT value FROM jsonb_array_elements(x->'vinculos') LOOP
  IF vinculo->>'estado' IS DISTINCT FROM 'activo'
  OR vinculo->>'vigente_desde' IS NULL OR vinculo->>'vigente_hasta' IS NULL
  OR NOT isfinite((vinculo->>'vigente_desde')::timestamptz)
  OR NOT isfinite((vinculo->>'vigente_hasta')::timestamptz)
  OR ahora<(vinculo->>'vigente_desde')::timestamptz OR ahora>=(vinculo->>'vigente_hasta')::timestamptz THEN
   RAISE EXCEPTION 'Personal27: vínculo caducado' USING ERRCODE='42501'; END IF;
 END LOOP;
 recibo:='relacionrpt:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_relacion_para_rpt(recibo_ref,relacion_ref,material_sha256,
  decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,relacion,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 -- Cualquier error, incluido 40001 después de consumir, revierte consumo,
 -- auditoría y recibo al revertir la transacción completa. Aquí no se captura.
 RETURN jsonb_build_object(
  'relacion',jsonb_build_object('empleado_ref',hecho.empleado_ref,'relacion_ref',hecho.relacion_ref,
    'organismo_ref',hecho.organismo_ref,'version',hecho.revision,'estado',hecho.estado,
    'periodo',jsonb_build_object('desde',hecho.vigente_desde::text,'hasta',coalesce(hecho.vigente_hasta::text,'')),
    'procedencia',jsonb_build_object('acto_ref',hecho.acto_ref,'fuente_ref',hecho.fuente_ref,
      'fuente_version',hecho.fuente_version::text,'certeza','no_acreditado')),
  'corte',jsonb_build_object('vigente_en',m->>'vigente_en','conocido_en',m->>'conocido_en'),
  'cobertura','no_acreditada',
  'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
    'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
    'auditoria_ref',consumo.auditoria_ref,'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_ejecutor;

DO $post$
DECLARE o record; f record; permitido oid;
BEGIN
 FOR o IN SELECT c.oid,c.relname,c.relowner,c.relacl,c.relrowsecurity,c.relforcerowsecurity,c.reltype
  FROM pg_class c WHERE c.oid IN ('vec_personal.control_generacion_relacion_rpt'::regclass,
    'vec_personal.recibo_relacion_para_rpt'::regclass) LOOP
  IF o.relowner<>'vec_personal_propietario'::regrole OR NOT o.relrowsecurity OR NOT o.relforcerowsecurity
  OR EXISTS (SELECT 1 FROM aclexplode(coalesce(o.relacl,acldefault('r',o.relowner))) a
     WHERE a.grantee<>o.relowner OR a.grantor<>o.relowner OR a.is_grantable)
  OR EXISTS (SELECT 1 FROM pg_type t CROSS JOIN LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
     WHERE t.oid=o.reltype AND (a.grantee<>t.typowner OR a.grantor<>t.typowner OR a.is_grantable))
  OR (SELECT count(*) FROM pg_policy p WHERE p.polrelid=o.oid)<>1
  OR NOT EXISTS (SELECT 1 FROM pg_policy p WHERE p.polrelid=o.oid AND p.polname='propietario_interno'
     AND p.polroles=ARRAY['vec_personal_propietario'::regrole::oid] AND p.polcmd='*' AND p.polpermissive)
  THEN RAISE EXCEPTION 'Personal27: ACL/RLS de tabla divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOR f IN SELECT p.* FROM pg_proc p WHERE p.oid IN (
  'vec_personal.consultar_relacion_para_rpt_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_personal.avanzar_generacion_relacion_rpt_v1()'::regprocedure) LOOP
  permitido:=CASE f.proname WHEN 'consultar_relacion_para_rpt_v1' THEN 'vec_personal_ejecutor'::regrole::oid
    ELSE f.proowner END;
  IF f.proowner<>'vec_personal_propietario'::regrole OR f.provolatile<>'v' OR f.proparallel<>'u'
  OR f.prosecdef IS DISTINCT FROM (f.proname<>'avanzar_generacion_relacion_rpt_v1')
  OR (SELECT count(*) FROM unnest(f.proconfig) configuracion WHERE configuracion LIKE 'search_path=%')<>1
  OR NOT (f.proconfig @> ARRAY['search_path=pg_catalog, pg_temp','row_security=on'])
  OR (SELECT count(*) FROM aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))))<>(CASE WHEN permitido=f.proowner THEN 1 ELSE 2 END)
  OR EXISTS (SELECT 1 FROM aclexplode(coalesce(f.proacl,acldefault('f',f.proowner))) a
     WHERE a.privilege_type<>'EXECUTE' OR a.grantee NOT IN (f.proowner,permitido) OR a.grantor<>f.proowner OR a.is_grantable)
  THEN RAISE EXCEPTION 'Personal27: ACL de función divergente' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM vec_personal.control_generacion_relacion_rpt g FULL JOIN
    (SELECT relacion_ref,max(revision)::bigint AS generacion FROM vec_personal.relacion_servicio_historia GROUP BY relacion_ref) h
    USING(relacion_ref) WHERE g.generacion IS DISTINCT FROM h.generacion)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.relacion_servicio_historia'::regclass
    AND t.tgname='relacion_rpt_generacion_insertada' AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=5
    AND t.tgfoid='vec_personal.avanzar_generacion_relacion_rpt_v1()'::regprocedure)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.control_generacion_relacion_rpt'::regclass
    AND t.tgname='no_borrar' AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=11
    AND t.tgfoid='vec_personal.rechazar_mutacion_registro_empleado_v1()'::regprocedure)
 OR NOT EXISTS (SELECT 1 FROM pg_trigger t WHERE t.tgrelid='vec_personal.control_generacion_relacion_rpt'::regclass
    AND t.tgname='no_truncar' AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgtype=34
    AND t.tgfoid='vec_personal.rechazar_mutacion_registro_empleado_v1()'::regprocedure)
 THEN RAISE EXCEPTION 'Personal27: control divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
