\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:migracion:000019', 0));

-- Preimagen causal: propietario, superficie ACL y protección de las tablas
-- que consume esta migración. No depende del LOGIN o entorno del ensayo.
DO $frontera_preimagen$
DECLARE nombre text;tabla regclass;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_contexto_actor_v1_propietario' AND NOT(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_contexto_actor_v1' AND nspowner=to_regrole('vec_contexto_actor_v1_propietario'))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='vec_contexto_actor_v1' AND (a.grantee=0 OR (a.privilege_type='CREATE' AND a.grantee<>n.nspowner)))
 THEN RAISE EXCEPTION 'ADMIN: namespace o propietario divergente' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['perfil_actual','perfil_versiones','vinculo_contexto_actual','vinculo_contexto_versiones'] LOOP
  tabla:=to_regclass('vec_contexto_actor_v1.'||nombre);
  IF tabla IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=tabla AND relowner=to_regrole('vec_contexto_actor_v1_propietario') AND relrowsecurity=false AND relforcerowsecurity=false)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=tabla AND a.grantee<>c.relowner)
  THEN RAISE EXCEPTION 'ADMIN: tabla o ACL divergente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $frontera_preimagen$;

DO $preimagen$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper)
     OR pg_catalog.to_regrole('vec_contexto_actor_v1_propietario') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.perfil_actual') IS NULL
     OR pg_catalog.to_regclass('vec_contexto_actor_v1.vinculo_contexto_actual') IS NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.listar_perfiles_cuenta_v1(text,text)') IS NOT NULL
     OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.impedir_revivir_perfil_v1()') IS NOT NULL
     OR EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.perfil_actual a
       JOIN vec_contexto_actor_v1.perfil_versiones v USING (perfil_ref, version)
       WHERE v.estado = 'activo' AND EXISTS (
         SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones h
         WHERE h.perfil_ref = a.perfil_ref AND h.estado = 'revocado' AND h.version < a.version))
     OR EXISTS (
       SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
       JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING (vinculo_ref, version)
       WHERE v.estado = 'activo' AND EXISTS (
         SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones h
         WHERE h.vinculo_ref = a.vinculo_ref AND h.estado = 'revocado' AND h.version < a.version))
  THEN RAISE EXCEPTION 'CA19: preimagen incompatible' USING ERRCODE = '55000'; END IF;
END $preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
CREATE FUNCTION vec_contexto_actor_v1.impedir_revivir_perfil_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE nueva_estado text; antigua_estado text;
BEGIN
  IF TG_OP = 'DELETE' THEN
    RAISE EXCEPTION 'puntero de perfil no eliminable' USING ERRCODE = '23514';
  END IF;
  IF TG_TABLE_NAME = 'perfil_actual' THEN
    SELECT estado INTO STRICT nueva_estado FROM vec_contexto_actor_v1.perfil_versiones
      WHERE perfil_ref = NEW.perfil_ref AND version = NEW.version;
    IF TG_OP = 'UPDATE' THEN
      SELECT estado INTO STRICT antigua_estado FROM vec_contexto_actor_v1.perfil_versiones
        WHERE perfil_ref = OLD.perfil_ref AND version = OLD.version;
      IF NEW.perfil_ref IS DISTINCT FROM OLD.perfil_ref OR NEW.version <= OLD.version THEN
        RAISE EXCEPTION 'avance de perfil invalido' USING ERRCODE = '23514';
      END IF;
    END IF;
    IF nueva_estado = 'activo' AND (antigua_estado = 'revocado' OR EXISTS (
      SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones h
      WHERE h.perfil_ref = NEW.perfil_ref AND h.estado = 'revocado' AND h.version < NEW.version)) THEN
      RAISE EXCEPTION 'perfil revocado no revivible' USING ERRCODE = '23514';
    END IF;
  ELSE
    SELECT estado INTO STRICT nueva_estado FROM vec_contexto_actor_v1.vinculo_contexto_versiones
      WHERE vinculo_ref = NEW.vinculo_ref AND version = NEW.version;
    IF TG_OP = 'UPDATE' THEN
      SELECT estado INTO STRICT antigua_estado FROM vec_contexto_actor_v1.vinculo_contexto_versiones
        WHERE vinculo_ref = OLD.vinculo_ref AND version = OLD.version;
      IF NEW.vinculo_ref IS DISTINCT FROM OLD.vinculo_ref OR NEW.version <= OLD.version THEN
        RAISE EXCEPTION 'avance de vinculo invalido' USING ERRCODE = '23514';
      END IF;
    END IF;
    IF nueva_estado = 'activo' AND (antigua_estado = 'revocado' OR EXISTS (
      SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones h
      WHERE h.vinculo_ref = NEW.vinculo_ref AND h.estado = 'revocado' AND h.version < NEW.version)) THEN
      RAISE EXCEPTION 'vinculo revocado no revivible' USING ERRCODE = '23514';
    END IF;
  END IF;
  RETURN NEW;
