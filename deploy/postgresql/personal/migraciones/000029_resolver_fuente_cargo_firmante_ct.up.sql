\set ON_ERROR_STOP on
-- Personal 29: resolver privado de fuente nominal para el plan de firma CT.
-- Los cinco selectores vienen del plan gobernado y de fuentes acreditadas;
-- ninguna búsqueda por RolID, perfil o nombre concede ejercicio del cargo.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000029:fuente_cargo_ct',0));
DO $pre$
BEGIN
 IF current_user<>'vec_personal_propietario'
  OR to_regclass('vec_personal.cargo_competencial_actual') IS NULL
  OR to_regclass('vec_personal.cargo_competencial_historia') IS NULL
  OR to_regclass('vec_personal.enlace_cargo_competencial_actual') IS NULL
  OR to_regclass('vec_personal.enlace_cargo_competencial_historia') IS NULL
  OR to_regprocedure('vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)') IS NULL
  OR EXISTS(SELECT 1 FROM pg_roles WHERE rolname IN
    ('vec_personal_propietario','vec_autorizacion_propietario') AND rolcanlogin)
  OR NOT has_schema_privilege('vec_autorizacion_propietario','vec_personal','USAGE')
  OR NOT has_function_privilege('vec_autorizacion_propietario',
    'vec_personal.leer_revalidar_cargo_ocupante_ct_v1(bytea)','EXECUTE')
  OR to_regprocedure('vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'Personal 29: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- El llamador AUT interno llega después del consumo V3 en la transacción CT.
-- Se conserva el mismo cerrojo de publicación que Personal28 y sus bloqueos
-- de punteros y versiones hasta COMMIT. La fachada Personal28 vuelve a probar
-- vigencia, órgano, puesto, relación y ocupación antes de entregar la fuente.
CREATE FUNCTION vec_personal.resolver_fuente_cargo_ocupante_ct_v1(
 cargo_ref text,enlace_ref text,persona_ref text,organizacion_ref text,unidad_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
#variable_conflict use_variable
DECLARE c record; e record; t record; v_delegacion jsonb; v_personal jsonb;
 v_contexto jsonb; v_fuente jsonb; v_fecha timestamptz(6);
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR current_setting('TimeZone')<>'UTC'
  OR current_setting('role')<>'none'
  OR NOT pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_personal_propietario','MEMBER')
  OR (cargo_ref ~ '^car_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (enlace_ref ~ '^enc_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$') IS NOT TRUE
  OR (organizacion_ref ~ '^[a-z][a-z0-9_:-]{2,127}$') IS NOT TRUE
  OR (unidad_ref ~ '^[a-z][a-z0-9_:-]{2,127}$') IS NOT TRUE
 THEN RAISE EXCEPTION 'cargo_nominal_resolucion_denegada' USING ERRCODE='42501'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('vec_personal:cargos_competenciales:v1',0));
 LOCK TABLE vec_personal.org_nodo_historia,vec_personal.puesto_rpt_historia IN SHARE MODE;
 SELECT a.huella_sha256 AS puntero_sha256,h.* INTO STRICT c
 FROM vec_personal.cargo_competencial_actual a
 JOIN vec_personal.cargo_competencial_historia h USING(cargo_ref,version)
 WHERE a.cargo_ref=$1 FOR SHARE OF a,h;
 SELECT a.huella_sha256 AS puntero_sha256,h.* INTO STRICT e
 FROM vec_personal.enlace_cargo_competencial_actual a
 JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
 WHERE a.enlace_ref=$2 FOR SHARE OF a,h;
 IF e.clase='titular' THEN
  t:=e;
 ELSE
  IF e.clase NOT IN ('delegacion_competencia','delegacion_firma','suplencia')
   OR e.titular_enlace_ref IS NULL THEN
   RAISE EXCEPTION 'cargo_nominal_ejercicio_invalido' USING ERRCODE='42501'; END IF;
  SELECT a.huella_sha256 AS puntero_sha256,h.* INTO STRICT t
  FROM vec_personal.enlace_cargo_competencial_actual a
  JOIN vec_personal.enlace_cargo_competencial_historia h USING(enlace_ref,version)
  WHERE a.enlace_ref=e.titular_enlace_ref FOR SHARE OF a,h;
 END IF;

 IF c.puntero_sha256 IS DISTINCT FROM c.huella_sha256
  OR e.puntero_sha256 IS DISTINCT FROM e.huella_sha256
  OR t.puntero_sha256 IS DISTINCT FROM t.huella_sha256
  OR c.estado<>'vigente' OR e.estado<>'vigente' OR t.estado<>'vigente'
  OR c.organizacion_ref IS DISTINCT FROM organizacion_ref
  OR c.unidad_ref IS DISTINCT FROM unidad_ref
  OR e.persona_ref IS DISTINCT FROM persona_ref
  OR e.cargo_ref IS DISTINCT FROM cargo_ref OR e.cargo_version IS DISTINCT FROM c.version
  OR t.cargo_ref IS DISTINCT FROM cargo_ref OR t.cargo_version IS DISTINCT FROM c.version
  OR t.clase<>'titular'
  OR (e.clase<>'titular' AND (
   e.titular_enlace_ref IS DISTINCT FROM t.enlace_ref
   OR e.titular_enlace_version IS DISTINCT FROM t.version
   OR e.titular_enlace_sha256 IS DISTINCT FROM t.huella_sha256
   OR e.delegante_persona_ref IS DISTINCT FROM t.persona_ref
   OR e.persona_ref IS NOT DISTINCT FROM t.persona_ref))
  OR e.accion_ref IS DISTINCT FROM t.accion_ref
  OR e.recurso_ref IS DISTINCT FROM t.recurso_ref
  OR e.finalidad_ref IS DISTINCT FROM t.finalidad_ref
 THEN RAISE EXCEPTION 'cargo_nominal_fuente_divergente' USING ERRCODE='42501'; END IF;

 v_delegacion:=NULL;
 IF e.clase<>'titular' THEN
  v_delegacion:=jsonb_build_object(
   'acto',jsonb_build_object('referencia',e.acto_ref,'version',e.acto_version,
     'huella_sha256',e.acto_huella_sha256),
   'delegante_persona_ref',e.delegante_persona_ref,
   'delegado_persona_ref',e.persona_ref,'cargo_ref',e.cargo_ref,
   'vigente_desde',e.vigente_desde,'vigente_hasta',e.vigente_hasta);
 END IF;
 v_personal:=jsonb_build_object(
  'cargo',jsonb_build_object('referencia',c.cargo_ref,'version',c.version,
   'huella_sha256',c.huella_sha256),
  'enlace_ocupante',jsonb_build_object('referencia',t.enlace_ref,'version',t.version,
   'huella_sha256',t.huella_sha256),
  'ocupante_persona_ref',t.persona_ref,'cargo_ref_enlace',t.cargo_ref,
  'cargo_vigente_desde',c.vigente_desde,'cargo_vigente_hasta',c.vigente_hasta,
  'enlace_vigente_desde',t.vigente_desde,'enlace_vigente_hasta',t.vigente_hasta,
  'delegacion',v_delegacion);
 v_fecha:=clock_timestamp();
 v_contexto:=jsonb_build_object('esquema','vec.competencia-firmante.historica.v1',
  'fecha_historica',v_fecha,'personal',v_personal,
  'identidad',jsonb_build_object('persona_ref',persona_ref),
  'recurso',jsonb_build_object('organizacion_ref',organizacion_ref,
   'unidad_ref',unidad_ref,'recurso_autorizable_ref',e.recurso_ref),
  'accion',e.accion_ref,'finalidad',e.finalidad_ref);
 v_fuente:=vec_personal.leer_revalidar_cargo_ocupante_ct_v1(
  convert_to(v_contexto::text,'UTF8'));
 IF v_fuente->>'esquema' IS DISTINCT FROM 'vec.personal.cargo-ocupante.ct.v1'
  OR v_fuente#>>'{cargo,referencia}' IS DISTINCT FROM cargo_ref
  OR v_fuente#>>'{cargo,version}' IS DISTINCT FROM c.version::text
  OR v_fuente#>>'{cargo,huella_sha256}' IS DISTINCT FROM c.huella_sha256
  OR v_fuente#>>'{enlace_ocupante,referencia}' IS DISTINCT FROM t.enlace_ref
  OR v_fuente#>>'{enlace_ocupante,version}' IS DISTINCT FROM t.version::text
  OR v_fuente#>>'{enlace_ocupante,huella_sha256}' IS DISTINCT FROM t.huella_sha256
  OR v_fuente->>'persona_ejerciente_ref' IS DISTINCT FROM persona_ref
  OR v_fuente->>'tipo_ejercicio' IS DISTINCT FROM e.clase
  OR v_fuente->>'organizacion_ref' IS DISTINCT FROM organizacion_ref
  OR v_fuente->>'unidad_ref' IS DISTINCT FROM unidad_ref
  OR v_fuente->'delegacion' IS DISTINCT FROM COALESCE(v_delegacion,'null'::jsonb)
 THEN RAISE EXCEPTION 'cargo_nominal_revalidacion_divergente' USING ERRCODE='42501'; END IF;

 -- Estos atributos son hechos de Personal para cotejo por AUT35. No son una
 -- concesión: la acción, el recurso y la finalidad siguen sujetos a V3 y AUT.
 RETURN v_fuente || jsonb_build_object(
  'accion_ref',e.accion_ref,'recurso_autorizable_ref',e.recurso_ref,
  'finalidad_ref',e.finalidad_ref,
  'cargo_procedencia',jsonb_build_object('acto_ref',c.acto_ref,
   'acto_version',c.acto_version,'acto_huella_sha256',c.acto_huella_sha256,
   'fuente_ref',c.fuente_ref,'fuente_version',c.fuente_version,
   'fuente_huella_sha256',c.fuente_huella_sha256,'recibo_ref',c.recibo_ref),
  'enlace_ejerciente',jsonb_build_object('referencia',e.enlace_ref,
   'version',e.version,'huella_sha256',e.huella_sha256),
  'ejerciente_procedencia',jsonb_build_object('acto_ref',e.acto_ref,
   'acto_version',e.acto_version,'acto_huella_sha256',e.acto_huella_sha256,
   'fuente_ref',e.fuente_ref,'fuente_version',e.fuente_version,
   'fuente_huella_sha256',e.fuente_huella_sha256,'recibo_ref',e.recibo_ref));
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'cargo_nominal_fuente_ausente_o_ambigua' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_personal.resolver_fuente_cargo_ocupante_ct_v1(text,text,text,text,text)
 TO vec_autorizacion_propietario;
COMMIT;
