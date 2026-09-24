\set ON_ERROR_STOP on
-- D7. Instalar tras Base 000010–000011, Composición 000010a y las cuatro fachadas AD3-59.
-- La alta inicial v1 consume autorización propia y referencias explícitas; no
-- crea datos de asignación desde una sesión, un cargo ni una relación inferida.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000012:asignacion-dietas:v1',0));
DO $rol$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_d7_ejecutor')
 THEN RAISE EXCEPTION 'Personal 000012: rol D7 preexistente o migrador no autorizado'
      USING ERRCODE='55000'; END IF;
END $rol$;
CREATE ROLE vec_personal_d7_ejecutor
 NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_personal_propietario;

DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.asignacion_dietas') IS NULL
    OR to_regprocedure('vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)') IS NOT NULL
    OR to_regprocedure('vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_grupo_dieta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_grupo_dieta_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_d7_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'Personal 000012: falta postimagen AD3 nominal o estado incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_personal.recibo_asignacion_dietas (
 referencia text PRIMARY KEY CHECK(referencia~'^rad_[0-9a-f]{32}$'),
 operacion text NOT NULL CHECK(operacion IN ('alta_inicial','consulta','correccion','correccion_grupo')),
 clave_idempotencia text CHECK(clave_idempotencia IS NULL OR clave_idempotencia~'^[A-Za-z0-9_-]{16,128}$'),
 huella_semantica_sha256 text CHECK(huella_semantica_sha256 IS NULL OR huella_semantica_sha256~'^[0-9a-f]{64}$'),
 asignacion_ref text NOT NULL REFERENCES vec_personal.asignacion_dietas(asignacion_ref),
 relacion_ref text NOT NULL, persona_ref text NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 unidad_ref text NOT NULL, version bigint NOT NULL CHECK(version>0),
 decision_ref text NOT NULL, efecto_ref text NOT NULL, consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'), auditoria_ad3_ref text NOT NULL,
 registrada_en timestamptz(6) NOT NULL,
 UNIQUE(relacion_ref,clave_idempotencia),
 CHECK((operacion='consulta')=(clave_idempotencia IS NULL AND huella_semantica_sha256 IS NULL))
);
CREATE TABLE vec_personal.evidencia_asignacion_dietas (
 evidencia_ref text PRIMARY KEY CHECK(evidencia_ref~'^ead_[0-9a-f]{32}$'), recibo_ref text NOT NULL UNIQUE REFERENCES vec_personal.recibo_asignacion_dietas(referencia),
 tipo text NOT NULL CHECK(tipo IN ('alta_inicial_asignacion_dietas_autorizada','consulta_asignacion_dietas_autorizada','correccion_asignacion_dietas_autorizada','correccion_grupo_dieta_autorizada')), registrada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_personal.recibo_asignacion_dietas ENABLE ROW LEVEL SECURITY; ALTER TABLE vec_personal.recibo_asignacion_dietas FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evidencia_asignacion_dietas ENABLE ROW LEVEL SECURITY; ALTER TABLE vec_personal.evidencia_asignacion_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY recibo_asignacion_lectura_contextual ON vec_personal.recibo_asignacion_dietas
 FOR SELECT TO vec_personal_propietario
 USING(persona_ref=current_setting('vec.dietas.persona_ref',true)
   AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL);
CREATE POLICY recibo_asignacion_alta_contextual ON vec_personal.recibo_asignacion_dietas
 FOR INSERT TO vec_personal_propietario
 WITH CHECK(persona_ref=current_setting('vec.dietas.persona_ref',true)
   AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL);
CREATE POLICY evidencia_asignacion_alta_contextual ON vec_personal.evidencia_asignacion_dietas
 FOR INSERT TO vec_personal_propietario
 WITH CHECK(EXISTS(SELECT 1 FROM vec_personal.recibo_asignacion_dietas r
   WHERE r.referencia=recibo_ref
     AND r.persona_ref=current_setting('vec.dietas.persona_ref',true)));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_asignacion_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_asignacion_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.evidencia_asignacion_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.evidencia_asignacion_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();

-- Dietas invoca esta comprobación desde su función propietaria terminal. La
-- referencia y versión deben designar exactamente la última asignación vigente.
CREATE FUNCTION vec_personal.revalidar_asignacion_dietas_v1(
 p_relacion text,p_persona text,p_unidad text,p_asignacion text,p_version bigint,p_grupo smallint,p_centro text,p_administrativo text,p_responsable text,p_fecha date)
RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE r vec_personal.relacion_empleado_dietas%ROWTYPE;
        a vec_personal.asignacion_dietas%ROWTYPE;
        hoy date := (statement_timestamp() AT TIME ZONE 'UTC')::date;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members m
       WHERE m.member=session_user::regrole
         AND m.roleid='vec_dietas_ejecutor'::regrole
         AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR EXISTS(SELECT 1 FROM pg_roles v
       WHERE left(v.rolname,4)='vec_' AND v.rolname<>session_user
         AND v.rolname<>'vec_dietas_ejecutor'
         AND pg_has_role(session_user,v.oid,'MEMBER'))
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_relacion !~ '^rel_[A-Za-z0-9_-]{22,128}$'
    OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_unidad IS NULL OR p_asignacion !~ '^ads_[A-Za-z0-9_-]{22,128}$'
    OR p_version IS NULL OR p_version<1 OR p_grupo NOT BETWEEN 1 AND 3
    OR p_fecha IS DISTINCT FROM hoy
    OR p_centro IS NULL OR p_administrativo !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR p_responsable !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_fecha IS NULL
 THEN RAISE EXCEPTION 'sello de asignación Personal inválido' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',p_persona,true);
 SELECT * INTO r FROM vec_personal.relacion_empleado_dietas
 WHERE relacion_ref=p_relacion AND persona_ref=p_persona AND unidad_ref=p_unidad
 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'relación Personal no disponible' USING ERRCODE='P7201'; END IF;
 PERFORM vec_personal.revalidar_relacion_dietas_v1(
  r.relacion_ref,r.persona_ref,r.empleado_ref,r.unidad_ref,r.desde::text,
  coalesce(r.hasta::text,''),r.version,r.procedencia_acto_ref,r.fuente_ref,r.fuente_version,p_fecha);
 SELECT * INTO a FROM vec_personal.asignacion_dietas
 WHERE relacion_ref=p_relacion AND persona_ref=p_persona AND unidad_ref=p_unidad
   AND vigente_desde<=p_fecha
 ORDER BY version DESC LIMIT 1 FOR SHARE;
 IF NOT FOUND OR a.asignacion_ref IS DISTINCT FROM p_asignacion
    OR a.version IS DISTINCT FROM p_version OR a.grupo_dieta IS DISTINCT FROM p_grupo
    OR a.centro_ref IS DISTINCT FROM p_centro
    OR a.administrativo_persona_ref IS DISTINCT FROM p_administrativo
    OR a.responsable_persona_ref IS DISTINCT FROM p_responsable
 THEN RAISE EXCEPTION 'sello de asignación Personal no vigente' USING ERRCODE='P7201'; END IF;
 RETURN true;
END; $f$;

-- Un único núcleo conserva las mismas columnas y controles de consumo para
-- consulta, corrección ordinaria y corrección de grupo, con ACL de entrada distintas.
CREATE FUNCTION vec_personal.ejecutar_asignacion_dietas_interna_v1(
 p_material text,p_modo text,p_capacidad bytea,p_decision bytea,p_motivo bytea,
 p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,registrada_en timestamptz,estado_local text,
 asignacion_ref text,relacion_ref text,persona_ref text,unidad_ref text,centro_ref text,
 administrativo_persona_ref text,responsable_persona_ref text,grupo_dieta smallint,
 vigente_desde date,version bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; i jsonb; c jsonb; d jsonb; x jsonb;
 r vec_personal.relacion_empleado_dietas%ROWTYPE;
 a vec_personal.asignacion_dietas%ROWTYPE;
 vigente vec_personal.asignacion_dietas%ROWTYPE;
 previo vec_personal.recibo_asignacion_dietas%ROWTYPE;
 z record;
 actor_persona text; actor_empleado text; sujeto_persona text; sujeto_empleado text;
 relacion text; unidad text; clave text; sem text; material_sha text; recurso_sha text;
 amb text; atr text; accion text; audiencia text; finalidad text; tipo text;
 fecha date; hoy date := (statement_timestamp() AT TIME ZONE 'UTC')::date;
 ahora timestamptz(6); ref text; nueva_ref text;
 recibo_operacion text; nueva_version bigint;
 campos jsonb:='["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]'::jsonb;
BEGIN
 IF p_modo NOT IN ('registrar_inicial','consultar','corregir','grupo_corregir')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_d7_ejecutor','MEMBER')
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members m
       WHERE m.member=session_user::regrole
         AND m.roleid='vec_personal_d7_ejecutor'::regrole
         AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)<>1
    OR EXISTS(SELECT 1 FROM pg_roles v
       WHERE left(v.rolname,4)='vec_' AND v.rolname<>session_user
         AND v.rolname<>'vec_personal_d7_ejecutor'
         AND pg_has_role(session_user,v.oid,'MEMBER'))
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL
    OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL
    OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
    OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'operación de asignación Personal rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de asignación Personal inválido' USING ERRCODE='22023';
 END;
 i:=m->'identidad';
 IF jsonb_typeof(m)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
       ARRAY['administrativo_persona_ref','centro_ref','clave_idempotencia',
        'empleado_ref','esquema','fecha_referencia','grupo_dieta','identidad',
        'motivo_revision','operacion','persona_ref','procedencia_acto_ref',
        'relacion_ref','responsable_persona_ref','unidad_ref','version_esperada','vigente_desde']
    OR m->>'esquema'<>'vec.personal.asignacion-dietas.v1'
    OR m->>'operacion' IS DISTINCT FROM p_modo
    OR m->>'relacion_ref' !~ '^rel_[A-Za-z0-9_-]{22,128}$'
    OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR m->>'fecha_referencia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR jsonb_typeof(i)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(i) ORDER BY 1) IS DISTINCT FROM
       ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref',
        'cuenta_version','empleado_ref','perfil_ref','perfil_version',
        'persona_ref','persona_version']
 THEN RAISE EXCEPTION 'material de asignación no canónico' USING ERRCODE='22023'; END IF;
 relacion:=m->>'relacion_ref'; sujeto_persona:=m->>'persona_ref';
 sujeto_empleado:=m->>'empleado_ref'; unidad:=m->>'unidad_ref';
 fecha:=(m->>'fecha_referencia')::date;
 IF p_modo='consultar' AND fecha IS DISTINCT FROM hoy
 THEN RAISE EXCEPTION 'consulta histórica de asignación no disponible' USING ERRCODE='P7201'; END IF;
 SELECT e.valor->>'referencia' INTO actor_empleado
 FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
 WHERE jsonb_typeof(e.valor)='object'
   AND e.valor->>'tipo'='empleado' AND e.valor->>'estado'='activo';
 actor_persona:=x->>'persona_ref';
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 amb:='{"empleado_ref":'||to_jsonb(sujeto_empleado)::text||
      ',"persona_ref":'||to_jsonb(sujeto_persona)::text||
      ',"relacion_ref":'||to_jsonb(relacion)::text||
      ',"unidad_ref":'||to_jsonb(unidad)::text||'}';
 atr:='{"fecha_referencia":'||to_jsonb(fecha::text)::text||
      ',"material_sha256":'||to_jsonb(material_sha)::text||
      ',"operacion":'||to_jsonb(p_modo)::text||'}';
 recurso_sha:=encode(sha256(convert_to('{"ambitos":'||amb||',"atributos":'||atr||'}','UTF8')),'hex');
 accion:=CASE p_modo
   WHEN 'registrar_inicial' THEN 'personal.asignacion_dietas.registrar_inicial'
   WHEN 'consultar' THEN 'personal.asignacion_dietas.consultar'
   WHEN 'corregir' THEN 'personal.asignacion_dietas.corregir'
   ELSE 'personal.asignacion_dietas.grupo_corregir' END;
 audiencia:=CASE p_modo
   WHEN 'registrar_inicial' THEN 'vec_personal.asignacion_dietas.registrar_inicial.v1'
   WHEN 'consultar' THEN 'vec_personal.asignacion_dietas.consultar.v1'
   WHEN 'corregir' THEN 'vec_personal.asignacion_dietas.corregir.v1'
   ELSE 'vec_personal.asignacion_dietas.grupo_corregir.v1' END;
 finalidad:=CASE p_modo
   WHEN 'registrar_inicial' THEN 'registrar_asignacion_dietas_inicial'
   WHEN 'consultar' THEN 'preparar_borrador_dietas'
   WHEN 'corregir' THEN 'corregir_asignacion_dietas'
   ELSE 'corregir_grupo_dieta' END;
 tipo:=CASE p_modo
   WHEN 'registrar_inicial' THEN 'alta_inicial_asignacion_dietas_autorizada'
   WHEN 'consultar' THEN 'consulta_asignacion_dietas_autorizada'
   WHEN 'corregir' THEN 'correccion_asignacion_dietas_autorizada'
   ELSE 'correccion_grupo_dieta_autorizada' END;
 IF x->>'esquema'<>'vec.contexto-actor.vinculado.v2'
    OR actor_persona !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR actor_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR (SELECT count(*) FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
        WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
          AND e.valor->>'estado'='activo')<>1
    OR i->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR i->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR i->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR i->>'persona_ref' IS DISTINCT FROM actor_persona
    OR i->>'empleado_ref' IS DISTINCT FROM actor_empleado
    OR i->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR i->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR coalesce((x->>'persona_version')::numeric,0)<>p_persona_version
    OR coalesce((x->>'perfil_version')::numeric,0)<>p_perfil_version
    OR (p_modo='consultar' AND
        (sujeto_persona IS DISTINCT FROM actor_persona
         OR sujeto_empleado IS DISTINCT FROM actor_empleado))
    OR (p_modo='grupo_corregir' AND
        sujeto_persona IS NOT DISTINCT FROM actor_persona)
    OR unidad IS NULL OR length(unidad) NOT BETWEEN 1 AND 256
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR c->>'operacion' IS DISTINCT FROM accion
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'accion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'asignacion_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM relacion
    OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM recurso_sha
 THEN RAISE EXCEPTION 'capacidad de asignación Personal no corresponde' USING ERRCODE='42501'; END IF;
 IF p_modo='consultar' THEN
  IF m->'clave_idempotencia' IS DISTINCT FROM '""'::jsonb
     OR m->'version_esperada' IS DISTINCT FROM '0'::jsonb
     OR m->'centro_ref' IS DISTINCT FROM '""'::jsonb
     OR m->'administrativo_persona_ref' IS DISTINCT FROM '""'::jsonb
     OR m->'responsable_persona_ref' IS DISTINCT FROM '""'::jsonb
     OR m->'grupo_dieta' IS DISTINCT FROM '0'::jsonb
     OR m->'vigente_desde' IS DISTINCT FROM '""'::jsonb
     OR m->'motivo_revision' IS DISTINCT FROM '""'::jsonb
     OR m->'procedencia_acto_ref' IS DISTINCT FROM '""'::jsonb
  THEN RAISE EXCEPTION 'consulta de asignación no canónica' USING ERRCODE='22023'; END IF;
 ELSE
  IF m->>'clave_idempotencia' IS NULL
     OR m->>'clave_idempotencia' !~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
     OR (p_modo='registrar_inicial' AND m->'version_esperada' IS DISTINCT FROM '0'::jsonb)
     OR (p_modo<>'registrar_inicial' AND coalesce((m->>'version_esperada')::bigint,0)<1)
     OR m->>'vigente_desde' IS NULL
     OR m->>'vigente_desde' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
     OR (m->>'vigente_desde')::date>fecha
     OR length(coalesce(m->>'centro_ref','')) NOT BETWEEN 1 AND 160
     OR m->>'administrativo_persona_ref' IS NULL
     OR m->>'administrativo_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR m->>'responsable_persona_ref' IS NULL
     OR m->>'responsable_persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
     OR m->>'administrativo_persona_ref'=m->>'responsable_persona_ref'
     OR m->>'administrativo_persona_ref'=sujeto_persona
     OR m->>'responsable_persona_ref'=sujeto_persona
     OR coalesce((m->>'grupo_dieta')::smallint,0) NOT BETWEEN 1 AND 3
     OR length(coalesce(m->>'motivo_revision','')) NOT BETWEEN 3 AND 500
     OR m->>'motivo_revision' ~ '[[:cntrl:]]'
     OR length(coalesce(m->>'procedencia_acto_ref','')) NOT BETWEEN 1 AND 256
  THEN RAISE EXCEPTION 'corrección de asignación no canónica' USING ERRCODE='22023'; END IF;
 END IF;
 PERFORM set_config('vec.dietas.persona_ref',sujeto_persona,true);
 -- AD3 consume y audita en la misma transacción antes de leer Personal.
 IF p_modo='registrar_inicial' THEN
  SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSIF p_modo='consultar' THEN
  SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSIF p_modo='corregir' THEN
  SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT z FROM vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_grupo_dieta_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF z.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'asignación requiere consumo fresco' USING ERRCODE='P0573';
 END IF;
 SELECT * INTO r FROM vec_personal.relacion_empleado_dietas v
 WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
   AND v.empleado_ref=sujeto_empleado AND v.unidad_ref=unidad
   AND v.estado='activa' AND v.desde<=fecha AND (v.hasta IS NULL OR fecha<v.hasta)
 FOR SHARE;
 IF NOT FOUND THEN RAISE EXCEPTION 'relación/unidad Personal no vigente' USING ERRCODE='P7201'; END IF;
 IF p_modo='consultar' THEN
  SELECT * INTO a FROM vec_personal.asignacion_dietas v
  WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
    AND v.unidad_ref=unidad AND v.vigente_desde<=fecha
  ORDER BY v.version DESC LIMIT 1 FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'asignación Personal no disponible' USING ERRCODE='P7201'; END IF;
  ahora:=clock_timestamp();
  ref:='rad_'||substr(encode(sha256(convert_to(z.consumo_huella_sha256||'|'||material_sha||'|'||ahora::text,'UTF8')),'hex'),1,32);
  INSERT INTO vec_personal.recibo_asignacion_dietas VALUES(
   ref,'consulta',NULL,NULL,a.asignacion_ref,relacion,sujeto_persona,unidad,
   a.version,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,z.auditoria_ref,ahora);
  INSERT INTO vec_personal.evidencia_asignacion_dietas VALUES(
   'ead_'||substr(encode(sha256(convert_to(ref||'|'||z.consumo_huella_sha256,'UTF8')),'hex'),1,32),
   ref,tipo,ahora);
  RETURN QUERY SELECT ref,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,
   z.auditoria_ref,ahora,'consultada'::text,a.asignacion_ref,a.relacion_ref,
   a.persona_ref,a.unidad_ref,a.centro_ref,a.administrativo_persona_ref,
   a.responsable_persona_ref,a.grupo_dieta,a.vigente_desde,a.version;
  RETURN;
 END IF;
 clave:=m->>'clave_idempotencia'; sem:=material_sha;
 IF p_modo='registrar_inicial' THEN recibo_operacion:='alta_inicial';
 ELSIF p_modo='corregir' THEN recibo_operacion:='correccion';
 ELSE recibo_operacion:='correccion_grupo'; END IF;
 SELECT * INTO previo FROM vec_personal.recibo_asignacion_dietas v
 WHERE v.relacion_ref=relacion AND v.clave_idempotencia=clave;
 IF FOUND THEN
  IF previo.huella_semantica_sha256 IS DISTINCT FROM sem
     OR previo.operacion IS DISTINCT FROM recibo_operacion
  THEN RAISE EXCEPTION 'conflicto de idempotencia de asignación' USING ERRCODE='P7204'; END IF;
  SELECT * INTO a FROM vec_personal.asignacion_dietas v
  WHERE v.asignacion_ref=previo.asignacion_ref AND v.relacion_ref=relacion
    AND v.persona_ref=sujeto_persona AND v.unidad_ref=unidad;
  IF NOT FOUND THEN RAISE EXCEPTION 'recibo de asignación incompleto' USING ERRCODE='55000'; END IF;
  -- Recuperar el mismo recibo exige relación y competencia vigentes hoy,
  -- además de un consumo AD3 nuevo ligado al mismo material.
  IF r.desde>hoy
     OR (r.hasta IS NOT NULL AND hoy>=r.hasta)
  THEN RAISE EXCEPTION 'relación Personal no vigente al recuperar' USING ERRCODE='P7201'; END IF;
  IF p_modo IN ('registrar_inicial','corregir') THEN
   SELECT * INTO vigente FROM vec_personal.asignacion_dietas v
   WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
     AND v.unidad_ref=unidad AND v.vigente_desde<=hoy
   ORDER BY v.version DESC LIMIT 1 FOR SHARE;
   IF NOT FOUND OR vigente.administrativo_persona_ref IS DISTINCT FROM actor_persona
      OR actor_persona IS NOT DISTINCT FROM sujeto_persona
   THEN RAISE EXCEPTION 'administrativo actual no competente' USING ERRCODE='42501'; END IF;
  END IF;
  RETURN QUERY SELECT previo.referencia,z.decision_ref,z.efecto_ref,
   z.consumo_huella_sha256,z.auditoria_ref,previo.registrada_en,
   'replay_confirmado'::text,a.asignacion_ref,a.relacion_ref,a.persona_ref,
   a.unidad_ref,a.centro_ref,a.administrativo_persona_ref,
   a.responsable_persona_ref,a.grupo_dieta,a.vigente_desde,a.version;
  RETURN;
 END IF;
 IF fecha IS DISTINCT FROM hoy
 THEN RAISE EXCEPTION 'fecha de asignación no actual' USING ERRCODE='P7201'; END IF;
 IF p_modo<>'registrar_inicial' AND (m->>'vigente_desde')::date IS DISTINCT FROM fecha
 THEN RAISE EXCEPTION 'corrección debe iniciar hoy' USING ERRCODE='P7201'; END IF;
 SELECT * INTO a FROM vec_personal.asignacion_dietas v
 WHERE v.relacion_ref=relacion AND v.persona_ref=sujeto_persona
   AND v.unidad_ref=unidad AND v.vigente_desde<=fecha
 ORDER BY v.version DESC LIMIT 1 FOR UPDATE;
 IF p_modo='registrar_inicial' THEN
  IF FOUND OR EXISTS(SELECT 1 FROM vec_personal.asignacion_dietas v WHERE v.relacion_ref=relacion)
  THEN RAISE EXCEPTION 'asignación inicial ya existe' USING ERRCODE='P7203'; END IF;
  IF sujeto_persona IS NOT DISTINCT FROM actor_persona
     OR sujeto_empleado IS NOT DISTINCT FROM actor_empleado
     OR actor_persona IS DISTINCT FROM m->>'administrativo_persona_ref'
     OR (m->>'vigente_desde')::date<r.desde
  THEN RAISE EXCEPTION 'alta inicial sin separación o vigencia válida' USING ERRCODE='42501'; END IF;
 ELSE
  IF NOT FOUND OR a.version IS DISTINCT FROM (m->>'version_esperada')::bigint
  THEN RAISE EXCEPTION 'versión de asignación no vigente' USING ERRCODE='P7203'; END IF;
 END IF;
 IF p_modo<>'registrar_inicial' AND (m->>'vigente_desde')::date<a.vigente_desde
 THEN RAISE EXCEPTION 'vigencia de asignación retroactiva' USING ERRCODE='P7201'; END IF;
 IF p_modo='corregir' AND (
    actor_persona IS DISTINCT FROM a.administrativo_persona_ref
    OR actor_persona IS NOT DISTINCT FROM sujeto_persona)
 THEN RAISE EXCEPTION 'administrativo anterior no competente' USING ERRCODE='42501'; END IF;
 IF p_modo='corregir' AND a.grupo_dieta IS DISTINCT FROM (m->>'grupo_dieta')::smallint
 THEN RAISE EXCEPTION 'grupo de dieta requiere concesión propia' USING ERRCODE='42501'; END IF;
 IF p_modo='grupo_corregir' AND (
   a.grupo_dieta IS NOT DISTINCT FROM (m->>'grupo_dieta')::smallint
   OR a.centro_ref IS DISTINCT FROM m->>'centro_ref'
   OR a.administrativo_persona_ref IS DISTINCT FROM m->>'administrativo_persona_ref'
   OR a.responsable_persona_ref IS DISTINCT FROM m->>'responsable_persona_ref')
 THEN RAISE EXCEPTION 'corrección de grupo altera otra asignación' USING ERRCODE='42501'; END IF;
 IF p_modo='registrar_inicial' THEN nueva_version:=1;
 ELSE nueva_version:=a.version+1; END IF;
 ahora:=clock_timestamp();
 nueva_ref:='ads_'||substr(encode(sha256(convert_to(relacion||'|'||sem||'|'||ahora::text,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.asignacion_dietas VALUES(
  nueva_ref,relacion,sujeto_persona,unidad,nueva_version,m->>'centro_ref',
  m->>'administrativo_persona_ref',m->>'responsable_persona_ref',
  (m->>'grupo_dieta')::smallint,(m->>'vigente_desde')::date,
  m->>'motivo_revision',m->>'procedencia_acto_ref',i->>'actor_ref',ahora);
 ref:='rad_'||substr(encode(sha256(convert_to(nueva_ref||'|'||z.consumo_huella_sha256,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.recibo_asignacion_dietas VALUES(
  ref,recibo_operacion,
  clave,sem,nueva_ref,relacion,sujeto_persona,unidad,nueva_version,
  z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,z.auditoria_ref,ahora);
 INSERT INTO vec_personal.evidencia_asignacion_dietas VALUES(
  'ead_'||substr(encode(sha256(convert_to(ref||'|'||z.consumo_huella_sha256,'UTF8')),'hex'),1,32),
  ref,tipo,ahora);
 RETURN QUERY SELECT ref,z.decision_ref,z.efecto_ref,z.consumo_huella_sha256,
  z.auditoria_ref,ahora,'registrada'::text,nueva_ref,relacion,sujeto_persona,
  unidad,m->>'centro_ref',m->>'administrativo_persona_ref',
  m->>'responsable_persona_ref',(m->>'grupo_dieta')::smallint,
  (m->>'vigente_desde')::date,nueva_version;
END; $f$;

CREATE FUNCTION vec_personal.registrar_asignacion_dietas_inicial_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,registrada_en timestamptz,estado_local text,
 asignacion_ref text,relacion_ref text,persona_ref text,unidad_ref text,centro_ref text,
 administrativo_persona_ref text,responsable_persona_ref text,grupo_dieta smallint,
 vigente_desde date,version bigint)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_asignacion_dietas_interna_v1(
  p_material,'registrar_inicial',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_personal.consultar_asignacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,registrada_en timestamptz,estado_local text,
 asignacion_ref text,relacion_ref text,persona_ref text,unidad_ref text,centro_ref text,
 administrativo_persona_ref text,responsable_persona_ref text,grupo_dieta smallint,
 vigente_desde date,version bigint)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_asignacion_dietas_interna_v1(
  p_material,'consultar',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_personal.corregir_asignacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,registrada_en timestamptz,estado_local text,
 asignacion_ref text,relacion_ref text,persona_ref text,unidad_ref text,centro_ref text,
 administrativo_persona_ref text,responsable_persona_ref text,grupo_dieta smallint,
 vigente_desde date,version bigint)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_asignacion_dietas_interna_v1(
  p_material,'corregir',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
CREATE FUNCTION vec_personal.corregir_grupo_dieta_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(
 recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,registrada_en timestamptz,estado_local text,
 asignacion_ref text,relacion_ref text,persona_ref text,unidad_ref text,centro_ref text,
 administrativo_persona_ref text,responsable_persona_ref text,grupo_dieta smallint,
 vigente_desde date,version bigint)
LANGUAGE sql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_personal.ejecutar_asignacion_dietas_interna_v1(
  p_material,'grupo_corregir',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz)
$f$;
REVOKE ALL ON TABLE vec_personal.recibo_asignacion_dietas,
 vec_personal.evidencia_asignacion_dietas
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_dietas_ejecutor,vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_personal.ejecutar_asignacion_dietas_interna_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_dietas_ejecutor,vec_dietas_propietario;
REVOKE ALL ON FUNCTION
 vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date),
 vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_ejecutor,vec_personal_d7_ejecutor,vec_dietas_ejecutor,vec_dietas_propietario;
GRANT USAGE ON SCHEMA vec_personal TO vec_dietas_ejecutor,vec_dietas_propietario,vec_personal_ejecutor,vec_personal_d7_ejecutor;
GRANT EXECUTE ON FUNCTION
 vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_d7_ejecutor;
GRANT EXECUTE ON FUNCTION
 vec_personal.revalidar_asignacion_dietas_v1(text,text,text,text,bigint,smallint,text,text,text,date)
 TO vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION
 vec_personal.registrar_asignacion_dietas_inicial_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_personal.corregir_grupo_dieta_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_d7_ejecutor;
COMMIT;
