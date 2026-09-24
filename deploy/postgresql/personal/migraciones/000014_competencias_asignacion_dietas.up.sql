\set ON_ERROR_STOP on
-- D7b. Consulta inversa nominal: actor -> asignaciones actuales en que figura
-- como administrativo o responsable. AD3 consume antes de leer Personal.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000014:competencias-asignacion-dietas:v1',0));

DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.asignacion_dietas') IS NULL
    OR to_regclass('vec_personal.relacion_empleado_dietas') IS NULL
    OR to_regprocedure('vec_personal.consultar_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_personal_d7_ejecutor'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regclass('vec_personal.recibo_competencias_asignacion_dietas') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_competencias_asignacion_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal 000014: falta D7/AD3 o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- La política solo expone filas candidatas al propietario. La función vuelve
-- a comprobar la última versión sin esta política para no aceptar un actor
-- sustituido en una corrección posterior.
CREATE POLICY competencias_actor_candidato ON vec_personal.asignacion_dietas
 FOR SELECT TO vec_personal_propietario
 USING (current_setting('vec.dietas.competencias_actor_persona_ref',true) IS NOT NULL
    AND (administrativo_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true)
      OR responsable_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true)));

CREATE TABLE vec_personal.recibo_competencias_asignacion_dietas (
 referencia text PRIMARY KEY CHECK(referencia~'^rca_[0-9a-f]{32}$'),
 actor_persona_ref text NOT NULL CHECK(actor_persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 decision_ref text NOT NULL, efecto_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ad3_ref text NOT NULL,
 fecha_referencia date NOT NULL,
 cardinalidad integer NOT NULL CHECK(cardinalidad BETWEEN 0 AND 100),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 contexto_sha256 text NOT NULL CHECK(contexto_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL
);
CREATE TABLE vec_personal.evidencia_competencias_asignacion_dietas (
 evidencia_ref text PRIMARY KEY CHECK(evidencia_ref~'^eca_[0-9a-f]{32}$'),
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_personal.recibo_competencias_asignacion_dietas(referencia),
 tipo text NOT NULL CHECK(tipo='consulta_competencias_asignacion_dietas_autorizada'),
 registrada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_personal.recibo_competencias_asignacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_competencias_asignacion_dietas FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evidencia_competencias_asignacion_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evidencia_competencias_asignacion_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY competencias_recibo_lectura_actor ON vec_personal.recibo_competencias_asignacion_dietas
 FOR SELECT TO vec_personal_propietario
 USING(actor_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true));
CREATE POLICY competencias_recibo_alta_actor ON vec_personal.recibo_competencias_asignacion_dietas
 FOR INSERT TO vec_personal_propietario
 WITH CHECK(actor_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true));
CREATE POLICY competencias_evidencia_alta_actor ON vec_personal.evidencia_competencias_asignacion_dietas
 FOR INSERT TO vec_personal_propietario
 WITH CHECK(EXISTS(SELECT 1 FROM vec_personal.recibo_competencias_asignacion_dietas r
   WHERE r.referencia=recibo_ref AND r.actor_persona_ref=current_setting('vec.dietas.competencias_actor_persona_ref',true)));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_competencias_asignacion_dietas
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_competencias_asignacion_dietas
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.evidencia_competencias_asignacion_dietas
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.evidencia_competencias_asignacion_dietas
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();

CREATE FUNCTION vec_personal.consultar_competencias_asignacion_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,
 p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,
 auditoria_ad3_ref text,consultada_en timestamptz,cardinalidad integer,competencias json)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='5s' AS $fn$
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; x jsonb;
 v_actor text; v_empleado text; v_fecha date; v_material_sha text;
 v_amb text; v_atr text; v_recurso_sha text; v_campos jsonb :=
 '["asignacion_ref","auditoria_ad3_ref","cardinalidad","competencias","consultada_en","consumo_huella_sha256","decision_ref","efecto_ref","recibo_ref","relacion_ref","rol","unidad_ref","version","vigente_desde"]'::jsonb;
 v_consumo record; candidato record; actual vec_personal.asignacion_dietas%ROWTYPE;
 relacion vec_personal.relacion_empleado_dietas%ROWTYPE;
 v_lista jsonb:='[]'::jsonb; v_n integer:=0; v_candidatos integer:=0; v_rol text;
 v_ahora timestamptz(6); v_recibo text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR current_setting('TimeZone')<>'UTC'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_personal_d7_ejecutor','MEMBER')
    OR NOT EXISTS(SELECT 1 FROM pg_auth_members miembros
       WHERE miembros.member=session_user::regrole
         AND miembros.roleid='vec_personal_d7_ejecutor'::regrole
         AND NOT miembros.admin_option AND miembros.inherit_option
         AND NOT miembros.set_option)
    OR (SELECT count(*) FROM pg_auth_members miembros
        WHERE miembros.member=session_user::regrole)<>1
    OR EXISTS(SELECT 1 FROM pg_roles rol
       WHERE left(rol.rolname,4)='vec_' AND rol.rolname<>session_user
         AND rol.rolname<>'vec_personal_d7_ejecutor'
         AND pg_has_role(session_user,rol.oid,'MEMBER'))
    OR pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL
    OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL
    OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL
    OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'consulta de competencias Personal rechazada' USING ERRCODE='42501'; END IF;
 BEGIN
  m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de competencias Personal inválido' USING ERRCODE='22023';
 END;
 i:=m->'identidad';
 IF jsonb_typeof(m)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM
       ARRAY['esquema','fecha_referencia','identidad']
    OR m->>'esquema'<>'vec.personal.asignacion-dietas.competencias.v1'
    OR m->>'fecha_referencia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR jsonb_typeof(i)<>'object'
    OR ARRAY(SELECT jsonb_object_keys(i) ORDER BY 1) IS DISTINCT FROM
       ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref',
        'cuenta_version','empleado_ref','perfil_ref','perfil_version',
        'persona_ref','persona_version']
 THEN RAISE EXCEPTION 'material de competencias no canónico' USING ERRCODE='22023'; END IF;
 v_fecha:=(m->>'fecha_referencia')::date;
 IF v_fecha IS DISTINCT FROM (clock_timestamp() AT TIME ZONE 'UTC')::date
 THEN RAISE EXCEPTION 'competencias históricas no disponibles' USING ERRCODE='P7201'; END IF;
 v_actor:=x->>'persona_ref';
 SELECT e.valor->>'referencia' INTO v_empleado
 FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
 WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
   AND e.valor->>'estado'='activo';
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_amb:='{"persona_ref":'||to_jsonb(v_actor)::text||'}';
 v_atr:='{"fecha_referencia":'||to_jsonb(v_fecha::text)::text||
        ',"material_sha256":'||to_jsonb(v_material_sha)::text||',"operacion":"lista"}';
 v_recurso_sha:=encode(sha256(convert_to('{"ambitos":'||v_amb||',"atributos":'||v_atr||'}','UTF8')),'hex');
 IF jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR jsonb_typeof(x) IS DISTINCT FROM 'object'
    OR x->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR v_actor IS NULL OR v_actor !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR v_empleado IS NULL OR v_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR (SELECT count(*) FROM jsonb_array_elements(coalesce(x->'vinculos','[]'::jsonb)) e(valor)
        WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado'
          AND e.valor->>'estado'='activo')<>1
    OR i->>'actor_ref' IS DISTINCT FROM x->>'principal_ref'
    OR i->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref'
    OR i->>'contexto_version' IS DISTINCT FROM x->>'contexto_version'
    OR i->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref'
    OR i->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version'
    OR i->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR i->>'persona_ref' IS DISTINCT FROM v_actor
    OR i->>'empleado_ref' IS DISTINCT FROM v_empleado
    OR i->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR i->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR coalesce((x->>'persona_version')::numeric,0)<>p_persona_version
    OR coalesce((x->>'perfil_version')::numeric,0)<>p_perfil_version
    OR c->>'audiencia_consumo'<>'vec_personal.asignacion_dietas.competencias.v1'
    OR c->>'operacion'<>'personal.asignacion_dietas.competencias_consultar'
    OR d->>'concedida'<>'true'
    OR d->>'accion'<>'personal.asignacion_dietas.competencias_consultar'
    OR d->>'modulo_id'<>'personal'
    OR d->>'tipo_recurso'<>'asignacion_dietas_competencias'
    OR d->>'recurso_ref' IS DISTINCT FROM v_actor
    OR d->>'finalidad'<>'tramitar_dietas_asignadas'
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM v_campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha
 THEN RAISE EXCEPTION 'capacidad de competencias Personal no corresponde' USING ERRCODE='42501'; END IF;

 PERFORM set_config('vec.dietas.competencias_actor_persona_ref',v_actor,true);
 SELECT * INTO STRICT v_consumo FROM
  vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
   p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE
 THEN RAISE EXCEPTION 'competencias requieren consumo fresco' USING ERRCODE='P0573'; END IF;

 -- Primero se localizan relaciones donde el actor figuró alguna vez. Para
 -- cada una, la política original por sujeto permite comprobar la última
 -- asignación vigente y la relación jurídica actual. Una versión vieja no
 -- concede competencia tras ser reemplazada.
 FOR candidato IN
  SELECT DISTINCT ON (a.relacion_ref) a.relacion_ref,a.persona_ref,a.unidad_ref
  FROM vec_personal.asignacion_dietas a
  WHERE a.administrativo_persona_ref=v_actor OR a.responsable_persona_ref=v_actor
  ORDER BY a.relacion_ref,a.version DESC
  LIMIT 1001
 LOOP
  v_candidatos:=v_candidatos+1;
  IF v_candidatos>1000
  THEN RAISE EXCEPTION 'consulta supera 1000 relaciones candidatas' USING ERRCODE='P7202'; END IF;
  PERFORM set_config('vec.dietas.persona_ref',candidato.persona_ref,true);
  SELECT * INTO actual FROM vec_personal.asignacion_dietas a
  WHERE a.relacion_ref=candidato.relacion_ref AND a.persona_ref=candidato.persona_ref
    AND a.unidad_ref=candidato.unidad_ref AND a.vigente_desde<=v_fecha
  ORDER BY a.version DESC LIMIT 1;
  IF NOT FOUND OR (actual.administrativo_persona_ref IS DISTINCT FROM v_actor
       AND actual.responsable_persona_ref IS DISTINCT FROM v_actor) THEN CONTINUE; END IF;
  SELECT * INTO relacion FROM vec_personal.relacion_empleado_dietas r
  WHERE r.relacion_ref=actual.relacion_ref AND r.persona_ref=actual.persona_ref
    AND r.unidad_ref=actual.unidad_ref AND r.estado='activa'
    AND r.desde<=v_fecha AND (r.hasta IS NULL OR v_fecha<r.hasta);
  IF NOT FOUND THEN CONTINUE; END IF;
  v_n:=v_n+1;
  IF v_n>100 THEN RAISE EXCEPTION 'consulta supera 100 competencias' USING ERRCODE='P7202'; END IF;
  v_rol:=CASE WHEN actual.administrativo_persona_ref=v_actor
              THEN 'administrativo' ELSE 'responsable' END;
  v_lista:=v_lista||jsonb_build_array(jsonb_build_object(
   'asignacion_ref',actual.asignacion_ref,'relacion_ref',actual.relacion_ref,
   'unidad_ref',actual.unidad_ref,'rol',v_rol,
   'vigente_desde',actual.vigente_desde::text,'version',actual.version));
 END LOOP;
 PERFORM set_config('vec.dietas.persona_ref','',true);
 v_ahora:=clock_timestamp();
 v_recibo:='rca_'||substr(encode(sha256(convert_to(
  v_consumo.consumo_huella_sha256||'|'||v_material_sha||'|'||v_ahora::text,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.recibo_competencias_asignacion_dietas VALUES(
  v_recibo,v_actor,v_consumo.decision_ref,v_consumo.efecto_ref,
  v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_fecha,v_n,
  v_material_sha,encode(sha256(p_contexto),'hex'),v_ahora);
 INSERT INTO vec_personal.evidencia_competencias_asignacion_dietas VALUES(
  'eca_'||substr(encode(sha256(convert_to(v_recibo||'|'||v_consumo.consumo_huella_sha256,'UTF8')),'hex'),1,32),
  v_recibo,'consulta_competencias_asignacion_dietas_autorizada',v_ahora);
 RETURN QUERY SELECT v_recibo,v_consumo.decision_ref,v_consumo.efecto_ref,
  v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_ahora,v_n,v_lista::json;
END $fn$;

REVOKE ALL ON TABLE vec_personal.recibo_competencias_asignacion_dietas,
 vec_personal.evidencia_competencias_asignacion_dietas
 FROM PUBLIC,vec_personal_d7_ejecutor,vec_personal_ejecutor,
 vec_dietas_ejecutor,vec_dietas_propietario;
REVOKE ALL ON FUNCTION vec_personal.consultar_competencias_asignacion_dietas_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_personal_d7_ejecutor,vec_personal_ejecutor,
 vec_dietas_ejecutor,vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_competencias_asignacion_dietas_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_personal_d7_ejecutor;
COMMENT ON FUNCTION vec_personal.consultar_competencias_asignacion_dietas_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 IS 'D7b: lectura nominal de competencias actuales de Personal, con consumo AD3 y recibo; sin SELECT Dietas.';
COMMIT;
