\set ON_ERROR_STOP on
-- Fachada mínima de Personal para Dietas. No concede a Dietas SELECT sobre
-- relaciones; la comprobación bloqueante se hace por SECURITY DEFINER.
BEGIN;
SET LOCAL ROLE vec_personal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000007:relacion-dietas:v1',0));

DO $pre$
BEGIN
  IF current_user <> 'vec_personal_propietario'
     OR to_regclass('vec_personal.relacion_empleado_dietas') IS NOT NULL
     OR to_regprocedure('vec_personal.resolver_relacion_dietas_v1(text,text,text,date)') IS NOT NULL
     OR to_regprocedure('vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date)') IS NOT NULL
     OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_propietario'
                      AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls) THEN
    RAISE EXCEPTION 'Personal 000007: precondición incompatible' USING ERRCODE='55000';
  END IF;
END $pre$;

CREATE TABLE vec_personal.relacion_empleado_dietas (
  relacion_ref text PRIMARY KEY CHECK (relacion_ref ~ '^rel_[A-Za-z0-9_-]{22,128}$'),
  persona_ref text NOT NULL CHECK (persona_ref ~ '^per_[A-Za-z0-9_-]{22,128}$'),
  empleado_ref text NOT NULL CHECK (empleado_ref ~ '^emp_[A-Za-z0-9_-]{22,128}$'),
  unidad_ref text NOT NULL CHECK (length(unidad_ref) BETWEEN 1 AND 256),
  estado text NOT NULL CHECK (estado='activa'),
  desde date NOT NULL,
  hasta date,
  version bigint NOT NULL CHECK (version>0),
  procedencia_acto_ref text NOT NULL CHECK (length(procedencia_acto_ref) BETWEEN 1 AND 256),
  fuente_ref text NOT NULL CHECK (length(fuente_ref) BETWEEN 1 AND 256),
  fuente_version bigint NOT NULL CHECK (fuente_version>0),
  registrada_en timestamptz(6) NOT NULL DEFAULT clock_timestamp(),
  CHECK (hasta IS NULL OR desde<hasta),
  UNIQUE(persona_ref,empleado_ref,relacion_ref,version,procedencia_acto_ref,fuente_ref,fuente_version)
);
ALTER TABLE vec_personal.relacion_empleado_dietas ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_personal.relacion_empleado_dietas FORCE ROW LEVEL SECURITY;
CREATE POLICY contexto_dietas_por_persona ON vec_personal.relacion_empleado_dietas FOR ALL
 TO vec_personal_propietario
 USING (current_setting('vec.dietas.persona_ref',true) IS NOT NULL AND persona_ref=current_setting('vec.dietas.persona_ref',true))
 WITH CHECK (current_setting('vec.dietas.persona_ref',true) IS NOT NULL AND persona_ref=current_setting('vec.dietas.persona_ref',true));
CREATE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog AS $$ BEGIN RAISE EXCEPTION 'historia Personal/Dietas inmutable' USING ERRCODE='55000'; END $$;
ALTER FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1() OWNER TO vec_personal_propietario;
CREATE TRIGGER historia_inmutable BEFORE UPDATE OR DELETE ON vec_personal.relacion_empleado_dietas FOR EACH ROW EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON vec_personal.relacion_empleado_dietas FOR EACH STATEMENT EXECUTE FUNCTION vec_personal.rechazar_mutacion_relacion_dietas_v1();

CREATE FUNCTION vec_personal.resolver_relacion_dietas_v1(p_persona text,p_empleado text,p_relacion text,p_fecha date)
RETURNS TABLE(persona_ref text,empleado_ref text,relacion_ref text,unidad_ref text,estado text,desde date,hasta date,version bigint,procedencia_acto_ref text,fuente_ref text,fuente_version bigint)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $$
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER') OR pg_has_role(session_user,'vec_personal_propietario','MEMBER') OR pg_has_role(session_user,'vec_personal_migrador','MEMBER') OR p_persona IS NULL OR p_empleado IS NULL OR p_relacion IS NULL OR p_fecha IS NULL OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$' OR p_relacion !~ '^rel_[A-Za-z0-9_-]{22,128}$' THEN RAISE EXCEPTION 'selector Personal inválido' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',p_persona,true);
 RETURN QUERY SELECT r.persona_ref,r.empleado_ref,r.relacion_ref,r.unidad_ref,r.estado,r.desde,r.hasta,r.version,r.procedencia_acto_ref,r.fuente_ref,r.fuente_version FROM vec_personal.relacion_empleado_dietas r WHERE r.persona_ref=p_persona AND r.empleado_ref=p_empleado AND r.relacion_ref=p_relacion AND r.estado='activa' AND r.desde<=p_fecha AND (r.hasta IS NULL OR p_fecha<r.hasta);
 IF NOT FOUND THEN RAISE EXCEPTION 'relación Personal no disponible' USING ERRCODE='P7201'; END IF;