END $funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.impedir_revivir_perfil_v1() FROM PUBLIC, vec_contexto_actor_v1_runtime;
CREATE TRIGGER perfil_sin_revivir_v1 BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.perfil_actual
FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.impedir_revivir_perfil_v1();
CREATE TRIGGER vinculo_sin_revivir_v1 BEFORE INSERT OR UPDATE OR DELETE ON vec_contexto_actor_v1.vinculo_contexto_actual
FOR EACH ROW EXECUTE FUNCTION vec_contexto_actor_v1.impedir_revivir_perfil_v1();

-- Fachada mínima. Sólo el propietario puede ejecutarla hasta disponer del consumidor
-- atestado y del LOGIN administrativo; no se concede SELECT sobre las tablas.
CREATE FUNCTION vec_contexto_actor_v1.listar_perfiles_cuenta_v1(p_cuenta_ref text, p_persona_ref text)
RETURNS TABLE (perfil_ref text, vinculo_ref text, perfil_version numeric, vinculo_version numeric)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path = pg_catalog, pg_temp AS $funcion$
  SELECT p.perfil_ref, v.vinculo_ref, p.version, v.version
  FROM vec_contexto_actor_v1.vinculo_contexto_actual va
  JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING (vinculo_ref, version)
  JOIN vec_contexto_actor_v1.perfil_actual pa ON pa.perfil_ref = v.perfil_ref
  JOIN vec_contexto_actor_v1.perfil_versiones p ON p.perfil_ref = pa.perfil_ref AND p.version = pa.version
  JOIN vec_contexto_actor_v1.persona_actual pea ON pea.persona_ref = p.persona_ref
  JOIN vec_contexto_actor_v1.persona_versiones pe ON pe.persona_ref = pea.persona_ref AND pe.version = pea.version
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_actual ca ON ca.cuenta_ref = v.cuenta_ref
  JOIN vec_contexto_actor_v1.proyeccion_cuenta_versiones c ON c.cuenta_ref = ca.cuenta_ref AND c.version = ca.version
  WHERE v.cuenta_ref = p_cuenta_ref AND p.persona_ref = p_persona_ref
    AND v.persona_ref = p.persona_ref
    AND c.estado = 'activo' AND pe.estado = 'activo'
    AND p.estado = 'activo' AND v.estado = 'activo'
    AND c.procedencia_autoridad = 'autoridad_maestra_acreditada'
    AND pe.procedencia_autoridad = 'autoridad_maestra_acreditada'
    AND p.procedencia_autoridad = 'autoridad_maestra_acreditada'
    AND v.procedencia_autoridad = 'autoridad_maestra_acreditada'
    AND pg_catalog.clock_timestamp() >= GREATEST(c.vigente_desde, pe.vigente_desde, p.vigente_desde, v.vigente_desde)
    AND pg_catalog.clock_timestamp() < LEAST(c.vigente_hasta, pe.vigente_hasta, p.vigente_hasta, v.vigente_hasta)
  ORDER BY p.perfil_ref, v.vinculo_ref
$funcion$;
REVOKE ALL ON FUNCTION vec_contexto_actor_v1.listar_perfiles_cuenta_v1(text,text) FROM PUBLIC, vec_contexto_actor_v1_runtime;
COMMIT;
