\set ON_ERROR_STOP on
-- Fachada de lectura propia Personal -> Dietas. Se instala únicamente después
-- de AD45, que consume la capacidad atestada antes de que esta función lea una
-- relación. No publica ni reutiliza el resolutor interno 000007.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000008:consulta-relacion-propia-dietas:v1',0));

DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.recibo_consulta_relacion_propia_dietas') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls) THEN
   RAISE EXCEPTION 'Personal 000008: falta AD45 o estado incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

CREATE TABLE vec_personal.recibo_consulta_relacion_propia_dietas (
 referencia text PRIMARY KEY CHECK(referencia~'^rpd_[0-9a-f]{32}$'),
 decision_ref text NOT NULL,
 efecto_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL CHECK(consumo_huella_sha256~'^[0-9a-f]{64}$'),
 auditoria_ad3_ref text NOT NULL,
 persona_ref text NOT NULL CHECK(persona_ref~'^per_[A-Za-z0-9_-]{22,128}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref~'^emp_[A-Za-z0-9_-]{22,128}$'),
 operacion text NOT NULL CHECK(operacion IN ('lista','detalle')),
 fecha_referencia date NOT NULL,
 relacion_ref text,
 cardinalidad integer NOT NULL CHECK(cardinalidad>=0),
 material_sha256 text NOT NULL CHECK(material_sha256~'^[0-9a-f]{64}$'),
 contexto_sha256 text NOT NULL CHECK(contexto_sha256~'^[0-9a-f]{64}$'),
 registrada_en timestamptz(6) NOT NULL,
 CHECK((operacion='detalle')=(relacion_ref IS NOT NULL))
);
CREATE TABLE vec_personal.evidencia_consulta_relacion_propia_dietas (
 evidencia_ref text PRIMARY KEY CHECK(evidencia_ref~'^epd_[0-9a-f]{32}$'),
 recibo_ref text NOT NULL UNIQUE REFERENCES vec_personal.recibo_consulta_relacion_propia_dietas(referencia),
 tipo text NOT NULL CHECK(tipo='consulta_relacion_propia_dietas_autorizada'),
 registrada_en timestamptz(6) NOT NULL
);
ALTER TABLE vec_personal.recibo_consulta_relacion_propia_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_consulta_relacion_propia_dietas FORCE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evidencia_consulta_relacion_propia_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.evidencia_consulta_relacion_propia_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY consulta_propietaria ON vec_personal.recibo_consulta_relacion_propia_dietas FOR ALL TO vec_personal_propietario
 USING(persona_ref=current_setting('vec.dietas.persona_ref',true) AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL)
 WITH CHECK(persona_ref=current_setting('vec.dietas.persona_ref',true) AND current_setting('vec.dietas.persona_ref',true) IS NOT NULL);
CREATE POLICY evidencia_consulta_propietaria ON vec_personal.evidencia_consulta_relacion_propia_dietas FOR ALL TO vec_personal_propietario
 USING(EXISTS(SELECT 1 FROM vec_personal.recibo_consulta_relacion_propia_dietas r WHERE r.referencia=recibo_ref AND r.persona_ref=current_setting('vec.dietas.persona_ref',true)))
 WITH CHECK(EXISTS(SELECT 1 FROM vec_personal.recibo_consulta_relacion_propia_dietas r WHERE r.referencia=recibo_ref AND r.persona_ref=current_setting('vec.dietas.persona_ref',true)));
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_consulta_relacion_propia_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_consulta_relacion_propia_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.evidencia_consulta_relacion_propia_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.evidencia_consulta_relacion_propia_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();

CREATE FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE(recibo_ref text,decision_ref text,efecto_ref text,consumo_huella_sha256 text,auditoria_ad3_ref text,consultada_en timestamptz,cardinalidad integer,relaciones json)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $fn$
DECLARE m jsonb; i jsonb; c jsonb; d jsonb; x jsonb; v_empleado text; v_persona text; v_fecha date; v_selector text; v_operacion text; v_n integer; v_consumo record; v_recibo text; v_ahora timestamptz(6); v_campos jsonb := '["desde","empleado_ref","estado","fuente_ref","fuente_version","hasta","persona_ref","procedencia_acto_ref","relacion_ref","unidad_ref","version"]'::jsonb; v_material_sha text; v_recurso_sha text; v_amb text; v_atr text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
    OR current_user<>'vec_personal_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER') OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
   RAISE EXCEPTION 'consulta propia Personal rechazada' USING ERRCODE='42501';
 END IF;
 BEGIN m:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; x:=convert_from(p_contexto,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'material de consulta propio inválido' USING ERRCODE='22023'; END; i:=m->'identidad';
 IF jsonb_typeof(m)<>'object' OR ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY['esquema','fecha_referencia','identidad','operacion','relacion_ref']
    OR m->>'esquema'<>'vec.personal.relacion-propia-dietas.v1' OR m->>'operacion' NOT IN ('lista','detalle') OR m->>'fecha_referencia' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
    OR jsonb_typeof(i)<>'object' OR ARRAY(SELECT jsonb_object_keys(i) ORDER BY 1) IS DISTINCT FROM ARRAY['actor_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','empleado_ref','perfil_ref','perfil_version','persona_ref','persona_version']
    OR (m->'relacion_ref' IS NOT NULL AND m->'relacion_ref'<>'null'::jsonb AND (jsonb_typeof(m->'relacion_ref')<>'string' OR m->>'relacion_ref' !~ '^rel_[A-Za-z0-9_-]{22,128}$')) THEN
   RAISE EXCEPTION 'material de consulta propio no canónico' USING ERRCODE='22023';
 END IF;
 v_operacion:=m->>'operacion'; v_selector:=NULLIF(m->>'relacion_ref',''); v_fecha:=(m->>'fecha_referencia')::date;
 IF (v_operacion='detalle' AND v_selector IS NULL) OR (v_operacion='lista' AND v_selector IS NOT NULL) THEN RAISE EXCEPTION 'selector incompatible con operación' USING ERRCODE='22023'; END IF;
 v_persona:=x->>'persona_ref';
 SELECT e.valor->>'referencia' INTO v_empleado FROM jsonb_array_elements(COALESCE(x->'vinculos','[]'::jsonb)) e(valor)
  WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado' AND e.valor->>'estado'='activo';
 v_material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 v_amb:='{"empleado_ref":'||to_jsonb(v_empleado)::text||',"persona_ref":'||to_jsonb(v_persona)::text||'}';
 v_atr:='{"fecha_referencia":'||to_jsonb(v_fecha::text)::text||',"material_sha256":"'||v_material_sha||'","operacion":'||to_jsonb(v_operacion)::text||',"relacion_ref":'||to_jsonb(coalesce(v_selector,'sin_seleccion'))::text||'}';
 v_recurso_sha:=encode(sha256(convert_to('{"ambitos":'||v_amb||',"atributos":'||v_atr||'}','UTF8')),'hex');
 IF x->>'esquema'<>'vec.contexto-actor.vinculado.v2' OR v_persona !~ '^per_[A-Za-z0-9_-]{22,128}$' OR v_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR (SELECT count(*) FROM jsonb_array_elements(COALESCE(x->'vinculos','[]'::jsonb)) e(valor) WHERE jsonb_typeof(e.valor)='object' AND e.valor->>'tipo'='empleado' AND e.valor->>'estado'='activo')<>1
    OR i->>'actor_ref' IS DISTINCT FROM x->>'principal_ref' OR i->>'contexto_actor_ref' IS DISTINCT FROM x->>'contexto_actor_ref' OR i->>'contexto_version' IS DISTINCT FROM x->>'contexto_version' OR i->>'cuenta_ref' IS DISTINCT FROM x->>'cuenta_ref' OR i->>'cuenta_version' IS DISTINCT FROM x->>'cuenta_version' OR i->>'perfil_ref' IS DISTINCT FROM x->>'perfil_activo_ref' OR i->>'persona_ref' IS DISTINCT FROM v_persona OR i->>'empleado_ref' IS DISTINCT FROM v_empleado
    OR i->>'persona_version' IS DISTINCT FROM p_persona_version::text OR i->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR coalesce((x->>'persona_version')::numeric,0)<>p_persona_version OR coalesce((x->>'perfil_version')::numeric,0)<>p_perfil_version
    OR c->>'audiencia_consumo'<>'vec_personal.relacion_propia.consultar_dietas.v1' OR c->>'operacion'<>'personal.relacion.propia.consultar_dietas'
    OR d->>'concedida'<>'true' OR d->>'accion'<>'personal.relacion.propia.consultar_dietas' OR d->>'finalidad'<>'preparar_borrador_dietas' OR d->>'modulo_id'<>'personal' OR d->>'tipo_recurso'<>'relacion_empleado_dietas' OR d->>'recurso_ref' IS DISTINCT FROM v_empleado
    OR d->>'principal_id' IS DISTINCT FROM i->>'actor_ref' OR d->>'perfil_activo_ref' IS DISTINCT FROM i->>'perfil_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM v_campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM v_recurso_sha THEN
   RAISE EXCEPTION 'capacidad Personal/Dietas no corresponde a la consulta' USING ERRCODE='42501';
 END IF;
 PERFORM set_config('vec.dietas.persona_ref',v_persona,true);
 -- AD45 es el único consumidor de capacidad: la lectura no se intenta si no
 -- hay consumo nuevo atestado y ligado al mismo material/contexto/versiones.
 SELECT * INTO STRICT v_consumo FROM vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'consulta propia requiere consumo fresco' USING ERRCODE='P0573'; END IF;
 SELECT count(*) INTO v_n FROM (SELECT 1 FROM vec_personal.relacion_empleado_dietas r WHERE r.persona_ref=v_persona AND r.empleado_ref=v_empleado AND r.estado='activa' AND r.desde<=v_fecha AND (r.hasta IS NULL OR v_fecha<r.hasta) AND (v_selector IS NULL OR r.relacion_ref=v_selector) LIMIT 101) q;
 IF v_n>100 THEN RAISE EXCEPTION 'consulta propia supera límite de 100 relaciones' USING ERRCODE='P7202'; END IF;
 v_ahora:=clock_timestamp(); v_recibo:='rpd_'||substr(encode(sha256(convert_to(v_consumo.consumo_huella_sha256||'|'||v_material_sha||'|'||v_ahora::text,'UTF8')),'hex'),1,32);
 INSERT INTO vec_personal.recibo_consulta_relacion_propia_dietas VALUES(v_recibo,v_consumo.decision_ref,v_consumo.efecto_ref,v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_persona,v_empleado,v_operacion,v_fecha,v_selector,v_n,v_material_sha,encode(sha256(p_contexto),'hex'),v_ahora);
 INSERT INTO vec_personal.evidencia_consulta_relacion_propia_dietas VALUES('epd_'||substr(encode(sha256(convert_to(v_recibo||'|'||v_consumo.consumo_huella_sha256,'UTF8')),'hex'),1,32),v_recibo,'consulta_relacion_propia_dietas_autorizada',v_ahora);
 RETURN QUERY SELECT v_recibo,v_consumo.decision_ref,v_consumo.efecto_ref,v_consumo.consumo_huella_sha256,v_consumo.auditoria_ref,v_ahora,v_n,coalesce((SELECT json_agg(json_build_object('desde',r.desde::text,'empleado_ref',r.empleado_ref,'estado',r.estado,'fuente_ref',r.fuente_ref,'fuente_version',r.fuente_version,'hasta',r.hasta::text,'persona_ref',r.persona_ref,'procedencia_acto_ref',r.procedencia_acto_ref,'relacion_ref',r.relacion_ref,'unidad_ref',r.unidad_ref,'version',r.version) ORDER BY r.desde,r.relacion_ref) FROM vec_personal.relacion_empleado_dietas r WHERE r.persona_ref=v_persona AND r.empleado_ref=v_empleado AND r.estado='activa' AND r.desde<=v_fecha AND (r.hasta IS NULL OR v_fecha<r.hasta) AND (v_selector IS NULL OR r.relacion_ref=v_selector)),'[]'::json);
END $fn$;

ALTER FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_personal_propietario;
REVOKE ALL ON TABLE vec_personal.recibo_consulta_relacion_propia_dietas,vec_personal.evidencia_consulta_relacion_propia_dietas FROM PUBLIC,vec_personal_ejecutor,vec_dietas_propietario,vec_dietas_ejecutor;
REVOKE ALL ON FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_ejecutor,vec_dietas_propietario;
GRANT USAGE ON SCHEMA vec_personal TO vec_dietas_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_ejecutor;
COMMENT ON FUNCTION vec_personal.consultar_relaciones_propias_dietas_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) IS 'Lectura propia atestada Personal->Dietas; AD45 se consume antes de leer y no expone el resolutor 000007.';
COMMIT;
