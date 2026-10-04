\set ON_ERROR_STOP on
-- Personal26: consulta mínima propia de la pareja histórica Personal16.
-- Orden causal 03/10/2026: Personal16/CA7 -> AD149 -> Personal26.
-- Personal16/CA7 y AD149 nominal son dependencias reales; no usa AD143..148.
-- No consulta vigencia laboral ni inventa un corte. Contexto y autorización
-- actuales se revalidan en la TX SERIALIZABLE de lectura/consumo/recibo.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000026:crn11',0));
DO $pre$ BEGIN
 IF current_user<>'vec_personal_propietario'
    OR to_regclass('vec_personal.proyeccion_empleado_persona_historia') IS NULL
    OR to_regprocedure('vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(text)') IS NULL
    OR to_regprocedure('vec_personal.resolver_empleado_canonico_persona_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_personal.rechazar_mutacion_proyeccion_empleado_v1()') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR NOT has_function_privilege('vec_personal_propietario',
      'vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
    OR to_regclass('vec_personal.recibo_vinculo_propio_crn11') IS NOT NULL
    OR to_regprocedure('vec_personal.consultar_vinculo_propio_historico_crn11_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
  RAISE EXCEPTION 'Personal26: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;

-- Recibo mínimo de acceso. La auditoría común la registra el consumo V3.
-- No se duplica la pareja ni la fuente en otro agregado, ni se crea outbox.
CREATE TABLE vec_personal.recibo_vinculo_propio_crn11 (
 recibo_ref text PRIMARY KEY CHECK(recibo_ref ~ '^vinculocrn11:[0-9a-f-]{36}$'),
 empleado_ref text NOT NULL CHECK(empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
 material_sha256 text NOT NULL CHECK(material_sha256 ~ '^[0-9a-f]{64}$'),
 decision_ref text NOT NULL,
 auditoria_ref text NOT NULL,
 consumo_huella_sha256 text NOT NULL UNIQUE CHECK(consumo_huella_sha256 ~ '^[0-9a-f]{64}$'),
 consultada_en timestamptz(6) NOT NULL CHECK(isfinite(consultada_en))
);
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.recibo_vinculo_propio_crn11
 FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.recibo_vinculo_propio_crn11
 FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_proyeccion_empleado_v1();
ALTER TABLE vec_personal.recibo_vinculo_propio_crn11 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.recibo_vinculo_propio_crn11 FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_interno ON vec_personal.recibo_vinculo_propio_crn11
 FOR ALL TO vec_personal_propietario USING(true) WITH CHECK(true);
REVOKE ALL ON TABLE vec_personal.recibo_vinculo_propio_crn11 FROM PUBLIC,vec_personal_ejecutor;

CREATE FUNCTION vec_personal.consultar_vinculo_propio_historico_crn11_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 m jsonb; d jsonb; c jsonb; ctx jsonb; proyeccion record; fuente record; consumo record;
 empleado text; persona text; material_canon text; material_sha text;
 contexto_canon text; contexto_sha text; ahora timestamptz(6); recibo text;
 k text; vinculos integer;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
    OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
       AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolbypassrls)
    OR NOT pg_has_role(session_user,'vec_personal_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_personal_migrador','MEMBER')
    OR p_material IS NULL OR octet_length(p_material)>4096
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL THEN
  RAISE EXCEPTION 'vínculo CRN11 denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
  m:=p_material::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; ctx:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material de vínculo CRN11 inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(ctx) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'material de vínculo CRN11 inválido' USING ERRCODE='22023';
 END IF;
 IF ARRAY(SELECT jsonb_object_keys(m) ORDER BY 1) IS DISTINCT FROM ARRAY[
   'actor_ref','contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version',
   'empleado_ref','esquema','perfil_ref','perfil_version','persona_ref','persona_version',
   'vinculo_ref','vinculo_version']
    OR m->>'esquema' IS DISTINCT FROM 'vec.personal.vinculo-propio-crn11.consulta.v1'
    OR m->>'empleado_ref' IS NULL OR m->>'empleado_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR m->>'persona_ref' IS NULL OR m->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR m->>'actor_ref' IS DISTINCT FROM m->>'persona_ref'
    OR m->>'cuenta_ref' IS NULL OR m->>'cuenta_ref' !~ '^cta_[A-Za-z0-9_-]{22,128}$'
    OR m->>'perfil_ref' IS NULL OR m->>'perfil_ref' !~ '^prf_[A-Za-z0-9_-]{22,128}$'
    OR m->>'contexto_actor_ref' IS NULL OR m->>'contexto_actor_ref' !~ '^[a-z][A-Za-z0-9_:-]{2,159}$'
    OR m->>'vinculo_ref' IS NULL OR m->>'vinculo_ref' !~ '^pep_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'material de vínculo CRN11 incompatible' USING ERRCODE='42501';
 END IF;
 FOREACH k IN ARRAY ARRAY['contexto_version','cuenta_version','perfil_version','persona_version','vinculo_version'] LOOP
  IF jsonb_typeof(m->k) IS DISTINCT FROM 'number'
     OR m->>k IS NULL OR m->>k !~ '^[1-9][0-9]{0,18}$'
     OR (m->>k)::numeric>9223372036854775807 THEN
   RAISE EXCEPTION 'versión de vínculo CRN11 inválida' USING ERRCODE='22023';
  END IF;
 END LOOP;
 empleado:=m->>'empleado_ref'; persona:=m->>'persona_ref';
 IF m->>'persona_version' IS DISTINCT FROM p_persona_version::text
    OR m->>'perfil_version' IS DISTINCT FROM p_perfil_version::text
    OR persona IS DISTINCT FROM d->>'principal_id'
    OR m->>'perfil_ref' IS DISTINCT FROM d->>'perfil_activo_ref' THEN
  RAISE EXCEPTION 'actor de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 -- Orden exacto de json.Marshal(MaterialVinculoPropioCRN11).
 material_canon:='{"esquema":"vec.personal.vinculo-propio-crn11.consulta.v1","empleado_ref":'||to_jsonb(empleado)::text||
  ',"actor_ref":'||to_jsonb(m->>'actor_ref')::text||',"contexto_actor_ref":'||to_jsonb(m->>'contexto_actor_ref')::text||
  ',"contexto_version":'||(m->>'contexto_version')||',"cuenta_ref":'||to_jsonb(m->>'cuenta_ref')::text||
  ',"cuenta_version":'||(m->>'cuenta_version')||',"perfil_ref":'||to_jsonb(m->>'perfil_ref')::text||
  ',"perfil_version":'||(m->>'perfil_version')||',"persona_ref":'||to_jsonb(persona)::text||
  ',"persona_version":'||(m->>'persona_version')||',"vinculo_ref":'||to_jsonb(m->>'vinculo_ref')::text||
  ',"vinculo_version":'||(m->>'vinculo_version')||'}';
 IF p_material IS DISTINCT FROM material_canon THEN
  RAISE EXCEPTION 'material de vínculo CRN11 no canónico' USING ERRCODE='22023';
 END IF;
 material_sha:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_canon:='{"ambitos":{"empleado_ref":'||to_jsonb(empleado)::text||
  '},"atributos":{"material_sha256":"'||material_sha||'","operacion":"vinculo_propio_historico_crn11"}}';
 contexto_sha:=encode(sha256(convert_to(contexto_canon,'UTF8')),'hex');
 IF d->>'concedida' IS DISTINCT FROM 'true' OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_historico_propio_crn11'
    OR d->>'finalidad' IS DISTINCT FROM 'acreditar_vinculo_historico_propio_crn11'
    OR d->'campos_permitidos' IS DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS DISTINCT FROM empleado
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.vinculo_propio.crn11.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM empleado
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'concesión de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 -- El canon del contexto y su uso vigente los verifica el núcleo V3/CA7.
 -- El material además debe referirse a la entrada exacta del empleado.
 IF ctx->>'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
    OR ctx->>'principal_ref' IS DISTINCT FROM m->>'actor_ref'
    OR ctx->>'persona_ref' IS DISTINCT FROM persona
    OR ctx->>'perfil_activo_ref' IS DISTINCT FROM m->>'perfil_ref'
    OR ctx->>'estado' IS DISTINCT FROM 'activo'
    OR jsonb_typeof(ctx->'vinculos') IS DISTINCT FROM 'array' THEN
  RAISE EXCEPTION 'contexto de vínculo CRN11 incompatible' USING ERRCODE='42501';
 END IF;
 FOREACH k IN ARRAY ARRAY['contexto_actor_ref','contexto_version','cuenta_ref','cuenta_version','persona_version','perfil_version'] LOOP
  IF ctx->k IS DISTINCT FROM m->k THEN
   RAISE EXCEPTION 'contexto de vínculo CRN11 divergente' USING ERRCODE='42501';
  END IF;
 END LOOP;
 SELECT count(*) INTO vinculos FROM jsonb_array_elements(ctx->'vinculos') v(e)
  WHERE v.e->>'tipo'='empleado';
 IF vinculos<>1 OR NOT EXISTS(SELECT 1 FROM jsonb_array_elements(ctx->'vinculos') v(e)
     WHERE v.e->>'tipo'='empleado' AND v.e->>'referencia'=empleado
       AND v.e->>'vinculo_ref'=m->>'vinculo_ref' AND v.e->'version'=m->'vinculo_version'
       AND v.e->>'estado'='activo') THEN
  RAISE EXCEPTION 'contexto de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 PERFORM vec_personal.bloquear_generacion_proyeccion_empleado_persona_v1(persona);
 SELECT * INTO STRICT proyeccion FROM vec_personal.resolver_empleado_canonico_persona_v1(persona,clock_timestamp());
 IF proyeccion.resultado IS DISTINCT FROM 'empleado'
    OR proyeccion.persona_ref IS DISTINCT FROM persona OR proyeccion.empleado_ref IS DISTINCT FROM empleado
    OR proyeccion.proyeccion_ref IS DISTINCT FROM m->>'vinculo_ref'
    OR proyeccion.version::text IS DISTINCT FROM m->>'vinculo_version' THEN
  RAISE EXCEPTION 'vínculo actual CRN11 divergente' USING ERRCODE='42501';
 END IF;
 -- Consumo nuevo: V3 revalida permiso/contexto/revocación antes de la lectura.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR consumo.efecto_ref IS DISTINCT FROM empleado OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'consumo de vínculo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 SELECT h.* INTO STRICT fuente FROM vec_personal.proyeccion_empleado_persona_historia h
  WHERE h.proyeccion_ref=m->>'vinculo_ref' AND h.version=(m->>'vinculo_version')::bigint
    AND h.persona_ref=persona AND h.empleado_ref=empleado;
 IF fuente.procedencia_ref IS DISTINCT FROM proyeccion.procedencia_ref
    OR fuente.procedencia_ref IS NULL OR fuente.procedencia_ref !~ '^prc_[A-Za-z0-9_-]{22,128}$' THEN
  RAISE EXCEPTION 'fuente de vínculo CRN11 divergente' USING ERRCODE='55000';
 END IF;
 ahora:=clock_timestamp();
 IF d->>'valida_hasta' IS NULL OR ahora>=(d->>'valida_hasta')::timestamptz
    OR ahora<proyeccion.vigente_desde OR ahora>=proyeccion.vigente_hasta THEN
  RAISE EXCEPTION 'vínculo CRN11 caducado' USING ERRCODE='42501';
 END IF;
 recibo:='vinculocrn11:'||gen_random_uuid()::text;
 INSERT INTO vec_personal.recibo_vinculo_propio_crn11(
  recibo_ref,empleado_ref,material_sha256,decision_ref,auditoria_ref,consumo_huella_sha256,consultada_en)
 VALUES(recibo,empleado,material_sha,consumo.decision_ref,consumo.auditoria_ref,consumo.consumo_huella_sha256,ahora);
 RETURN jsonb_build_object('vinculo',jsonb_build_object(
  'persona_ref',persona,'empleado_ref',empleado,'vinculo_ref',fuente.proyeccion_ref,
  'fuente_ref',fuente.procedencia_ref,'version',fuente.version),
  'evidencia',jsonb_build_object('recibo_ref',recibo,'decision_ref',consumo.decision_ref,
  'efecto_ref',consumo.efecto_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
  'auditoria_ref',consumo.auditoria_ref,
  'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_personal.consultar_vinculo_propio_historico_crn11_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.consultar_vinculo_propio_historico_crn11_v1(
 text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_ejecutor;
COMMIT;
