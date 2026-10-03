\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000022', 0));

-- Preimagen causal: propietario, superficie ACL y protección de las tablas
-- que consume esta migración. No depende del LOGIN o entorno del ensayo.
DO $frontera_preimagen$
DECLARE nombre text;tabla regclass;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_propietario' AND NOT(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='vec_autorizacion' AND nspowner=to_regrole('vec_autorizacion_propietario'))
 OR EXISTS(SELECT 1 FROM pg_namespace n CROSS JOIN LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a WHERE n.nspname='vec_autorizacion' AND (a.grantee=0 OR (a.privilege_type='CREATE' AND a.grantee<>n.nspowner)))
 THEN RAISE EXCEPTION 'ADMIN: namespace o propietario divergente' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['version_rol','asignacion_perfil','asignacion_perfil_actual','control_vigencia_version_rol','control_vigencia_version_rol_actual'] LOOP
  tabla:=to_regclass('vec_autorizacion.'||nombre);
  IF tabla IS NULL OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=tabla AND relowner=to_regrole('vec_autorizacion_propietario') AND relrowsecurity=true AND relforcerowsecurity=true)
  OR EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=tabla AND a.grantee<>c.relowner)
  THEN RAISE EXCEPTION 'ADMIN: tabla o ACL divergente %',nombre USING ERRCODE='55000'; END IF;
 END LOOP;
END $frontera_preimagen$;
DO $preimagen$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = current_user AND rolsuper)
     OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
     OR pg_catalog.to_regclass('vec_autorizacion.version_rol') IS NULL
     OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_actual') IS NULL
     OR pg_catalog.to_regprocedure('vec_autorizacion.impedir_revivir_asignacion_v1()') IS NOT NULL
     OR EXISTS (SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id = 'administracion_perfiles')
     OR EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual a
       JOIN vec_autorizacion.asignacion_perfil v ON v.asignacion_ref = a.asignacion_ref
       WHERE v.documento->>'estado' = 'activa' AND EXISTS (
         SELECT 1 FROM vec_autorizacion.asignacion_perfil h
         WHERE h.asignacion_id = v.asignacion_id AND h.version < v.version
           AND h.documento->>'estado' = 'revocada'))
  THEN RAISE EXCEPTION 'AUT22: preimagen incompatible' USING ERRCODE = '55000'; END IF;
END $preimagen$;
SET LOCAL ROLE vec_autorizacion_propietario;

CREATE FUNCTION vec_autorizacion.impedir_revivir_asignacion_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER
SET search_path = pg_catalog, pg_temp AS $funcion$
DECLARE nueva vec_autorizacion.asignacion_perfil%ROWTYPE;
        antigua vec_autorizacion.asignacion_perfil%ROWTYPE;
BEGIN
  SELECT * INTO STRICT nueva FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref = NEW.asignacion_ref;
  IF TG_OP = 'UPDATE' THEN
    SELECT * INTO STRICT antigua FROM vec_autorizacion.asignacion_perfil
    WHERE asignacion_ref = OLD.asignacion_ref;
    IF NEW.perfil_activo_ref IS DISTINCT FROM OLD.perfil_activo_ref
       OR nueva.asignacion_id IS DISTINCT FROM antigua.asignacion_id
       OR nueva.principal_id IS DISTINCT FROM antigua.principal_id
       OR nueva.version <= antigua.version THEN
      RAISE EXCEPTION 'avance de asignacion invalido' USING ERRCODE = '23514';
    END IF;
  END IF;
  IF nueva.documento->>'estado' = 'activa'
     AND (antigua.documento->>'estado' = 'revocada' OR EXISTS (
       SELECT 1 FROM vec_autorizacion.asignacion_perfil h
       WHERE h.asignacion_id = nueva.asignacion_id AND h.version < nueva.version
         AND h.documento->>'estado' = 'revocada')) THEN
    RAISE EXCEPTION 'asignacion revocada no revivible' USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion.impedir_revivir_asignacion_v1() FROM PUBLIC;
CREATE TRIGGER asignacion_sin_revivir_v1 BEFORE INSERT OR UPDATE ON vec_autorizacion.asignacion_perfil_actual
FOR EACH ROW EXECUTE FUNCTION vec_autorizacion.impedir_revivir_asignacion_v1();

-- Versión fija con una sola concesión de lectura mínima. Las operaciones de
-- escritura esperan sus actos, doble control y consumidor V3 en otro corte.
DO $rol$
DECLARE instante text;
        rol jsonb;
        control jsonb;
        rol_bytes text;
        control_bytes text;
        rol_ref constant text := 'rol:administracion_perfiles:v1';
BEGIN
  instante := pg_catalog.to_char(pg_catalog.date_trunc('second',pg_catalog.clock_timestamp()) AT TIME ZONE 'UTC',
                                 'YYYY-MM-DD"T"HH24:MI:SS"Z"');
  -- Orden y omisión de campos opcionales idénticos a VersionRol y
  -- ControlVigenciaVersionRol de domain/autorizacion.go (json.Marshal).
  rol_bytes := pg_catalog.format('{"rol_id":"administracion_perfiles","version":1,"nombre":"administracion.perfiles.rol","estado":"publicada","concesiones":[{"accion":"administracion.perfiles.consultar","modulo_id":"administracion","tipo_recurso":"perfil","finalidades":["gestion_perfiles"],"garantia_minima":"alto","campos_permitidos":["perfil_ref","vinculo_ref","version"]}],"publicada_por":"migracion:autorizacion:000022","publicada_en":"%s","retirada_en":"0001-01-01T00:00:00Z"}', instante);
  rol := rol_bytes::jsonb;
  IF NOT vec_autorizacion.concesiones_positivas_validas(rol) THEN
    RAISE EXCEPTION 'AUT22: rol no válido' USING ERRCODE = '23514';
  END IF;
  INSERT INTO vec_autorizacion.version_rol
    (version_rol_ref,rol_id,version,huella_sha256,publicada_en,documento)
  VALUES (rol_ref,'administracion_perfiles',1,
    pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(rol_bytes,'UTF8')),'hex'),
    instante::timestamptz,rol);
  control_bytes := pg_catalog.format('{"version_rol_ref":"%s","revision":1,"estado":"habilitada","actualizado_por":"migracion:autorizacion:000022","actualizado_en":"%s"}',rol_ref,instante);
  control := control_bytes::jsonb;
  INSERT INTO vec_autorizacion.control_vigencia_version_rol
    (version_rol_ref,revision,estado,huella_sha256,actualizado_en,documento)
  VALUES (rol_ref,1,'habilitada',
    pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(control_bytes,'UTF8')),'hex'),
    instante::timestamptz,control);
  INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
    (version_rol_ref,revision,actualizada_en,actualizada_por,acto_ref)
  VALUES (rol_ref,1,instante::timestamptz,'migracion:autorizacion:000022','migracion:autorizacion:000022');
END $rol$;
COMMIT;