END $$;

CREATE FUNCTION vec_personal.revalidar_relacion_dietas_v1(p_relacion text,p_persona text,p_empleado text,p_unidad text,p_desde text,p_hasta text,p_version bigint,p_acto text,p_fuente text,p_fuente_version bigint,p_fecha date)
RETURNS boolean LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' AS $$
DECLARE r vec_personal.relacion_empleado_dietas%ROWTYPE;
BEGIN
 IF current_user<>'vec_personal_propietario' OR session_user=current_user OR NOT pg_has_role(session_user,'vec_dietas_ejecutor','MEMBER') OR pg_has_role(session_user,'vec_personal_propietario','MEMBER') OR pg_has_role(session_user,'vec_personal_migrador','MEMBER') OR p_relacion IS NULL OR p_persona IS NULL OR p_empleado IS NULL OR p_unidad IS NULL OR p_desde IS NULL OR p_hasta IS NULL OR p_version IS NULL OR p_acto IS NULL OR p_fuente IS NULL OR p_fuente_version IS NULL OR p_fecha IS NULL OR p_relacion !~ '^rel_[A-Za-z0-9_-]{22,128}$' OR p_persona !~ '^per_[A-Za-z0-9_-]{22,128}$' OR p_empleado !~ '^emp_[A-Za-z0-9_-]{22,128}$' OR p_unidad='' OR p_desde !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' OR (p_hasta<>'' AND p_hasta !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$') OR p_version<1 OR p_acto='' OR p_fuente='' OR p_fuente_version<1 THEN RAISE EXCEPTION 'sello Personal inválido' USING ERRCODE='42501'; END IF;
 PERFORM set_config('vec.dietas.persona_ref',p_persona,true);
 SELECT * INTO r FROM vec_personal.relacion_empleado_dietas WHERE relacion_ref=p_relacion FOR SHARE;
 IF NOT FOUND OR r.persona_ref IS DISTINCT FROM p_persona OR r.empleado_ref IS DISTINCT FROM p_empleado OR r.unidad_ref IS DISTINCT FROM p_unidad OR r.estado IS DISTINCT FROM 'activa' OR r.desde::text IS DISTINCT FROM p_desde OR coalesce(r.hasta::text,'') IS DISTINCT FROM p_hasta OR r.version IS DISTINCT FROM p_version OR r.procedencia_acto_ref IS DISTINCT FROM p_acto OR r.fuente_ref IS DISTINCT FROM p_fuente OR r.fuente_version IS DISTINCT FROM p_fuente_version OR r.desde>p_fecha OR (r.hasta IS NOT NULL AND p_fecha>=r.hasta) THEN RAISE EXCEPTION 'sello Personal no vigente' USING ERRCODE='P7201'; END IF;
 RETURN true;
END $$;

ALTER FUNCTION vec_personal.resolver_relacion_dietas_v1(text,text,text,date) OWNER TO vec_personal_propietario;
ALTER FUNCTION vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date) OWNER TO vec_personal_propietario;
REVOKE ALL ON TABLE vec_personal.relacion_empleado_dietas FROM PUBLIC,vec_personal_ejecutor,vec_dietas_propietario;
-- Resolver es una pieza interna de preparación todavía sin consumidor AD3: no se
-- publica a ningún rol runtime. Dietas sólo puede revalidar dentro de su efecto.
REVOKE ALL ON FUNCTION vec_personal.resolver_relacion_dietas_v1(text,text,text,date),vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date) FROM PUBLIC,vec_personal_ejecutor,vec_dietas_propietario,vec_dietas_ejecutor;
GRANT USAGE ON SCHEMA vec_personal TO vec_dietas_propietario;
REVOKE USAGE ON SCHEMA vec_personal FROM vec_dietas_ejecutor;
GRANT EXECUTE ON FUNCTION vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date) TO vec_dietas_propietario;
COMMENT ON FUNCTION vec_personal.revalidar_relacion_dietas_v1(text,text,text,text,text,text,bigint,text,text,bigint,date) IS 'Fachada nominal bloqueante para Dietas; no concede SELECT de relaciones.';
COMMIT;
